package livecontrol_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/livecontrol"
)

func TestBreakGlassNeedsASevereIncidentWithAnId(t *testing.T) {
	cases := []struct {
		name     string
		incident livecontrol.Incident
		valid    bool
	}{
		{"sev1", livecontrol.Incident{Severity: "SEV1", ID: "INC-12"}, true},
		{"sev2", livecontrol.Incident{Severity: "SEV2", ID: "INC-12"}, true},
		{"longest id", livecontrol.Incident{Severity: "SEV1", ID: strings.Repeat("a", 128)}, true},
		{"sev3", livecontrol.Incident{Severity: "SEV3", ID: "INC-12"}, false},
		{"lower case severity", livecontrol.Incident{Severity: "sev1", ID: "INC-12"}, false},
		{"no severity", livecontrol.Incident{ID: "INC-12"}, false},
		{"no id", livecontrol.Incident{Severity: "SEV1"}, false},
		{"blank id", livecontrol.Incident{Severity: "SEV1", ID: "   "}, false},
		{"padded id", livecontrol.Incident{Severity: "SEV1", ID: " INC-12"}, false},
		{"control character", livecontrol.Incident{Severity: "SEV1", ID: "INC-12\n"}, false},
		{"too long", livecontrol.Incident{Severity: "SEV1", ID: strings.Repeat("a", 129)}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.incident.Validate()
			if c.valid && err != nil {
				t.Fatalf("refused: %v", err)
			}
			if !c.valid && !errors.Is(err, livecontrol.ErrIncident) {
				t.Fatalf("got %v, want ErrIncident", err)
			}
		})
	}
}

func TestParseChecksTheControls(t *testing.T) {
	cases := []struct {
		name  string
		raw   string
		valid bool
	}{
		{"version only", `{"minimum_app_version":"1.4.0"}`, true},
		{"enforced allowlist", `{"minimum_app_version":"1.4.0","build_allowlist_enforced":true,"build_allowlist":["abc"]}`, true},
		{"listed but not enforced", `{"minimum_app_version":"1.4.0","build_allowlist":["abc"]}`, true},
		{"both kill switches", `{"minimum_app_version":"1.4.0","purchases_disabled":true,"ranked_disabled":true}`, true},
		{"no version", `{}`, false},
		{"bad version", `{"minimum_app_version":"latest"}`, false},
		{"enforced and empty", `{"minimum_app_version":"1.4.0","build_allowlist_enforced":true}`, false},
		{"empty hash", `{"minimum_app_version":"1.4.0","build_allowlist":[""]}`, false},
		{"unknown field", `{"minimum_app_version":"1.4.0","purchases_off":true}`, false},
		{"not json", `{`, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := livecontrol.Parse([]byte(c.raw))
			if c.valid && err != nil {
				t.Fatalf("refused: %v", err)
			}
			if !c.valid && !errors.Is(err, livecontrol.ErrInvalid) {
				t.Fatalf("got %v, want ErrInvalid", err)
			}
		})
	}
}

func TestMaintenanceCarriesItsEndToTheAdmission(t *testing.T) {
	controls, err := livecontrol.Parse([]byte(`{"minimum_app_version":"1.0.0","maintenance_ends_at":"2026-11-09T06:00:00Z"}`))
	if err != nil {
		t.Fatal(err)
	}
	if admission := controls.Admission(); !admission.Maintenance() || !admission.MaintenanceEndsAt.Equal(time.Date(2026, 11, 9, 6, 0, 0, 0, time.UTC)) {
		t.Fatalf("admission %+v", admission)
	}
	if off, _ := livecontrol.Parse([]byte(`{"minimum_app_version":"1.0.0","maintenance_ends_at":null}`)); off.Admission().Maintenance() {
		t.Fatal("a cleared end still means maintenance")
	}
}

func TestAdmissionCarriesTheVersionAndTheAllowlist(t *testing.T) {
	controls, err := livecontrol.Parse([]byte(`{"minimum_app_version":"1.4.0","build_allowlist_enforced":true,"build_allowlist":["abc"]}`))
	if err != nil {
		t.Fatal(err)
	}
	admission := controls.Admission()
	if !admission.Admits("1.4.0", "abc") || admission.Admits("1.3.9", "abc") || admission.Admits("1.4.0", "leaked") {
		t.Fatalf("admission %+v", admission)
	}
}

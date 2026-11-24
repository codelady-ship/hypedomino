package adminapi_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

func TestTheLiveopsKeysArePublishedRolledBackAndAudited(t *testing.T) {
	a := newAPI(t)
	a.create("leyla@example.com", rbac.LiveOps)
	liveOps := a.signIn("leyla")
	path := "/admin/config/liveops"

	status, body := a.call("GET", path, liveOps, nil)
	keys := body["keys"].([]any)
	frames := keys[2].(map[string]any)
	if value, set := ruleValue(t, body, "ContentGateMinFrames"); status != http.StatusOK || body["config_version"] != float64(0) || len(keys) != 14 || value != 20 || set {
		t.Fatalf("defaults %d %v", status, body)
	}
	if _, bounded := frames["max"]; bounded || frames["min"] != float64(0) || frames["default"] != float64(20) || frames["type"] != "integer" {
		t.Fatalf("frames key %v", frames)
	}
	batch := keys[6].(map[string]any)
	if batch["key"] != "AnalyticsBatchSize" || batch["min"] != float64(1) || batch["max"] != float64(400) || batch["default"] != float64(50) || batch["unit"] != "events" {
		t.Fatalf("analytics batch key %v", batch)
	}
	appeal := keys[11].(map[string]any)
	if appeal["key"] != "AppealSlaHours" || appeal["min"] != float64(1) || appeal["max"] != float64(720) || appeal["default"] != float64(72) || appeal["unit"] != "hours" {
		t.Fatalf("appeal sla key %v", appeal)
	}
	status, body = a.call("PUT", path, liveOps, map[string]any{"values": map[string]int{"ContentGateMinFrames": 25}, "reason": "more frames at launch"})
	if value, set := ruleValue(t, body, "ContentGateMinFrames"); status != http.StatusOK || body["config_version"] != float64(1) || value != 25 || !set {
		t.Fatalf("publish %d %v", status, body)
	}
	status, body = a.call("PUT", path, liveOps, map[string]any{"values": map[string]int{"ContentGateMinFrames": 0}, "reason": "waived"})
	expect(t, status, body, http.StatusOK, "")

	a.now = start.Add(11 * time.Minute)
	status, body = a.call("POST", path+"/rollback", liveOps, map[string]any{"version": 1, "reason": "waiver withdrawn"})
	expect(t, status, body, http.StatusForbidden, "step_up_required")
	a.call("POST", "/admin/session/step-up", liveOps, map[string]string{"id_token": "leyla", "nonce": "nonce"})
	status, body = a.call("POST", path+"/rollback", liveOps, map[string]any{"version": 1, "reason": "waiver withdrawn"})
	if value, _ := ruleValue(t, body, "ContentGateMinFrames"); status != http.StatusOK || body["config_version"] != float64(3) || value != 25 {
		t.Fatalf("rollback %d %v", status, body)
	}

	status, body = a.call("GET", path+"/history", liveOps, nil)
	versions := body["versions"].([]any)
	if status != http.StatusOK || len(versions) != 3 || versions[1].(map[string]any)["values"].(map[string]any)["ContentGateMinFrames"] != float64(0) {
		t.Fatalf("history %d %v", status, body)
	}
	entries := a.entries(liveOps, "object_type=game_config&object_id=liveops")
	if len(entries) != 3 || entries[0]["action"] != "rollback" || entries[0]["reason"] != "waiver withdrawn" || entries[2]["reason"] != "more frames at launch" {
		t.Fatalf("entries %v", entries)
	}
}

func TestTheLiveopsKeysAreWholeNumbersFromZeroAndFollowTheRoleMatrix(t *testing.T) {
	a := newAPI(t)
	a.create("leyla@example.com", rbac.LiveOps)
	a.create("orxan@example.com", rbac.Support)
	liveOps, content, support := a.signIn("leyla"), a.signIn("ayla"), a.signIn("orxan")
	path := "/admin/config/liveops"
	tests := []struct {
		name   string
		values map[string]any
	}{
		{"negative", map[string]any{"ContentGateMinEmoji": -1}},
		{"fraction", map[string]any{"ContentGateMinEmoji": 1.5}},
		{"string", map[string]any{"ContentGateMinEmoji": "40"}},
		{"unknown key", map[string]any{"ContentGateMinChests": 3}},
		{"rules key", map[string]any{"TurnSeconds": 20}},
		{"analytics batch above the body limit", map[string]any{"AnalyticsBatchSize": 401}},
		{"analytics flush below its minimum", map[string]any{"AnalyticsFlushSeconds": 9}},
		{"analytics queue below the batch", map[string]any{"AnalyticsBatchSize": 100, "AnalyticsQueueLimit": 99}},
		{"review sla of zero", map[string]any{"ReviewSlaHours": 0}},
		{"report sla above a month", map[string]any{"ReportSlaHours": 721}},
	}
	for _, test := range tests {
		status, body := a.call("PUT", path, liveOps, map[string]any{"values": test.values, "reason": "tune"})
		if status != http.StatusBadRequest || body["error"] != "invalid" || body["detail"] == "" {
			t.Fatalf("%s: %d %v", test.name, status, body)
		}
	}
	for _, token := range []string{content, support} {
		status, body := a.call("PUT", path, token, map[string]any{"values": map[string]int{"ContentGateMinEmoji": 41}, "reason": "tune"})
		expect(t, status, body, http.StatusForbidden, "forbidden")
		status, body = a.call("GET", path, token, nil)
		expect(t, status, body, http.StatusOK, "")
	}
	if entries := a.entries(a.signIn("root"), "object_type=game_config"); len(entries) != 0 {
		t.Fatalf("refused publishes left entries: %v", entries)
	}
}

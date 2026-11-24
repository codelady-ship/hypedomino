package adminapi_test

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

func (a *api) publishTwoEnglishBundles(token string) {
	a.t.Helper()
	a.call("PUT", "/admin/strings/match.result.win", token, winKey)
	for _, text := range []string{"Victory!", "You win!"} {
		a.call("PUT", "/admin/strings/match.result.win/values/en", token, map[string]any{"text": text, "status": "approved"})
		if status, body := a.call("POST", "/admin/strings/bundles/en", token, map[string]any{}); status != http.StatusOK {
			a.t.Fatalf("publish en: %d %v", status, body)
		}
	}
	a.call("PUT", "/admin/strings/match.result.win/values/az", token, map[string]any{"text": "Qələbə!", "status": "approved"})
	if status, body := a.call("POST", "/admin/strings/bundles/az", token, map[string]any{}); status != http.StatusOK {
		a.t.Fatalf("publish az: %d %v", status, body)
	}
}

func bundlesOf(body map[string]any) map[string]any {
	return body["bundles"].(map[string]any)
}

func TestActiveVersionsArePublishedAndRolledBackWithAReasonAndAFreshSignIn(t *testing.T) {
	a := newAPI(t)
	content := a.signIn("ayla")
	a.publishTwoEnglishBundles(content)

	status, body := a.call("GET", "/admin/strings/active", content, nil)
	chains := body["fallback_chains"].(map[string]any)
	if status != http.StatusOK || body["config_version"] != float64(0) || len(bundlesOf(body)) != 0 || !reflect.DeepEqual(chains["az"], []any{"az", "tr", "en"}) {
		t.Fatalf("nothing active %d %v", status, body)
	}
	status, body = a.call("PUT", "/admin/strings/active", content, map[string]any{"bundles": map[string]int{"en": 1}})
	if status != http.StatusOK || body["config_version"] != float64(1) || bundlesOf(body)["en"] != float64(1) {
		t.Fatalf("first publish %d %v", status, body)
	}
	status, body = a.call("PUT", "/admin/strings/active", content, map[string]any{
		"bundles": map[string]int{"en": 2, "az": 1}, "fallback_chains": map[string][]string{"az": {"az", "en"}},
	})
	if status != http.StatusOK || body["config_version"] != float64(2) || !reflect.DeepEqual(body["fallback_chains"].(map[string]any)["az"], []any{"az", "en"}) {
		t.Fatalf("second publish %d %v", status, body)
	}

	a.now = start.Add(5 * time.Minute)
	for _, bundles := range []map[string]int{{"en": 1, "az": 1}, {"en": 2}} {
		status, body = a.call("PUT", "/admin/strings/active", content, map[string]any{"bundles": bundles})
		expect(t, status, body, http.StatusBadRequest, "reason_required")
	}
	a.now = start.Add(11 * time.Minute)
	status, body = a.call("PUT", "/admin/strings/active", content, map[string]any{"bundles": map[string]int{"en": 1, "az": 1}, "reason": "bad wording"})
	expect(t, status, body, http.StatusForbidden, "step_up_required")
	status, body = a.call("POST", "/admin/session/step-up", content, map[string]string{"id_token": "ayla", "nonce": "nonce"})
	expect(t, status, body, http.StatusOK, "")
	status, body = a.call("PUT", "/admin/strings/active", content, map[string]any{"bundles": map[string]int{"en": 1, "az": 1}, "reason": "bad wording", "ref": "INC-7"})
	if status != http.StatusOK || body["config_version"] != float64(3) || bundlesOf(body)["en"] != float64(1) {
		t.Fatalf("rollback %d %v", status, body)
	}
	status, body = a.call("GET", "/admin/strings/active", content, nil)
	if status != http.StatusOK || body["config_version"] != float64(3) || bundlesOf(body)["en"] != float64(1) {
		t.Fatalf("active after rollback %d %v", status, body)
	}

	entries := a.entries(content, "object_type=snapshots&object_id=localization")
	if len(entries) != 3 {
		t.Fatalf("entries %v", entries)
	}
	rollback, second, first := entries[0], entries[1], entries[2]
	if rollback["action"] != "rollback" || rollback["reason"] != "bad wording" || rollback["ref"] != "INC-7" || rollback["config_version"] != float64(3) ||
		rollback["before"].(map[string]any)["bundles"].(map[string]any)["en"] != float64(2) ||
		rollback["after"].(map[string]any)["bundles"].(map[string]any)["en"] != float64(1) {
		t.Fatalf("rollback entry %v", rollback)
	}
	if second["action"] != "publish" || second["config_version"] != float64(2) || first["action"] != "publish" || first["before"] != nil {
		t.Fatalf("publish entries %v %v", second, first)
	}
}

func TestActiveVersionsMustResolveAndNeedPublish(t *testing.T) {
	a := newAPI(t)
	a.create("leyla@example.com", rbac.LiveOps)
	a.create("orxan@example.com", rbac.Support)
	content := a.signIn("ayla")
	a.publishTwoEnglishBundles(content)
	tests := map[string]struct {
		body   map[string]any
		status int
		error  string
	}{
		"unpublished version": {map[string]any{"bundles": map[string]int{"en": 5}}, http.StatusBadRequest, "invalid"},
		"no en bundle":        {map[string]any{"bundles": map[string]int{"az": 1}}, http.StatusBadRequest, "invalid"},
		"unknown language":    {map[string]any{"bundles": map[string]int{"en": 1, "de": 1}}, http.StatusBadRequest, "invalid"},
		"chain without en":    {map[string]any{"bundles": map[string]int{"en": 1}, "fallback_chains": map[string][]string{"az": {"az", "tr"}}}, http.StatusBadRequest, "invalid"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			status, body := a.call("PUT", "/admin/strings/active", content, test.body)
			expect(t, status, body, test.status, test.error)
		})
	}
	status, body := a.call("PUT", "/admin/strings/active", a.signIn("leyla"), map[string]any{"bundles": map[string]int{"en": 1}})
	expect(t, status, body, http.StatusForbidden, "forbidden")
	status, body = a.call("GET", "/admin/strings/active", a.signIn("orxan"), nil)
	expect(t, status, body, http.StatusOK, "")
	if entries := a.entries(content, "object_type=snapshots"); len(entries) != 0 {
		t.Fatalf("refused activations left entries: %v", entries)
	}
}

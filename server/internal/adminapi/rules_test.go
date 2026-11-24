package adminapi_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
	"github.com/Alievs-corp/hypedomino/server/internal/rulesconfig"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
)

func ruleValue(t *testing.T, body map[string]any, name string) (float64, bool) {
	t.Helper()
	for _, key := range body["keys"].([]any) {
		if view := key.(map[string]any); view["key"] == name {
			return view["value"].(float64), view["set"].(bool)
		}
	}
	t.Fatalf("key %s missing from %v", name, body)
	return 0, false
}

func (a *api) serverTurnSeconds() int {
	a.t.Helper()
	watcher := snapshot.New(a.pool, rulesconfig.Domain, rulesconfig.Parse, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if _, err := watcher.Load(context.Background()); err != nil && !errors.Is(err, snapshot.ErrNone) {
		a.t.Fatal(err)
	}
	tuning, _, _ := watcher.Current()
	return tuning.Apply(rules.DefaultConfig()).TurnSeconds
}

func TestAPublishedTurnTimeReachesTheServerAndCanBeRolledBack(t *testing.T) {
	a := newAPI(t)
	a.create("leyla@example.com", rbac.LiveOps)
	liveOps := a.signIn("leyla")

	status, body := a.call("GET", "/admin/config/rules", liveOps, nil)
	if value, set := ruleValue(t, body, "TurnSeconds"); status != http.StatusOK || body["config_version"] != float64(0) || value != 20 || set {
		t.Fatalf("defaults %d %v", status, body)
	}
	status, body = a.call("PUT", "/admin/config/rules", liveOps, map[string]any{"values": map[string]int{"TurnSeconds": 25}, "reason": "faster matches"})
	if value, set := ruleValue(t, body, "TurnSeconds"); status != http.StatusOK || body["config_version"] != float64(1) || value != 25 || !set {
		t.Fatalf("publish %d %v", status, body)
	}
	if got := a.serverTurnSeconds(); got != 25 {
		t.Fatalf("server turn seconds %d after publish", got)
	}
	status, body = a.call("PUT", "/admin/config/rules", liveOps, map[string]any{"values": map[string]int{"TurnSeconds": 30, "MatchLoadingSeconds": 15}})
	expect(t, status, body, http.StatusOK, "")

	a.now = start.Add(5 * time.Minute)
	status, body = a.call("POST", "/admin/config/rules/rollback", liveOps, map[string]any{"version": 1})
	expect(t, status, body, http.StatusBadRequest, "reason_required")
	a.now = start.Add(11 * time.Minute)
	status, body = a.call("POST", "/admin/config/rules/rollback", liveOps, map[string]any{"version": 1, "reason": "too slow"})
	expect(t, status, body, http.StatusForbidden, "step_up_required")
	a.call("POST", "/admin/session/step-up", liveOps, map[string]string{"id_token": "leyla", "nonce": "nonce"})
	status, body = a.call("POST", "/admin/config/rules/rollback", liveOps, map[string]any{"version": 99, "reason": "too slow"})
	expect(t, status, body, http.StatusNotFound, "not_found")
	status, body = a.call("POST", "/admin/config/rules/rollback", liveOps, map[string]any{"version": 1, "reason": "too slow"})
	loading, loadingSet := ruleValue(t, body, "MatchLoadingSeconds")
	if value, _ := ruleValue(t, body, "TurnSeconds"); status != http.StatusOK || body["config_version"] != float64(3) || value != 25 || loading != 10 || loadingSet {
		t.Fatalf("rollback %d %v", status, body)
	}
	if got := a.serverTurnSeconds(); got != 25 {
		t.Fatalf("server turn seconds %d after rollback", got)
	}

	status, body = a.call("GET", "/admin/config/rules/history", liveOps, nil)
	versions := body["versions"].([]any)
	if status != http.StatusOK || len(versions) != 3 || versions[0].(map[string]any)["config_version"] != float64(3) ||
		versions[1].(map[string]any)["values"].(map[string]any)["MatchLoadingSeconds"] != float64(15) {
		t.Fatalf("history %d %v", status, body)
	}
	entries := a.entries(liveOps, "object_type=game_config&object_id=rules")
	if len(entries) != 3 {
		t.Fatalf("entries %v", entries)
	}
	rollback, second, first := entries[0], entries[1], entries[2]
	if rollback["action"] != "rollback" || rollback["reason"] != "too slow" || rollback["config_version"] != float64(3) ||
		rollback["before"].(map[string]any)["TurnSeconds"] != float64(30) || rollback["after"].(map[string]any)["TurnSeconds"] != float64(25) {
		t.Fatalf("rollback entry %v", rollback)
	}
	if second["action"] != "publish" || first["before"] != nil || first["reason"] != "faster matches" {
		t.Fatalf("publish entries %v %v", second, first)
	}
}

func TestRulesValuesStayInsideTheirBoundsAndFollowTheRoleMatrix(t *testing.T) {
	a := newAPI(t)
	a.create("leyla@example.com", rbac.LiveOps)
	a.create("orxan@example.com", rbac.Support)
	liveOps, content, support := a.signIn("leyla"), a.signIn("ayla"), a.signIn("orxan")
	for name, values := range map[string]any{
		"below the minimum": map[string]any{"TurnSeconds": 5},
		"above the maximum": map[string]any{"AfkConsecutiveTimeouts": 11},
		"not an integer":    map[string]any{"TurnSeconds": "25"},
		"a fraction":        map[string]any{"TurnSeconds": 25.5},
		"an unknown key":    map[string]any{"HandSize": 9},
	} {
		status, body := a.call("PUT", "/admin/config/rules", liveOps, map[string]any{"values": values})
		if status != http.StatusBadRequest || body["error"] != "invalid" || body["detail"] == "" {
			t.Fatalf("%s: %d %v", name, status, body)
		}
	}
	for _, token := range []string{content, support} {
		status, body := a.call("PUT", "/admin/config/rules", token, map[string]any{"values": map[string]int{"TurnSeconds": 25}})
		expect(t, status, body, http.StatusForbidden, "forbidden")
		status, body = a.call("GET", "/admin/config/rules", token, nil)
		expect(t, status, body, http.StatusOK, "")
	}
	if entries := a.entries(a.signIn("root"), "object_type=game_config"); len(entries) != 0 {
		t.Fatalf("refused publishes left entries: %v", entries)
	}
	if got := a.serverTurnSeconds(); got != rules.DefaultConfig().TurnSeconds {
		t.Fatalf("server turn seconds %d", got)
	}
}

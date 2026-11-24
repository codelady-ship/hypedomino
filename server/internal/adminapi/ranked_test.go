package adminapi_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/gamestrings"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

func stamp(at time.Time) string {
	return at.UTC().Format(time.RFC3339)
}

func (a *api) seasonWindows(token string) map[string][2]string {
	a.t.Helper()
	status, body := a.call("GET", "/admin/ranked/seasons", token, nil)
	if status != http.StatusOK {
		a.t.Fatalf("seasons %d %v", status, body)
	}
	windows := map[string][2]string{}
	for _, raw := range body["seasons"].([]any) {
		season := raw.(map[string]any)
		windows[season["id"].(string)] = [2]string{season["start"].(string), season["end"].(string)}
	}
	return windows
}

func expectWindows(t *testing.T, got map[string][2]string, want map[string][2]time.Time) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("seasons %v, want %v", got, want)
	}
	for id, window := range want {
		if got[id] != [2]string{stamp(window[0]), stamp(window[1])} {
			t.Fatalf("%s is %v, want %v", id, got[id], window)
		}
	}
}

func TestRankedSeasonsAreScheduledEditedExtendedAndRemovedWithAudit(t *testing.T) {
	a := newAPI(t)
	a.create("leyla@example.com", rbac.LiveOps)
	a.create("orxan@example.com", rbac.Support)
	liveOps, support, content := a.signIn("leyla"), a.signIn("orxan"), a.signIn("ayla")
	day := 24 * time.Hour
	oneStart := start.Add(time.Hour)
	oneEnd := oneStart.Add(30 * day)

	status, body := a.call("GET", "/admin/ranked/seasons", support, nil)
	if status != http.StatusOK || body["next_start"] != nil || len(body["seasons"].([]any)) != 0 || body["server_time"] != stamp(start) {
		t.Fatalf("empty %d %v", status, body)
	}
	status, body = a.call("GET", "/admin/ranked/seasons", content, nil)
	expect(t, status, body, http.StatusForbidden, "forbidden")
	one := map[string]any{"id": "season.one", "start": oneStart, "end": oneEnd, "reason": "launch"}
	status, body = a.call("POST", "/admin/ranked/seasons", support, one)
	expect(t, status, body, http.StatusForbidden, "forbidden")
	status, body = a.call("POST", "/admin/ranked/seasons", liveOps, map[string]any{"id": "season.one", "start": oneStart, "end": oneEnd})
	expect(t, status, body, http.StatusBadRequest, "reason_required")
	status, body = a.call("POST", "/admin/ranked/seasons", liveOps, map[string]any{"id": "season.one", "start": start.Add(-time.Hour), "end": oneEnd, "reason": "launch"})
	expect(t, status, body, http.StatusBadRequest, "invalid")
	for _, step := range []struct {
		name   string
		status gamestrings.Status
	}{{"no name", ""}, {"a draft name", gamestrings.Draft}} {
		if step.status != "" {
			a.nameKeys(step.status, "season.one.name")
		}
		status, body = a.call("POST", "/admin/ranked/seasons", liveOps, one)
		if status != http.StatusBadRequest || body["error"] != "invalid" || !strings.HasSuffix(body["detail"].(string), "no approved en name for season.one.name") || !refusedKeys(body, "season.one.name") {
			t.Fatalf("%s: %d %v", step.name, status, body)
		}
	}
	a.nameKeys(gamestrings.Approved, "season.one.name", "season.two.name")

	status, body = a.call("POST", "/admin/ranked/seasons", liveOps, one)
	if status != http.StatusCreated || body["state"] != "scheduled" || body["name_key"] != "season.one.name" {
		t.Fatalf("create %d %v", status, body)
	}
	status, body = a.call("POST", "/admin/ranked/seasons", liveOps, map[string]any{"id": "season.two", "start": oneEnd.Add(time.Hour), "end": oneEnd.Add(30 * day), "reason": "next"})
	expect(t, status, body, http.StatusBadRequest, "invalid")
	status, body = a.call("POST", "/admin/ranked/seasons", liveOps, map[string]any{"id": "season.two", "start": oneEnd, "end": oneEnd.Add(30 * day), "no_rewards": true, "reason": "next"})
	if status != http.StatusCreated || body["no_rewards"] != true {
		t.Fatalf("create second %d %v", status, body)
	}
	status, body = a.call("GET", "/admin/ranked/seasons", support, nil)
	if status != http.StatusOK || body["next_start"] != stamp(oneEnd.Add(30*day)) {
		t.Fatalf("next start %d %v", status, body)
	}

	status, body = a.call("PATCH", "/admin/ranked/seasons/season.two", liveOps, map[string]any{"start": oneEnd.Add(day), "end": oneEnd.Add(30 * day), "reason": "later"})
	expect(t, status, body, http.StatusBadRequest, "invalid")
	status, body = a.call("PATCH", "/admin/ranked/seasons/season.one", liveOps, map[string]any{"start": oneStart, "end": oneEnd.Add(2 * day), "reason": "longer"})
	expect(t, status, body, http.StatusOK, "")
	expectWindows(t, a.seasonWindows(support), map[string][2]time.Time{
		"season.one": {oneStart, oneEnd.Add(2 * day)},
		"season.two": {oneEnd.Add(2 * day), oneEnd.Add(32 * day)},
	})
	status, body = a.call("PATCH", "/admin/ranked/seasons/season.one", liveOps, map[string]any{"start": start.Add(30 * time.Minute), "end": oneEnd.Add(-day), "reason": "shorter"})
	expect(t, status, body, http.StatusOK, "")
	oneStart = start.Add(30 * time.Minute)
	expectWindows(t, a.seasonWindows(support), map[string][2]time.Time{
		"season.one": {oneStart, oneEnd.Add(-day)},
		"season.two": {oneEnd.Add(-day), oneEnd.Add(29 * day)},
	})
	status, body = a.call("PATCH", "/admin/ranked/seasons/season.missing", liveOps, map[string]any{"start": oneStart, "end": oneEnd, "reason": "none"})
	expect(t, status, body, http.StatusNotFound, "not_found")

	status, body = a.call("DELETE", "/admin/ranked/seasons/season.one", liveOps, map[string]any{"reason": "drop"})
	expect(t, status, body, http.StatusBadRequest, "invalid")
	status, body = a.call("DELETE", "/admin/ranked/seasons/season.two", support, map[string]any{"reason": "drop"})
	expect(t, status, body, http.StatusForbidden, "forbidden")
	status, body = a.call("DELETE", "/admin/ranked/seasons/season.two", liveOps, map[string]any{"reason": "drop"})
	expect(t, status, body, http.StatusNoContent, "")
	status, body = a.call("POST", "/admin/ranked/seasons/season.one/extend", liveOps, map[string]any{"end": oneEnd, "reason": "early"})
	expect(t, status, body, http.StatusConflict, "season_not_live")

	a.now = start.Add(2 * time.Hour)
	liveOps, support = a.signIn("leyla"), a.signIn("orxan")
	status, body = a.call("POST", "/admin/ranked/seasons", liveOps, map[string]any{"id": "season.two", "start": oneEnd.Add(-day), "end": oneEnd.Add(29 * day), "reason": "again"})
	expect(t, status, body, http.StatusCreated, "")
	status, body = a.call("PATCH", "/admin/ranked/seasons/season.one", liveOps, map[string]any{"start": oneStart, "end": oneEnd, "reason": "too late"})
	expect(t, status, body, http.StatusConflict, "season_started")
	status, body = a.call("DELETE", "/admin/ranked/seasons/season.one", liveOps, map[string]any{"reason": "too late"})
	expect(t, status, body, http.StatusConflict, "season_started")
	status, body = a.call("POST", "/admin/ranked/seasons/season.one/extend", liveOps, map[string]any{"end": oneEnd.Add(-2 * day), "reason": "shorter"})
	expect(t, status, body, http.StatusBadRequest, "invalid")
	status, body = a.call("POST", "/admin/ranked/seasons/season.two/extend", liveOps, map[string]any{"end": oneEnd.Add(40 * day), "reason": "scheduled"})
	expect(t, status, body, http.StatusConflict, "season_not_live")
	status, body = a.call("POST", "/admin/ranked/seasons/season.one/extend", liveOps, map[string]any{"end": oneEnd.Add(3 * day), "reason": "holiday"})
	if status != http.StatusOK || body["state"] != "live" || body["end"] != stamp(oneEnd.Add(3*day)) {
		t.Fatalf("extend %d %v", status, body)
	}
	expectWindows(t, a.seasonWindows(support), map[string][2]time.Time{
		"season.one": {oneStart, oneEnd.Add(3 * day)},
		"season.two": {oneEnd.Add(3 * day), oneEnd.Add(33 * day)},
	})

	root := a.signIn("root")
	var actions []string
	for _, entry := range a.entries(root, "object_type=ranked_seasons&object_id=season.one") {
		actions = append(actions, entry["action"].(string))
	}
	want := []string{"extend_season", "reschedule_season", "reschedule_season", "create_season"}
	if len(actions) != len(want) {
		t.Fatalf("season.one audit %v", actions)
	}
	for i := range want {
		if actions[i] != want[i] {
			t.Fatalf("season.one audit %v, want %v", actions, want)
		}
	}
	removed := a.entries(root, "object_type=ranked_seasons&object_id=season.two")
	if len(removed) != 3 || removed[1]["action"] != "delete_season" || removed[1]["after"] != nil || removed[1]["before"].(map[string]any)["no_rewards"] != true {
		t.Fatalf("season.two audit %v", removed)
	}
}

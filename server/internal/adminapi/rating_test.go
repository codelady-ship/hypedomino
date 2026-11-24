package adminapi_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

func ratingMode(config map[string]any, mode string) map[string]any {
	return config["modes"].(map[string]any)[mode].(map[string]any)
}

func TestRatingConfigFreezesStructuralKeysWhileASeasonIsOpen(t *testing.T) {
	a := newAPI(t)
	a.publishExampleCatalog()
	a.create("leyla@example.com", rbac.LiveOps)
	a.create("orxan@example.com", rbac.Support)
	liveOps, support, content := a.signIn("leyla"), a.signIn("orxan"), a.signIn("ayla")

	status, body := a.call("GET", "/admin/config/rating", support, nil)
	rules := body["rules"].(map[string]any)
	if status != http.StatusOK || body["catalog"] != nil || rules["open_season"] != "" || len(rules["tiers"].([]any)) != 10 || len(rules["division_ranks"].([]any)) != 28 || rules["defaults"] == nil {
		t.Fatalf("empty %d %v", status, body)
	}
	status, body = a.call("GET", "/admin/config/rating", content, nil)
	expect(t, status, body, http.StatusForbidden, "forbidden")
	status, body = a.call("PUT", "/admin/config/rating", support, map[string]any{"catalog": exampleAt(t, "ranking/examples/rating")})
	expect(t, status, body, http.StatusForbidden, "forbidden")

	status, body = a.call("PUT", "/admin/config/rating", liveOps, map[string]any{"catalog": exampleAt(t, "ranking/examples/rating"), "reason": "launch ranking"})
	first, _ := body["config_version"].(float64)
	if status != http.StatusOK || first < 1 {
		t.Fatalf("publish %d %v", status, body)
	}
	invalid := exampleAt(t, "ranking/examples/rating")
	ratingMode(invalid, "1v1")["demotionLp"] = 150
	status, body = a.call("PUT", "/admin/config/rating", liveOps, map[string]any{"catalog": invalid})
	if status != http.StatusBadRequest || body["error"] != "invalid" || !strings.Contains(body["detail"].(string), "demotionLp") {
		t.Fatalf("invalid %d %v", status, body)
	}
	unlisted := exampleAt(t, "ranking/examples/rating")
	ratingMode(unlisted, "1v1")["rankRewards"].(map[string]any)["Gold"].(map[string]any)["medals"] = []string{"medal.wins_250"}
	status, body = a.call("PUT", "/admin/config/rating", liveOps, map[string]any{"catalog": unlisted})
	if status != http.StatusBadRequest || !strings.Contains(body["detail"].(string), "medal.wins_250") {
		t.Fatalf("a medal not listing the bundle: %d %v", status, body)
	}
	medals := exampleCatalog(t, "medals")
	a.passMedalArt(medals)
	a.nameExamples()
	wins250 := listed(medals, "medals", "medal.wins_250")
	wins250["sources"] = append(wins250["sources"].([]any), map[string]any{"type": "rank_reward", "ref": "bundle.rank_season_1_gold"})
	status, body = a.call("PUT", "/admin/config/medals", content, map[string]any{"catalog": medals})
	expect(t, status, body, http.StatusOK, "")
	status, body = a.call("PUT", "/admin/config/rating", liveOps, map[string]any{"catalog": unlisted, "reason": "gold medal"})
	expect(t, status, body, http.StatusOK, "")
	status, body = a.call("PUT", "/admin/config/medals", content, map[string]any{"catalog": exampleCatalog(t, "medals")})
	if status != http.StatusBadRequest || !strings.Contains(body["detail"].(string), "medal.wins_250") {
		t.Fatalf("a medal dropping a rank reward source: %d %v", status, body)
	}
	structural := exampleAt(t, "ranking/examples/rating")
	ratingMode(structural, "2v2")["divisionLp"] = 120
	status, body = a.call("PUT", "/admin/config/rating", liveOps, map[string]any{"catalog": structural, "reason": "no season yet"})
	expect(t, status, body, http.StatusOK, "")

	if _, err := a.pool.Exec(context.Background(), `insert into ranked_seasons (id, name_key, starts_at, ends_at, rewards, created_at) values ('season.one', 'season.one.name', $1, $2, true, $1)`, start.Add(-time.Hour), start.Add(48*time.Hour)); err != nil {
		t.Fatal(err)
	}
	status, body = a.call("GET", "/admin/config/rating", support, nil)
	if rules, _ := body["rules"].(map[string]any); status != http.StatusOK || rules["open_season"] != "season.one" {
		t.Fatalf("open season %d %v", status, body)
	}
	ratingMode(structural, "2v2")["divisionLp"] = 100
	status, body = a.call("PUT", "/admin/config/rating", liveOps, map[string]any{"catalog": structural})
	if detail, _ := body["detail"].(string); status != http.StatusConflict || body["error"] != "frozen" || !strings.Contains(detail, "2v2 divisionLp") {
		t.Fatalf("structural while live: %d %v", status, body)
	}
	coefficients := exampleAt(t, "ranking/examples/rating")
	ratingMode(coefficients, "2v2")["divisionLp"] = 120
	ratingMode(coefficients, "1v1")["mmrK"] = 24
	status, body = a.call("PUT", "/admin/config/rating", liveOps, map[string]any{"catalog": coefficients, "reason": "faster mmr"})
	expect(t, status, body, http.StatusOK, "")

	a.now = start.Add(72 * time.Hour)
	liveOps, support = a.signIn("leyla"), a.signIn("orxan")
	status, body = a.call("PUT", "/admin/config/rating", liveOps, map[string]any{"catalog": structural})
	expect(t, status, body, http.StatusConflict, "frozen")
	if _, err := a.pool.Exec(context.Background(), `update ranked_seasons set closed_1v1_at = $1, closed_2v2_at = $1`, a.now); err != nil {
		t.Fatal(err)
	}
	status, body = a.call("PUT", "/admin/config/rating", liveOps, map[string]any{"catalog": structural, "reason": "after the close"})
	expect(t, status, body, http.StatusOK, "")

	status, body = a.call("POST", "/admin/config/rating/rollback", liveOps, map[string]any{"version": first, "reason": "back to launch"})
	if status != http.StatusOK || ratingMode(body["catalog"].(map[string]any), "1v1")["mmrK"] != float64(20) || ratingMode(body["catalog"].(map[string]any), "2v2")["divisionLp"] != float64(100) {
		t.Fatalf("rollback %d %v", status, body)
	}
	status, body = a.call("GET", "/admin/config/rating/history", support, nil)
	if status != http.StatusOK || len(body["versions"].([]any)) != 6 {
		t.Fatalf("history %d %v", status, body)
	}
	entries := a.entries(a.signIn("root"), "object_type=ranked_seasons&object_id=rating")
	if len(entries) != 6 || entries[0]["action"] != "rollback" || entries[5]["reason"] != "launch ranking" {
		t.Fatalf("entries %v", entries)
	}
}

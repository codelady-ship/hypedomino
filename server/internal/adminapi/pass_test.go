package adminapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
)

func passWithListedCosmetics(t *testing.T) map[string]any {
	t.Helper()
	config := exampleCatalog(t, "pass")
	for _, season := range config["seasons"].([]any) {
		for _, track := range []string{"free", "premium"} {
			levels := season.(map[string]any)[track].(map[string]any)
			for level, bundle := range levels {
				delete(bundle.(map[string]any), "cosmetics")
				if bundle.(map[string]any)["coins"] == nil && bundle.(map[string]any)["crystals"] == nil {
					delete(levels, level)
				}
			}
		}
	}
	premiumLevel(config, 0, "30")["cosmetics"] = []string{"avatar.tile_wizard"}
	return config
}

func premiumLevel(config map[string]any, season int, level string) map[string]any {
	return config["seasons"].([]any)[season].(map[string]any)["premium"].(map[string]any)[level].(map[string]any)
}

func withSecondSeason(config map[string]any) map[string]any {
	first := config["seasons"].([]any)[0].(map[string]any)
	second := map[string]any{"id": "season_2", "start": "2027-03-01T00:00:00Z", "end": "2027-06-01T00:00:00Z", "curve": first["curve"], "free": first["free"], "premium": map[string]any{}}
	for level, bundle := range first["premium"].(map[string]any) {
		copied := map[string]any{}
		for key, value := range bundle.(map[string]any) {
			copied[key] = value
		}
		delete(copied, "cosmetics")
		second["premium"].(map[string]any)[level] = copied
	}
	config["seasons"] = append(config["seasons"].([]any), second)
	return config
}

func seasonIDs(config map[string]any) []string {
	ids := []string{}
	for _, season := range config["seasons"].([]any) {
		ids = append(ids, season.(map[string]any)["id"].(string))
	}
	return ids
}

func (a *api) publishItems(edit map[string]map[string]any) {
	a.t.Helper()
	manifest := exampleAt(a.t, "cosmetics/examples/catalog")
	for _, item := range manifest["items"].([]any) {
		entry := item.(map[string]any)
		for key, value := range edit[entry["id"].(string)] {
			entry[key] = value
		}
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		a.t.Fatal(err)
	}
	if _, err := snapshot.Publish(context.Background(), a.pool, cosmetics.Domain, raw); err != nil {
		a.t.Fatal(err)
	}
}

func TestARunningSeasonReplacesADisabledItemWithALiveOneOfTheSameTypeAndRarity(t *testing.T) {
	a := newAPI(t)
	a.create("leyla@example.com", rbac.LiveOps)
	liveOps := a.signIn("leyla")
	levelSource := []any{map[string]any{"type": "pass", "ref": "pass.season_1.premium.level_30"}}
	a.publishItems(map[string]map[string]any{"avatar.comet_fox": {"sources": levelSource}})
	config := passWithListedCosmetics(t)
	premiumLevel(config, 0, "30")["cosmetics"] = []string{"avatar.comet_fox"}
	status, body := a.call("PUT", "/admin/config/pass", liveOps, map[string]any{"catalog": config, "reason": "season 1"})
	expect(t, status, body, http.StatusOK, "")

	a.now = time.Date(2026, 12, 15, 9, 0, 0, 0, time.UTC)
	liveOps = a.signIn("leyla")
	a.publishItems(map[string]map[string]any{
		"avatar.comet_fox":  {"sources": levelSource, "availability": "disabled"},
		"avatar.moon_owl":   {"sources": levelSource},
		"avatar.paper_kite": {"sources": levelSource},
	})
	swapped := func(item string) map[string]any {
		next := passWithListedCosmetics(t)
		level := premiumLevel(next, 0, "30")
		level["cosmetics"], level["revision"] = []string{item}, level["revision"].(float64)+1
		return next
	}
	status, body = a.call("PUT", "/admin/config/pass", liveOps, map[string]any{"catalog": swapped("avatar.paper_kite"), "reason": "withdrawn"})
	expect(t, status, body, http.StatusConflict, "frozen")
	status, body = a.call("PUT", "/admin/config/pass", liveOps, map[string]any{"catalog": swapped("avatar.moon_owl")})
	expect(t, status, body, http.StatusBadRequest, "reason_required")
	status, body = a.call("PUT", "/admin/config/pass", liveOps, map[string]any{"catalog": swapped("avatar.moon_owl"), "reason": "comet fox withdrawn"})
	expect(t, status, body, http.StatusOK, "")
	if got := premiumLevel(body["catalog"].(map[string]any), 0, "30")["cosmetics"].([]any); len(got) != 1 || got[0] != "avatar.moon_owl" {
		t.Fatalf("replaced level %v", got)
	}
	entries := a.entries(a.signIn("root"), "object_type=pass_seasons&object_id=pass")
	if len(entries) != 2 || entries[0]["reason"] != "comet fox withdrawn" {
		t.Fatalf("entries %v", entries)
	}
}

func TestPassSeasonsArePublishedWithListedTrackItemsAndFrozenWhileRunning(t *testing.T) {
	a := newAPI(t)
	a.publishExampleCatalog()
	a.create("leyla@example.com", rbac.LiveOps)
	liveOps, content := a.signIn("leyla"), a.signIn("ayla")

	status, body := a.call("GET", "/admin/config/pass", content, nil)
	if status != http.StatusOK || body["catalog"] != nil || body["rules"].(map[string]any)["server_time"] == "" {
		t.Fatalf("empty %d %v", status, body)
	}
	status, body = a.call("PUT", "/admin/config/pass", liveOps, map[string]any{"catalog": exampleCatalog(t, "pass")})
	if status != http.StatusBadRequest || body["error"] != "invalid" || !strings.Contains(body["detail"].(string), "whose sources do not list it") {
		t.Fatalf("unlisted track items: %d %v", status, body)
	}
	for name, edit := range map[string]func(map[string]any){
		"an item listing another level": func(c map[string]any) { premiumLevel(c, 0, "31")["cosmetics"] = []string{"avatar.tile_wizard"} },
		"a mythic item":                 func(c map[string]any) { premiumLevel(c, 0, "30")["cosmetics"] = []string{"table.night_market"} },
		"a premium level without bundle": func(c map[string]any) {
			delete(c["seasons"].([]any)[0].(map[string]any)["premium"].(map[string]any), "7")
		},
	} {
		config := passWithListedCosmetics(t)
		edit(config)
		status, body := a.call("PUT", "/admin/config/pass", liveOps, map[string]any{"catalog": config})
		if status != http.StatusBadRequest || body["error"] != "invalid" || body["detail"] == "" {
			t.Fatalf("%s: %d %v", name, status, body)
		}
	}
	status, body = a.call("PUT", "/admin/config/pass", content, map[string]any{"catalog": passWithListedCosmetics(t)})
	expect(t, status, body, http.StatusForbidden, "forbidden")

	status, body = a.call("PUT", "/admin/config/pass", liveOps, map[string]any{"catalog": passWithListedCosmetics(t), "reason": "season 1"})
	first, _ := body["config_version"].(float64)
	if status != http.StatusOK || first < 1 {
		t.Fatalf("publish %d %v", status, body)
	}
	status, body = a.call("PUT", "/admin/config/pass", liveOps, map[string]any{"catalog": withSecondSeason(passWithListedCosmetics(t)), "reason": "plan season 2"})
	if status != http.StatusOK || len(seasonIDs(body["catalog"].(map[string]any))) != 2 {
		t.Fatalf("second season %d %v", status, body)
	}
	status, body = a.call("PUT", "/admin/config/pass", liveOps, map[string]any{"catalog": passWithListedCosmetics(t), "reason": "season 2 not yet"})
	if status != http.StatusOK || len(seasonIDs(body["catalog"].(map[string]any))) != 1 {
		t.Fatalf("a season that has not started may be dropped: %d %v", status, body)
	}

	a.now = time.Date(2026, 12, 15, 9, 0, 0, 0, time.UTC)
	liveOps = a.signIn("leyla")
	curve := passWithListedCosmetics(t)
	curve["seasons"].([]any)[0].(map[string]any)["curve"].(map[string]any)["xpMax"] = 1200
	status, body = a.call("PUT", "/admin/config/pass", liveOps, map[string]any{"catalog": curve})
	expect(t, status, body, http.StatusConflict, "frozen")
	track := passWithListedCosmetics(t)
	premiumLevel(track, 0, "2")["crystals"] = 5
	status, body = a.call("PUT", "/admin/config/pass", liveOps, map[string]any{"catalog": track})
	expect(t, status, body, http.StatusConflict, "frozen")
	dropped := passWithListedCosmetics(t)
	dropped["seasons"] = []any{}
	status, body = a.call("PUT", "/admin/config/pass", liveOps, map[string]any{"catalog": dropped})
	if status != http.StatusBadRequest || !strings.Contains(body["detail"].(string), "season_1") {
		t.Fatalf("a started season dropped: %d %v", status, body)
	}
	xp := withSecondSeason(passWithListedCosmetics(t))
	xp["matchXp1v1"].(map[string]any)["win"] = 120
	status, body = a.call("PUT", "/admin/config/pass", liveOps, map[string]any{"catalog": xp, "reason": "more xp for a win, plan season 2"})
	if status != http.StatusOK {
		t.Fatalf("xp sources change mid-season: %d %v", status, body)
	}

	a.now = time.Date(2027, 3, 15, 9, 0, 0, 0, time.UTC)
	liveOps, content = a.signIn("leyla"), a.signIn("ayla")
	status, body = a.call("POST", "/admin/config/pass/rollback", liveOps, map[string]any{"version": first, "reason": "back to the launch xp"})
	restored := body["catalog"].(map[string]any)
	if status != http.StatusOK || restored["matchXp1v1"].(map[string]any)["win"] != float64(100) {
		t.Fatalf("rollback %d %v", status, body)
	}
	if ids := seasonIDs(restored); len(ids) != 2 || ids[1] != "season_2" {
		t.Fatalf("a started season missing from the target version is kept: %v", ids)
	}

	status, body = a.call("GET", "/admin/config/pass/history", content, nil)
	if status != http.StatusOK || len(body["versions"].([]any)) != 5 {
		t.Fatalf("history %d %v", status, body)
	}
	entries := a.entries(a.signIn("root"), "object_type=pass_seasons&object_id=pass")
	if len(entries) != 5 || entries[0]["action"] != "rollback" || entries[4]["reason"] != "season 1" {
		t.Fatalf("entries %v", entries)
	}
}

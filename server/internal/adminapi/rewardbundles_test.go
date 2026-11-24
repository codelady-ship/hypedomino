package adminapi_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/gamestrings"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

func bundleRows(t *testing.T, a *api, token, id string) []map[string]any {
	t.Helper()
	status, body := a.call("GET", "/admin/reward-bundles?limit=200&q="+url.QueryEscape(id), token, nil)
	expect(t, status, body, http.StatusOK, "")
	var rows []map[string]any
	for _, row := range body["bundles"].([]any) {
		if entry := row.(map[string]any); entry["id"] == id {
			rows = append(rows, entry)
		}
	}
	return rows
}

func grantingMission(id string, coins int, cosmetics ...any) map[string]any {
	return map[string]any{
		"id": id, "type": "achievement", "nameKey": "mission." + id + ".name", "event": "match_won", "target": 10, "filters": map[string]any{},
		"availability": "live",
		"bundle":       map[string]any{"id": "bundle.wins", "revision": 1, "coins": coins, "cosmetics": cosmetics, "medals": []any{}, "passXp": 0},
	}
}

func publishMissions(t *testing.T, a *api, token string, missions ...any) {
	t.Helper()
	catalog := map[string]any{"dailySlots": []any{}, "weeklySlots": []any{}, "missions": missions}
	a.nameEntries(gamestrings.Approved, catalog, "missions")
	status, body := a.call("PUT", "/admin/config/missions", token, map[string]any{"catalog": catalog, "reason": "grants"})
	expect(t, status, body, http.StatusOK, "")
}

func TestRewardBundlesListTheDefaultsAnyRoleCanSee(t *testing.T) {
	a := newAPI(t)
	a.create("leyla@example.com", rbac.Analyst)
	analyst := a.signIn("leyla")

	if daily := bundleRows(t, a, analyst, "bundle.daily_login_day_1"); len(daily) != 0 {
		t.Fatalf("an unpublished calendar grants nothing, yet it lists %v", daily)
	}
	gold := bundleRows(t, a, analyst, "bundle.rank_gold")
	if len(gold) != 1 || gold[0]["differs"] != false || len(gold[0]["usedBy"].([]any)) != 2 || gold[0]["guideValue"].(float64) != 700 {
		t.Fatalf("rank gold %v", gold)
	}
}

func bundleIDs(t *testing.T, a *api, token, query string) []string {
	t.Helper()
	status, body := a.call("GET", "/admin/reward-bundles?"+query, token, nil)
	expect(t, status, body, http.StatusOK, "")
	var ids []string
	for _, row := range body["bundles"].([]any) {
		ids = append(ids, row.(map[string]any)["id"].(string))
	}
	return ids
}

func TestRewardBundlesPageAndFilterOnTheServer(t *testing.T) {
	a := newAPI(t)
	a.create("leyla@example.com", rbac.Analyst)
	analyst := a.signIn("leyla")

	every := bundleIDs(t, a, analyst, "limit=200")
	if len(every) < 3 {
		t.Fatalf("too few default bundles to page: %v", every)
	}
	first, second := bundleIDs(t, a, analyst, "limit=1"), bundleIDs(t, a, analyst, "limit=1&after="+url.QueryEscape(every[0]))
	if strings.Join(first, ",") != every[0] || strings.Join(second, ",") != every[1] {
		t.Fatalf("pages %v %v of %v", first, second, every)
	}
	if gold := bundleIDs(t, a, analyst, "q=RANK_GOLD"); strings.Join(gold, ",") != "bundle.rank_gold" {
		t.Fatalf("search %v", gold)
	}
	if rated := bundleIDs(t, a, analyst, "domain=rating&q=rank_gold"); strings.Join(rated, ",") != "bundle.rank_gold" {
		t.Fatalf("rating %v", rated)
	}
	if none := bundleIDs(t, a, analyst, "domain=shop"); len(none) != 0 {
		t.Fatalf("shop %v", none)
	}
	for _, query := range []string{"limit=x", "differing=yes"} {
		status, body := a.call("GET", "/admin/reward-bundles?"+query, analyst, nil)
		expect(t, status, body, http.StatusBadRequest, "invalid")
	}
}

func TestRewardBundlesTrackUsesValueAndDifferingContent(t *testing.T) {
	a := newAPI(t)
	a.create("leyla@example.com", rbac.LiveOps)
	content, liveOps := a.signIn("ayla"), a.signIn("leyla")

	status, body := a.call("GET", "/admin/config/cosmetics", liveOps, nil)
	expect(t, status, body, http.StatusOK, "")
	builtin := body["rules"].(map[string]any)["builtin"].(map[string]any)
	fox := foxAvatar()
	fox["sources"] = []any{
		map[string]any{"type": "mission", "ref": "mission.first"},
		map[string]any{"type": "mission", "ref": "mission.second"},
		map[string]any{"type": "mission", "ref": "mission.third"},
	}
	catalog := map[string]any{"items": append(append([]any{}, builtin["items"].([]any)...), fox)}
	a.nameItems(gamestrings.Approved, catalog["items"].([]any)...)
	a.passCosmeticArt(fox)
	status, body = a.call("PUT", "/admin/config/cosmetics", content, map[string]any{"catalog": catalog, "reason": "fox"})
	expect(t, status, body, http.StatusOK, "")

	publishMissions(t, a, liveOps, grantingMission("mission.first", 100, "avatar.fox"), grantingMission("mission.second", 100, "avatar.fox"))
	rows := bundleRows(t, a, liveOps, "bundle.wins")
	if len(rows) != 1 || rows[0]["differs"] != false || len(rows[0]["usedBy"].([]any)) != 2 {
		t.Fatalf("same content %v", rows)
	}
	if rows[0]["guideValue"].(float64) != 1600 {
		t.Fatalf("value %v, want the coins plus the rare avatar guide price", rows[0]["guideValue"])
	}
	use := rows[0]["usedBy"].([]any)[0].(map[string]any)
	if use["domain"] != "missions" || use["object"] != "mission.first" {
		t.Fatalf("use %v", use)
	}

	publishMissions(t, a, liveOps, grantingMission("mission.first", 100, "avatar.fox"), grantingMission("mission.second", 100, "avatar.fox"), grantingMission("mission.third", 250, "avatar.fox"))
	rows = bundleRows(t, a, liveOps, "bundle.wins")
	if len(rows) != 2 || rows[0]["differs"] != true || rows[1]["differs"] != true {
		t.Fatalf("differing content %v", rows)
	}
	if differing := bundleIDs(t, a, liveOps, "differing=true&q=bundle.wins&limit=1"); strings.Join(differing, ",") != "bundle.wins,bundle.wins" {
		t.Fatalf("a page keeps every row of its last id: %v", differing)
	}
}

func TestRewardBundlesCoverTheShopThePassAndPromoCampaigns(t *testing.T) {
	a := newPromoAPI(t)
	a.publishShopItems()
	a.nameOffers()
	liveOps := a.signIn("leyla")

	status, body := a.call("PUT", "/admin/config/shop", liveOps, map[string]any{"catalog": catalog(weekendPack(start, start.Add(time.Hour)), cometFox()), "reason": "weekend"})
	expect(t, status, body, http.StatusOK, "")
	status, body = a.call("PUT", "/admin/config/pass", liveOps, map[string]any{"catalog": passWithListedCosmetics(t), "reason": "season"})
	expect(t, status, body, http.StatusOK, "")
	for _, id := range []string{"promo.launch", "promo.launch_two"} {
		status, body = a.call("POST", "/admin/promo-codes", liveOps, promoBody(id, 300, nil))
		expect(t, status, body, http.StatusCreated, "")
	}

	offer := bundleRows(t, a, liveOps, "bundle.weekend_pack")
	if len(offer) != 1 || offer[0]["usedBy"].([]any)[0].(map[string]any)["object"] != "offer.weekend_pack" {
		t.Fatalf("offer %v", offer)
	}
	codes := bundleRows(t, a, liveOps, "bundle.launch_coins")
	if len(codes) != 1 || len(codes[0]["usedBy"].([]any)) != 1 || codes[0]["usedBy"].([]any)[0].(map[string]any)["object"] != "campaign.launch_week" {
		t.Fatalf("a campaign is one use however many codes it has: %v", codes)
	}
	status, body = a.call("GET", "/admin/reward-bundles?domain=pass&limit=200", liveOps, nil)
	expect(t, status, body, http.StatusOK, "")
	if !grantsFrom(body, "pass", "premium.30") {
		t.Fatalf("no pass level use in %v", body)
	}
}

func grantsFrom(body map[string]any, domain, detail string) bool {
	for _, row := range body["bundles"].([]any) {
		for _, use := range row.(map[string]any)["usedBy"].([]any) {
			if entry := use.(map[string]any); entry["domain"] == domain && entry["detail"] == detail {
				return true
			}
		}
	}
	return false
}

func editedBundle(revision, coins, passXP int, cosmetics ...any) map[string]any {
	return map[string]any{"revision": revision, "coins": coins, "crystals": 0, "cosmetics": cosmetics, "medals": []any{}, "passXp": passXP, "reason": "retune"}
}

func publishSharedBundle(t *testing.T, a *api, content, liveOps string) {
	t.Helper()
	status, body := a.call("GET", "/admin/config/cosmetics", liveOps, nil)
	expect(t, status, body, http.StatusOK, "")
	fox := foxAvatar()
	fox["sources"] = []any{map[string]any{"type": "mission", "ref": "mission.first"}, map[string]any{"type": "mission", "ref": "mission.second"}, map[string]any{"type": "shop", "ref": "offer.weekend_pack"}}
	owl := foxAvatar()
	owl["id"], owl["sources"] = "avatar.owl", fox["sources"]
	comet := foxAvatar()
	comet["id"], comet["sources"] = "avatar.comet_fox", []any{map[string]any{"type": "shop", "ref": "offer.avatar_comet_fox"}}
	items := append(append([]any{}, body["rules"].(map[string]any)["builtin"].(map[string]any)["items"].([]any)...), fox, owl, comet)
	a.nameItems(gamestrings.Approved, items...)
	a.passCosmeticArt(fox, owl, comet)
	status, body = a.call("PUT", "/admin/config/cosmetics", content, map[string]any{"catalog": map[string]any{"items": items}, "reason": "fox"})
	expect(t, status, body, http.StatusOK, "")
	publishMissions(t, a, liveOps, grantingMission("mission.first", 0, "avatar.fox"), grantingMission("mission.second", 0, "avatar.fox"))
	a.nameOffers()
	pack := weekendPack(start, start.Add(time.Hour))
	pack["bundle"] = map[string]any{"id": "bundle.wins", "revision": 1, "cosmetics": []string{"avatar.fox"}, "medals": []string{}}
	status, body = a.call("PUT", "/admin/config/shop", liveOps, map[string]any{"catalog": catalog(pack, cometFox()), "reason": "pack"})
	expect(t, status, body, http.StatusOK, "")
}

func TestARewardBundleEditChangesEveryHolderOrNone(t *testing.T) {
	a := newAPI(t)
	a.create("leyla@example.com", rbac.LiveOps)
	content, liveOps := a.signIn("ayla"), a.signIn("leyla")
	publishSharedBundle(t, a, content, liveOps)

	rows := bundleRows(t, a, liveOps, "bundle.wins")
	if len(rows) != 1 || rows[0]["editable"] != true || len(rows[0]["usedBy"].([]any)) != 3 {
		t.Fatalf("shared %v", rows)
	}
	revision := int(rows[0]["revision"].(float64))

	status, body := a.call("PUT", "/admin/reward-bundles/bundle.wins", content, editedBundle(revision, 0, 50, "avatar.fox"))
	expect(t, status, body, http.StatusForbidden, "forbidden")
	stale := editedBundle(revision+5, 0, 50, "avatar.fox")
	status, body = a.call("PUT", "/admin/reward-bundles/bundle.wins", liveOps, stale)
	expect(t, status, body, http.StatusConflict, "stale_bundle")
	status, body = a.call("PUT", "/admin/reward-bundles/bundle.wins", liveOps, editedBundle(revision, 0, 0, "avatar.fox"))
	expect(t, status, body, http.StatusBadRequest, "invalid")
	status, body = a.call("PUT", "/admin/reward-bundles/bundle.none", liveOps, editedBundle(1, 5, 0))
	expect(t, status, body, http.StatusNotFound, "not_found")

	status, missionsBefore := a.call("GET", "/admin/config/missions", liveOps, nil)
	expect(t, status, missionsBefore, http.StatusOK, "")
	status, body = a.call("PUT", "/admin/reward-bundles/bundle.wins", liveOps, editedBundle(revision, 5, 0, "avatar.fox"))
	expect(t, status, body, http.StatusBadRequest, "invalid")
	status, missionsAfter := a.call("GET", "/admin/config/missions", liveOps, nil)
	expect(t, status, missionsAfter, http.StatusOK, "")
	if missionsBefore["config_version"] != missionsAfter["config_version"] {
		t.Fatalf("the shop refused coins, yet missions moved from %v to %v", missionsBefore["config_version"], missionsAfter["config_version"])
	}

	status, body = a.call("PUT", "/admin/reward-bundles/bundle.wins", liveOps, editedBundle(revision, 0, 0, "avatar.fox", "avatar.owl"))
	expect(t, status, body, http.StatusOK, "")
	rows = bundleRows(t, a, liveOps, "bundle.wins")
	if len(rows) != 1 || len(rows[0]["cosmetics"].([]any)) != 2 || int(rows[0]["revision"].(float64)) != revision+1 || len(rows[0]["usedBy"].([]any)) != 3 {
		t.Fatalf("edited %v", rows)
	}
}

func TestARewardBundleWithAnotherHolderOrDifferingContentIsNotEditedHere(t *testing.T) {
	a := newAPI(t)
	a.create("leyla@example.com", rbac.LiveOps)
	liveOps := a.signIn("leyla")

	status, body := a.call("PUT", "/admin/config/daily-login", liveOps, map[string]any{"catalog": calendarOf(30, 30, 50, 50, 70, 70, 150), "reason": "launch"})
	expect(t, status, body, http.StatusOK, "")
	daily := bundleRows(t, a, liveOps, "bundle.daily_login_day_1")
	if daily[0]["editable"] != false {
		t.Fatalf("daily %v", daily)
	}
	status, body = a.call("PUT", "/admin/reward-bundles/bundle.daily_login_day_1", liveOps, editedBundle(1, 40, 0))
	expect(t, status, body, http.StatusConflict, "bundle_not_editable")

	publishMissions(t, a, liveOps, grantingMission("mission.first", 100), grantingMission("mission.second", 200))
	status, body = a.call("PUT", "/admin/reward-bundles/bundle.wins", liveOps, editedBundle(1, 300, 0))
	expect(t, status, body, http.StatusConflict, "bundle_not_editable")
}

func passBundleWithCoins(t *testing.T, a *api, token string) map[string]any {
	t.Helper()
	status, body := a.call("GET", "/admin/reward-bundles?domain=pass&limit=200", token, nil)
	expect(t, status, body, http.StatusOK, "")
	for _, row := range body["bundles"].([]any) {
		entry := row.(map[string]any)
		if entry["coins"].(float64) > 0 && entry["editable"] == true && entry["usedBy"].([]any)[0].(map[string]any)["domain"] == "pass" {
			return entry
		}
	}
	t.Fatalf("no editable pass bundle in %v", body)
	return nil
}

func TestARewardBundleEditReachesThePassTracks(t *testing.T) {
	a := newAPI(t)
	a.publishExampleCatalog()
	a.create("leyla@example.com", rbac.LiveOps)
	liveOps := a.signIn("leyla")
	status, body := a.call("PUT", "/admin/config/pass", liveOps, map[string]any{"catalog": passWithListedCosmetics(t), "reason": "season"})
	expect(t, status, body, http.StatusOK, "")

	row := passBundleWithCoins(t, a, liveOps)
	id := row["id"].(string)
	coins, revision := int(row["coins"].(float64)), int(row["revision"].(float64))
	status, body = a.call("PUT", "/admin/reward-bundles/"+id, liveOps, editedBundle(revision, coins-1, 0))
	expect(t, status, body, http.StatusOK, "")
	rows := bundleRows(t, a, liveOps, id)
	if len(rows) != 1 || int(rows[0]["coins"].(float64)) != coins-1 || int(rows[0]["revision"].(float64)) != revision+1 {
		t.Fatalf("edited %v", rows)
	}
}

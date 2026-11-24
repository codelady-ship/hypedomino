package adminapi_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

func newBundle(id string, coins int, cosmetics ...any) map[string]any {
	return map[string]any{"id": id, "coins": coins, "crystals": 0, "cosmetics": cosmetics, "medals": []any{}, "passXp": 0, "reason": "new pack"}
}

func TestANewRewardBundleIsCreatedOnceWithPublishedContent(t *testing.T) {
	a := newAPI(t)
	a.create("leyla@example.com", rbac.LiveOps)
	content, liveOps := a.signIn("ayla"), a.signIn("leyla")
	a.publishShopItems()

	status, body := a.call("POST", "/admin/reward-bundles", content, newBundle("bundle.weekend_pack", 50))
	expect(t, status, body, http.StatusForbidden, "forbidden")
	unnoted := newBundle("bundle.weekend_pack", 50)
	unnoted["reason"] = " "
	status, body = a.call("POST", "/admin/reward-bundles", liveOps, unnoted)
	expect(t, status, body, http.StatusBadRequest, "reason_required")
	for name, refused := range map[string]map[string]any{
		"id":      newBundle("weekend_pack", 50),
		"empty":   newBundle("bundle.weekend_pack", 0),
		"unknown": newBundle("bundle.weekend_pack", 0, "avatar.nobody"),
	} {
		status, body = a.call("POST", "/admin/reward-bundles", liveOps, refused)
		if status != http.StatusBadRequest || body["error"] != "invalid" {
			t.Fatalf("%s: got %d %v", name, status, body)
		}
	}
	status, body = a.call("POST", "/admin/reward-bundles", liveOps, newBundle("bundle.rank_gold", 50))
	expect(t, status, body, http.StatusConflict, "bundle_exists")

	status, body = a.call("POST", "/admin/reward-bundles", liveOps, newBundle("bundle.weekend_pack", 50))
	expect(t, status, body, http.StatusCreated, "")
	if body["bundle"].(map[string]any)["revision"].(float64) != 1 || body["config_version"].(float64) < 1 {
		t.Fatalf("created %v", body)
	}
	status, body = a.call("POST", "/admin/reward-bundles", liveOps, newBundle("bundle.weekend_pack", 70))
	expect(t, status, body, http.StatusConflict, "bundle_exists")

	rows := bundleRows(t, a, liveOps, "bundle.weekend_pack")
	if len(rows) != 1 || rows[0]["editable"] != true || rows[0]["coins"].(float64) != 50 {
		t.Fatalf("created row %v", rows)
	}
	use := rows[0]["usedBy"].([]any)[0].(map[string]any)
	if len(rows[0]["usedBy"].([]any)) != 1 || use["domain"] != "reward_bundles" || use["object"] != "bundle.weekend_pack" {
		t.Fatalf("uses %v", rows[0]["usedBy"])
	}
	if listed := bundleIDs(t, a, liveOps, "domain=reward_bundles"); len(listed) != 1 || listed[0] != "bundle.weekend_pack" {
		t.Fatalf("domain filter %v", listed)
	}
}

func TestACreatedRewardBundleIsEditedAndThenGrantedByAnOffer(t *testing.T) {
	a := newAPI(t)
	a.create("leyla@example.com", rbac.LiveOps)
	liveOps := a.signIn("leyla")
	a.publishShopItems()
	a.nameOffers()

	status, body := a.call("POST", "/admin/reward-bundles", liveOps, newBundle("bundle.weekend_pack", 50))
	expect(t, status, body, http.StatusCreated, "")
	status, body = a.call("PUT", "/admin/reward-bundles/bundle.weekend_pack", liveOps, editedBundle(1, 0, 0, "avatar.comet_fox"))
	expect(t, status, body, http.StatusOK, "")

	pack := weekendPack(start, start.Add(time.Hour))
	pack["bundle"] = map[string]any{"id": "bundle.weekend_pack", "revision": 2, "cosmetics": []string{"avatar.comet_fox"}}
	status, body = a.call("PUT", "/admin/config/shop", liveOps, map[string]any{"catalog": catalog(pack, cometFox()), "reason": "weekend"})
	expect(t, status, body, http.StatusOK, "")

	rows := bundleRows(t, a, liveOps, "bundle.weekend_pack")
	if len(rows) != 1 || rows[0]["differs"] != false || rows[0]["revision"].(float64) != 2 || len(rows[0]["usedBy"].([]any)) != 2 {
		t.Fatalf("one bundle in the library and the shop: %v", rows)
	}
}

package adminapi_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/adminapi"
	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/gamestrings"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/shop"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
)

func weekendPack(opens, closes time.Time) map[string]any {
	return map[string]any{
		"id": "offer.weekend_pack", "kind": "bundle", "section": "featured", "priceCrystals": 90,
		"bundle":  map[string]any{"id": "bundle.weekend_pack", "revision": 1, "cosmetics": []string{"avatar.comet_fox"}},
		"window":  map[string]any{"start": opens, "end": closes},
		"nameKey": "shop.offer.weekend_pack",
	}
}

func cometFox() map[string]any {
	return map[string]any{
		"id": "offer.avatar_comet_fox", "kind": "cosmetic", "section": "daily_rotation", "priceCoins": 400, "limitPerAccount": 1,
		"bundle":  map[string]any{"id": "bundle.avatar_comet_fox", "revision": 1, "cosmetics": []string{"avatar.comet_fox"}},
		"window":  map[string]any{"start": "2026-01-01T00:00:00Z"},
		"nameKey": "shop.offer.avatar_comet_fox",
	}
}

func catalog(offers ...map[string]any) map[string]any {
	return map[string]any{
		"schema": 1, "offers": offers,
		"rotations": []map[string]any{{"period": "daily", "start": "2026-11-01T00:00:00Z", "offers": []string{"offer.avatar_comet_fox"}}},
	}
}

func (a *api) publishShopItems() {
	a.t.Helper()
	sources := []any{}
	for _, offer := range []string{"offer.avatar_comet_fox", "offer.starter", "offer.weekend_pack"} {
		sources = append(sources, map[string]any{"type": "shop", "ref": offer})
	}
	a.publishItems(map[string]map[string]any{"avatar.comet_fox": {"sources": sources}})
}

func (a *api) nameOffers() {
	a.t.Helper()
	a.nameKeys(gamestrings.Approved, "shop.offer.avatar_comet_fox", "shop.offer.weekend_pack", "shop.offer.chest_avatar")
}

func TestAnOfferNeedsAnApprovedEnglishNameToBePublishedOrRestored(t *testing.T) {
	a := newAPI(t)
	a.create("leyla@example.com", rbac.LiveOps)
	liveOps := a.signIn("leyla")
	a.publishShopItems()
	a.nameKeys(gamestrings.Approved, "shop.offer.weekend_pack")
	weekend := catalog(cometFox(), weekendPack(start.Add(time.Hour), start.Add(2*time.Hour)))

	for _, step := range []struct {
		name   string
		status gamestrings.Status
	}{{"no value", ""}, {"a draft value", gamestrings.Draft}} {
		if step.status != "" {
			a.nameKeys(step.status, "shop.offer.avatar_comet_fox")
		}
		status, body := a.call("PUT", "/admin/config/shop", liveOps, map[string]any{"catalog": weekend, "reason": "weekend"})
		if status != http.StatusBadRequest || body["error"] != "invalid" || !strings.HasSuffix(body["detail"].(string), "no approved en name for shop.offer.avatar_comet_fox") || !refusedKeys(body, "shop.offer.avatar_comet_fox") {
			t.Fatalf("%s: %d %v", step.name, status, body)
		}
	}
	a.nameKeys(gamestrings.Approved, "shop.offer.avatar_comet_fox")
	status, body := a.call("PUT", "/admin/config/shop", liveOps, map[string]any{"catalog": weekend, "reason": "weekend"})
	expect(t, status, body, http.StatusOK, "")
	status, body = a.call("PUT", "/admin/config/shop", liveOps, map[string]any{"catalog": weekend, "reason": "again"})
	expect(t, status, body, http.StatusOK, "")

	a.nameKeys(gamestrings.Draft, "shop.offer.weekend_pack")
	status, body = a.call("POST", "/admin/config/shop/rollback", liveOps, map[string]any{"version": 2, "reason": "restore"})
	if status != http.StatusBadRequest || body["error"] != "invalid" || !strings.HasSuffix(body["detail"].(string), "no approved en name for shop.offer.weekend_pack") || !refusedKeys(body, "shop.offer.weekend_pack") {
		t.Fatalf("rollback %d %v", status, body)
	}
}

func TestAVariantNameKeyNeedsAnApprovedEnglishName(t *testing.T) {
	a := newAPI(t)
	a.create("leyla@example.com", rbac.LiveOps)
	liveOps := a.signIn("leyla")
	a.publishShopItems()
	a.nameOffers()
	pack := weekendPack(start.Add(time.Hour), start.Add(2*time.Hour))
	pack["variants"] = []any{map[string]any{"id": "bold", "nameKey": "shop.offer.weekend_pack_bold", "position": 1}}

	status, body := a.call("PUT", "/admin/config/shop", liveOps, map[string]any{"catalog": catalog(cometFox(), pack), "reason": "variant"})
	if status != http.StatusBadRequest || body["error"] != "invalid" || !strings.HasSuffix(body["detail"].(string), "no approved en name for shop.offer.weekend_pack_bold") || !refusedKeys(body, "shop.offer.weekend_pack_bold") {
		t.Fatalf("unnamed variant %d %v", status, body)
	}
	a.nameKeys(gamestrings.Approved, "shop.offer.weekend_pack_bold")
	status, body = a.call("PUT", "/admin/config/shop", liveOps, map[string]any{"catalog": catalog(cometFox(), pack), "reason": "variant"})
	expect(t, status, body, http.StatusOK, "")
	status, body = a.call("GET", "/admin/config/shop", liveOps, nil)
	expect(t, status, body, http.StatusOK, "")
	for _, offer := range body["catalog"].(map[string]any)["offers"].([]any) {
		if entry := offer.(map[string]any); entry["id"] == "offer.weekend_pack" && len(entry["variants"].([]any)) != 1 {
			t.Fatalf("variants %v", entry)
		}
	}
}

func (a *api) featuredAt(at time.Time) []string {
	a.t.Helper()
	watcher := snapshot.New(a.pool, shop.Domain, shop.Parser(shop.DefaultConfig()), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if _, err := watcher.Load(context.Background()); err != nil && !errors.Is(err, snapshot.ErrNone) {
		a.t.Fatal(err)
	}
	current, _, _ := watcher.Current()
	ids := []string{}
	if current == nil {
		return ids
	}
	for _, section := range current.Sections(shop.DefaultConfig(), at) {
		if section.Name == shop.SectionFeatured {
			for _, offer := range section.Offers {
				ids = append(ids, offer.ID)
			}
		}
	}
	return ids
}

func TestAScheduledOfferOpensAndClosesOnServerTimeAndCanBeRolledBack(t *testing.T) {
	a := newAPI(t)
	a.create("leyla@example.com", rbac.LiveOps)
	liveOps := a.signIn("leyla")
	opens, closes := start.Add(time.Hour), start.Add(3*time.Hour)
	a.nameOffers()

	status, body := a.call("GET", "/admin/config/shop", liveOps, nil)
	if status != http.StatusOK || body["config_version"] != float64(0) || body["catalog"] != nil {
		t.Fatalf("empty %d %v", status, body)
	}
	status, body = a.call("PUT", "/admin/config/shop", liveOps, map[string]any{"catalog": catalog(cometFox()), "reason": "too early"})
	if status != http.StatusBadRequest || !strings.Contains(body["detail"].(string), "offer.avatar_comet_fox grants avatar.comet_fox, whose sources do not list it") {
		t.Fatalf("offer before its item %d %v", status, body)
	}
	a.publishShopItems()
	status, body = a.call("PUT", "/admin/config/shop", liveOps, map[string]any{"catalog": catalog(cometFox(), weekendPack(opens, closes)), "reason": "weekend sale"})
	if status != http.StatusOK || body["config_version"] != float64(2) || len(body["catalog"].(map[string]any)["offers"].([]any)) != 2 {
		t.Fatalf("publish %d %v", status, body)
	}
	for at, want := range map[time.Time]int{start: 0, opens: 1, closes.Add(-time.Second): 1, closes: 0} {
		if got := a.featuredAt(at); len(got) != want {
			t.Fatalf("featured at %s: %v", at.Format(time.RFC3339), got)
		}
	}

	status, body = a.call("PUT", "/admin/config/shop", liveOps, map[string]any{"catalog": catalog(cometFox())})
	if status != http.StatusBadRequest || body["error"] != "invalid" || !strings.Contains(body["detail"].(string), "offer.weekend_pack") {
		t.Fatalf("removed offer %d %v", status, body)
	}
	status, body = a.call("PUT", "/admin/config/shop", liveOps, map[string]any{"catalog": catalog(cometFox(), weekendPack(opens, opens.Add(time.Hour)))})
	if status != http.StatusOK || body["config_version"] != float64(3) {
		t.Fatalf("shortened %d %v", status, body)
	}
	if got := a.featuredAt(opens.Add(90 * time.Minute)); len(got) != 0 {
		t.Fatalf("featured after the shortened window: %v", got)
	}

	a.now = start.Add(11 * time.Minute)
	status, body = a.call("POST", "/admin/config/shop/rollback", liveOps, map[string]any{"version": 2, "reason": "sale extended"})
	expect(t, status, body, http.StatusForbidden, "step_up_required")
	a.call("POST", "/admin/session/step-up", liveOps, map[string]string{"id_token": "leyla", "nonce": "nonce"})
	status, body = a.call("POST", "/admin/config/shop/rollback", liveOps, map[string]any{"version": 7, "reason": "sale extended"})
	expect(t, status, body, http.StatusNotFound, "not_found")
	status, body = a.call("POST", "/admin/config/shop/rollback", liveOps, map[string]any{"version": 2, "reason": "sale extended"})
	if status != http.StatusOK || body["config_version"] != float64(4) {
		t.Fatalf("rollback %d %v", status, body)
	}
	if got := a.featuredAt(opens.Add(90 * time.Minute)); len(got) != 1 {
		t.Fatalf("featured after rollback: %v", got)
	}

	status, body = a.call("GET", "/admin/config/shop/history", liveOps, nil)
	versions := body["versions"].([]any)
	if status != http.StatusOK || len(versions) != 3 || versions[0].(map[string]any)["config_version"] != float64(4) {
		t.Fatalf("history %d %v", status, body)
	}
	entries := a.entries(liveOps, "object_type=shop_offers&object_id=shop")
	if len(entries) != 3 {
		t.Fatalf("entries %v", entries)
	}
	rollback, shortened, first := entries[0], entries[1], entries[2]
	if rollback["action"] != "rollback" || rollback["reason"] != "sale extended" || rollback["config_version"] != float64(4) || rollback["before"] == nil {
		t.Fatalf("rollback entry %v", rollback)
	}
	if shortened["action"] != "publish" || first["before"] != nil || first["reason"] != "weekend sale" {
		t.Fatalf("publish entries %v %v", shortened, first)
	}
}

func TestACatalogIsCheckedLikeTheGameServerAndFollowsTheRoleMatrix(t *testing.T) {
	a := newAPI(t)
	a.create("leyla@example.com", rbac.LiveOps)
	a.create("orxan@example.com", rbac.Support)
	liveOps, content, support := a.signIn("leyla"), a.signIn("ayla"), a.signIn("orxan")
	bothPrices := weekendPack(start, start.Add(time.Hour))
	bothPrices["productId"] = "offer_starter"
	lateRotation := catalog(cometFox())
	lateRotation["rotations"] = []map[string]any{{"period": "daily", "start": "2026-11-01T05:00:00Z", "offers": []string{"offer.avatar_comet_fox"}}}
	for name, request := range map[string]map[string]any{
		"no catalog":                 {"reason": "x"},
		"another schema":             {"catalog": map[string]any{"schema": 2, "offers": []any{}}},
		"coins and a product":        {"catalog": catalog(cometFox(), bothPrices)},
		"a rotation off its period":  {"catalog": lateRotation},
		"an offer listed twice":      {"catalog": catalog(cometFox(), cometFox())},
		"a catalog that is a string": {"catalog": "shop"},
	} {
		status, body := a.call("PUT", "/admin/config/shop", liveOps, request)
		if status != http.StatusBadRequest || body["error"] != "invalid" || body["detail"] == "" {
			t.Fatalf("%s: %d %v", name, status, body)
		}
	}
	huge := catalog(cometFox())
	huge["padding"] = strings.Repeat("x", 1<<20)
	status, body := a.call("PUT", "/admin/config/shop", liveOps, map[string]any{"catalog": huge})
	expect(t, status, body, http.StatusRequestEntityTooLarge, "too_large")
	for _, token := range []string{content, support} {
		status, body := a.call("PUT", "/admin/config/shop", token, map[string]any{"catalog": catalog(cometFox())})
		expect(t, status, body, http.StatusForbidden, "forbidden")
		status, body = a.call("GET", "/admin/config/shop", token, nil)
		expect(t, status, body, http.StatusOK, "")
	}
	if entries := a.entries(a.signIn("root"), "object_type=shop_offers"); len(entries) != 0 {
		t.Fatalf("refused publishes left entries: %v", entries)
	}
	if got := a.featuredAt(start); len(got) != 0 {
		t.Fatalf("featured %v", got)
	}
}

func avatarChest() map[string]any {
	return map[string]any{
		"id": "offer.chest_avatar", "kind": "chest", "section": "chests", "priceCoins": 500,
		"bundle":  map[string]any{"id": "bundle.chest_avatar", "revision": 1, "cosmetics": []string{"avatar.paper_kite", "avatar.moon_owl"}},
		"window":  map[string]any{"start": "2026-01-01T00:00:00Z"},
		"nameKey": "shop.offer.chest_avatar",
	}
}

func TestAChestPoolMustHoldPublishedCommonOrRareItemsOfTheChest(t *testing.T) {
	a := newAPI(t)
	a.create("leyla@example.com", rbac.LiveOps)
	liveOps := a.signIn("leyla")
	withChest := catalog(cometFox(), avatarChest())
	withChest["chests"] = map[string]any{"rarityPercent": map[string]int{"common": 85, "rare": 15}, "rarePityCount": 10}
	a.nameOffers()

	status, body := a.call("GET", "/admin/cosmetics/items", liveOps, nil)
	if status != http.StatusOK || len(body["items"].([]any)) != 5 {
		t.Fatalf("built-in items %d %v", status, body)
	}
	status, body = a.call("PUT", "/admin/config/shop", liveOps, map[string]any{"catalog": withChest})
	if status != http.StatusBadRequest || body["error"] != "invalid" || !strings.Contains(body["detail"].(string), "unknown item avatar.paper_kite") {
		t.Fatalf("chest before its items %d %v", status, body)
	}
	raw, err := os.ReadFile("../../../docs/cosmetics/examples/catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := snapshot.Publish(context.Background(), a.pool, cosmetics.Domain, raw); err != nil {
		t.Fatal(err)
	}
	status, body = a.call("GET", "/admin/cosmetics/items", liveOps, nil)
	items := body["items"].([]any)
	if status != http.StatusOK || len(items) != 16 {
		t.Fatalf("published items %d %v", status, body)
	}
	kite := items[6].(map[string]any)
	if kite["id"] != "avatar.paper_kite" || kite["rarity"] != "common" || kite["availability"] != "live" || kite["sources"].([]any)[0].(map[string]any)["type"] != "chest" {
		t.Fatalf("chest item %v", kite)
	}
	status, body = a.call("PUT", "/admin/config/shop", liveOps, map[string]any{"catalog": withChest})
	if status != http.StatusOK || body["config_version"] != float64(2) {
		t.Fatalf("chest after its items %d %v", status, body)
	}
	withChest["offers"] = []map[string]any{cometFox(), avatarChest()}
	withChest["offers"].([]map[string]any)[1]["bundle"].(map[string]any)["cosmetics"] = []string{"avatar.paper_kite", "frame.season_1_top"}
	status, body = a.call("PUT", "/admin/config/shop", liveOps, map[string]any{"catalog": withChest})
	if status != http.StatusBadRequest || !strings.Contains(body["detail"].(string), "rarity epic") {
		t.Fatalf("epic pool item %d %v", status, body)
	}
}

func TestAShopPreviewShowsTheSectionsOfADraftAtAGivenTime(t *testing.T) {
	a := newAPI(t)
	a.create("orxan@example.com", rbac.Support)
	support := a.signIn("orxan")
	opens, closes := start.Add(time.Hour), start.Add(3*time.Hour)
	draft := catalog(cometFox(), weekendPack(opens, closes))
	shown := func(body map[string]any, section string) []any {
		for _, entry := range body["sections"].([]any) {
			if entry.(map[string]any)["section"] == section {
				return entry.(map[string]any)["offers"].([]any)
			}
		}
		t.Fatalf("no %s in %v", section, body)
		return nil
	}
	status, body := a.call("POST", "/admin/config/shop/preview", support, map[string]any{"catalog": draft, "at": opens})
	expect(t, status, body, http.StatusOK, "")
	if featured, daily := shown(body, "featured"), shown(body, "daily_rotation"); len(featured) != 1 || featured[0].(map[string]any)["id"] != "offer.weekend_pack" || featured[0].(map[string]any)["name_key"] != "shop.offer.weekend_pack" || len(daily) != 1 || daily[0].(map[string]any)["id"] != "offer.avatar_comet_fox" {
		t.Fatalf("at opening %v", body)
	}
	status, body = a.call("POST", "/admin/config/shop/preview", support, map[string]any{"catalog": draft, "at": closes})
	expect(t, status, body, http.StatusOK, "")
	if featured := shown(body, "featured"); len(featured) != 0 {
		t.Fatalf("at closing %v", body)
	}
	status, body = a.call("POST", "/admin/config/shop/preview", support, map[string]any{"catalog": draft})
	if status != http.StatusOK || body["at"] != start.Format(time.RFC3339) {
		t.Fatalf("server time %d %v", status, body)
	}
	status, body = a.call("POST", "/admin/config/shop/preview", support, map[string]any{"catalog": catalog(cometFox(), cometFox())})
	expect(t, status, body, http.StatusBadRequest, "invalid")
	if entries := a.entries(a.signIn("root"), "object_type=shop_offers"); len(entries) != 0 {
		t.Fatalf("a preview left entries: %v", entries)
	}
}

func TestAShopPreviewShowsTheChosenVariantOfAnOffer(t *testing.T) {
	a := newAPI(t)
	a.create("orxan@example.com", rbac.Support)
	support := a.signIn("orxan")
	opens := start.Add(time.Hour)
	second := weekendPack(opens, opens.Add(time.Hour))
	second["id"], second["nameKey"] = "offer.weekend_pack_two", "shop.offer.weekend_pack_two"
	second["bundle"] = map[string]any{"id": "bundle.weekend_pack_two", "revision": 1, "cosmetics": []string{"avatar.comet_fox"}}
	second["variants"] = []any{map[string]any{"id": "top", "nameKey": "shop.offer.weekend_pack_top", "position": 1}}
	draft := catalog(cometFox(), weekendPack(opens, opens.Add(time.Hour)), second)
	featured := func(variants map[string]string) string {
		status, body := a.call("POST", "/admin/config/shop/preview", support, map[string]any{"catalog": draft, "at": opens, "variants": variants})
		expect(t, status, body, http.StatusOK, "")
		for _, entry := range body["sections"].([]any) {
			if entry.(map[string]any)["section"] != "featured" {
				continue
			}
			var shown []string
			for _, offer := range entry.(map[string]any)["offers"].([]any) {
				offer := offer.(map[string]any)
				shown = append(shown, fmt.Sprint(offer["id"], " ", offer["name_key"], " ", offer["variant"]))
			}
			return strings.Join(shown, ", ")
		}
		t.Fatalf("no featured in %v", body)
		return ""
	}

	if got := featured(nil); got != "offer.weekend_pack shop.offer.weekend_pack <nil>, offer.weekend_pack_two shop.offer.weekend_pack_two <nil>" {
		t.Fatalf("as written: %s", got)
	}
	if got := featured(map[string]string{"offer.weekend_pack_two": "top"}); got != "offer.weekend_pack_two shop.offer.weekend_pack_top top, offer.weekend_pack shop.offer.weekend_pack <nil>" {
		t.Fatalf("variant top: %s", got)
	}
	for name, variants := range map[string]map[string]string{
		"unknown variant": {"offer.weekend_pack_two": "bold"},
		"unknown offer":   {"offer.missing": "top"},
		"no variants":     {"offer.weekend_pack": "top"},
	} {
		status, body := a.call("POST", "/admin/config/shop/preview", support, map[string]any{"catalog": draft, "at": opens, "variants": variants})
		if status != http.StatusBadRequest || body["error"] != "invalid" {
			t.Fatalf("%s: %d %v", name, status, body)
		}
	}
}

func TestTheShopCatalogCarriesTheRotationRulesOfTheServer(t *testing.T) {
	a := newAPIWith(t, func(c *adminapi.Config) {
		c.Shop = shop.Config{DailySlots: 6, WeeklySlots: 3, ResetHourUTC: 4, ReferencePriceDays: 14}
	})
	a.create("leyla@example.com", rbac.LiveOps)
	status, body := a.call("GET", "/admin/config/shop", a.signIn("leyla"), nil)
	want := map[string]any{"daily_slots": float64(6), "weekly_slots": float64(3), "reset_hour_utc": float64(4)}
	if status != http.StatusOK || !reflect.DeepEqual(body["rotation"], want) || body["reference_price_days"] != float64(14) {
		t.Fatalf("got %d %v %v, want rotation %v and 14 reference price days", status, body["rotation"], body["reference_price_days"], want)
	}
}

func TestPublishRefusesAReferencePriceThatIsNotTheLowestEarlierPrice(t *testing.T) {
	a := newAPI(t)
	a.create("leyla@example.com", rbac.LiveOps)
	liveOps := a.signIn("leyla")
	a.publishShopItems()
	a.nameOffers()
	discounted := func(price, reference int) map[string]any {
		offer := cometFox()
		offer["priceCoins"], offer["referencePriceCoins"] = price, reference
		return catalog(offer)
	}
	status, body := a.call("PUT", "/admin/config/shop", liveOps, map[string]any{"catalog": discounted(300, 400)})
	if status != http.StatusBadRequest || body["error"] != "invalid" || !strings.Contains(body["detail"].(string), "offer.avatar_comet_fox has no earlier price") {
		t.Fatalf("reference before any published price %d %v", status, body)
	}
	if status, body = a.call("PUT", "/admin/config/shop", liveOps, map[string]any{"catalog": catalog(cometFox())}); status != http.StatusOK {
		t.Fatalf("full price %d %v", status, body)
	}
	status, body = a.call("PUT", "/admin/config/shop", liveOps, map[string]any{"catalog": discounted(300, 500)})
	if status != http.StatusBadRequest || !strings.Contains(body["detail"].(string), "reference price 500 is not its lowest price 400") {
		t.Fatalf("invented reference %d %v", status, body)
	}
	if status, body = a.call("PUT", "/admin/config/shop", liveOps, map[string]any{"catalog": discounted(300, 400)}); status != http.StatusOK {
		t.Fatalf("honest reference %d %v", status, body)
	}
}

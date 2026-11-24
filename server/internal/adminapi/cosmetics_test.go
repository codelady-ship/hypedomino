package adminapi_test

import (
	"context"
	"encoding/json"
	"image"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/adminapi"
	"github.com/Alievs-corp/hypedomino/server/internal/contentpublish"
	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/gamestrings"
	"github.com/Alievs-corp/hypedomino/server/internal/objectstore"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

func cosmeticItem(catalog map[string]any, id string) map[string]any {
	return listed(catalog, "items", id)
}

func foxAvatar() map[string]any {
	return map[string]any{
		"id": "avatar.fox", "type": "avatar", "rarityId": "rare", "animated": true, "availability": "live",
		"sources": []any{map[string]any{"type": "shop", "ref": "offer.fox"}}, "artSource": "handmade", "culturalReview": "approved",
	}
}

func (a *api) nameItems(status gamestrings.Status, items ...any) {
	a.t.Helper()
	store := gamestrings.NewStore(a.pool)
	for _, item := range items {
		key := "cosmetic." + item.(map[string]any)["id"].(string) + ".name"
		if err := store.SaveEntry(context.Background(), gamestrings.Entry{Key: key, Note: "Cosmetic item name.", MaxLength: 40}, "content-1", start); err != nil {
			a.t.Fatal(err)
		}
		if err := store.SetValue(context.Background(), key, gamestrings.English, gamestrings.Value{Text: "Item", Status: status}, "content-1", start); err != nil {
			a.t.Fatal(err)
		}
	}
}

func (a *api) passCosmeticArt(items ...any) {
	a.t.Helper()
	for _, item := range items {
		id := item.(map[string]any)["id"].(string)
		address := "cosmetic/" + strings.Replace(id, ".", "/", 1) + "/static"
		_, err := a.pool.Exec(context.Background(), `insert into cosmetic_assets (id, item_id, slot, address, object_key, status, uploaded_by, created_at)
			values (gen_random_uuid(), $1, 'static', $2, $2 || '/' || gen_random_uuid() || '.png', 'passed', 'art', $3)`, id, address, start)
		if err != nil {
			a.t.Fatal(err)
		}
	}
}

func TestCosmeticsPublishNeedsAnApprovedEnglishName(t *testing.T) {
	a := newAPI(t)
	content := a.signIn("ayla")
	_, body := a.call("GET", "/admin/config/cosmetics", content, nil)
	builtin := body["rules"].(map[string]any)["builtin"].(map[string]any)["items"].([]any)
	catalog := map[string]any{"items": append(append([]any{}, builtin...), foxAvatar())}
	a.nameItems(gamestrings.Approved, builtin...)
	a.passCosmeticArt(foxAvatar())

	steps := []struct {
		name   string
		status gamestrings.Status
		want   int
	}{
		{"no value", "", http.StatusBadRequest},
		{"a draft value", gamestrings.Draft, http.StatusBadRequest},
		{"an approved value", gamestrings.Approved, http.StatusOK},
	}
	for _, step := range steps {
		if step.status != "" {
			a.nameItems(step.status, foxAvatar())
		}
		status, body := a.call("PUT", "/admin/config/cosmetics", content, map[string]any{"catalog": catalog, "reason": "fox avatar"})
		if step.want == http.StatusOK {
			expect(t, status, body, http.StatusOK, "")
			continue
		}
		if status != step.want || body["error"] != "invalid" || !strings.Contains(body["detail"].(string), "V-16 no approved en name for avatar.fox") || !refusedKeys(body, cosmetics.NameKeyOf("avatar.fox")) {
			t.Errorf("%s: %d %v", step.name, status, body)
		}
	}
}

func TestCosmeticsPublishStampsItemsAndKeepsThemForever(t *testing.T) {
	a := newAPI(t)
	a.create("leyla@example.com", rbac.LiveOps)
	content, liveOps := a.signIn("ayla"), a.signIn("leyla")

	status, body := a.call("GET", "/admin/config/cosmetics", liveOps, nil)
	rules, _ := body["rules"].(map[string]any)
	if status != http.StatusOK || body["catalog"] != nil || len(rules["types"].([]any)) != 5 || len(rules["source_types"].([]any)) != 9 {
		t.Fatalf("empty %d %v", status, body)
	}
	builtin := rules["builtin"].(map[string]any)
	status, body = a.call("PUT", "/admin/config/cosmetics", liveOps, map[string]any{"catalog": builtin, "reason": "no"})
	expect(t, status, body, http.StatusForbidden, "forbidden")

	first := map[string]any{"items": append(append([]any{}, builtin["items"].([]any)...), foxAvatar())}
	a.nameItems(gamestrings.Approved, first["items"].([]any)...)
	a.passCosmeticArt(foxAvatar())
	status, body = a.call("PUT", "/admin/config/cosmetics", content, map[string]any{"catalog": first, "reason": "fox avatar"})
	expect(t, status, body, http.StatusOK, "")
	published := body["catalog"].(map[string]any)
	fox := cosmeticItem(published, "avatar.fox")
	if published["manifestVersion"] != builtin["manifestVersion"].(float64)+1 || fox["revision"] != 1.0 || fox["nameKey"] != "cosmetic.avatar.fox.name" || fox["animationAddress"] != "cosmetic/avatar/fox/anim" {
		t.Fatalf("stamped %v", published)
	}
	starter := cosmeticItem(published, "avatar.starter")
	if starter["revision"] != cosmeticItem(builtin, "avatar.starter")["revision"] {
		t.Fatalf("an unchanged builtin item keeps its revision: %v", starter)
	}

	refused := []struct {
		name   string
		change func(map[string]any)
		detail string
	}{
		{"a price", func(c map[string]any) { cosmeticItem(c, "avatar.fox")["price"] = 5 }, "V-1"},
		{"a pending review", func(c map[string]any) { cosmeticItem(c, "avatar.fox")["culturalReview"] = "pending" }, "V-1"},
		{"an unknown rarity", func(c map[string]any) { cosmeticItem(c, "avatar.fox")["rarityId"] = "ultra" }, "V-14"},
		{"a new type", func(c map[string]any) { cosmeticItem(c, "avatar.fox")["type"] = "frame" }, "frame"},
		{"a removed item", func(c map[string]any) { c["items"] = without(c, "items", "avatar.fox")["items"] }, "V-12"},
		{"a builtin default removed", func(c map[string]any) { c["items"] = without(c, "items", "avatar.starter")["items"] }, "default"},
	}
	for _, c := range refused {
		var catalog map[string]any
		raw, _ := json.Marshal(published)
		if err := json.Unmarshal(raw, &catalog); err != nil {
			t.Fatal(err)
		}
		c.change(catalog)
		status, body = a.call("PUT", "/admin/config/cosmetics", content, map[string]any{"catalog": catalog, "reason": "refused"})
		if status != http.StatusBadRequest || body["error"] != "invalid" || !strings.Contains(body["detail"].(string), c.detail) {
			t.Errorf("%s: %d %v", c.name, status, body)
		}
	}

	cosmeticItem(published, "avatar.fox")["rarityId"] = "epic"
	status, body = a.call("PUT", "/admin/config/cosmetics", content, map[string]any{"catalog": published, "reason": "rarer fox"})
	expect(t, status, body, http.StatusOK, "")
	if got := cosmeticItem(body["catalog"].(map[string]any), "avatar.fox"); got["revision"] != 2.0 || got["rarityId"] != "epic" {
		t.Fatalf("edited %v", got)
	}

	status, body = a.call("GET", "/admin/config/cosmetics/history", content, nil)
	versions := body["versions"].([]any)
	if status != http.StatusOK || len(versions) != 2 {
		t.Fatalf("history %d %v", status, body)
	}
	earlier := versions[1].(map[string]any)["config_version"]
	status, body = a.call("POST", "/admin/config/cosmetics/rollback", content, map[string]any{"version": earlier, "reason": "back to rare"})
	expect(t, status, body, http.StatusOK, "")
	rolled := body["catalog"].(map[string]any)
	if got := cosmeticItem(rolled, "avatar.fox"); got["revision"] != 3.0 || got["rarityId"] != "rare" {
		t.Fatalf("rollback %v", got)
	}

	entries := a.entries(a.signIn("root"), "object_type=cosmetic_items")
	if len(entries) != 3 {
		t.Fatalf("audit %v", entries)
	}
}

func cosmeticOnSale(id string, fields map[string]any) map[string]any {
	kind, slug, _ := strings.Cut(id, ".")
	item := map[string]any{
		"id": id, "type": kind, "rarityId": "rare", "availability": "live",
		"sources": []any{map[string]any{"type": "shop", "ref": "offer." + slug}}, "artSource": "handmade", "culturalReview": "approved",
	}
	for key, value := range fields {
		item[key] = value
	}
	return item
}

func TestEveryCosmeticTypeIsCreatedEditedAndPublished(t *testing.T) {
	a := newAPI(t)
	content := a.signIn("ayla")
	_, body := a.call("GET", "/admin/config/cosmetics", content, nil)
	builtin := body["rules"].(map[string]any)["builtin"].(map[string]any)["items"].([]any)
	layers := []any{map[string]any{"name": "back", "animated": false}, map[string]any{"name": "waves", "animated": true}}
	created := []any{
		cosmeticOnSale("avatar.fox", map[string]any{"animated": true}),
		cosmeticOnSale("frame.ember", map[string]any{"animated": true}),
		cosmeticOnSale("table.reef", map[string]any{"animated": true, "layers": layers}),
		cosmeticOnSale("tile.storm", map[string]any{"animated": true}),
		cosmeticOnSale("emoji.wave", map[string]any{"animated": false, "group": "basic"}),
	}
	catalog := map[string]any{"items": append(append([]any{}, builtin...), created...)}
	a.nameItems(gamestrings.Approved, catalog["items"].([]any)...)
	a.passCosmeticArt(created...)

	status, body := a.call("PUT", "/admin/config/cosmetics", content, map[string]any{"catalog": catalog, "reason": "one item of every type"})
	expect(t, status, body, http.StatusOK, "")
	published := body["catalog"].(map[string]any)
	for _, entry := range created {
		id := entry.(map[string]any)["id"].(string)
		item, base := cosmeticItem(published, id), "cosmetic/"+strings.Replace(id, ".", "/", 1)
		if item["revision"] != 1.0 || item["staticAddress"] != base+"/static" {
			t.Errorf("created %v", item)
		}
	}
	for _, id := range []string{"avatar.fox", "frame.ember", "tile.storm"} {
		if item := cosmeticItem(published, id); item["animationAddress"] != "cosmetic/"+strings.Replace(id, ".", "/", 1)+"/anim" {
			t.Errorf("animated %v", item)
		}
	}
	reef := cosmeticItem(published, "table.reef")["layers"].([]any)
	if len(reef) != 2 || reef[0].(map[string]any)["address"] != "cosmetic/table/reef/layer/back" || reef[1].(map[string]any)["address"] != "cosmetic/table/reef/layer/waves" {
		t.Errorf("table layers %v", reef)
	}
	if wave := cosmeticItem(published, "emoji.wave"); wave["group"] != "basic" || wave["animationAddress"] != nil {
		t.Errorf("static emoji %v", wave)
	}

	for _, entry := range created {
		cosmeticItem(published, entry.(map[string]any)["id"].(string))["rarityId"] = "epic"
	}
	status, body = a.call("PUT", "/admin/config/cosmetics", content, map[string]any{"catalog": published, "reason": "every type rarer"})
	expect(t, status, body, http.StatusOK, "")
	edited := body["catalog"].(map[string]any)
	for _, entry := range created {
		if item := cosmeticItem(edited, entry.(map[string]any)["id"].(string)); item["revision"] != 2.0 || item["rarityId"] != "epic" {
			t.Errorf("edited %v", item)
		}
	}
}

func TestCosmeticsPublishKeepsTheSourcesOtherDomainsGrantFrom(t *testing.T) {
	a := newAPI(t)
	a.publishExampleCatalog()
	a.create("leyla@example.com", rbac.LiveOps)
	content, liveOps := a.signIn("ayla"), a.signIn("leyla")

	status, body := a.call("PUT", "/admin/config/rating", liveOps, map[string]any{"catalog": exampleAt(t, "ranking/examples/rating"), "reason": "rank rewards"})
	expect(t, status, body, http.StatusOK, "")

	status, body = a.call("GET", "/admin/config/cosmetics", content, nil)
	expect(t, status, body, http.StatusOK, "")
	catalog := body["catalog"].(map[string]any)
	frame := cosmeticItem(catalog, "frame.season_1_top")
	frame["sources"] = []any{map[string]any{"type": "event", "ref": "event.other"}}
	status, body = a.call("PUT", "/admin/config/cosmetics", content, map[string]any{"catalog": catalog, "reason": "drop the rank source"})
	if status != http.StatusBadRequest || body["error"] != "invalid" || !strings.Contains(body["detail"].(string), "bundle.rank_season_1_gold") {
		t.Fatalf("dropped source %d %v", status, body)
	}
}

func (a *api) uploadedBy(email, itemID string) {
	a.t.Helper()
	account, err := a.accounts.ByEmail(context.Background(), email)
	if err != nil {
		a.t.Fatal(err)
	}
	_, err = a.pool.Exec(context.Background(), `insert into cosmetic_assets (id, item_id, slot, address, object_key, status, uploaded_by, created_at)
		values (gen_random_uuid(), $1, 'static', $2, $2 || '/x.png', 'passed', $3, $4)`, itemID, "cosmetic/"+itemID+"/static", account.ID, start)
	if err != nil {
		a.t.Fatal(err)
	}
}

func TestTheUploaderOfANewItemsArtCannotPublishItsReview(t *testing.T) {
	a := newAPI(t)
	content, root := a.signIn("ayla"), a.signIn("root")
	a.uploadedBy("ayla@example.com", "avatar.fox")
	_, body := a.call("GET", "/admin/config/cosmetics", content, nil)
	builtin := body["rules"].(map[string]any)["builtin"].(map[string]any)
	catalog := map[string]any{"items": append(append([]any{}, builtin["items"].([]any)...), foxAvatar())}
	a.nameItems(gamestrings.Approved, catalog["items"].([]any)...)

	status, body := a.call("PUT", "/admin/config/cosmetics", content, map[string]any{"catalog": catalog, "reason": "my own fox"})
	expect(t, status, body, http.StatusForbidden, "own_upload")
	status, body = a.call("PUT", "/admin/config/cosmetics", root, map[string]any{"catalog": catalog, "reason": "reviewed fox"})
	expect(t, status, body, http.StatusOK, "")
	published := body["catalog"].(map[string]any)
	cosmeticItem(published, "avatar.fox")["rarityId"] = "epic"
	status, body = a.call("PUT", "/admin/config/cosmetics", content, map[string]any{"catalog": published, "reason": "rarer fox"})
	expect(t, status, body, http.StatusOK, "")
}

func (a *api) publishFox(token string) (int, map[string]any) {
	a.t.Helper()
	_, body := a.call("GET", "/admin/config/cosmetics", token, nil)
	builtin := body["rules"].(map[string]any)["builtin"].(map[string]any)
	catalog := map[string]any{"items": append(append([]any{}, builtin["items"].([]any)...), foxAvatar())}
	a.nameItems(gamestrings.Approved, catalog["items"].([]any)...)
	a.passCosmeticArt(foxAvatar())
	return a.call("PUT", "/admin/config/cosmetics", token, map[string]any{"catalog": catalog, "reason": "fox avatar"})
}

func TestEveryPublishedManifestIsUploadedUnderItsHash(t *testing.T) {
	a := newAPI(t)
	content := a.signIn("ayla")
	status, body := a.publishFox(content)
	expect(t, status, body, http.StatusOK, "")
	published := body["catalog"].(map[string]any)
	cosmeticItem(published, "avatar.fox")["rarityId"] = "epic"
	status, body = a.call("PUT", "/admin/config/cosmetics", content, map[string]any{"catalog": published, "reason": "rarer fox"})
	expect(t, status, body, http.StatusOK, "")
	_, history := a.call("GET", "/admin/config/cosmetics/history", content, nil)
	earlier := history["versions"].([]any)[1].(map[string]any)["config_version"]
	status, body = a.call("POST", "/admin/config/cosmetics/rollback", content, map[string]any{"version": earlier, "reason": "back to rare"})
	expect(t, status, body, http.StatusOK, "")

	versions := map[float64]bool{}
	for key, file := range a.content.files {
		if key != cosmetics.File(cosmetics.Hash(file.Body)) || file.ContentType != "application/json" || file.CacheControl != contentpublish.ImmutableCacheControl {
			t.Fatalf("file %s %s %s", key, file.ContentType, file.CacheControl)
		}
		var manifest map[string]any
		if err := json.Unmarshal(file.Body, &manifest); err != nil {
			t.Fatal(err)
		}
		versions[manifest["manifestVersion"].(float64)] = true
	}
	latest := body["catalog"].(map[string]any)["manifestVersion"].(float64)
	if len(versions) != 3 || !versions[latest] || !versions[latest-1] || !versions[latest-2] {
		t.Fatalf("uploaded versions %v, latest %v", versions, latest)
	}
}

func TestAManifestTheContentStoreRefusesIsNotPublished(t *testing.T) {
	cases := map[string]struct {
		adjust func(*adminapi.Config, *contentStore)
		want   string
	}{
		"no content store":      {func(c *adminapi.Config, _ *contentStore) { c.Content = nil }, "content_store_off"},
		"content store offline": {func(_ *adminapi.Config, store *contentStore) { store.offline = true }, "content_store_unavailable"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			store := &contentStore{files: map[string]objectstore.Object{}}
			a := newAPIWith(t, func(config *adminapi.Config) {
				config.Content = store
				c.adjust(config, store)
			})
			content := a.signIn("ayla")
			status, body := a.publishFox(content)
			expect(t, status, body, http.StatusServiceUnavailable, c.want)
			if _, body := a.call("GET", "/admin/config/cosmetics", content, nil); body["catalog"] != nil {
				t.Fatalf("published %v", body["catalog"])
			}
		})
	}
}

func TestLiveOpsSwitchesOffAnItemForAnIncident(t *testing.T) {
	a := newAPI(t)
	a.create("leyla@example.com", rbac.LiveOps)
	content, liveOps := a.signIn("ayla"), a.signIn("leyla")
	_, body := a.call("GET", "/admin/config/cosmetics", content, nil)
	builtin := body["rules"].(map[string]any)["builtin"].(map[string]any)
	catalog := map[string]any{"items": append(append([]any{}, builtin["items"].([]any)...), foxAvatar())}
	a.nameItems(gamestrings.Approved, catalog["items"].([]any)...)
	a.passCosmeticArt(foxAvatar())
	status, body := a.call("PUT", "/admin/config/cosmetics", content, map[string]any{"catalog": catalog, "reason": "fox avatar"})
	expect(t, status, body, http.StatusOK, "")
	a.nameItems(gamestrings.Draft, foxAvatar())

	path := "/admin/cosmetics/items/avatar.fox/switch-off"
	incident := map[string]any{"reason": "offensive art reported", "ref": "SEV2-17"}
	status, body = a.call("POST", path, content, incident)
	expect(t, status, body, http.StatusForbidden, "forbidden")
	status, body = a.call("POST", path, liveOps, map[string]any{"reason": "offensive art reported"})
	expect(t, status, body, http.StatusBadRequest, "invalid")
	status, body = a.call("POST", path, liveOps, map[string]any{"ref": "SEV2-17"})
	expect(t, status, body, http.StatusBadRequest, "reason_required")
	status, body = a.call("POST", "/admin/cosmetics/items/avatar.nobody/switch-off", liveOps, incident)
	expect(t, status, body, http.StatusNotFound, "not_found")

	status, body = a.call("POST", path, liveOps, incident)
	expect(t, status, body, http.StatusOK, "")
	if fox := cosmeticItem(body["catalog"].(map[string]any), "avatar.fox"); fox["availability"] != "disabled" || fox["revision"] != 2.0 {
		t.Fatalf("switched off %v", fox)
	}
	entries := a.entries(a.signIn("root"), "object_type=cosmetic_items&action=retire")
	if len(entries) != 1 || entries[0]["ref"] != "SEV2-17" || entries[0]["reason"] != "offensive art reported" {
		t.Fatalf("audit %v", entries)
	}
	a.now = start.Add(15 * time.Minute)
	status, body = a.call("POST", path, liveOps, incident)
	expect(t, status, body, http.StatusForbidden, "step_up_required")
}

func TestANewItemIsPublishedOnlyOnceItsStaticPassed(t *testing.T) {
	cases := []struct {
		name    string
		prepare func(t *testing.T, a *api, store *memoryStore, token string)
		status  int
		refusal string
	}{
		{"no upload", func(*testing.T, *api, *memoryStore, string) {}, http.StatusConflict, "cosmetic_art_missing"},
		{"awaiting its file", func(t *testing.T, a *api, _ *memoryStore, token string) { startUpload(t, a, token) }, http.StatusConflict, "cosmetic_art_missing"},
		{"rejected", func(t *testing.T, a *api, store *memoryStore, token string) {
			checkFoxStatic(t, a, store, token, image.NewGray(image.Rect(0, 0, 256, 256)))
		}, http.StatusConflict, "cosmetic_art_missing"},
		{"an upload of another item", func(t *testing.T, a *api, store *memoryStore, token string) {
			status, body := a.call("POST", "/admin/assets/uploads", token, map[string]any{"item_id": "avatar.owl", "slot": "static"})
			expect(t, status, body, http.StatusOK, "")
			store.upload(body["upload_url"].(string), encodedPNG(t, image.NewNRGBA(image.Rect(0, 0, 256, 256))))
			status, body = a.call("POST", "/admin/assets/"+body["asset"].(map[string]any)["id"].(string)+"/check", token, nil)
			expect(t, status, body, http.StatusOK, "")
		}, http.StatusConflict, "cosmetic_art_missing"},
		{"passed", func(t *testing.T, a *api, store *memoryStore, token string) {
			checkFoxStatic(t, a, store, token, image.NewNRGBA(image.Rect(0, 0, 256, 256)))
		}, http.StatusOK, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, store := newAssetAPI(t)
			content, root := a.signIn("ayla"), a.signIn("root")
			c.prepare(t, a, store, root)
			_, body := a.call("GET", "/admin/config/cosmetics", content, nil)
			builtin := body["rules"].(map[string]any)["builtin"].(map[string]any)
			catalog := map[string]any{"items": append(append([]any{}, builtin["items"].([]any)...), foxAvatar())}
			a.nameItems(gamestrings.Approved, catalog["items"].([]any)...)
			status, body := a.call("PUT", "/admin/config/cosmetics", content, map[string]any{"catalog": catalog, "reason": "fox avatar"})
			expect(t, status, body, c.status, c.refusal)
			if detail, _ := body["detail"].(string); c.status != http.StatusOK && !strings.HasSuffix(detail, ": avatar.fox") {
				t.Fatalf("detail %v", body)
			}
		})
	}
}

func TestAPublishedItemIsEditedWithoutANewUpload(t *testing.T) {
	a := newAPI(t)
	content := a.signIn("ayla")
	status, body := a.publishFox(content)
	expect(t, status, body, http.StatusOK, "")
	if _, err := a.pool.Exec(context.Background(), `delete from cosmetic_assets where item_id = 'avatar.fox'`); err != nil {
		t.Fatal(err)
	}
	published := body["catalog"].(map[string]any)
	cosmeticItem(published, "avatar.fox")["rarityId"] = "epic"
	status, body = a.call("PUT", "/admin/config/cosmetics", content, map[string]any{"catalog": published, "reason": "rarer fox"})
	expect(t, status, body, http.StatusOK, "")
}

func checkFoxStatic(t *testing.T, a *api, store *memoryStore, token string, picture image.Image) {
	t.Helper()
	id, uploadURL := startUpload(t, a, token)
	store.upload(uploadURL, encodedPNG(t, picture))
	status, body := a.call("POST", "/admin/assets/"+id+"/check", token, nil)
	expect(t, status, body, http.StatusOK, "")
}

package cosmetics

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func manifestOf(t *testing.T, items ...map[string]any) Manifest {
	t.Helper()
	m := Manifest{SchemaVersion: SchemaVersion, ManifestVersion: 2}
	for _, item := range items {
		raw, err := json.Marshal(item)
		if err != nil {
			t.Fatal(err)
		}
		m.Items = append(m.Items, raw)
	}
	return m
}

func avatar() map[string]any {
	return map[string]any{
		"id": "avatar.fox", "type": "avatar", "revision": 1, "nameKey": "cosmetic.avatar.fox.name", "rarityId": "rare",
		"animated": true, "staticAddress": "cosmetic/avatar/fox/static", "animationAddress": "cosmetic/avatar/fox/anim",
		"availability": "live", "sources": []any{map[string]any{"type": "shop", "ref": "offer.fox"}}, "artSource": "handmade", "culturalReview": "approved",
	}
}

func TestPublishedExamplesAndTheBuiltinManifestAreStrictlyValid(t *testing.T) {
	if err := ValidateManifest(exampleManifest(t), DefaultRarities()); err != nil {
		t.Fatal(err)
	}
	if err := ValidateManifest(BuiltinManifest(), DefaultRarities()); err != nil {
		t.Fatal(err)
	}
}

func TestValidateManifestRefusesWhatTheSchemaRefuses(t *testing.T) {
	table := func() map[string]any {
		item := avatar()
		delete(item, "animationAddress")
		item["id"], item["type"], item["nameKey"], item["staticAddress"] = "table.reef", "table", "cosmetic.table.reef.name", "cosmetic/table/reef/static"
		item["layers"] = []any{map[string]any{"name": "back", "address": "cosmetic/table/reef/layer/back", "animated": true}}
		return item
	}
	emoji := func() map[string]any {
		item := avatar()
		item["id"], item["type"], item["nameKey"], item["staticAddress"], item["animationAddress"] = "emoji.wink", "emoji", "cosmetic.emoji.wink.name", "cosmetic/emoji/wink/static", "cosmetic/emoji/wink/anim"
		item["group"] = "faces"
		return item
	}
	for name, item := range map[string]map[string]any{"avatar": avatar(), "table": table(), "emoji": emoji()} {
		if err := ValidateManifest(manifestOf(t, item), DefaultRarities()); err != nil {
			t.Errorf("valid %s: %v", name, err)
		}
	}
	cases := []struct {
		name   string
		item   func() map[string]any
		change func(map[string]any)
		rule   string
	}{
		{"a price field", avatar, func(i map[string]any) { i["price"] = 100 }, "V-1"},
		{"an unknown source type", avatar, func(i map[string]any) { i["sources"] = []any{map[string]any{"type": "gift", "ref": "x"}} }, "V-1"},
		{"a bad source ref", avatar, func(i map[string]any) { i["sources"] = []any{map[string]any{"type": "shop", "ref": "Bad Ref"}} }, "V-1"},
		{"a price on a source", avatar, func(i map[string]any) { i["sources"] = []any{map[string]any{"type": "shop", "ref": "x", "price": 1}} }, "V-1"},
		{"an unknown art source", avatar, func(i map[string]any) { i["artSource"] = "stolen" }, "V-1"},
		{"a pending review", avatar, func(i map[string]any) { i["culturalReview"] = "pending" }, "V-1"},
		{"no animated flag", avatar, func(i map[string]any) { delete(i, "animated") }, "V-1"},
		{"animated without an animation", avatar, func(i map[string]any) { delete(i, "animationAddress") }, "V-3"},
		{"static with an animation", avatar, func(i map[string]any) { i["animated"] = false }, "V-3"},
		{"a model on an avatar", avatar, func(i map[string]any) { i["modelAddress"] = "cosmetic/avatar/fox/model" }, "V-1"},
		{"clips on an avatar", avatar, func(i map[string]any) { i["clips"] = map[string]any{"idle": "cosmetic/avatar/fox/clip/idle"} }, "V-1"},
		{"an animated table without layers", table, func(i map[string]any) { delete(i, "layers") }, "V-3"},
		{"no animated layer", table, func(i map[string]any) {
			i["layers"] = []any{map[string]any{"name": "back", "address": "cosmetic/table/reef/layer/back", "animated": false}}
		}, "V-5"},
		{"layer names repeated", table, func(i map[string]any) {
			layer := map[string]any{"name": "back", "address": "cosmetic/table/reef/layer/back", "animated": true}
			i["layers"] = []any{layer, layer}
		}, "V-5"},
		{"an emoji without a group", emoji, func(i map[string]any) { delete(i, "group") }, "V-6"},
		{"a group on an avatar", avatar, func(i map[string]any) { i["group"] = "faces" }, "V-6"},
		{"another name key", avatar, func(i map[string]any) { i["nameKey"] = "cosmetic.avatar.other.name" }, "V-9"},
		{"another address", avatar, func(i map[string]any) { i["staticAddress"] = "cosmetic/avatar/other/static" }, "V-10"},
		{"an unknown rarity", avatar, func(i map[string]any) { i["rarityId"] = "ultra" }, "V-14"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			item := c.item()
			c.change(item)
			err := ValidateManifest(manifestOf(t, item), DefaultRarities())
			if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), c.rule) {
				t.Fatalf("err = %v, want %s", err, c.rule)
			}
		})
	}
}

func decodedItems(t *testing.T, m Manifest) map[string]map[string]any {
	t.Helper()
	items := map[string]map[string]any{}
	for _, raw := range m.Items {
		var fields map[string]any
		if err := json.Unmarshal(raw, &fields); err != nil {
			t.Fatal(err)
		}
		items[fields["id"].(string)] = fields
	}
	return items
}

func submit(t *testing.T, items ...map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"items": items})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestStampDerivesTheServerFieldsAndRaisesRevisionsOnce(t *testing.T) {
	fresh := func() map[string]any {
		item := avatar()
		for _, field := range []string{"revision", "nameKey", "staticAddress", "animationAddress"} {
			delete(item, field)
		}
		return item
	}
	previous := manifestOf(t, avatar())
	previous.ManifestVersion = 7

	stamped, err := Stamp(previous, submit(t, fresh()))
	if err != nil {
		t.Fatal(err)
	}
	item := decodedItems(t, stamped)["avatar.fox"]
	if stamped.ManifestVersion != 8 || stamped.SchemaVersion != SchemaVersion || item["revision"] != 1.0 {
		t.Fatalf("an unchanged item keeps its revision: %d %v", stamped.ManifestVersion, item)
	}
	if item["nameKey"] != "cosmetic.avatar.fox.name" || item["staticAddress"] != "cosmetic/avatar/fox/static" || item["animationAddress"] != "cosmetic/avatar/fox/anim" {
		t.Fatalf("derived %v", item)
	}

	changed := fresh()
	changed["rarityId"] = "epic"
	changed["revision"] = 99
	stamped, err = Stamp(previous, submit(t, changed))
	if err != nil {
		t.Fatal(err)
	}
	if got := decodedItems(t, stamped)["avatar.fox"]["revision"]; got != 2.0 {
		t.Fatalf("a changed item takes the next revision, whatever was sent: %v", got)
	}
}

func TestStampGivesEveryTypeItsAddresses(t *testing.T) {
	table := map[string]any{"id": "table.reef", "type": "table", "animated": true, "layers": []any{map[string]any{"name": "back", "animated": true}}}
	still := map[string]any{"id": "frame.plain", "type": "frame", "animated": false}
	tile := map[string]any{"id": "tile.storm", "type": "tile", "animated": true}
	stamped, err := Stamp(manifestOf(t, avatar()), submit(t, table, still, tile))
	if err != nil {
		t.Fatal(err)
	}
	items := decodedItems(t, stamped)
	if layer := items["table.reef"]["layers"].([]any)[0].(map[string]any); layer["address"] != "cosmetic/table/reef/layer/back" {
		t.Errorf("table %v", layer)
	}
	if items["tile.storm"]["animationAddress"] != "cosmetic/tile/storm/anim" || items["tile.storm"]["staticAddress"] != "cosmetic/tile/storm/static" {
		t.Errorf("tile %v", items["tile.storm"])
	}
	if _, has := items["frame.plain"]["animationAddress"]; has || items["frame.plain"]["revision"] != 1.0 {
		t.Errorf("static frame %v", items["frame.plain"])
	}
}

func TestStampRefusesAnItemWithoutAValidID(t *testing.T) {
	for _, id := range []string{"hat.red", "character.ali", "action.wave"} {
		if _, err := Stamp(manifestOf(t, avatar()), submit(t, map[string]any{"id": id})); !errors.Is(err, ErrInvalid) {
			t.Fatalf("%s: err = %v", id, err)
		}
	}
	if _, err := Stamp(manifestOf(t, avatar()), []byte(`{`)); !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v", err)
	}
}

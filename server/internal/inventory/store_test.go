package inventory_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/inventory"
	"github.com/Alievs-corp/hypedomino/server/internal/profile"
)

var start = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

func catalog(t *testing.T) *cosmetics.Catalog {
	t.Helper()
	c, err := cosmetics.Load("../../../docs/cosmetics/examples/catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func player(t *testing.T, pool *pgxpool.Pool, nickname string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `insert into accounts default values returning id`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if nickname != "" {
		if _, err := profile.NewStore(pool, profile.DefaultConfig()).SetNickname(context.Background(), id, nickname, start); err != nil {
			t.Fatal(err)
		}
	}
	return id
}

var (
	shopFox   = cosmetics.Source{Type: "shop", Ref: "offer.avatar_comet_fox"}
	passWiz   = cosmetics.Source{Type: "pass", Ref: "pass.season_1.premium.level_30"}
	promoNite = cosmetics.Source{Type: "promo", Ref: "campaign.launch_week"}
)

func TestDefaultsAreOwnedByEveryoneAndGrantsNeedAListedSource(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := inventory.NewStore(pool, catalog(t), inventory.DefaultConfig())
	ctx := context.Background()
	ayla := player(t, pool, "Ayla")
	owned, err := store.Owned(ctx, ayla)
	if err != nil || len(owned) != 5 || owned[0].ID != "avatar.starter" || owned[0].Source.Type != cosmetics.SourceDefault || !owned[0].GrantedAt.IsZero() {
		t.Fatalf("fresh inventory %+v %v", owned, err)
	}
	if owns, err := store.Owns(ctx, ayla, "table.starter"); err != nil || !owns {
		t.Fatalf("default table owned %v %v", owns, err)
	}
	if owns, err := store.Owns(ctx, ayla, "avatar.comet_fox"); err != nil || owns {
		t.Fatalf("unowned fox %v %v", owns, err)
	}
	if _, err := store.Grant(ctx, ayla, "avatar.nope", shopFox, start); !errors.Is(err, inventory.ErrUnknownItem) {
		t.Fatalf("unknown item granted: %v", err)
	}
	if _, err := store.Grant(ctx, ayla, "avatar.comet_fox", cosmetics.Source{Type: "mission", Ref: "mission.x"}, start); !errors.Is(err, inventory.ErrSourceNotListed) {
		t.Fatalf("unlisted source granted: %v", err)
	}
	if _, err := store.Grant(ctx, ayla, "avatar.starter", cosmetics.Source{Type: cosmetics.SourceDefault}, start); !errors.Is(err, inventory.ErrDefaultItem) {
		t.Fatalf("default granted explicitly: %v", err)
	}
	granted, err := store.Grant(ctx, ayla, "avatar.comet_fox", shopFox, start)
	if err != nil || !granted {
		t.Fatalf("grant %v %v", granted, err)
	}
	if granted, err := store.Grant(ctx, ayla, "avatar.comet_fox", shopFox, start.Add(time.Hour)); err != nil || granted {
		t.Fatalf("second grant of an owned item %v %v", granted, err)
	}
	owned, err = store.Owned(ctx, ayla)
	if err != nil || len(owned) != 6 || owned[5].ID != "avatar.comet_fox" || owned[5].Type != "avatar" || owned[5].Source != shopFox || !owned[5].GrantedAt.Equal(start) {
		t.Fatalf("inventory after grant %+v %v", owned, err)
	}
	if owns, err := store.Owns(ctx, ayla, "avatar.comet_fox"); err != nil || !owns {
		t.Fatalf("fox owned %v %v", owns, err)
	}
}

func TestEquipAcceptsOnlyOwnedLiveItemsOfTheSlotType(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := inventory.NewStore(pool, catalog(t), inventory.Config{EmojiSlots: 2, ActionSlots: 1})
	ctx := context.Background()
	ayla, fresh := player(t, pool, "Ayla"), player(t, pool, "")
	for _, c := range []struct {
		slot string
		ids  []string
		err  error
	}{
		{"hat", []string{"avatar.starter"}, inventory.ErrUnknownSlot},
		{"avatar", nil, inventory.ErrTooMany},
		{"avatar", []string{"avatar.starter", "avatar.comet_fox"}, inventory.ErrTooMany},
		{"emoji", []string{"emoji.smile", "emoji.smile"}, inventory.ErrTooMany},
		{"emoji", []string{"emoji.smile", "emoji.laugh_spin", "emoji.smile"}, inventory.ErrTooMany},
		{"avatar", []string{"avatar.nope"}, inventory.ErrUnknownItem},
		{"avatar", []string{"frame.starter"}, inventory.ErrWrongType},
		{"avatar", []string{"avatar.comet_fox"}, inventory.ErrNotOwned},
		{"frame", []string{"frame.withdrawn"}, inventory.ErrDisabled},
		{"emoji", []string{"emoji.laugh_spin"}, inventory.ErrNotOwned},
	} {
		if _, err := store.Equip(ctx, ayla, c.slot, c.ids, start); !errors.Is(err, c.err) {
			t.Fatalf("equip %s %v: %v, want %v", c.slot, c.ids, err, c.err)
		}
	}
	if _, err := store.Equip(ctx, fresh, "avatar", []string{"avatar.starter"}, start); !errors.Is(err, inventory.ErrNoProfile) {
		t.Fatalf("equip before setup: %v", err)
	}
	if _, err := store.Grant(ctx, ayla, "avatar.comet_fox", shopFox, start); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Grant(ctx, ayla, "frame.withdrawn", cosmetics.Source{Type: "shop", Ref: "offer.frame_withdrawn"}, start); err != nil {
		t.Fatal(err)
	}
	loadout, err := store.Equip(ctx, ayla, "avatar", []string{"avatar.comet_fox"}, start)
	if err != nil || loadout.Avatar != "avatar.comet_fox" || loadout.Frame != "" {
		t.Fatalf("equip owned avatar %+v %v", loadout, err)
	}
	if _, err := store.Equip(ctx, ayla, "frame", []string{"frame.withdrawn"}, start); !errors.Is(err, inventory.ErrDisabled) {
		t.Fatalf("owned but disabled item equipped: %v", err)
	}
	loadout, err = store.Equip(ctx, ayla, "emoji", []string{"emoji.smile"}, start)
	if err != nil || len(loadout.Emojis) != 1 || loadout.Avatar != "avatar.comet_fox" {
		t.Fatalf("equip emoji %+v %v", loadout, err)
	}
	loadout, err = store.Equip(ctx, ayla, "emoji", nil, start)
	if err != nil || len(loadout.Emojis) != 0 {
		t.Fatalf("clear emojis %+v %v", loadout, err)
	}
	p, _, err := profile.NewStore(pool, profile.DefaultConfig()).Get(ctx, ayla)
	if err != nil || p.Loadout.Avatar != "avatar.comet_fox" {
		t.Fatalf("stored loadout %+v %v", p.Loadout, err)
	}
}

func TestFreezeFallsBackToDefaultsAndRevokeUnequips(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := inventory.NewStore(pool, catalog(t), inventory.DefaultConfig())
	ctx := context.Background()
	ayla, fresh := player(t, pool, "Ayla"), player(t, pool, "")
	frozen, err := store.Freeze(ctx, fresh)
	if err != nil || frozen.Avatar != "avatar.starter" || frozen.Frame != "frame.starter" || frozen.Character != "character.starter" || frozen.Table != "table.starter" || len(frozen.Emojis) != 0 {
		t.Fatalf("freeze without a profile %+v %v", frozen, err)
	}
	for id, source := range map[string]cosmetics.Source{"avatar.comet_fox": shopFox, "character.tile_wizard": passWiz, "table.night_market": promoNite} {
		if _, err := store.Grant(ctx, ayla, id, source, start); err != nil {
			t.Fatal(id, err)
		}
	}
	for slot, id := range map[string]string{"avatar": "avatar.comet_fox", "character": "character.tile_wizard", "table": "table.night_market"} {
		if _, err := store.Equip(ctx, ayla, slot, []string{id}, start); err != nil {
			t.Fatal(slot, err)
		}
	}
	if _, err := pool.Exec(ctx, `update profiles set loadout = loadout || '{"frame":"frame.withdrawn","emojis":["emoji.smile","emoji.laugh_spin","emoji.smile"],"actions":["action.victory_dance"]}' where account_id = $1`, ayla); err != nil {
		t.Fatal(err)
	}
	frozen, err = store.Freeze(ctx, ayla)
	if err != nil || frozen.Avatar != "avatar.comet_fox" || frozen.Character != "character.tile_wizard" || frozen.Table != "table.night_market" || frozen.Frame != "frame.starter" || len(frozen.Emojis) != 1 || frozen.Emojis[0] != "emoji.smile" || len(frozen.Actions) != 0 {
		t.Fatalf("frozen loadout %+v %v", frozen, err)
	}
	if err := store.Revoke(ctx, ayla, "avatar.starter", start); !errors.Is(err, inventory.ErrDefaultItem) {
		t.Fatalf("default revoked: %v", err)
	}
	if err := store.Revoke(ctx, ayla, "frame.season_1_top", start); !errors.Is(err, inventory.ErrNotOwned) {
		t.Fatalf("unowned revoked: %v", err)
	}
	if err := store.Revoke(ctx, ayla, "avatar.comet_fox", start.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := store.Revoke(ctx, ayla, "avatar.comet_fox", start.Add(time.Hour)); !errors.Is(err, inventory.ErrNotOwned) {
		t.Fatalf("revoked twice: %v", err)
	}
	p, _, err := profile.NewStore(pool, profile.DefaultConfig()).Get(ctx, ayla)
	if err != nil || p.Loadout.Avatar != "avatar.starter" || p.Loadout.Character != "character.tile_wizard" || p.Loadout.Frame != "frame.starter" {
		t.Fatalf("loadout after revoke %+v %v", p.Loadout, err)
	}
	owned, err := store.Owned(ctx, ayla)
	if err != nil || len(owned) != 7 {
		t.Fatalf("inventory after revoke %+v %v", owned, err)
	}
	if granted, err := store.Grant(ctx, ayla, "avatar.comet_fox", cosmetics.Source{Type: "event", Ref: "event.spring_cup"}, start.Add(2*time.Hour)); err != nil || !granted {
		t.Fatalf("regrant after revoke %v %v", granted, err)
	}
	if owns, err := store.Owns(ctx, ayla, "avatar.comet_fox"); err != nil || !owns {
		t.Fatalf("owned after regrant %v %v", owns, err)
	}
}

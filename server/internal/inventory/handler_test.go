package inventory_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/inventory"
	"github.com/Alievs-corp/hypedomino/server/internal/profile"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

type fakeVerifier struct{}

func (fakeVerifier) Verify(access string) (token.Claims, error) {
	switch {
	case access == "expired":
		return token.Claims{}, token.ErrExpired
	case strings.HasPrefix(access, "access:"):
		return token.Claims{Account: strings.TrimPrefix(access, "access:")}, nil
	}
	return token.Claims{}, token.ErrSignature
}

type fakeInventory struct {
	loadouts map[string]profile.Loadout
	equips   []string
}

func (f *fakeInventory) Owned(_ context.Context, account string) ([]inventory.Owned, error) {
	if account == "broken" {
		return nil, errors.New("database down")
	}
	return []inventory.Owned{
		{ID: "avatar.starter", Type: "avatar", Source: cosmetics.Source{Type: "default"}},
		{ID: "avatar.comet_fox", Type: "avatar", Source: cosmetics.Source{Type: "shop", Ref: "offer.avatar_comet_fox"}, GrantedAt: start},
	}, nil
}

func (f *fakeInventory) Equip(_ context.Context, account, slot string, ids []string, _ time.Time) (profile.Loadout, error) {
	f.equips = append(f.equips, account+" "+slot+" "+strings.Join(ids, ","))
	switch {
	case slot == "hat":
		return profile.Loadout{}, inventory.ErrUnknownSlot
	case len(ids) > 1:
		return profile.Loadout{}, inventory.ErrTooMany
	case len(ids) == 1 && ids[0] == "avatar.nope":
		return profile.Loadout{}, inventory.ErrUnknownItem
	case len(ids) == 1 && ids[0] == "avatar.locked":
		return profile.Loadout{}, inventory.ErrNotOwned
	case len(ids) == 1 && ids[0] == "frame.starter":
		return profile.Loadout{}, inventory.ErrWrongType
	case len(ids) == 1 && ids[0] == "avatar.gone":
		return profile.Loadout{}, inventory.ErrDisabled
	case account == "new":
		return profile.Loadout{}, inventory.ErrNoProfile
	case account == "broken":
		return profile.Loadout{}, errors.New("database down")
	}
	l := f.loadouts[account]
	l.Avatar = ids[0]
	f.loadouts[account] = l
	return l, nil
}

func (f *fakeInventory) Freeze(_ context.Context, account string) (profile.Loadout, error) {
	l := f.loadouts[account]
	if l.Avatar == "" {
		l.Avatar = "avatar.starter"
	}
	return l, nil
}

func (f *fakeInventory) EmojiSlots() int {
	return 6
}

type observedAccounts struct {
	accounts []string
}

func (o *observedAccounts) Observe(_ context.Context, account string) {
	o.accounts = append(o.accounts, account)
}

type fixture struct {
	t         *testing.T
	server    *httptest.Server
	inventory *fakeInventory
	unowned   *observedAccounts
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	inv := &fakeInventory{loadouts: map[string]profile.Loadout{}}
	handler := inventory.NewHandler(fakeVerifier{}, inv, slog.New(slog.NewTextHandler(io.Discard, nil)), func() time.Time { return start })
	unowned := &observedAccounts{}
	handler.FlagUnowned(unowned)
	mux := http.NewServeMux()
	mux.Handle("/inventory", handler)
	mux.Handle("/inventory/", handler)
	f := &fixture{t: t, server: httptest.NewServer(mux), inventory: inv, unowned: unowned}
	t.Cleanup(f.server.Close)
	return f
}

func (f *fixture) call(method, path, bearer, body string) (int, map[string]any) {
	f.t.Helper()
	request, err := http.NewRequest(method, f.server.URL+path, strings.NewReader(body))
	if err != nil {
		f.t.Fatal(err)
	}
	if bearer != "" {
		request.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		f.t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.Header.Get("Cache-Control") != "no-store" {
		f.t.Fatalf("headers %v", resp.Header)
	}
	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		f.t.Fatal(err)
	}
	return resp.StatusCode, decoded
}

func TestInventoryRoutesNeedAValidAccessToken(t *testing.T) {
	f := newFixture(t)
	for _, route := range []struct{ method, path string }{{"GET", "/inventory"}, {"PUT", "/inventory/equip"}} {
		if status, body := f.call(route.method, route.path, "", `{"slot":"avatar","items":["avatar.starter"]}`); status != http.StatusUnauthorized || body["error"] != "access_rejected" {
			t.Fatalf("%s %s without a token answered %d %v", route.method, route.path, status, body)
		}
		if status, body := f.call(route.method, route.path, "expired", `{}`); status != http.StatusUnauthorized || body["error"] != "access_expired" {
			t.Fatalf("%s %s with an expired token answered %d %v", route.method, route.path, status, body)
		}
	}
}

func TestTheInventoryListsOwnedItemsAndTheEffectiveLoadout(t *testing.T) {
	f := newFixture(t)
	status, body := f.call("GET", "/inventory", "access:ayla", "")
	if status != http.StatusOK {
		t.Fatalf("inventory answered %d %v", status, body)
	}
	items := body["items"].([]any)
	first, second := items[0].(map[string]any), items[1].(map[string]any)
	if len(items) != 2 || first["id"] != "avatar.starter" || first["source_type"] != "default" || first["granted_at"] != nil || first["source_ref"] != nil {
		t.Fatalf("default item %v", first)
	}
	if second["id"] != "avatar.comet_fox" || second["source_type"] != "shop" || second["source_ref"] != "offer.avatar_comet_fox" || second["granted_at"] != "2026-10-30T12:00:00Z" {
		t.Fatalf("granted item %v", second)
	}
	if body["loadout"].(map[string]any)["avatar"] != "avatar.starter" {
		t.Fatalf("loadout %v", body["loadout"])
	}
	if body["emoji_slots"] != float64(6) {
		t.Fatalf("emoji slots %v", body["emoji_slots"])
	}
	if status, body := f.call("GET", "/inventory", "access:broken", ""); status != http.StatusInternalServerError || body["error"] != "internal" {
		t.Fatalf("failing store answered %d %v", status, body)
	}
}

func TestEquipRefusalsCarryTheirReason(t *testing.T) {
	f := newFixture(t)
	cases := []struct {
		account string
		body    string
		status  int
		reason  string
	}{
		{"ayla", `{"slot":`, http.StatusBadRequest, "malformed"},
		{"ayla", `{"slot":"hat","items":["avatar.starter"]}`, http.StatusBadRequest, "slot.unknown"},
		{"ayla", `{"slot":"avatar","items":["avatar.starter","avatar.comet_fox"]}`, http.StatusBadRequest, "slot.too_many"},
		{"ayla", `{"slot":"avatar","items":["avatar.nope"]}`, http.StatusNotFound, "item.unknown"},
		{"ayla", `{"slot":"avatar","items":["avatar.locked"]}`, http.StatusForbidden, "item.not_owned"},
		{"ayla", `{"slot":"avatar","items":["frame.starter"]}`, http.StatusConflict, "item.wrong_type"},
		{"ayla", `{"slot":"avatar","items":["avatar.gone"]}`, http.StatusConflict, "item.disabled"},
		{"new", `{"slot":"avatar","items":["avatar.starter"]}`, http.StatusConflict, "profile.setup_required"},
		{"broken", `{"slot":"avatar","items":["avatar.starter"]}`, http.StatusInternalServerError, "internal"},
	}
	for _, c := range cases {
		if status, body := f.call("PUT", "/inventory/equip", "access:"+c.account, c.body); status != c.status || body["error"] != c.reason {
			t.Fatalf("%s %s answered %d %v, want %d %s", c.account, c.body, status, body, c.status, c.reason)
		}
	}
	if len(f.unowned.accounts) != 1 || f.unowned.accounts[0] != "ayla" {
		t.Fatalf("unowned attempts observed %v, want only the not_owned refusal", f.unowned.accounts)
	}
	status, body := f.call("PUT", "/inventory/equip", "access:ayla", `{"slot":"avatar","items":["avatar.comet_fox"]}`)
	if status != http.StatusOK || body["loadout"].(map[string]any)["avatar"] != "avatar.comet_fox" {
		t.Fatalf("equip answered %d %v", status, body)
	}
	if last := f.inventory.equips[len(f.inventory.equips)-1]; last != "ayla avatar avatar.comet_fox" {
		t.Fatalf("store received %q", last)
	}
}

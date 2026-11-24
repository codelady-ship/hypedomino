package main

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/promo"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

var forgedValues = map[string]any{
	"balance": 9_999_999, "coins": 9_999_999, "coin": 9_999_999, "crystals": 9_999_999, "crystal": 9_999_999,
	"price": 0, "priceCoins": 0, "price_coins": 0, "priceCrystals": 0, "cost": 0, "amount": 9_999_999, "quantity": 100,
	"reward": map[string]any{"coins": 9_999_999, "crystals": 9_999_999}, "rewards": []any{map[string]any{"coins": 9_999_999}},
	"bundle": map[string]any{"id": "bundle.forged", "revision": 1, "coins": 9_999_999, "cosmetics": []string{"frame.golden_dragon_mythic"}},
	"grant":  map[string]any{"coins": 9_999_999}, "score": 999, "points": 999, "rating": 3000, "mmr": 3000, "lp": 999,
	"xp": 9_999_999, "level": 100, "rank": "apex", "tier": "apex", "result": "win", "winner": true,
	"inventory": []string{"frame.golden_dragon_mythic", "table.night_market"}, "owned": true, "items_owned": []string{"frame.golden_dragon_mythic"},
	"premium": true, "state": "purchased", "environment": "Production", "granted": true, "repeated": false,
}

type economyStep struct {
	name   string
	method string
	path   string
	body   func(a *attacker) map[string]any
	want   int
}

func fixed(body map[string]any) func(*attacker) map[string]any {
	return func(*attacker) map[string]any { return maps.Clone(body) }
}

func forge(body map[string]any, victim *attacker) map[string]any {
	forged := maps.Clone(body)
	for key, value := range forgedValues {
		if _, legitimate := body[key]; !legitimate {
			forged[key] = value
		}
	}
	for _, key := range []string{"account", "account_id", "sub", "player"} {
		if _, legitimate := body[key]; !legitimate {
			forged[key] = victim.account
		}
	}
	return forged
}

func (h *redTeam) multiUseCode(t *testing.T, id string) string {
	t.Helper()
	hasher, err := promo.NewHasher(h.promoKey)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	created, err := promo.NewStore(h.pool, nil, nil, hasher, promo.DefaultConfig()).Create(context.Background(), promo.Definition{
		ID: id, Campaign: "campaign.redteam", Kind: promo.MultiUse, Cap: 100, Creator: "admin.redteam",
		Bundle: reward.Bundle{ID: "bundle.redteam_promo", Revision: 1, Coins: 100}, Start: now.Add(-time.Hour), End: now.Add(24 * time.Hour),
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	return created.Plaintext
}

func (h *redTeam) write(t *testing.T, a *attacker, method, path string, body map[string]any) reply {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return h.signed(t, a, method, path, raw)
}

func TestRedTeamForgedValuesInEveryWriteAreIgnored(t *testing.T) {
	h := startRedTeam(t)
	bystander := h.attacker(t)
	bystanderBefore := h.economy(t, bystander)
	code := h.multiUseCode(t, "promo.redteam_forged")
	rounds := [][]economyStep{
		{
			{"shop purchase", http.MethodPost, "/shop/buy", fixed(map[string]any{"offer": "offer.avatar_comet_fox", "request_id": "buy-1"}), http.StatusOK},
			{"store checkout", http.MethodPost, "/shop/begin", fixed(map[string]any{"offer": "offer.crystals_tier1"}), http.StatusOK},
			{"store receipt", http.MethodPost, "/shop/complete", func(a *attacker) map[string]any {
				return map[string]any{"offer": "offer.crystals_tier1", "platform": "google", "token": "receipt-" + a.account}
			}, http.StatusOK},
			{"equip", http.MethodPut, "/inventory/equip", fixed(map[string]any{"slot": "avatar", "items": []string{"avatar.comet_fox"}}), http.StatusOK},
			{"equip unowned", http.MethodPut, "/inventory/equip", fixed(map[string]any{"slot": "table", "items": []string{"table.night_market"}}), http.StatusForbidden},
		},
		{
			{"pass levels", http.MethodPost, "/pass/levels", fixed(map[string]any{"count": 1, "currency": "coin", "request_id": "levels-1"}), http.StatusOK},
			{"daily login", http.MethodPost, "/daily-login/claim", fixed(map[string]any{}), http.StatusOK},
			{"medal showcase", http.MethodPut, "/medals/showcase", fixed(map[string]any{"medals": []string{}}), http.StatusOK},
			{"mission reroll", http.MethodPost, "/missions/reroll", fixed(map[string]any{"mission": "mission.forged"}), http.StatusConflict},
			{"promo code", http.MethodPost, "/promo/redeem", fixed(map[string]any{"code": code}), http.StatusOK},
		},
	}
	for _, steps := range rounds {
		honest, forger := h.attacker(t), h.attacker(t)
		for _, a := range []*attacker{honest, forger} {
			h.fund(t, a, wallet.Coin, startingCoins)
			h.fund(t, a, wallet.Crystal, 500)
			h.play.sell("receipt-"+a.account, "crystals_tier1", a.account, false)
		}
		for _, step := range steps {
			h.compare(t, step, honest, forger, bystander)
		}
	}
	h.unchanged(t, bystander, bystanderBefore, "values naming another account")
}

func (h *redTeam) compare(t *testing.T, step economyStep, honest, forger, bystander *attacker) {
	t.Helper()
	plain := h.write(t, honest, step.method, step.path, step.body(honest))
	if plain.status != step.want {
		t.Fatalf("%s: the plain request answered %d %v, want %d", step.name, plain.status, plain.body, step.want)
	}
	forgerBefore := h.economy(t, forger)
	forged := h.write(t, forger, step.method, step.path, forge(step.body(forger), bystander))
	if forged.status == http.StatusBadRequest && plain.status != forged.status {
		h.unchanged(t, forger, forgerBefore, step.name+" with forged values refused as "+forged.code())
		forged = h.write(t, forger, step.method, step.path, step.body(forger))
	}
	if plain.status != forged.status || plain.code() != forged.code() {
		t.Fatalf("%s: plain answered %d %s, forged answered %d %s", step.name, plain.status, plain.code(), forged.status, forged.code())
	}
	if honestState, forgedState := h.economy(t, honest), h.economy(t, forger); honestState != forgedState {
		t.Fatalf("%s: forged values changed the outcome:\nplain\n%s\nforged\n%s", step.name, honestState, forgedState)
	}
}

func TestRedTeamSpendsAreBoundedIdempotentAndSigned(t *testing.T) {
	h := startRedTeam(t)

	t.Run("pass level counts outside the season", func(t *testing.T) {
		for _, count := range []int{0, -1, math.MinInt, 1000, 1 << 62, math.MaxInt - 1, math.MaxInt} {
			a := h.attacker(t)
			h.fund(t, a, wallet.Coin, startingCoins)
			h.fund(t, a, wallet.Crystal, 500)
			before := h.economy(t, a)
			for _, currency := range []string{"coin", "crystal"} {
				r := h.write(t, a, http.MethodPost, "/pass/levels", map[string]any{"count": count, "currency": currency, "request_id": "count-" + currency})
				if r.status != http.StatusConflict || r.code() != "pass.level_count" {
					t.Fatalf("%d levels for %s answered %d %v", count, currency, r.status, r.body)
				}
			}
			h.unchanged(t, a, before, fmt.Sprintf("%d pass levels", count))
		}
	})

	t.Run("one purchase request sent twice", func(t *testing.T) {
		a := h.attacker(t)
		h.fund(t, a, wallet.Coin, startingCoins)
		buy := map[string]any{"offer": "offer.avatar_comet_fox", "request_id": "twice-1"}
		if r := h.write(t, a, http.MethodPost, "/shop/buy", buy); r.status != http.StatusOK {
			t.Fatalf("purchase answered %d %v", r.status, r.body)
		}
		after := h.economy(t, a)
		if r := h.write(t, a, http.MethodPost, "/shop/buy", buy); r.status != http.StatusOK {
			t.Fatalf("the repeated request answered %d %v", r.status, r.body)
		}
		h.unchanged(t, a, after, "a repeated purchase request")
		buy["request_id"] = "twice-2"
		if r := h.write(t, a, http.MethodPost, "/shop/buy", buy); r.status != http.StatusConflict || r.code() != "offer.limit_reached" {
			t.Fatalf("a second purchase of a one-per-account offer answered %d %v", r.status, r.body)
		}
		h.unchanged(t, a, after, "a second purchase of a one-per-account offer")
	})

	t.Run("concurrent purchases against one balance", func(t *testing.T) {
		a := h.attacker(t)
		h.fund(t, a, wallet.Coin, 500)
		var wg sync.WaitGroup
		statuses := make(chan int, 5)
		for attempt := range 5 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				statuses <- h.write(t, a, http.MethodPost, "/shop/buy", map[string]any{"offer": "offer.chest_avatar", "request_id": fmt.Sprintf("race-%d", attempt)}).status
			}()
		}
		wg.Wait()
		close(statuses)
		granted := 0
		for status := range statuses {
			if status == http.StatusOK {
				granted++
			}
		}
		var balance, chests int64
		if err := h.pool.QueryRow(context.Background(), `select (select balance from wallets where account_id = $1 and currency = 'coin'), (select count(*) from ledger_entries where account_id = $1 and reason = 'chest_purchase')`, a.account).Scan(&balance, &chests); err != nil {
			t.Fatal(err)
		}
		if granted != 1 || chests != 1 || balance != 0 {
			t.Fatalf("five concurrent purchases with coins for one: %d granted, %d chests charged, balance %d", granted, chests, balance)
		}
	})

	t.Run("writes without a valid request signature", func(t *testing.T) {
		a, victim := h.attacker(t), h.attacker(t)
		h.fund(t, a, wallet.Coin, startingCoins)
		before := h.economy(t, a)
		cheap, err := json.Marshal(map[string]any{"offer": "offer.avatar_comet_fox", "request_id": "mac-1"})
		if err != nil {
			t.Fatal(err)
		}
		expensive, err := json.Marshal(map[string]any{"offer": "offer.table_night_market", "request_id": "mac-1"})
		if err != nil {
			t.Fatal(err)
		}
		attempts := map[string]string{
			"no signature":               "",
			"signature of another body":  token.SignRequest(a.pair.RequestKey, http.MethodPost, "/shop/buy", a.pair.AccessToken, cheap),
			"signature of another path":  token.SignRequest(a.pair.RequestKey, http.MethodPost, "/shop/begin", a.pair.AccessToken, expensive),
			"signature with another key": token.SignRequest(victim.pair.RequestKey, http.MethodPost, "/shop/buy", a.pair.AccessToken, expensive),
			"signature of another token": token.SignRequest(a.pair.RequestKey, http.MethodPost, "/shop/buy", victim.pair.AccessToken, expensive),
		}
		for name, mac := range attempts {
			if r := h.call(t, a, http.MethodPost, "/shop/buy", expensive, mac); r.status != http.StatusUnauthorized || r.code() != "bad_mac" {
				t.Fatalf("%s answered %d %v", name, r.status, r.body)
			}
		}
		h.unchanged(t, a, before, "writes without a valid request signature")
		h.awaitSignal(t, a, "mac_failure")
		if r := h.call(t, &attacker{address: a.address}, http.MethodPost, "/shop/buy", expensive, ""); r.status != http.StatusUnauthorized {
			t.Fatalf("a write without an access token answered %d %v", r.status, r.body)
		}
	})
}

func TestRedTeamFloodsAreRateLimited(t *testing.T) {
	h := startRedTeam(t)

	t.Run("sign-in from one address", func(t *testing.T) {
		limited := 0
		for range 10 {
			if r := h.call(t, &attacker{address: "203.0.113.7"}, http.MethodPost, "/auth/signin", []byte(`{"provider":"google","id_token":"forged"}`), ""); r.status == http.StatusTooManyRequests && r.code() == "rate_limited" {
				limited++
			}
		}
		if limited == 0 {
			t.Fatal("ten sign-ins from one address in a row were never rate limited")
		}
	})

	t.Run("requests from one account", func(t *testing.T) {
		a := h.attacker(t)
		limited := 0
		for range 40 {
			if r := h.call(t, a, http.MethodGet, "/wallet", nil, ""); r.status == http.StatusTooManyRequests {
				limited++
			}
		}
		if limited == 0 {
			t.Fatal("forty requests from one account in a row were never rate limited")
		}
	})

	t.Run("spending from one account", func(t *testing.T) {
		a := h.attacker(t)
		h.fund(t, a, wallet.Coin, startingCoins)
		before := h.economy(t, a)
		limited := 0
		for attempt := range 10 {
			r := h.write(t, a, http.MethodPost, "/shop/buy", map[string]any{"offer": "offer.unknown", "request_id": fmt.Sprintf("flood-%d", attempt)})
			if r.status == http.StatusTooManyRequests {
				limited++
			}
		}
		if limited == 0 {
			t.Fatal("ten purchases from one account in a row were never rate limited")
		}
		h.unchanged(t, a, before, "a purchase flood")
	})
}

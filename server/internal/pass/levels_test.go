package pass_test

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/matchstore"
	"github.com/Alievs-corp/hypedomino/server/internal/pass"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

func TestBoughtLevelsKeepProgressInsideTheLevelAndChargeOnce(t *testing.T) {
	f := newFixture(t)
	account := f.account(t)
	if _, err := f.wallet.Apply(context.Background(), wallet.Entry{Account: account, Amount: 1000, Reason: wallet.Promo, Source: wallet.Source{Type: "promo", Ref: "campaign.test"}, Key: "seed"}, now); err != nil {
		t.Fatal(err)
	}
	loss := pass.Match{ID: "match-loss", Account: account, Mode: rules.Mode1v1, Kind: matchstore.KindRanked, EndedAt: now}
	if _, err := f.store.AwardMatch(context.Background(), loss); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		purchase, err := f.store.BuyLevels(context.Background(), account, 3, wallet.Coin, "request-1", now)
		if err != nil {
			t.Fatal(err)
		}
		if purchase.Level != 4 || purchase.XP != 360 || purchase.Currency != wallet.Coin || purchase.Price != 600 || purchase.Balance != 20+1000-600+20 {
			t.Fatalf("purchase = %+v", purchase)
		}
	}
	if got := f.balance(t, account); got != 440 {
		t.Fatalf("balance = %d, want 440", got)
	}
	cases := []struct {
		name    string
		count   int
		request string
		want    error
	}{
		{"zero levels", 0, "request-2", pass.ErrLevelCount},
		{"past the last level", 27, "request-3", pass.ErrLevelCount},
		{"a count that overflows the level", math.MaxInt, "request-5", pass.ErrLevelCount},
		{"a count that wraps the price to a credit", math.MaxInt - 1, "request-6", pass.ErrLevelCount},
		{"not enough coins", 3, "request-4", wallet.ErrInsufficient},
		{"no request id", 1, "", pass.ErrRequest},
	}
	for _, c := range cases {
		if _, err := f.store.BuyLevels(context.Background(), account, c.count, wallet.Coin, c.request, now); !errors.Is(err, c.want) {
			t.Fatalf("%s: %v", c.name, err)
		}
	}
	if got := f.xp(t, account); got != 360 {
		t.Fatalf("xp after refusals = %d", got)
	}
}

func TestLevelsCanBeBoughtWithCrystals(t *testing.T) {
	f := newFixture(t)
	account := f.account(t)
	if _, err := f.wallet.Apply(context.Background(), wallet.Entry{Account: account, Currency: wallet.Crystal, Amount: 100, Reason: wallet.Purchase, Source: wallet.Source{Type: "store", Ref: "t1"}, Key: "purchase:google:t1"}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.BuyLevels(context.Background(), account, 3, wallet.Crystal, "request-1", now); !errors.Is(err, wallet.ErrInsufficient) {
		t.Fatalf("three levels for 100 crystals: %v", err)
	}
	purchase, err := f.store.BuyLevels(context.Background(), account, 2, wallet.Crystal, "request-2", now)
	if err != nil || purchase.Currency != wallet.Crystal || purchase.Price != 2*40 || purchase.Balance != 20 || purchase.Level != 3 {
		t.Fatalf("purchase = %+v, %v", purchase, err)
	}
	if got := f.balance(t, account); got != 20+20 {
		t.Fatalf("coins after a crystal purchase = %d, want the free rewards of levels 1 and 3 only", got)
	}
	if _, err := f.store.BuyLevels(context.Background(), account, 1, "gem", "request-3", now); !errors.Is(err, pass.ErrRequest) {
		t.Fatalf("unknown currency: %v", err)
	}
}

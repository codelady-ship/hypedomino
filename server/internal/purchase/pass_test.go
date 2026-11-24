package purchase_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/matchstore"
	"github.com/Alievs-corp/hypedomino/server/internal/pass"
	"github.com/Alievs-corp/hypedomino/server/internal/purchase"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
	"github.com/Alievs-corp/hypedomino/server/internal/shop"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

func withPass(t *testing.T, f *fixture) *pass.Store {
	t.Helper()
	watcher := pass.NewWatcher(f.pool, slog.New(slog.NewTextHandler(io.Discard, nil)))
	passes := pass.NewStore(f.pool, watcher)
	passes.SetRewards(f.engine)
	passes.SetWallet(f.wallet)
	season := pass.Season{ID: "season_1", Start: now.Add(-24 * time.Hour), End: now.Add(60 * 24 * time.Hour), Curve: pass.Curve{Levels: 30, XpBase: 100, XpStep: 1, XpMax: 100}, Free: map[int]reward.Bundle{}, Premium: map[int]reward.Bundle{}}
	for level := 1; level <= 30; level++ {
		if level%2 == 1 {
			season.Free[level] = reward.Bundle{ID: "bundle.test_free", Revision: level, Coins: 20}
		}
		season.Premium[level] = reward.Bundle{ID: "bundle.test_premium", Revision: level, Crystals: 2}
	}
	config := pass.Default()
	config.PremiumPlusLevels = 5
	config.Seasons = []pass.Season{season}
	raw, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := snapshot.Publish(context.Background(), f.pool, pass.Domain, raw); err != nil {
		t.Fatal(err)
	}
	if _, err := watcher.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	f.store.WithPass(passes)
	f.shop.WithPass(passes)
	return passes
}

func TestAPremiumPurchaseUnlocksThePremiumRewardsAlreadyReachedOnce(t *testing.T) {
	f := newFixture(t, purchase.Config{RefundsBeforePurchaseBlock: 2, RefundWindowDays: 30})
	passes := withPass(t, f)
	ctx := context.Background()
	ayla := f.account(t)
	for i := range 3 {
		match := pass.Match{ID: fmt.Sprintf("match-%d", i), Account: ayla, Mode: rules.Mode1v1, Kind: matchstore.KindRanked, Won: true, EndedAt: now}
		if _, err := passes.AwardMatch(ctx, match); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.shop.Begin(ctx, ayla, "offer.pass_premium", now); err != nil {
		t.Fatalf("checkout before premium: %v", err)
	}
	f.receipt("tok-pass", "pass_premium", ayla, purchase.StatePurchased)
	for i := range 2 {
		result, err := f.store.Complete(ctx, ayla, "offer.pass_premium", purchase.PlatformGoogle, "tok-pass", now)
		if err != nil {
			t.Fatal(err)
		}
		if result.State != purchase.RecordGranted || result.Repeated != (i == 1) || result.Crystals != 4*2+100 {
			t.Fatalf("complete %d: %+v", i, result)
		}
	}
	if _, err := f.shop.Begin(ctx, ayla, "offer.pass_premium", now); !errors.Is(err, shop.ErrOwned) {
		t.Fatalf("second checkout in the season: %v", err)
	}
	state, err := passes.Premium(ctx, ayla, now)
	if err != nil || !state.Premium || state.Level != 4 {
		t.Fatalf("premium state %+v, %v", state, err)
	}
	refund := purchase.Notice{ID: "r-pass", Kind: purchase.KindRefund, TransactionID: "tok-pass"}
	if handled, err := f.store.Notice(ctx, purchase.PlatformGoogle, refund, now.Add(time.Hour)); err != nil || !handled {
		t.Fatalf("refund %v %v", handled, err)
	}
	if coins, crystals := balances(t, f, ayla); coins != 40 || crystals != 0 {
		t.Fatalf("after the refund coins %d crystals %d, want 40 and 0", coins, crystals)
	}
	if state, err := passes.Premium(ctx, ayla, now); err != nil || state.Premium {
		t.Fatalf("premium after the refund %+v, %v", state, err)
	}
	reversal := purchase.Notice{ID: "rr-pass", Kind: purchase.KindReversal, TransactionID: "tok-pass"}
	if handled, err := f.store.Notice(ctx, purchase.PlatformGoogle, reversal, now.Add(2*time.Hour)); err != nil || !handled {
		t.Fatalf("reversal %v %v", handled, err)
	}
	if state, err := passes.Premium(ctx, ayla, now); err != nil || !state.Premium || state.Level != 4 {
		t.Fatalf("premium after the reversal %+v, %v", state, err)
	}
	if coins, crystals := balances(t, f, ayla); coins != 40 || crystals != 4*2 {
		t.Fatalf("after the reversal coins %d crystals %d, want the premium levels back", coins, crystals)
	}
	second := purchase.Notice{ID: "r-pass-2", Kind: purchase.KindRefund, TransactionID: "tok-pass"}
	if handled, err := f.store.Notice(ctx, purchase.PlatformGoogle, second, now.Add(3*time.Hour)); err != nil || !handled {
		t.Fatalf("second refund %v %v", handled, err)
	}
	if coins, crystals := balances(t, f, ayla); coins != 40 || crystals != 0 {
		t.Fatalf("after the second refund coins %d crystals %d, want 40 and 0", coins, crystals)
	}
}

func balances(t *testing.T, f *fixture, account string) (int64, int64) {
	t.Helper()
	coins, err := f.wallet.Balance(context.Background(), account, wallet.Coin)
	if err != nil {
		t.Fatal(err)
	}
	crystals, err := f.wallet.Balance(context.Background(), account, wallet.Crystal)
	if err != nil {
		t.Fatal(err)
	}
	return coins, crystals
}

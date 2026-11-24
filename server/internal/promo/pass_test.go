package promo_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/pass"
	"github.com/Alievs-corp/hypedomino/server/internal/promo"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

const premiumCrystalsPerLevel = 2

func (f *fixture) withPass(t *testing.T, start time.Time) *pass.Store {
	t.Helper()
	watcher := pass.NewWatcher(f.pool, slog.New(slog.NewTextHandler(io.Discard, nil)))
	passes := pass.NewStore(f.pool, watcher)
	passes.SetRewards(f.engine)
	passes.SetWallet(f.wallet)
	season := pass.Season{ID: "season_1", Start: start, End: start.Add(60 * 24 * time.Hour), Curve: pass.Curve{Levels: 30, XpBase: 100, XpStep: 1, XpMax: 100}, Free: map[int]reward.Bundle{}, Premium: map[int]reward.Bundle{}}
	for level := 1; level <= 30; level++ {
		season.Free[level] = reward.Bundle{ID: "bundle.test_free", Revision: level, Coins: 20}
		season.Premium[level] = reward.Bundle{ID: "bundle.test_premium", Revision: level, Crystals: premiumCrystalsPerLevel}
	}
	config := pass.Default()
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
	return passes
}

func (f *fixture) inTx(t *testing.T, run func(pgx.Tx) error) {
	t.Helper()
	ctx := context.Background()
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if err := run(tx); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func (f *fixture) balances(t *testing.T, account string) (int64, int64) {
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

func premiumDefinition(id, kind string) promo.Definition {
	d := definition(id, kind, coins)
	d.Campaign, d.PassPremium = "campaign.ideyanest_2026", true
	if kind == promo.MultiUse {
		d.Cap = 10
	}
	return d
}

func TestAPassPremiumCodeWaitsForASecondAdminAndUnlocksTheLevelsReached(t *testing.T) {
	f := newFixture(t, promo.DefaultConfig())
	passes := f.withPass(t, now.Add(-24*time.Hour))
	ctx := context.Background()
	backer := f.account(t)
	f.inTx(t, func(tx pgx.Tx) error {
		_, err := passes.AddIn(ctx, tx, backer, 300, cosmetics.Source{Type: "test", Ref: "warmup"}, "warmup", now)
		return err
	})
	before, err := passes.Premium(ctx, backer, now)
	if err != nil || before.Premium || before.Level < 2 {
		t.Fatalf("before the code %+v, %v", before, err)
	}
	coinsBefore, crystalsBefore := f.balances(t, backer)
	wantCoins := coinsBefore + coins.Coins
	if crystalsBefore != 0 {
		t.Fatalf("crystals before the code %d", crystalsBefore)
	}
	created := f.create(t, premiumDefinition("promo.backer_sky", promo.SingleUse))
	if created.Code.State != promo.StatePending || !created.Code.PassPremium {
		t.Fatalf("created %+v", created.Code)
	}
	if _, err := f.store.Redeem(ctx, backer, created.Plaintext, now); !errors.Is(err, promo.ErrNotActive) {
		t.Fatalf("pending redeemed: %v", err)
	}
	if _, err := f.store.Approve(ctx, "promo.backer_sky", "admin.alice", now); !errors.Is(err, promo.ErrApprover) {
		t.Fatalf("self approval: %v", err)
	}
	if _, err := f.store.Approve(ctx, "promo.backer_sky", "admin.bob", now); err != nil {
		t.Fatal(err)
	}
	redemption, err := f.store.Redeem(ctx, backer, created.Plaintext, now)
	if err != nil || !redemption.PassPremium {
		t.Fatalf("redemption %+v, %v", redemption, err)
	}
	after, err := passes.Premium(ctx, backer, now)
	if err != nil || !after.Premium || after.Level != before.Level {
		t.Fatalf("after the code %+v, %v", after, err)
	}
	wantCrystals := int64(after.Level * premiumCrystalsPerLevel)
	if coins, crystals := f.balances(t, backer); coins != wantCoins || crystals != wantCrystals {
		t.Fatalf("coins %d crystals %d, want %d and %d", coins, crystals, wantCoins, wantCrystals)
	}
	if _, err := f.store.Redeem(ctx, backer, created.Plaintext, now.Add(time.Minute)); !errors.Is(err, promo.ErrAlreadyRedeemed) {
		t.Fatalf("repeat: %v", err)
	}
	if coins, crystals := f.balances(t, backer); coins != wantCoins || crystals != wantCrystals {
		t.Fatalf("repeat granted again: coins %d crystals %d", coins, crystals)
	}
	stored, err := f.store.Get(ctx, "promo.backer_sky")
	if err != nil || !stored.PassPremium || stored.Redemptions != 1 {
		t.Fatalf("stored %+v, %v", stored, err)
	}
}

func TestAPassPremiumBatchWaitsForASecondAdmin(t *testing.T) {
	f := newFixture(t, promo.DefaultConfig())
	created, err := f.store.Batch(context.Background(), premiumDefinition("promo.backers", promo.SingleUse), 3, now)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range created {
		if c.Code.State != promo.StatePending || !c.Code.PassPremium {
			t.Fatalf("batch code %+v", c.Code)
		}
	}
	if _, err := f.store.ApproveBatch(context.Background(), "promo.backers", "admin.alice"); !errors.Is(err, promo.ErrApprover) {
		t.Fatalf("self approval: %v", err)
	}
	listed, err := f.store.List(context.Background(), promo.ListQuery{Campaign: "campaign.ideyanest_2026", Limit: 10})
	if err != nil || len(listed) != 3 || !listed[0].PassPremium {
		t.Fatalf("listed %+v, %v", listed, err)
	}
}

func TestAPassPremiumCodeRefusedForThePassStaysRedeemableAndNeverLocks(t *testing.T) {
	f := newFixture(t, promo.Config{CodeLength: 12, MaxValidityDays: 90, AttemptsPerHour: 10, FailedAttemptsBeforeLock: 2, LockMinutes: 60, CoinsApprovalThreshold: 2000, LockSignalWeight: 5, SpikePercent: 100, SpikeWindowMinutes: 60})
	seasonStart := now.Add(2 * time.Hour)
	passes := f.withPass(t, seasonStart)
	ctx := context.Background()
	created := f.create(t, premiumDefinition("promo.backer_storm", promo.MultiUse))
	if _, err := f.store.Approve(ctx, "promo.backer_storm", "admin.bob", now); err != nil {
		t.Fatal(err)
	}
	early, owner := f.account(t), f.account(t)
	inSeason := seasonStart.Add(time.Hour)
	f.inTx(t, func(tx pgx.Tx) error {
		_, err := passes.UnlockPremiumIn(ctx, tx, owner, "season_1", "purchase:google:tok-1", inSeason)
		return err
	})
	ownerCoins, ownerCrystals := f.balances(t, owner)
	cases := []struct {
		name    string
		account string
		at      time.Time
		want    error
	}{
		{"no season yet", early, now, promo.ErrNoPassSeason},
		{"no season yet again", early, now.Add(time.Minute), promo.ErrNoPassSeason},
		{"no season yet a third time", early, now.Add(2 * time.Minute), promo.ErrNoPassSeason},
		{"premium owned", owner, inSeason, promo.ErrPassOwned},
		{"premium owned again", owner, inSeason.Add(time.Minute), promo.ErrPassOwned},
		{"premium owned a third time", owner, inSeason.Add(2 * time.Minute), promo.ErrPassOwned},
	}
	for _, c := range cases {
		if _, err := f.store.Redeem(ctx, c.account, created.Plaintext, c.at); !errors.Is(err, c.want) {
			t.Fatalf("%s: %v", c.name, err)
		}
	}
	if got := f.redemptions(t, "promo.backer_storm"); got != 0 {
		t.Fatalf("redemptions after refusals %d", got)
	}
	if coins, crystals := f.balances(t, early); coins != 0 || crystals != 0 {
		t.Fatalf("refused account got coins %d crystals %d", coins, crystals)
	}
	if coins, crystals := f.balances(t, owner); coins != ownerCoins || crystals != ownerCrystals {
		t.Fatalf("owner got coins %d crystals %d, had %d and %d", coins, crystals, ownerCoins, ownerCrystals)
	}
	for _, account := range []string{early, owner} {
		if count, _, _ := f.lockSignal(t, account); count != 0 {
			t.Fatalf("lock signals for %s: %d", account, count)
		}
	}
	if _, err := f.store.Redeem(ctx, early, created.Plaintext, inSeason); err != nil {
		t.Fatalf("redeem once the season runs: %v", err)
	}
	state, err := passes.Premium(ctx, early, inSeason)
	if err != nil || !state.Premium {
		t.Fatalf("premium %+v, %v", state, err)
	}
	if got := f.redemptions(t, "promo.backer_storm"); got != 1 {
		t.Fatalf("redemptions %d", got)
	}
}

package pass_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/inventory"
	"github.com/Alievs-corp/hypedomino/server/internal/pass"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

var now = time.Date(2026, 12, 10, 12, 0, 0, 0, time.UTC)

const premiumItem = "avatar.tile_wizard"

type fixture struct {
	pool    *pgxpool.Pool
	watcher *snapshot.Watcher[*pass.Snapshot]
	store   *pass.Store
	wallet  *wallet.Store
	engine  *reward.Engine
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	_, pool := dbtest.Fresh(t)
	catalog, err := cosmetics.Load("../../../docs/cosmetics/examples/catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	walletStore := wallet.NewStore(pool)
	watcher := pass.NewWatcher(pool, slog.New(slog.NewTextHandler(io.Discard, nil)))
	f := &fixture{pool: pool, watcher: watcher, store: pass.NewStore(pool, watcher), wallet: walletStore}
	f.engine = reward.NewEngine(pool, walletStore, inventory.NewStore(pool, catalog, inventory.DefaultConfig()), nil, f.store, reward.DefaultConfig())
	f.store.SetRewards(f.engine)
	f.store.SetWallet(walletStore)
	f.publish(t, testSeason("season_1", now.Add(-10*24*time.Hour), now.Add(80*24*time.Hour)))
	return f
}

func testSeason(id string, start, end time.Time) pass.Season {
	season := pass.Season{
		ID:      id,
		Start:   start,
		End:     end,
		Curve:   pass.Curve{Levels: 30, XpBase: 100, XpStep: 1, XpMax: 100},
		Free:    map[int]reward.Bundle{},
		Premium: map[int]reward.Bundle{},
	}
	for level := 1; level <= 30; level++ {
		if level%2 == 1 {
			season.Free[level] = reward.Bundle{ID: "bundle.test_free", Revision: level, Coins: 20}
		}
		season.Premium[level] = reward.Bundle{ID: "bundle.test_premium", Revision: level, Crystals: 2}
	}
	season.Premium[30] = reward.Bundle{ID: "bundle.test_premium", Revision: 30, Crystals: 2, Cosmetics: []string{premiumItem}}
	return season
}

func (f *fixture) publish(t *testing.T, seasons ...pass.Season) {
	t.Helper()
	config := pass.Default()
	config.PremiumPlusLevels = 5
	config.Seasons = seasons
	raw, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pass.Parse(raw); err != nil {
		t.Fatal(err)
	}
	if _, err := snapshot.Publish(context.Background(), f.pool, pass.Domain, raw); err != nil {
		t.Fatal(err)
	}
	if _, err := f.watcher.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func (f *fixture) account(t *testing.T) string {
	t.Helper()
	var id string
	if err := f.pool.QueryRow(context.Background(), `insert into accounts (created_at) values ($1) returning id`, now.Add(-time.Hour)).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func (f *fixture) balance(t *testing.T, account string) int64 {
	t.Helper()
	balance, err := f.wallet.Balance(context.Background(), account, wallet.Coin)
	if err != nil {
		t.Fatal(err)
	}
	return balance
}

func (f *fixture) count(t *testing.T, query string, args ...any) int {
	t.Helper()
	var n int
	if err := f.pool.QueryRow(context.Background(), query, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestEnteringTheSeasonStartsAtLevelOneAndGrantsItsFreeRewardOnce(t *testing.T) {
	f := newFixture(t)
	account := f.account(t)
	for range 2 {
		season, progress, version, err := f.store.Enter(context.Background(), account, now)
		if err != nil {
			t.Fatal(err)
		}
		if season.ID != "season_1" || progress.XP != 0 || progress.Premium || version < 1 {
			t.Fatalf("season %s, progress %+v, version %d", season.ID, progress, version)
		}
	}
	if got := f.balance(t, account); got != 20 {
		t.Fatalf("balance = %d, want 20", got)
	}
	if got := f.count(t, `select count(*) from reward_grants where account_id = $1 and idempotency_key = 'pass:season_1:free:1'`, account); got != 1 {
		t.Fatalf("level 1 grants = %d", got)
	}
}

func TestTheViewPricesLevelsFromThePublishedConfig(t *testing.T) {
	f := newFixture(t)
	account := f.account(t)
	view, err := f.store.View(context.Background(), account, now)
	if err != nil {
		t.Fatal(err)
	}
	if view.Season != "season_1" || view.Level != 1 || view.LevelCoinPrice != 200 || view.LevelCrystalPrice != 40 {
		t.Fatalf("view %+v", view)
	}
	config := pass.Default()
	config.LevelCoinPrice, config.LevelCrystalPrice = 250, 45
	config.Seasons = []pass.Season{testSeason("season_1", now.Add(-10*24*time.Hour), now.Add(80*24*time.Hour))}
	raw, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := snapshot.Publish(context.Background(), f.pool, pass.Domain, raw); err != nil {
		t.Fatal(err)
	}
	if _, err := f.watcher.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	if view, err = f.store.View(context.Background(), account, now); err != nil {
		t.Fatal(err)
	}
	if view.LevelCoinPrice != 250 || view.LevelCrystalPrice != 45 || view.ConfigVersion < 2 {
		t.Fatalf("view after publish %+v", view)
	}
	if _, err := f.store.View(context.Background(), account, now.Add(81*24*time.Hour)); !errors.Is(err, pass.ErrNoSeason) {
		t.Fatalf("View after the season = %v", err)
	}
}

func TestThereIsNoPassOutsideASeason(t *testing.T) {
	f := newFixture(t)
	account := f.account(t)
	for _, at := range []time.Time{now.Add(-11 * 24 * time.Hour), now.Add(81 * 24 * time.Hour)} {
		if _, _, _, err := f.store.Enter(context.Background(), account, at); !errors.Is(err, pass.ErrNoSeason) {
			t.Fatalf("Enter(%v) = %v", at, err)
		}
	}
}

func TestAPublishedSeasonIsVisibleWithoutARestart(t *testing.T) {
	f := newFixture(t)
	account := f.account(t)
	later := now.Add(100 * 24 * time.Hour)
	if _, _, _, err := f.store.Enter(context.Background(), account, later); !errors.Is(err, pass.ErrNoSeason) {
		t.Fatalf("Enter before publish = %v", err)
	}
	f.publish(t, testSeason("season_1", now.Add(-10*24*time.Hour), now.Add(80*24*time.Hour)), testSeason("season_2", now.Add(90*24*time.Hour), now.Add(180*24*time.Hour)))
	season, _, _, err := f.store.Enter(context.Background(), account, later)
	if err != nil || season.ID != "season_2" {
		t.Fatalf("Enter after publish = %s, %v", season.ID, err)
	}
}

func TestTheSeasonEndGrantsEveryMissingLevelOnEveryUnlockedTrackOnce(t *testing.T) {
	f := newFixture(t)
	premium := f.account(t)
	free := f.account(t)
	for _, account := range []string{premium, free} {
		if _, _, _, err := f.store.Enter(context.Background(), account, now); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.pool.Exec(context.Background(), `update pass_progress set xp_total = 2900, premium_ref = 'purchase:test', premium_at = $2 where account_id = $1`, premium, now); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(context.Background(), `update pass_progress set xp_total = 450 where account_id = $1`, free); err != nil {
		t.Fatal(err)
	}
	end := now.Add(80 * 24 * time.Hour)
	if closed, err := f.store.CloseDue(context.Background(), end.Add(-time.Second)); err != nil || len(closed) != 0 {
		t.Fatalf("CloseDue before the end = %v, %v", closed, err)
	}
	closed, err := f.store.CloseDue(context.Background(), end)
	if err != nil {
		t.Fatal(err)
	}
	if len(closed) != 1 || closed[0].Season != "season_1" || closed[0].Accounts != 2 || closed[0].Granted != 14+30+2 {
		t.Fatalf("closed = %+v", closed)
	}
	if got, crystals := f.balance(t, premium), f.crystals(t, premium); got != 15*20 || crystals != 30*2 {
		t.Fatalf("premium coins %d crystals %d", got, crystals)
	}
	if got := f.balance(t, free); got != 3*20 {
		t.Fatalf("free balance = %d", got)
	}
	if got := f.count(t, `select count(*) from inventory_items where account_id = $1 and item_id = $2 and source_ref = 'pass.season_1.premium.level_30'`, premium, premiumItem); got != 1 {
		t.Fatalf("premium item rows = %d", got)
	}
	again, err := f.store.CloseDue(context.Background(), end.Add(time.Hour))
	if err != nil || len(again) != 0 {
		t.Fatalf("second CloseDue = %v, %v", again, err)
	}
	if got, crystals := f.balance(t, premium), f.crystals(t, premium); got != 15*20 || crystals != 30*2 {
		t.Fatalf("premium coins %d crystals %d after a second run", got, crystals)
	}
	if got := f.count(t, `select accounts from pass_season_ends where season_id = 'season_1'`); got != 2 {
		t.Fatalf("season end accounts = %d", got)
	}
}

func (f *fixture) crystals(t *testing.T, account string) int64 {
	t.Helper()
	balance, err := f.wallet.Balance(context.Background(), account, wallet.Crystal)
	if err != nil {
		t.Fatal(err)
	}
	return balance
}

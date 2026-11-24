package dailylogin_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/dailylogin"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/inventory"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

var start = time.Date(2026, 11, 3, 23, 0, 0, 0, time.UTC)

func TestADayIsClaimedOnceAndAMissedDayPausesTheCycle(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	ctx := context.Background()
	catalog, err := cosmetics.Load("../../../docs/cosmetics/examples/catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	walletStore := wallet.NewStore(pool)
	engine := reward.NewEngine(pool, walletStore, inventory.NewStore(pool, catalog, inventory.DefaultConfig()), nil, nil, reward.DefaultConfig())
	store := dailylogin.NewStore(pool, dailylogin.NewWatcher(pool, slog.New(slog.NewTextHandler(io.Discard, nil))), engine)
	var account string
	if err := pool.QueryRow(ctx, `insert into accounts (created_at) values ($1) returning id`, start.Add(-time.Hour)).Scan(&account); err != nil {
		t.Fatal(err)
	}
	status, err := store.Status(ctx, account, start)
	if err != nil || status.Day != 1 || status.ClaimedToday || !status.NextDayAt.Equal(time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("first status %+v, %v", status, err)
	}
	claimed, err := store.Claim(ctx, account, start)
	if err != nil || claimed.Day != 2 || !claimed.ClaimedToday || claimed.Granted.Coins != 30 {
		t.Fatalf("first claim %+v, %v", claimed, err)
	}
	if _, err := store.Claim(ctx, account, start.Add(59*time.Minute)); !errors.Is(err, dailylogin.ErrClaimed) {
		t.Fatalf("same economy day claim = %v", err)
	}
	if _, err := store.Claim(ctx, account, start.Add(time.Hour)); err != nil {
		t.Fatalf("claim after midnight UTC = %v", err)
	}
	afterGap, err := store.Claim(ctx, account, start.Add(5*24*time.Hour))
	if err != nil || afterGap.Granted.Coins != 50 || afterGap.Day != 4 {
		t.Fatalf("claim after missed days %+v, %v", afterGap, err)
	}
	if balance, err := walletStore.Balance(ctx, account, wallet.Coin); err != nil || balance != 110 {
		t.Fatalf("balance %d, %v", balance, err)
	}
}

func TestTheWeekEndsOnTheBigDayAndStartsAgain(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	ctx := context.Background()
	catalog, err := cosmetics.Load("../../../docs/cosmetics/examples/catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	walletStore := wallet.NewStore(pool)
	engine := reward.NewEngine(pool, walletStore, inventory.NewStore(pool, catalog, inventory.DefaultConfig()), nil, nil, reward.DefaultConfig())
	store := dailylogin.NewStore(pool, dailylogin.NewWatcher(pool, slog.New(slog.NewTextHandler(io.Discard, nil))), engine)
	var account string
	if err := pool.QueryRow(ctx, `insert into accounts (created_at) values ($1) returning id`, start.Add(-time.Hour)).Scan(&account); err != nil {
		t.Fatal(err)
	}
	var paid []int64
	for day := 0; day < 8; day++ {
		claimed, err := store.Claim(ctx, account, start.AddDate(0, 0, day))
		if err != nil {
			t.Fatal(err)
		}
		paid = append(paid, claimed.Granted.Coins)
	}
	if paid[6] != 150 || paid[7] != 30 {
		t.Fatalf("paid %v", paid)
	}
	if balance, err := walletStore.Balance(ctx, account, wallet.Coin); err != nil || balance != 480 {
		t.Fatalf("balance %d, %v", balance, err)
	}
}

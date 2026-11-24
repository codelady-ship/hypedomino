package matchreward_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/matchreward"
	"github.com/Alievs-corp/hypedomino/server/internal/matchstore"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

var start = time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)

func TestMatchCoinsStopAtTheDailyCapAndTheFirstWinPaysOncePerDay(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	ctx := context.Background()
	walletStore := wallet.NewStore(pool)
	store := matchreward.NewStore(pool, matchreward.NewWatcher(pool, slog.New(slog.NewTextHandler(io.Discard, nil))), walletStore)
	var account string
	if err := pool.QueryRow(ctx, `insert into accounts (created_at) values ($1) returning id`, start.Add(-time.Hour)).Scan(&account); err != nil {
		t.Fatal(err)
	}
	win := func(id string, at time.Time) matchreward.Match {
		return matchreward.Match{ID: id, Account: account, Mode: rules.Mode1v1, Kind: matchstore.KindRanked, Won: true, EndedAt: at}
	}
	first, err := store.Award(ctx, win("m0", start))
	if err != nil || first.Coins != 30 || first.FirstWin != 100 {
		t.Fatalf("first win %+v, %v", first, err)
	}
	if replay, err := store.Award(ctx, win("m0", start)); err != nil || replay.Coins != 0 || replay.FirstWin != 0 {
		t.Fatalf("replay %+v, %v", replay, err)
	}
	var total int64
	for i := 1; i <= 13; i++ {
		result, err := store.Award(ctx, win(fmt.Sprintf("m%d", i), start.Add(time.Duration(i)*time.Minute)))
		if err != nil || result.FirstWin != 0 {
			t.Fatalf("match %d %+v, %v", i, result, err)
		}
		total += result.Coins
	}
	if total != 370 {
		t.Fatalf("13 more wins paid %d, want the 370 left under the cap", total)
	}
	if custom, err := store.Award(ctx, matchreward.Match{ID: "c1", Account: account, Mode: rules.Mode1v1, Kind: matchstore.KindCustom, Won: true, EndedAt: start}); err != nil || custom != (matchreward.Result{}) {
		t.Fatalf("custom room %+v, %v", custom, err)
	}
	nextDay, err := store.Award(ctx, win("n1", start.Add(14*time.Hour)))
	if err != nil || nextDay.Coins != 30 || nextDay.FirstWin != 100 {
		t.Fatalf("next economy day %+v, %v", nextDay, err)
	}
	if balance, err := walletStore.Balance(ctx, account, wallet.Coin); err != nil || balance != 630 {
		t.Fatalf("balance %d, %v", balance, err)
	}
}

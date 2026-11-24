package risk_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/risk"
)

var waveNow = time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)

func newAccount(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var account string
	if err := pool.QueryRow(context.Background(), `insert into accounts (created_at) values (now()) returning id`).Scan(&account); err != nil {
		t.Fatal(err)
	}
	return account
}

func flag(t *testing.T, store *risk.Store, account, kind string, weight int, at time.Time) {
	t.Helper()
	dedupe := fmt.Sprintf("%s:%d:%d", kind, weight, at.UnixNano())
	if _, err := store.Record(context.Background(), risk.Signal{Account: account, Kind: kind, Weight: weight, Dedupe: dedupe, Evidence: map[string]any{}, At: at}); err != nil {
		t.Fatal(err)
	}
}

func openReviews(t *testing.T, pool *pgxpool.Pool, account string) (int, string) {
	t.Helper()
	var count int
	var cause string
	err := pool.QueryRow(context.Background(), `select count(*), coalesce(min(cause), '') from risk_reviews where account_id = $1 and closed_at is null`, account).Scan(&count, &cause)
	if err != nil {
		t.Fatal(err)
	}
	return count, cause
}

func shadowbanned(t *testing.T, pool *pgxpool.Pool, account string) bool {
	t.Helper()
	var banned bool
	if err := pool.QueryRow(context.Background(), `select shadowbanned_at is not null from accounts where id = $1`, account).Scan(&banned); err != nil {
		t.Fatal(err)
	}
	return banned
}

func TestAHardSignalOpensAReviewAtOnceWithoutASanction(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := risk.NewStore(pool)
	account := newAccount(t, pool)
	flag(t, store, account, risk.KindFakeReceipt, 1, waveNow)
	flag(t, store, account, risk.KindFakeReceipt, 1, waveNow.Add(time.Minute))
	if count, cause := openReviews(t, pool, account); count != 1 || cause != risk.CauseHardSignal {
		t.Fatalf("reviews %d cause %q", count, cause)
	}
	if shadowbanned(t, pool, account) {
		t.Fatal("a hard signal shadowbanned before a wave")
	}
	soft := newAccount(t, pool)
	flag(t, store, soft, risk.KindRefund, 100, waveNow)
	if count, _ := openReviews(t, pool, soft); count != 0 {
		t.Fatalf("a soft signal opened a review outside a wave: %d", count)
	}
}

func TestAWaveActsOnTheScoreOfTheDecayWindow(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := risk.NewStore(pool)
	config := risk.DefaultConfig()
	wave := risk.NewWave(pool, config, slog.New(slog.NewTextHandler(io.Discard, nil)))
	before := waveNow.Add(-time.Hour)

	quiet, reviewed, cracked, stale := newAccount(t, pool), newAccount(t, pool), newAccount(t, pool), newAccount(t, pool)
	flag(t, store, quiet, risk.KindReplay, config.ReviewScore-1, before)
	flag(t, store, reviewed, risk.KindReplay, config.ReviewScore, before)
	for index := range 8 {
		flag(t, store, cracked, risk.KindMacFailure, config.MacFailureWeight, before.Add(time.Duration(index)*time.Minute))
	}
	flag(t, store, stale, risk.KindReplay, config.ShadowbanScore, waveNow.Add(-config.Decay))

	result, err := wave.RunDue(context.Background(), waveNow)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ran || result.Reviewed != 2 || result.Shadowbanned != 1 {
		t.Fatalf("result %+v", result)
	}
	for account, want := range map[string]struct {
		reviews int
		banned  bool
	}{quiet: {0, false}, reviewed: {1, false}, cracked: {1, true}, stale: {0, false}} {
		count, _ := openReviews(t, pool, account)
		if count != want.reviews || shadowbanned(t, pool, account) != want.banned {
			t.Fatalf("account %s reviews %d, want %+v", account, count, want)
		}
	}
	var score int
	if err := pool.QueryRow(context.Background(), `select score from risk_actions where account_id = $1 and action = 'shadowban'`, cracked).Scan(&score); err != nil {
		t.Fatal(err)
	}
	if score != 8*config.MacFailureWeight {
		t.Fatalf("recorded score %d", score)
	}
}

func TestAWaveRunsOncePerIntervalAcrossNodes(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := risk.NewStore(pool)
	config := risk.DefaultConfig()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	account := newAccount(t, pool)

	first, err := risk.NewWave(pool, config, logger).RunDue(context.Background(), waveNow)
	if err != nil || !first.Ran {
		t.Fatalf("first %+v %v", first, err)
	}
	flag(t, store, account, risk.KindReplay, config.ShadowbanScore, waveNow)
	second, err := risk.NewWave(pool, config, logger).RunDue(context.Background(), waveNow.Add(time.Minute))
	if err != nil || second.Ran {
		t.Fatalf("a second node ran the same wave: %+v %v", second, err)
	}
	if shadowbanned(t, pool, account) {
		t.Fatal("a signal acted before its wave")
	}
	next, err := risk.NewWave(pool, config, logger).RunDue(context.Background(), waveNow.Add(config.BanWaveInterval))
	if err != nil || !next.Ran || next.Shadowbanned != 1 {
		t.Fatalf("next wave %+v %v", next, err)
	}
	again, err := risk.NewWave(pool, config, logger).RunDue(context.Background(), waveNow.Add(2*config.BanWaveInterval))
	if err != nil || !again.Ran || again.Shadowbanned != 0 {
		t.Fatalf("the wave after %+v %v", again, err)
	}
}

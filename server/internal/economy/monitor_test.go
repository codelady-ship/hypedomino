package economy_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/economy"
	"github.com/Alievs-corp/hypedomino/server/internal/risk"
)

var now = time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)

func TestConfigValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		adjust func(*economy.Config)
		valid  bool
	}{
		{"defaults", func(*economy.Config) {}, true},
		{"negative low end", func(c *economy.Config) { c.SinkRatioLowPercent = -1 }, false},
		{"high end not above low end", func(c *economy.Config) { c.SinkRatioHighPercent = c.SinkRatioLowPercent }, false},
		{"zero daily earn alert", func(c *economy.Config) { c.AccountDailyEarnAlertCoins = 0 }, false},
		{"zero median alert", func(c *economy.Config) { c.BalanceMedianAlertCoins = 0 }, false},
		{"negative farm weight", func(c *economy.Config) { c.FarmWeight = -1 }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := economy.DefaultConfig()
			tc.adjust(&config)
			if err := config.Validate(); (err == nil) != tc.valid {
				t.Fatalf("valid %v, got %v", tc.valid, err)
			}
		})
	}
}

type ledger struct {
	t    *testing.T
	pool *pgxpool.Pool
	seq  int
}

func (l *ledger) account() string {
	l.t.Helper()
	var id string
	if err := l.pool.QueryRow(context.Background(), `insert into accounts default values returning id`).Scan(&id); err != nil {
		l.t.Fatal(err)
	}
	return id
}

func (l *ledger) entry(account, currency, reason string, amount, balanceAfter int64, at time.Time) {
	l.t.Helper()
	l.seq++
	_, err := l.pool.Exec(context.Background(), `insert into ledger_entries (account_id, currency, amount, reason, source_type, source_ref, idempotency_key, config_version, balance_after, created_at)
		values ($1, $2, $3, $4, 'test', 'test', $5, 1, $6, $7)`, account, currency, amount, reason, fmt.Sprintf("k%d", l.seq), balanceAfter, at)
	if err != nil {
		l.t.Fatal(err)
	}
}

func seed(t *testing.T) (*pgxpool.Pool, string) {
	_, pool := dbtest.Fresh(t)
	l := &ledger{t: t, pool: pool}
	yesterday := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)
	farmer, player, gone := l.account(), l.account(), l.account()
	l.entry(farmer, "coin", "match_reward", 1000, 1000, yesterday)
	l.entry(farmer, "coin", "mission", 600, 1600, yesterday.Add(time.Hour))
	l.entry(player, "coin", "match_reward", 200, 200, yesterday.Add(-48*time.Hour))
	l.entry(player, "coin", "shop_purchase", -100, 100, yesterday.Add(-24*time.Hour))
	l.entry(player, "coin", "admin_adjust", 5000, 5100, yesterday)
	l.entry(player, "crystal", "purchase", 1000, 1000, yesterday)
	l.entry(gone, "coin", "match_reward", 90000, 90000, now.Add(-10*24*time.Hour))
	l.entry(farmer, "coin", "match_reward", 30, 1630, now)
	return pool, farmer
}

func TestReportMeasuresTheRollingWindowAndTheLastEconomyDay(t *testing.T) {
	pool, _ := seed(t)
	report, err := economy.NewMonitor(pool, economy.DefaultConfig(), slog.New(slog.NewTextHandler(io.Discard, nil))).Report(context.Background(), now)
	if err != nil {
		t.Fatal(err)
	}
	if report.CoinsCreated != 1800 || report.CoinsSpent != 100 || report.SinkRatioPercent == nil || *report.SinkRatioPercent != 5 {
		t.Fatalf("flow %+v", report)
	}
	if report.MedianBalance != 1600 {
		t.Fatalf("median %d", report.MedianBalance)
	}
	if !report.Day.Equal(time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)) || len(report.SourceMix) != 2 || report.SourceMix["match_reward"] != 1000 || report.SourceMix["mission"] != 600 {
		t.Fatalf("day %v mix %v", report.Day, report.SourceMix)
	}
	if report.AccountsOverDailyEarn != 1 || !slices.Equal(report.Alerts, []string{economy.AlertSinkRatio, economy.AlertAccountDailyEarn}) {
		t.Fatalf("alerts %d %v", report.AccountsOverDailyEarn, report.Alerts)
	}
}

func TestAnEmptyLedgerHasNoSinkRatioAndNoAlert(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	report, err := economy.NewMonitor(pool, economy.DefaultConfig(), slog.New(slog.NewTextHandler(io.Discard, nil))).Report(context.Background(), now)
	if err != nil {
		t.Fatal(err)
	}
	if report.SinkRatioPercent != nil || report.MedianBalance != 0 || len(report.Alerts) != 0 {
		t.Fatalf("%+v", report)
	}
}

func TestTheDailyRunFlagsFarmAccountsOncePerEconomyDay(t *testing.T) {
	pool, farmer := seed(t)
	monitor := economy.NewMonitor(pool, economy.DefaultConfig(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if _, ran, err := monitor.RunDue(context.Background(), now); err != nil || !ran {
		t.Fatalf("first run %v %v", ran, err)
	}
	if _, ran, err := monitor.RunDue(context.Background(), now.Add(time.Hour)); err != nil || ran {
		t.Fatalf("second run %v %v", ran, err)
	}
	signals, err := risk.NewStore(pool).Signals(context.Background(), farmer)
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) != 1 || signals[0].Kind != risk.KindFarm || signals[0].Weight != risk.DefaultConfig().CollusionWeight || signals[0].Dedupe != "2026-11-04" || signals[0].Evidence["coins"] != float64(1600) {
		t.Fatalf("signals %+v", signals)
	}
	if _, ran, err := monitor.RunDue(context.Background(), now.Add(24*time.Hour)); err != nil || !ran {
		t.Fatalf("next day %v %v", ran, err)
	}
}

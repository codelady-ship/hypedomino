package economy

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/matchreward"
	"github.com/Alievs-corp/hypedomino/server/internal/risk"
)

const (
	Window = 7 * 24 * time.Hour

	AlertSinkRatio        = "sink_ratio"
	AlertAccountDailyEarn = "account_daily_earn"
	AlertMedianBalance    = "median_balance"
)

var (
	ErrInvalidConfig = errors.New("economy: invalid config")

	SourceReasons    = []string{"match_reward", "first_win", "daily_login", "mission", "pass_reward", "rank_reward", "event_reward", "duplicate_compensation", "promo", "purchase", "first_purchase_bonus"}
	SinkReasons      = []string{"shop_purchase", "pass_level_purchase", "rename", "chest_purchase"}
	DailyEarnReasons = []string{"match_reward", "first_win", "daily_login", "mission"}
)

type Config struct {
	SinkRatioLowPercent        int
	SinkRatioHighPercent       int
	AccountDailyEarnAlertCoins int
	BalanceMedianAlertCoins    int
	FarmWeight                 int
}

func DefaultConfig() Config {
	return Config{SinkRatioLowPercent: 60, SinkRatioHighPercent: 120, AccountDailyEarnAlertCoins: 1500, BalanceMedianAlertCoins: 30000, FarmWeight: risk.DefaultConfig().CollusionWeight}
}

func (c Config) Validate() error {
	if c.SinkRatioLowPercent < 0 || c.SinkRatioHighPercent <= c.SinkRatioLowPercent {
		return fmt.Errorf("%w: the sink ratio band must not be negative and its high end above its low end", ErrInvalidConfig)
	}
	if c.AccountDailyEarnAlertCoins < 1 || c.BalanceMedianAlertCoins < 1 || c.FarmWeight < 0 {
		return fmt.Errorf("%w: earn and balance alerts must be positive and the farm weight not negative", ErrInvalidConfig)
	}
	return nil
}

type Report struct {
	From                  time.Time        `json:"from"`
	To                    time.Time        `json:"to"`
	Day                   time.Time        `json:"day"`
	CoinsCreated          int64            `json:"coinsCreated"`
	CoinsSpent            int64            `json:"coinsSpent"`
	SinkRatioPercent      *int64           `json:"sinkRatioPercent"`
	MedianBalance         int64            `json:"medianBalance"`
	SourceMix             map[string]int64 `json:"sourceMix"`
	AccountsOverDailyEarn int              `json:"accountsOverDailyEarn"`
	Alerts                []string         `json:"alerts"`
	earners               map[string]int64
}

type Monitor struct {
	pool   *pgxpool.Pool
	config Config
	logger *slog.Logger
}

func NewMonitor(pool *pgxpool.Pool, config Config, logger *slog.Logger) *Monitor {
	return &Monitor{pool: pool, config: config, logger: logger}
}

func (m *Monitor) Report(ctx context.Context, to time.Time) (Report, error) {
	day := matchreward.EconomyDay(to).Add(-24 * time.Hour)
	report := Report{From: to.Add(-Window).UTC(), To: to.UTC(), Day: day}
	if err := m.flow(ctx, &report); err != nil {
		return report, err
	}
	if err := m.median(ctx, &report); err != nil {
		return report, err
	}
	if err := m.sourceMix(ctx, &report); err != nil {
		return report, err
	}
	if err := m.dailyEarners(ctx, &report); err != nil {
		return report, err
	}
	report.Alerts = m.config.alerts(report)
	return report, nil
}

func (m *Monitor) flow(ctx context.Context, report *Report) error {
	err := m.pool.QueryRow(ctx, `select coalesce(sum(amount) filter (where amount > 0 and reason = any($3)), 0)::bigint,
		coalesce(-sum(amount) filter (where amount < 0 and reason = any($4)), 0)::bigint
		from ledger_entries where currency = 'coin' and created_at >= $1 and created_at < $2`,
		report.From, report.To, SourceReasons, SinkReasons).Scan(&report.CoinsCreated, &report.CoinsSpent)
	if err != nil || report.CoinsCreated == 0 {
		return err
	}
	ratio := report.CoinsSpent * 100 / report.CoinsCreated
	report.SinkRatioPercent = &ratio
	return nil
}

func (m *Monitor) median(ctx context.Context, report *Report) error {
	return m.pool.QueryRow(ctx, `select coalesce(percentile_disc(0.5) within group (order by balance_after), 0)::bigint from (
		select distinct on (account_id) balance_after from ledger_entries
		where currency = 'coin' and created_at < $2
		and account_id in (select account_id from ledger_entries where currency = 'coin' and created_at >= $1 and created_at < $2)
		order by account_id, created_at desc, id desc) latest`, report.From, report.To).Scan(&report.MedianBalance)
}

func (m *Monitor) sourceMix(ctx context.Context, report *Report) error {
	rows, err := m.pool.Query(ctx, `select reason, sum(amount)::bigint from ledger_entries
		where currency = 'coin' and amount > 0 and reason = any($3) and created_at >= $1 and created_at < $2 group by reason`,
		report.Day, report.Day.Add(24*time.Hour), SourceReasons)
	if err != nil {
		return err
	}
	report.SourceMix = map[string]int64{}
	var reason string
	var coins int64
	_, err = pgx.ForEachRow(rows, []any{&reason, &coins}, func() error {
		report.SourceMix[reason] = coins
		return nil
	})
	return err
}

func (m *Monitor) dailyEarners(ctx context.Context, report *Report) error {
	rows, err := m.pool.Query(ctx, `select account_id, sum(amount)::bigint from ledger_entries
		where currency = 'coin' and reason = any($3) and created_at >= $1 and created_at < $2
		group by account_id having sum(amount) > $4`,
		report.Day, report.Day.Add(24*time.Hour), DailyEarnReasons, m.config.AccountDailyEarnAlertCoins)
	if err != nil {
		return err
	}
	report.earners = map[string]int64{}
	var account string
	var coins int64
	_, err = pgx.ForEachRow(rows, []any{&account, &coins}, func() error {
		report.earners[account] = coins
		return nil
	})
	report.AccountsOverDailyEarn = len(report.earners)
	return err
}

func (c Config) alerts(report Report) []string {
	alerts := []string{}
	if ratio := report.SinkRatioPercent; ratio != nil && (*ratio < int64(c.SinkRatioLowPercent) || *ratio > int64(c.SinkRatioHighPercent)) {
		alerts = append(alerts, AlertSinkRatio)
	}
	if report.AccountsOverDailyEarn > 0 {
		alerts = append(alerts, AlertAccountDailyEarn)
	}
	if report.MedianBalance > int64(c.BalanceMedianAlertCoins) {
		alerts = append(alerts, AlertMedianBalance)
	}
	return alerts
}

func (m *Monitor) RunDue(ctx context.Context, now time.Time) (Report, bool, error) {
	to := matchreward.EconomyDay(now)
	var done bool
	if err := m.pool.QueryRow(ctx, `select exists (select 1 from economy_monitor_runs where day = $1)`, to.Add(-24*time.Hour)).Scan(&done); err != nil || done {
		return Report{}, false, err
	}
	report, err := m.Report(ctx, to)
	if err != nil {
		return report, false, err
	}
	if err := m.signalFarms(ctx, report, now); err != nil {
		return report, false, err
	}
	tag, err := m.pool.Exec(ctx, `insert into economy_monitor_runs (day, ran_at) values ($1, $2) on conflict (day) do nothing`, report.Day, now)
	if err != nil || tag.RowsAffected() == 0 {
		return report, false, err
	}
	m.logAlerts(report)
	return report, true, nil
}

func (m *Monitor) logAlerts(report Report) {
	for _, alert := range report.Alerts {
		m.logger.Warn("economy alert", "monitor", alert, "day", report.Day.Format(time.DateOnly), "sink_ratio_percent", report.SinkRatioPercent,
			"median_balance", report.MedianBalance, "accounts_over_daily_earn", report.AccountsOverDailyEarn)
	}
}

func (m *Monitor) signalFarms(ctx context.Context, report Report, now time.Time) error {
	day := report.Day.Format(time.DateOnly)
	for account, coins := range report.earners {
		signal := risk.Signal{Account: account, Kind: risk.KindFarm, Weight: m.config.FarmWeight, Dedupe: day, At: now,
			Evidence: map[string]any{"day": day, "coins": coins, "limit": m.config.AccountDailyEarnAlertCoins}}
		if _, err := risk.RecordIn(ctx, m.pool, signal); err != nil {
			return err
		}
	}
	return nil
}

func (m *Monitor) Run(ctx context.Context, every time.Duration, now func() time.Time) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		report, ran, err := m.RunDue(ctx, now())
		if err != nil {
			m.logger.Error("economy monitor failed", "error", err)
		}
		if ran {
			m.logger.Info("economy monitor ran", "day", report.Day.Format(time.DateOnly), "alerts", len(report.Alerts))
		}
	}
}

package risk

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	CauseScore      = "score"
	CauseHardSignal = "hard_signal"
	ActionShadowban = "shadowban"
)

func isHard(kind string) bool {
	return kind == KindFakeReceipt
}

func openReview(ctx context.Context, db Executor, account, cause string, at time.Time) error {
	_, err := db.Exec(ctx, `insert into risk_reviews (account_id, cause, opened_at) values ($1, $2, $3)
		on conflict (account_id) where closed_at is null do nothing`, account, cause, at)
	return err
}

type Wave struct {
	pool   *pgxpool.Pool
	config Config
	logger *slog.Logger
	hide   func(ctx context.Context, account string) error
}

func NewWave(pool *pgxpool.Pool, config Config, logger *slog.Logger) *Wave {
	return &Wave{pool: pool, config: config, logger: logger}
}

func (w *Wave) OnShadowban(hide func(ctx context.Context, account string) error) {
	w.hide = hide
}

type WaveResult struct {
	Ran          bool
	Reviewed     int
	Shadowbanned int
}

type standing struct {
	account string
	score   int
}

func (w *Wave) RunDue(ctx context.Context, now time.Time) (WaveResult, error) {
	slot := now.UTC().Truncate(w.config.BanWaveInterval)
	tag, err := w.pool.Exec(ctx, `insert into risk_waves (wave, started_at) values ($1, $2) on conflict (wave) do nothing`, slot, now)
	if err != nil || tag.RowsAffected() == 0 {
		return WaveResult{}, err
	}
	standings, err := w.overReview(ctx, now)
	if err != nil {
		return WaveResult{Ran: true}, err
	}
	result := WaveResult{Ran: true}
	for _, current := range standings {
		shadowbanned, err := w.apply(ctx, current, slot, now)
		if err != nil {
			return result, err
		}
		result.Reviewed++
		if shadowbanned {
			result.Shadowbanned++
			w.hideFromBoards(ctx, current.account)
		}
	}
	return result, nil
}

func (w *Wave) hideFromBoards(ctx context.Context, account string) {
	if w.hide == nil {
		return
	}
	if err := w.hide(ctx, account); err != nil {
		w.logger.Error("shadowbanned account still on a board until it rebuilds", "account", account, "error", err)
	}
}

func (w *Wave) overReview(ctx context.Context, now time.Time) ([]standing, error) {
	rows, err := w.pool.Query(ctx, `select s.account_id, sum(s.weight)::int from risk_signals s join accounts a on a.id = s.account_id
		where s.created_at > $1 and (a.risk_reviewed_at is null or s.created_at > a.risk_reviewed_at)
		group by s.account_id having sum(s.weight) >= $2`, now.Add(-w.config.Decay), w.config.ReviewScore)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (standing, error) {
		var current standing
		err := row.Scan(&current.account, &current.score)
		return current, err
	})
}

func (w *Wave) apply(ctx context.Context, current standing, slot, now time.Time) (bool, error) {
	shadowbanned := false
	err := pgx.BeginFunc(ctx, w.pool, func(tx pgx.Tx) error {
		if err := openReview(ctx, tx, current.account, CauseScore, now); err != nil {
			return err
		}
		if current.score < w.config.ShadowbanScore {
			return nil
		}
		tag, err := tx.Exec(ctx, `update accounts set shadowbanned_at = $2 where id = $1 and shadowbanned_at is null`, current.account, now)
		if err != nil || tag.RowsAffected() == 0 {
			return err
		}
		shadowbanned = true
		_, err = tx.Exec(ctx, `insert into risk_actions (account_id, action, score, wave, created_at) values ($1, $2, $3, $4, $5)`, current.account, ActionShadowban, current.score, slot, now)
		return err
	})
	return shadowbanned, err
}

func (w *Wave) Run(ctx context.Context, every time.Duration, now func() time.Time) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		result, err := w.RunDue(ctx, now())
		if err != nil {
			w.logger.Error("risk wave failed", "error", err)
		}
		if result.Ran {
			w.logger.Info("risk wave ran", "reviewed", result.Reviewed, "shadowbanned", result.Shadowbanned)
		}
	}
}

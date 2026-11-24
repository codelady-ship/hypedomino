package pass

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

const (
	SourcePass   = "pass"
	endBatchSize = 500
)

var (
	ErrNoSeason   = errors.New("pass: no active season")
	ErrNoSnapshot = errors.New("pass: no config snapshot")
)

type Rewarder interface {
	ApplyIn(ctx context.Context, tx pgx.Tx, grant reward.Grant, now time.Time) (reward.Report, error)
	RevokeIn(ctx context.Context, tx pgx.Tx, account, key string, reason wallet.Reason, revokeKey string, now time.Time) (reward.Report, error)
	ReinstateIn(ctx context.Context, tx pgx.Tx, account, key string, reason wallet.Reason, reinstateKey string, now time.Time) (reward.Report, error)
}

type Progress struct {
	Season    string
	XP        int64
	Premium   bool
	EnteredAt time.Time
}

type Closed struct {
	Season   string
	Accounts int
	Granted  int
}

type Store struct {
	pool    *pgxpool.Pool
	configs *snapshot.Watcher[*Snapshot]
	rewards Rewarder
	wallet  Spender
}

func NewWatcher(pool *pgxpool.Pool, logger *slog.Logger) *snapshot.Watcher[*Snapshot] {
	watcher := snapshot.New(pool, Domain, Parse, logger)
	watcher.Seed(Default(), 0)
	return watcher
}

func NewStore(pool *pgxpool.Pool, configs *snapshot.Watcher[*Snapshot]) *Store {
	return &Store{pool: pool, configs: configs}
}

func (s *Store) current() (*Snapshot, int64, error) {
	current, version, ok := s.configs.Current()
	if !ok {
		return nil, 0, ErrNoSnapshot
	}
	return current, version, nil
}

func (s *Store) Enter(ctx context.Context, account string, now time.Time) (Season, Progress, int64, error) {
	current, version, err := s.current()
	if err != nil {
		return Season{}, Progress{}, 0, err
	}
	return s.enter(ctx, current, version, account, now)
}

func (s *Store) View(ctx context.Context, account string, now time.Time) (View, error) {
	current, version, err := s.current()
	if err != nil {
		return View{}, err
	}
	season, progress, _, err := s.enter(ctx, current, version, account, now)
	if err != nil {
		return View{}, err
	}
	return NewView(current, season, progress, version), nil
}

func (s *Store) enter(ctx context.Context, current *Snapshot, version int64, account string, now time.Time) (Season, Progress, int64, error) {
	season, ok := current.Active(now)
	if !ok {
		return Season{}, Progress{}, version, ErrNoSeason
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Season{}, Progress{}, 0, err
	}
	defer tx.Rollback(ctx)
	if err := reward.LockIn(ctx, tx, account); err != nil {
		return Season{}, Progress{}, 0, err
	}
	progress, err := s.enterIn(ctx, tx, account, season, version, now)
	if err != nil {
		return Season{}, Progress{}, 0, err
	}
	return season, progress, version, tx.Commit(ctx)
}

func (s *Store) enterIn(ctx context.Context, tx pgx.Tx, account string, season Season, version int64, now time.Time) (Progress, error) {
	tag, err := tx.Exec(ctx, `insert into pass_progress (account_id, season_id, entered_at) values ($1, $2, $3) on conflict do nothing`, account, season.ID, now)
	if err != nil {
		return Progress{}, err
	}
	progress, err := progressIn(ctx, tx, account, season.ID)
	if err != nil || tag.RowsAffected() == 0 {
		return progress, err
	}
	_, err = s.grantLevels(ctx, tx, account, season, version, []string{Free}, 1, 1, now)
	return progress, err
}

func (s *Store) SetRewards(rewards Rewarder) {
	s.rewards = rewards
}

func progressIn(ctx context.Context, tx pgx.Tx, account, season string) (Progress, error) {
	progress := Progress{Season: season}
	err := tx.QueryRow(ctx, `select xp_total, premium_ref is not null, entered_at from pass_progress where account_id = $1 and season_id = $2 for update`, account, season).Scan(&progress.XP, &progress.Premium, &progress.EnteredAt)
	return progress, err
}

func unlocked(progress Progress) []string {
	if progress.Premium {
		return []string{Free, Premium}
	}
	return []string{Free}
}

func (s *Store) grantReached(ctx context.Context, tx pgx.Tx, account string, season Season, version int64, progress Progress, now time.Time) (int, error) {
	return s.grantLevels(ctx, tx, account, season, version, unlocked(progress), 1, season.Curve.LevelFor(progress.XP), now)
}

func (s *Store) grantLevels(ctx context.Context, tx pgx.Tx, account string, season Season, version int64, tracks []string, from, to int, now time.Time) (int, error) {
	if s.rewards == nil {
		return 0, nil
	}
	granted := 0
	for level := from; level <= to; level++ {
		for _, track := range tracks {
			bundle, ok := season.Track(track)[level]
			if !ok {
				continue
			}
			grant := levelGrant(account, season.ID, track, level, bundle, version)
			report, err := s.rewards.ApplyIn(ctx, tx, grant, now)
			if err == nil && report.Repeated && report.Revoked() {
				report, err = s.rewards.ReinstateIn(ctx, tx, account, grant.Key, wallet.RefundReversal, grant.Key+":reinstate:"+report.RevokeKey, now)
				report.Repeated = false
			}
			if err != nil {
				return granted, fmt.Errorf("%s %s level %d: %w", season.ID, track, level, err)
			}
			if !report.Repeated {
				granted++
			}
		}
	}
	return granted, nil
}

func levelGrant(account, season, track string, level int, bundle reward.Bundle, version int64) reward.Grant {
	return reward.Grant{
		Account:       account,
		Bundle:        bundle,
		Source:        TrackSource(season, track, level),
		Reason:        wallet.PassReward,
		Key:           fmt.Sprintf("pass:%s:%s:%d", season, track, level),
		ConfigVersion: int(version),
	}
}

func (s *Store) CloseDue(ctx context.Context, now time.Time) ([]Closed, error) {
	current, version, err := s.current()
	if err != nil {
		return nil, err
	}
	var closed []Closed
	var failures []error
	for _, season := range current.Seasons {
		if now.Before(season.End) {
			continue
		}
		done, err := s.ended(ctx, season.ID)
		if err != nil {
			return closed, err
		}
		if done {
			continue
		}
		result, err := s.closeSeason(ctx, season, version, now)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		closed = append(closed, result)
	}
	return closed, errors.Join(failures...)
}

func (s *Store) ended(ctx context.Context, season string) (bool, error) {
	var done bool
	err := s.pool.QueryRow(ctx, `select exists (select 1 from pass_season_ends where season_id = $1)`, season).Scan(&done)
	return done, err
}

func (s *Store) closeSeason(ctx context.Context, season Season, version int64, now time.Time) (Closed, error) {
	result := Closed{Season: season.ID}
	var failures []error
	after := "00000000-0000-0000-0000-000000000000"
	for {
		accounts, err := s.accountsAfter(ctx, season.ID, after)
		if err != nil {
			return result, err
		}
		for _, account := range accounts {
			granted, err := s.settle(ctx, account, season, version, now)
			if err != nil {
				failures = append(failures, fmt.Errorf("account %s: %w", account, err))
				continue
			}
			result.Accounts++
			result.Granted += granted
		}
		if len(accounts) < endBatchSize {
			break
		}
		after = accounts[len(accounts)-1]
	}
	if err := errors.Join(failures...); err != nil {
		return result, fmt.Errorf("season %s end: %w", season.ID, err)
	}
	_, err := s.pool.Exec(ctx, `insert into pass_season_ends (season_id, config_version, accounts, closed_at) values ($1, $2, $3, $4) on conflict do nothing`, season.ID, version, result.Accounts, now)
	return result, err
}

func (s *Store) accountsAfter(ctx context.Context, season, after string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `select account_id from pass_progress where season_id = $1 and account_id > $2 order by account_id limit $3`, season, after, endBatchSize)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

func (s *Store) settle(ctx context.Context, account string, season Season, version int64, now time.Time) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	if err := reward.LockIn(ctx, tx, account); err != nil {
		return 0, err
	}
	progress, err := progressIn(ctx, tx, account, season.ID)
	if err != nil {
		return 0, err
	}
	granted, err := s.grantReached(ctx, tx, account, season, version, progress, now)
	if err != nil {
		return 0, err
	}
	return granted, tx.Commit(ctx)
}

func (s *Store) RunSeasonEnd(ctx context.Context, logger *slog.Logger, every time.Duration, now func() time.Time) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		closed, err := s.CloseDue(ctx, now())
		if err != nil {
			logger.Error("pass season end failed", "error", err)
		}
		for _, c := range closed {
			logger.Info("pass season closed", "season", c.Season, "accounts", c.Accounts, "granted", c.Granted)
		}
	}
}

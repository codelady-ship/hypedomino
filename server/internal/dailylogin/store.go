package dailylogin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

var (
	ErrNoCalendar = errors.New("daily_login.unavailable")
	ErrClaimed    = errors.New("daily_login.claimed")
)

type Rewards interface {
	ApplyIn(ctx context.Context, tx pgx.Tx, grant reward.Grant, now time.Time) (reward.Report, error)
}

type Status struct {
	Version      int64          `json:"version"`
	EconomyDay   string         `json:"economy_day"`
	Day          int            `json:"day"`
	ClaimedToday bool           `json:"claimed_today"`
	NextDayAt    time.Time      `json:"next_day_at"`
	Days         []Day          `json:"days"`
	Granted      *reward.Report `json:"granted,omitempty"`
}

type Store struct {
	pool     *pgxpool.Pool
	calendar *snapshot.Watcher[*Calendar]
	rewards  Rewards
}

func NewWatcher(pool *pgxpool.Pool, logger *slog.Logger) *snapshot.Watcher[*Calendar] {
	watcher := snapshot.New(pool, Domain, Parse, logger)
	watcher.Seed(Default(), 0)
	return watcher
}

func NewStore(pool *pgxpool.Pool, calendar *snapshot.Watcher[*Calendar], rewards Rewards) *Store {
	return &Store{pool: pool, calendar: calendar, rewards: rewards}
}

func (s *Store) current() (*Calendar, int64, error) {
	calendar, version, ok := s.calendar.Current()
	if !ok {
		return nil, 0, ErrNoCalendar
	}
	return calendar, version, nil
}

func progressIn(ctx context.Context, q pgx.Tx, account string) (int64, string, error) {
	var claims int64
	var last string
	err := q.QueryRow(ctx, `select claims, to_char(last_day, 'YYYY-MM-DD') from daily_logins where account_id = $1`, account).Scan(&claims, &last)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, "", nil
	}
	return claims, last, err
}

func status(calendar *Calendar, version int64, claims int64, last string, now time.Time) Status {
	today := EconomyDay(now)
	next := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day()+1, 0, 0, 0, 0, time.UTC)
	return Status{Version: version, EconomyDay: today, Day: calendar.Next(claims), ClaimedToday: last == today, NextDayAt: next, Days: calendar.Days}
}

func (s *Store) Status(ctx context.Context, account string, now time.Time) (Status, error) {
	calendar, version, err := s.current()
	if err != nil {
		return Status{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Status{}, err
	}
	defer tx.Rollback(ctx)
	claims, last, err := progressIn(ctx, tx, account)
	if err != nil {
		return Status{}, err
	}
	return status(calendar, version, claims, last, now), nil
}

func (s *Store) Claim(ctx context.Context, account string, now time.Time) (Status, error) {
	calendar, version, err := s.current()
	if err != nil {
		return Status{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Status{}, err
	}
	defer tx.Rollback(ctx)
	if err := reward.LockIn(ctx, tx, account); err != nil {
		return Status{}, err
	}
	claims, last, err := progressIn(ctx, tx, account)
	if err != nil {
		return Status{}, err
	}
	today := EconomyDay(now)
	if last == today {
		return Status{}, ErrClaimed
	}
	day := calendar.Next(claims)
	report, err := s.rewards.ApplyIn(ctx, tx, reward.Grant{
		Account:       account,
		Bundle:        calendar.Reward(day, version),
		Source:        cosmetics.Source{Type: Source, Ref: fmt.Sprintf("day_%d", day)},
		Reason:        wallet.DailyLogin,
		Key:           "daily_login:" + today,
		ConfigVersion: int(version),
	}, now)
	if err != nil {
		return Status{}, err
	}
	if _, err := tx.Exec(ctx, `insert into daily_logins (account_id, claims, last_day, updated_at) values ($1, 1, $2::date, $3)
		on conflict (account_id) do update set claims = daily_logins.claims + 1, last_day = excluded.last_day, updated_at = excluded.updated_at`, account, today, now); err != nil {
		return Status{}, err
	}
	out := status(calendar, version, claims+1, today, now)
	out.Granted = &report
	return out, tx.Commit(ctx)
}

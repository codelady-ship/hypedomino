package consent

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (s *Store) Choices(ctx context.Context, account string) (Choices, error) {
	return latest(ctx, s.pool, account)
}

func (s *Store) Granted(ctx context.Context, account string, purpose Purpose) (bool, error) {
	choices, err := s.Choices(ctx, account)
	return choices.Of(purpose), err
}

func (s *Store) Save(ctx context.Context, account string, change Change, now time.Time) (Choices, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Choices{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `select 1 from accounts where id = $1 for no key update`, account); err != nil {
		return Choices{}, err
	}
	current, err := latest(ctx, tx, account)
	if err != nil {
		return Choices{}, err
	}
	next := current.With(change)
	changed := current.Changed(next)
	if len(changed) == 0 {
		return current, nil
	}
	if change.PolicyVersion == "" || change.AppVersion == "" {
		return Choices{}, ErrContext
	}
	for _, purpose := range changed {
		if _, err := tx.Exec(ctx, `insert into consent_records (account_id, purpose, granted, policy_version, source, app_version, recorded_at)
			values ($1, $2, $3, $4, 'settings', $5, $6)`, account, string(purpose), next.Of(purpose), change.PolicyVersion, change.AppVersion, now); err != nil {
			return Choices{}, err
		}
	}
	return next, tx.Commit(ctx)
}

func (s *Store) SaveFirstLaunch(ctx context.Context, account string, first FirstLaunch, now time.Time) (Choices, error) {
	if len(first.Choices) == 0 {
		return s.Choices(ctx, account)
	}
	if first.PolicyVersion == "" || first.AppVersion == "" {
		return Choices{}, ErrContext
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Choices{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `select 1 from accounts where id = $1 for no key update`, account); err != nil {
		return Choices{}, err
	}
	var recorded bool
	if err := tx.QueryRow(ctx, `select exists (select 1 from consent_records where account_id = $1 and policy_version = $2)`, account, first.PolicyVersion).Scan(&recorded); err != nil {
		return Choices{}, err
	}
	if recorded {
		return latest(ctx, tx, account)
	}
	for _, choice := range first.Choices {
		if _, err := tx.Exec(ctx, `insert into consent_records (account_id, purpose, granted, policy_version, source, app_version, recorded_at)
			values ($1, $2, $3, $4, 'first_launch', $5, $6)`, account, string(choice.Purpose), choice.Granted, first.PolicyVersion, first.AppVersion, now); err != nil {
			return Choices{}, err
		}
	}
	choices, err := latest(ctx, tx, account)
	if err != nil {
		return Choices{}, err
	}
	return choices, tx.Commit(ctx)
}

func latest(ctx context.Context, q querier, account string) (Choices, error) {
	rows, err := q.Query(ctx, `select distinct on (purpose) purpose, granted from consent_records
		where account_id = $1 and purpose in ('analytics', 'diagnostics') order by purpose, recorded_at desc, id desc`, account)
	if err != nil {
		return Choices{}, err
	}
	defer rows.Close()
	var choices Choices
	for rows.Next() {
		var purpose string
		var granted bool
		if err := rows.Scan(&purpose, &granted); err != nil {
			return Choices{}, err
		}
		choices.set(Purpose(purpose), granted)
	}
	return choices, rows.Err()
}

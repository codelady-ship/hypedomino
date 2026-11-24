package dataexport

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const purgeBatch = 100

var (
	ErrCooldown    = errors.New("dataexport: cooldown")
	ErrNoAccount   = errors.New("dataexport: no such account")
	ErrBadSettings = errors.New("dataexport: link lifetime and cooldown must be positive")
)

type Settings struct {
	LinkTTL  time.Duration
	Cooldown time.Duration
}

func (s Settings) Validate() error {
	if s.LinkTTL <= 0 || s.Cooldown <= 0 {
		return ErrBadSettings
	}
	return nil
}

type Files interface {
	Put(ctx context.Context, key, contentType string, body []byte, now time.Time) error
	SignedGet(key string, now time.Time) string
	Delete(ctx context.Context, key string, now time.Time) error
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Reserve(ctx context.Context, account, key string, now time.Time, settings Settings) (time.Time, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return time.Time{}, err
	}
	defer tx.Rollback(ctx)
	err = tx.QueryRow(ctx, `select id from accounts where id = $1 for update`, account).Scan(new(any))
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, ErrNoAccount
	}
	if err != nil {
		return time.Time{}, err
	}
	var last *time.Time
	if err := tx.QueryRow(ctx, `select max(requested_at) from data_exports where account_id = $1`, account).Scan(&last); err != nil {
		return time.Time{}, err
	}
	if last != nil && now.Before(last.Add(settings.Cooldown)) {
		return last.Add(settings.Cooldown).UTC(), ErrCooldown
	}
	if _, err := tx.Exec(ctx, `insert into data_exports (account_id, object_key, requested_at, expires_at) values ($1, $2, $3, $4)`,
		account, key, now.UTC(), now.Add(settings.LinkTTL).UTC()); err != nil {
		return time.Time{}, err
	}
	return time.Time{}, tx.Commit(ctx)
}

func (s *Store) Release(ctx context.Context, key string) error {
	_, err := s.pool.Exec(ctx, `delete from data_exports where object_key = $1`, key)
	return err
}

func (s *Store) Purge(ctx context.Context, files Files, now time.Time) (int, error) {
	rows, err := s.pool.Query(ctx, `select object_key from data_exports where file_deleted_at is null and expires_at <= $1 order by expires_at limit $2`, now.UTC(), purgeBatch)
	if err != nil {
		return 0, err
	}
	keys, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return 0, err
	}
	for at, key := range keys {
		if err := files.Delete(ctx, key, now); err != nil {
			return at, fmt.Errorf("delete %s: %w", key, err)
		}
		if _, err := s.pool.Exec(ctx, `update data_exports set file_deleted_at = $2 where object_key = $1`, key, now.UTC()); err != nil {
			return at, err
		}
	}
	return len(keys), nil
}

func (s *Store) RunPurge(ctx context.Context, files Files, logger *slog.Logger, every time.Duration, now func() time.Time) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		purged, err := s.Purge(ctx, files, now())
		if err != nil {
			logger.Error("data export purge failed", "error", err)
		}
		if purged > 0 {
			logger.Info("data export files deleted", "count", purged)
		}
	}
}

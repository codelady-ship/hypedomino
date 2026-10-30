package snapshot

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNone = errors.New("snapshot: no published version")

type Parse[T any] func(raw []byte) (T, error)

type Watcher[T any] struct {
	pool    *pgxpool.Pool
	domain  string
	parse   Parse[T]
	logger  *slog.Logger
	mu      sync.RWMutex
	version int64
	value   T
	loaded  bool
}

func New[T any](pool *pgxpool.Pool, domain string, parse Parse[T], logger *slog.Logger) *Watcher[T] {
	return &Watcher[T]{pool: pool, domain: domain, parse: parse, logger: logger}
}

func (w *Watcher[T]) Seed(value T, version int64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.value, w.version, w.loaded = value, version, true
}

func (w *Watcher[T]) SeedFile(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	value, err := w.parse(raw)
	if err != nil {
		return err
	}
	w.Seed(value, 0)
	return nil
}

func (w *Watcher[T]) Current() (T, int64, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.value, w.version, w.loaded
}

func (w *Watcher[T]) Load(ctx context.Context) (bool, error) {
	var version int64
	var raw []byte
	err := w.pool.QueryRow(ctx, `select version, content from config_snapshots where domain = $1 order by version desc limit 1`, w.domain).Scan(&version, &raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrNone
	}
	if err != nil {
		return false, err
	}
	w.mu.RLock()
	current := w.version
	w.mu.RUnlock()
	if version == current {
		return false, nil
	}
	value, err := w.parse(raw)
	if err != nil {
		return false, err
	}
	w.Seed(value, version)
	return true, nil
}

func (w *Watcher[T]) Run(ctx context.Context, every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		changed, err := w.Load(ctx)
		switch {
		case err != nil && !errors.Is(err, ErrNone) && ctx.Err() == nil:
			w.logger.Error("config snapshot refresh failed", "domain", w.domain, "error", err)
		case changed:
			_, version, _ := w.Current()
			w.logger.Info("config snapshot applied", "domain", w.domain, "version", version)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func Publish(ctx context.Context, pool *pgxpool.Pool, domain string, content []byte) (int64, error) {
	var version int64
	err := pool.QueryRow(ctx, `insert into config_snapshots (domain, content) values ($1, $2) returning version`, domain, content).Scan(&version)
	return version, err
}

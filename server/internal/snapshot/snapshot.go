package snapshot

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
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
	applied func()
	synced  atomic.Bool
}

func New[T any](pool *pgxpool.Pool, domain string, parse Parse[T], logger *slog.Logger) *Watcher[T] {
	return &Watcher[T]{pool: pool, domain: domain, parse: parse, logger: logger}
}

func (w *Watcher[T]) OnApplied(applied func()) {
	w.applied = applied
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

type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func Latest(ctx context.Context, pool Querier, domain string) (int64, []byte, error) {
	var version int64
	var raw []byte
	err := pool.QueryRow(ctx, `select version, content from config_snapshots where domain = $1 order by version desc limit 1`, domain).Scan(&version, &raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil, ErrNone
	}
	return version, raw, err
}

func (w *Watcher[T]) Synced() bool {
	return w.synced.Load()
}

func (w *Watcher[T]) Load(ctx context.Context) (bool, error) {
	version, raw, err := Latest(ctx, w.pool, w.domain)
	if err == nil || errors.Is(err, ErrNone) {
		w.synced.Store(true)
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
			if w.applied != nil {
				w.applied()
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func Publish(ctx context.Context, pool Querier, domain string, content []byte) (int64, error) {
	var version int64
	err := pool.QueryRow(ctx, `insert into config_snapshots (domain, content) values ($1, $2) returning version`, domain, content).Scan(&version)
	return version, err
}

type Published struct {
	Version int64
	Content []byte
	At      time.Time
}

func Version(ctx context.Context, q Querier, domain string, version int64) (Published, error) {
	published := Published{Version: version}
	err := q.QueryRow(ctx, `select content, published_at from config_snapshots where domain = $1 and version = $2`, domain, version).Scan(&published.Content, &published.At)
	if errors.Is(err, pgx.ErrNoRows) {
		return Published{}, ErrNone
	}
	published.At = published.At.UTC()
	return published, err
}

type Active struct {
	Published
	Domain string
}

func Newest(ctx context.Context, q Querier) (Active, error) {
	var active Active
	err := q.QueryRow(ctx, `select version, domain, published_at from config_snapshots order by version desc limit 1`).Scan(&active.Version, &active.Domain, &active.At)
	if errors.Is(err, pgx.ErrNoRows) {
		return Active{}, ErrNone
	}
	active.At = active.At.UTC()
	return active, err
}

type Lister interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func History(ctx context.Context, q Lister, domain string, limit int) ([]Published, error) {
	rows, err := q.Query(ctx, `select version, content, published_at from config_snapshots where domain = $1 order by version desc limit $2`, domain, limit)
	if err != nil {
		return nil, err
	}
	return collectPublished(rows)
}

func InEffectWithinDays(ctx context.Context, q Lister, domain string, days int) ([]Published, error) {
	rows, err := q.Query(ctx, `select version, content, published_at from config_snapshots where domain = $1
		and version >= coalesce((select max(version) from config_snapshots where domain = $1 and published_at <= now() - make_interval(days => $2)), 0)
		order by version desc`, domain, days)
	if err != nil {
		return nil, err
	}
	return collectPublished(rows)
}

func collectPublished(rows pgx.Rows) ([]Published, error) {
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Published, error) {
		var p Published
		err := row.Scan(&p.Version, &p.Content, &p.At)
		p.At = p.At.UTC()
		return p, err
	})
}

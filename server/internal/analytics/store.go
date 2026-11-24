package analytics

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Record(ctx context.Context, account string, batch Batch, now time.Time) error {
	if len(batch.Events) == 0 {
		return nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `insert into analytics_ids (analytics_id, account_id, first_seen_at) values ($1, $2, $3) on conflict (analytics_id) do nothing`,
		batch.AnalyticsID, account, now); err != nil {
		return err
	}
	var owner string
	if err := tx.QueryRow(ctx, `select account_id::text from analytics_ids where analytics_id = $1`, batch.AnalyticsID).Scan(&owner); err != nil {
		return err
	}
	if owner != account {
		return ErrIDTaken
	}
	names, labels, values, clientTimes := columns(batch.Events)
	if _, err := tx.Exec(ctx, `insert into analytics_events (analytics_id, name, label, value, client_time_ms, received_at)
		select $1, e.name, e.label, e.value, e.client_time_ms, $6 from unnest($2::text[], $3::text[], $4::bigint[], $5::bigint[]) with ordinality as e(name, label, value, client_time_ms, position)
		order by e.position`,
		batch.AnalyticsID, names, labels, values, clientTimes, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func columns(events []Event) ([]string, []string, []int64, []int64) {
	names := make([]string, len(events))
	labels := make([]string, len(events))
	values := make([]int64, len(events))
	clientTimes := make([]int64, len(events))
	for i, e := range events {
		names[i], labels[i], values[i], clientTimes[i] = e.Name, e.Label, e.Value, e.ClientTimeMs
	}
	return names, labels, values, clientTimes
}

func (s *Store) PurgeRaw(ctx context.Context, receivedBefore time.Time) (int64, error) {
	tag, err := s.pool.Exec(ctx, `with sources as (delete from analytics_sources where recorded_at < $1)
		delete from analytics_events where received_at < $1`, receivedBefore)
	return tag.RowsAffected(), err
}

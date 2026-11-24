package analytics

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Alievs-corp/hypedomino/server/internal/matchstore"
)

const MatchEnd = "match_end"

func (s *Store) Publish(ctx context.Context, message matchstore.Pending) error {
	var outcome matchstore.Outcome
	if err := json.Unmarshal(message.Payload, &outcome); err != nil {
		return fmt.Errorf("analytics: outbox %d: %w", message.ID, err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `insert into analytics_sources (topic, key, recorded_at) values ($1, $2, $3) on conflict do nothing`,
		message.Topic, message.Key, outcome.EndedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	for _, player := range outcome.Players {
		if err := RecordServerIn(ctx, tx, MatchEnd, player.Account, 1, outcome.EndedAt); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

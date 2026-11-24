package rating

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Alievs-corp/hypedomino/server/internal/matchstore"
)

func (s *Store) Publish(ctx context.Context, message matchstore.Pending) error {
	var outcome matchstore.Outcome
	if err := json.Unmarshal(message.Payload, &outcome); err != nil {
		return fmt.Errorf("rating: outbox %d: %w", message.ID, err)
	}
	if outcome.Cancelled {
		return nil
	}
	result := Result{MatchID: outcome.Match, Mode: outcome.Mode, At: outcome.EndedAt, Penalties: outcome.Penalties}
	for _, player := range outcome.Players {
		if player.Team != 0 && player.Team != 1 {
			return fmt.Errorf("%w: match %s team %d", ErrResult, outcome.Match, player.Team)
		}
		result.Teams[player.Team] = append(result.Teams[player.Team], Participant{Account: player.Account, Offender: player.Offender})
		if player.Won {
			result.Winner = player.Team
		}
	}
	if _, err := s.Apply(ctx, result); err != nil {
		return fmt.Errorf("rating: match %s: %w", outcome.Match, err)
	}
	return nil
}

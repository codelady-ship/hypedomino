package matchreward

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Alievs-corp/hypedomino/server/internal/matchstore"
)

func (s *Store) Publish(ctx context.Context, message matchstore.Pending) error {
	var outcome matchstore.Outcome
	if err := json.Unmarshal(message.Payload, &outcome); err != nil {
		return fmt.Errorf("matchreward: outbox %d: %w", message.ID, err)
	}
	for _, player := range outcome.Players {
		match := Match{ID: outcome.Match, Account: player.Account, Mode: outcome.Mode, Kind: outcome.Kind, Won: player.Won, BotSeat: outcome.BotSeat, Offender: player.Offender && !player.Returned, Cancelled: outcome.Cancelled, EndedAt: outcome.EndedAt}
		if _, err := s.Award(ctx, match); err != nil {
			return fmt.Errorf("matchreward: match %s account %s: %w", outcome.Match, player.Account, err)
		}
	}
	return nil
}

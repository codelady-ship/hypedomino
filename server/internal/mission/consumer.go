package mission

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Alievs-corp/hypedomino/server/internal/matchstore"
)

func (s *Store) Publish(ctx context.Context, message matchstore.Pending) error {
	var outcome matchstore.Outcome
	if err := json.Unmarshal(message.Payload, &outcome); err != nil {
		return fmt.Errorf("mission: outbox %d: %w", message.ID, err)
	}
	for _, player := range outcome.Players {
		match := Match{Kind: outcome.Kind, Mode: outcome.Mode, Won: player.Won, BotSeat: outcome.BotSeat, Offender: player.Offender && !player.Returned, Cancelled: outcome.Cancelled, HandsWon: player.HandsWon, Dominoes: player.Dominoes, BoardPoints: player.BoardPoints, DoubleChains: player.DoubleChains}
		if _, err := s.ApplyMatch(ctx, player.Account, outcome.Match, match, outcome.EndedAt); err != nil {
			return fmt.Errorf("mission: match %s account %s: %w", outcome.Match, player.Account, err)
		}
	}
	return nil
}

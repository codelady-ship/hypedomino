package rating

import (
	"context"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

type QueueEntry struct {
	Mmr    int
	Rank   int
	Placed bool
	Avoid  []string
}

func (s *Store) Queue(ctx context.Context, account string, mode rules.Mode, now time.Time) (QueueEntry, error) {
	avoid, err := s.Avoid(ctx, account, mode, now)
	if err != nil {
		return QueueEntry{}, err
	}
	r, ok, err := s.Get(ctx, account, mode)
	if err != nil {
		return QueueEntry{}, err
	}
	if ok {
		return QueueEntry{Mmr: r.MMR, Rank: r.Rank.Index, Placed: r.Placed, Avoid: avoid}, nil
	}
	config, _, err := s.Config(mode)
	if err != nil {
		return QueueEntry{}, err
	}
	return QueueEntry{Mmr: config.InitialMmr, Avoid: avoid}, nil
}

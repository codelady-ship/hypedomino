package rating

import (
	"context"

	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

type QueueEntry struct {
	Mmr    int
	Rank   int
	Placed bool
}

func (s *Store) Queue(ctx context.Context, account string, mode rules.Mode) (QueueEntry, error) {
	r, ok, err := s.Get(ctx, account, mode)
	if err != nil {
		return QueueEntry{}, err
	}
	if ok {
		return QueueEntry{Mmr: r.MMR, Rank: r.Rank.Index, Placed: r.Placed}, nil
	}
	config, _, err := s.Config(mode)
	if err != nil {
		return QueueEntry{}, err
	}
	return QueueEntry{Mmr: config.InitialMmr}, nil
}

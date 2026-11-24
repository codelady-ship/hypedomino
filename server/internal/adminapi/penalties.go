package adminapi

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/penalties"
	"github.com/Alievs-corp/hypedomino/server/internal/rating"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

func penaltiesDomain() configDomain {
	return configDomain{name: penalties.Domain, object: rbac.GameConfig,
		prepare: func(ctx context.Context, tx pgx.Tx, _, after []byte) ([]byte, error) {
			next, err := penalties.Parse(after)
			if err != nil {
				return nil, err
			}
			ratings, err := publishedRating(ctx, tx)
			if err != nil {
				return nil, err
			}
			if ratings == nil {
				ratings = rating.Default()
			}
			return after, ratings.CheckOffensePenalty(next.OffenseRatingPenalty)
		},
		accept: func(_, after []byte) error {
			_, err := penalties.Parse(after)
			return err
		},
	}
}

func publishedPenalties(ctx context.Context, tx pgx.Tx) (penalties.Config, error) {
	_, raw, err := latestSnapshot(ctx, tx, penalties.Domain)
	if err != nil || raw == nil {
		return penalties.DefaultConfig(), err
	}
	return penalties.Parse(raw)
}

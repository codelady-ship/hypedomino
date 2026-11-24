package pass

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
)

func (s *Snapshot) Standing(ctx context.Context, db snapshot.Querier, account string, now time.Time) (int, int64, error) {
	season, ok := s.Active(now)
	if !ok {
		return 1, 0, nil
	}
	var xp int64
	err := db.QueryRow(ctx, `select xp_total from pass_progress where account_id = $1 and season_id = $2`, account, season.ID).Scan(&xp)
	if errors.Is(err, pgx.ErrNoRows) {
		return 1, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	return season.Curve.LevelFor(xp), xp, nil
}

func (s *Store) Standing(ctx context.Context, account string, now time.Time) (int, int64, error) {
	current, _, err := s.current()
	if err != nil {
		return 0, 0, err
	}
	return current.Standing(ctx, s.pool, account, now)
}

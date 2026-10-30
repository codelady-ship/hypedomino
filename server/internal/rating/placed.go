package rating

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func (s *Store) Placed(ctx context.Context, season string, mode rules.Mode, visit func(Entry) error) error {
	current, err := scanSeason(s.pool.QueryRow(ctx, `select `+seasonColumns+` from ranked_seasons where id = $1`, season))
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrSeasonUnknown
		}
		return err
	}
	if current.ClosedFor(mode) {
		return s.frozen(ctx, season, mode, visit)
	}
	rows, err := s.pool.Query(ctx, `select account_id, rank_index, lp, apex_tier, wins, lp_at, updated_at, coalesce(board_country, '') from ratings where mode = $1 and season_id = $2 and placed`, mode.String(), season)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var e Entry
		var apex *string
		var at *time.Time
		var updated time.Time
		if err := rows.Scan(&e.Account, &e.Rank.Index, &e.Rank.LP, &apex, &e.Wins, &at, &updated, &e.Country); err != nil {
			return err
		}
		e.Tier = tierOf(e.Rank, apex)
		if at != nil {
			e.LPAt = at.UTC()
		}
		e.Version = updated.UnixMilli()
		if err := visit(e); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (s *Store) frozen(ctx context.Context, season string, mode rules.Mode, visit func(Entry) error) error {
	rows, err := s.pool.Query(ctx, `select r.account_id, r.rank_index, r.lp, r.apex_tier, r.wins, r.position, s.ends_at, coalesce(r.country, '') from season_results r join ranked_seasons s on s.id = r.season_id where r.season_id = $1 and r.mode = $2 order by r.position`, season, mode.String())
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var e Entry
		var apex *string
		var position int
		var ended time.Time
		if err := rows.Scan(&e.Account, &e.Rank.Index, &e.Rank.LP, &apex, &e.Wins, &position, &ended, &e.Country); err != nil {
			return err
		}
		e.Tier = tierOf(e.Rank, apex)
		e.LPAt = ended.UTC().Add(time.Duration(position) * time.Second)
		e.Version = ended.UnixMilli()
		if err := visit(e); err != nil {
			return err
		}
	}
	return rows.Err()
}

func tierOf(rank Rank, apex *string) string {
	if apex != nil {
		return TierName(*apex)
	}
	return rank.Tier()
}

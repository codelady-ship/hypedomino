package rating

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

const notShadowbanned = `not exists (select 1 from accounts a where a.id = account_id and a.shadowbanned_at is not null)`

func (s *Store) listable(ctx context.Context, season string, mode rules.Mode, entries []Entry) ([]Entry, error) {
	accounts := make([]string, 0, len(entries))
	for _, e := range entries {
		accounts = append(accounts, e.Account)
	}
	rows, err := s.pool.Query(ctx, `select id::text from accounts where id = any($1::uuid[]) and shadowbanned_at is not null
		union select account_id::text from leaderboard_removals where account_id = any($1::uuid[]) and season_id = $2 and mode = $3`, accounts, season, mode.String())
	if err != nil {
		return nil, err
	}
	hidden, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}
	listed := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if !contains(hidden, e.Account) {
			listed = append(listed, e)
		}
	}
	return listed, nil
}

func (s *Store) shadowbanned(ctx context.Context, account string) (bool, error) {
	var hidden bool
	err := s.pool.QueryRow(ctx, `select exists (select 1 from accounts where id = $1 and shadowbanned_at is not null)`, account).Scan(&hidden)
	return hidden, err
}

func contains(accounts []string, account string) bool {
	for _, candidate := range accounts {
		if candidate == account {
			return true
		}
	}
	return false
}

func (s *Store) HideFromBoards(ctx context.Context, account string) error {
	if s.boards == nil {
		return nil
	}
	rows, err := s.pool.Query(ctx, `select season_id, mode, coalesce(board_country, '') from ratings where account_id = $1 and placed and season_id is not null`, account)
	if err != nil {
		return err
	}
	type placement struct {
		season, mode, country string
	}
	placements, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (placement, error) {
		var p placement
		err := row.Scan(&p.season, &p.mode, &p.country)
		return p, err
	})
	if err != nil {
		return err
	}
	for _, p := range placements {
		mode, err := rules.ParseMode(p.mode)
		if err != nil {
			return err
		}
		if err := s.boards.Remove(ctx, p.season, mode, []Entry{{Account: account, Country: p.country}}); err != nil {
			return err
		}
	}
	return nil
}

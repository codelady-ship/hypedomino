package rating

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

var (
	ErrNotOnBoard       = errors.New("rating.not_on_board")
	ErrRemovedFromBoard = errors.New("rating.removed_from_board")
)

type BoardRemoval struct {
	Season    string    `json:"season"`
	Mode      string    `json:"mode"`
	Account   string    `json:"account_id"`
	Country   string    `json:"country,omitempty"`
	RemovedAt time.Time `json:"removed_at"`
}

func notRemoved(season, mode, account string) string {
	return `not exists (select 1 from leaderboard_removals x where x.season_id = ` + season + ` and x.mode = ` + mode + ` and x.account_id = ` + account + `)`
}

func RemoveFromBoardIn(ctx context.Context, tx pgx.Tx, season string, mode rules.Mode, account string, now time.Time) (BoardRemoval, error) {
	removal := BoardRemoval{Season: season, Mode: mode.String(), Account: account, RemovedAt: now.UTC().Truncate(time.Microsecond)}
	err := tx.QueryRow(ctx, `select coalesce(board_country, '') from ratings where account_id = $1 and mode = $2 and season_id = $3 and placed
		union all select coalesce(country, '') from season_results where account_id = $1 and mode = $2 and season_id = $3 limit 1`, account, mode.String(), season).Scan(&removal.Country)
	if errors.Is(err, pgx.ErrNoRows) {
		return BoardRemoval{}, ErrNotOnBoard
	}
	if err != nil {
		return BoardRemoval{}, err
	}
	tag, err := tx.Exec(ctx, `insert into leaderboard_removals (season_id, mode, account_id, removed_at) values ($1, $2, $3, $4) on conflict do nothing`, season, mode.String(), account, removal.RemovedAt)
	if err != nil {
		return BoardRemoval{}, err
	}
	if tag.RowsAffected() == 0 {
		return BoardRemoval{}, ErrRemovedFromBoard
	}
	return removal, nil
}

func (s *Store) DropFromBoards(ctx context.Context, removal BoardRemoval) error {
	if s.boards == nil {
		return nil
	}
	mode, err := rules.ParseMode(removal.Mode)
	if err != nil {
		return err
	}
	return s.boards.Remove(ctx, removal.Season, mode, []Entry{{Account: removal.Account, Country: removal.Country}})
}

package matchstart

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/matchmaking"
	"github.com/Alievs-corp/hypedomino/server/internal/profile"
	"github.com/Alievs-corp/hypedomino/server/internal/room"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

type FreezeFunc func(ctx context.Context, account string) (profile.Loadout, error)

type RankLabelFunc func(ctx context.Context, account string, mode rules.Mode) (string, error)

type MutedFunc func(ctx context.Context, account string) (bool, error)

func Muted(pool *pgxpool.Pool, now func() time.Time) MutedFunc {
	return func(ctx context.Context, account string) (bool, error) {
		var muted bool
		err := pool.QueryRow(ctx, `select coalesce(muted_until > $2, false) from accounts where id = $1`, account, now()).Scan(&muted)
		return muted, err
	}
}

func ProfileRoster(profiles *profile.Store, freeze FreezeFunc, rankLabel RankLabelFunc, showcase func(ctx context.Context, account string) ([]string, error), muted MutedFunc) RosterFunc {
	return func(ctx context.Context, match matchmaking.Match) ([]room.Player, error) {
		roster := make([]room.Player, 0, len(match.Seats))
		for _, seat := range match.Seats {
			if seat.Bot {
				roster = append(roster, room.Player{Seat: seat.Seat, Bot: true})
				continue
			}
			p, _, err := profiles.Get(ctx, seat.Account)
			if err != nil {
				return nil, err
			}
			loadout, err := freeze(ctx, seat.Account)
			if err != nil {
				return nil, err
			}
			label, err := rankLabel(ctx, seat.Account, match.Mode)
			if err != nil {
				return nil, err
			}
			emblems, err := showcase(ctx, seat.Account)
			if err != nil {
				return nil, err
			}
			silenced, err := muted(ctx, seat.Account)
			if err != nil {
				return nil, err
			}
			roster = append(roster, room.Player{Seat: seat.Seat, Nickname: p.Nickname, Frame: loadout.Frame, Avatar: loadout.Avatar, Tile: loadout.Tile, RankLabel: label, Emblems: emblems, Emoji: loadout.Emojis, Muted: silenced})
		}
		return roster, nil
	}
}

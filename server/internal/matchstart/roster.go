package matchstart

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/matchmaking"
	"github.com/Alievs-corp/hypedomino/server/internal/profile"
	"github.com/Alievs-corp/hypedomino/server/internal/room"
)

func ProfileRoster(pool *pgxpool.Pool, profiles *profile.Store, now func() time.Time) RosterFunc {
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
			var muted bool
			if err := pool.QueryRow(ctx, `select coalesce(muted_until > $2, false) from accounts where id = $1`, seat.Account, now()).Scan(&muted); err != nil {
				return nil, err
			}
			roster = append(roster, room.Player{Seat: seat.Seat, Nickname: p.Nickname, Character: p.Loadout.Character, Frame: p.Loadout.Frame, Avatar: p.Loadout.Avatar, Emoji: p.Loadout.Emojis, Muted: muted})
		}
		return roster, nil
	}
}

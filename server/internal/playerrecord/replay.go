package playerrecord

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/replay"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

var ErrUnknownMatch = errors.New("playerrecord: no such match")

type Participant struct {
	Seat      rules.Seat `json:"seat"`
	Team      rules.Team `json:"team"`
	AccountID *string    `json:"account_id"`
	Nickname  *string    `json:"nickname"`
	Bot       bool       `json:"bot"`
	Result    *string    `json:"result"`
}

type MatchReplay struct {
	MatchID       string           `json:"match_id"`
	StartedAt     time.Time        `json:"started_at"`
	EndedAt       *time.Time       `json:"ended_at"`
	Mode          string           `json:"mode"`
	Kind          string           `json:"kind"`
	ConfigVersion int              `json:"config_version"`
	HandsPlayed   int              `json:"hands_played"`
	ScoreA        int              `json:"score_a"`
	ScoreB        int              `json:"score_b"`
	WinnerTeam    *rules.Team      `json:"winner_team"`
	Cancelled     bool             `json:"cancelled"`
	Participants  []Participant    `json:"participants"`
	Timeline      *replay.Timeline `json:"timeline"`
}

func LoadReplay(ctx context.Context, db Querier, matchID string) (MatchReplay, error) {
	if !isAccountID(matchID) {
		return MatchReplay{}, ErrUnknownMatch
	}
	var m MatchReplay
	var winner *int16
	var raw []byte
	err := db.QueryRow(ctx, `select id::text, started_at, ended_at, mode, kind, config_version, hands_played, score_a, score_b, winner_team, cancelled, replay
		from matches where id = $1::uuid`, matchID).Scan(&m.MatchID, &m.StartedAt, &m.EndedAt, &m.Mode, &m.Kind, &m.ConfigVersion, &m.HandsPlayed, &m.ScoreA, &m.ScoreB, &winner, &m.Cancelled, &raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return MatchReplay{}, ErrUnknownMatch
	}
	if err != nil {
		return MatchReplay{}, err
	}
	if winner != nil {
		team := rules.Team(*winner)
		m.WinnerTeam = &team
	}
	err = collect(ctx, db, &m.Participants, `select mp.seat, mp.team, mp.account_id::text, p.nickname, mp.bot, mp.result
		from match_participants mp left join profiles p on p.account_id = mp.account_id
		where mp.match_id = $1::uuid and mp.started_at = $2 order by mp.seat`, func(row pgx.CollectableRow, item *Participant) error {
		var seat, team int16
		if err := row.Scan(&seat, &team, &item.AccountID, &item.Nickname, &item.Bot, &item.Result); err != nil {
			return err
		}
		item.Seat, item.Team = rules.Seat(seat), rules.Team(team)
		return nil
	}, m.MatchID, m.StartedAt)
	if err != nil || raw == nil {
		return m, err
	}
	m.Timeline = timelineOf(raw)
	return m, nil
}

func timelineOf(raw []byte) *replay.Timeline {
	var record replay.Record
	if err := record.UnmarshalBinary(raw); err != nil {
		return &replay.Timeline{Hands: []replay.Hand{}, Problem: err.Error()}
	}
	timeline := replay.NewTimeline(record)
	return &timeline
}

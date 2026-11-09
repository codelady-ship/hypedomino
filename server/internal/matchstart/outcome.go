package matchstart

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Alievs-corp/hypedomino/server/internal/matchmaking"
	"github.com/Alievs-corp/hypedomino/server/internal/matchstore"
	"github.com/Alievs-corp/hypedomino/server/internal/profile"
	"github.com/Alievs-corp/hypedomino/server/internal/replay"
	"github.com/Alievs-corp/hypedomino/server/internal/room"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

type Archive interface {
	Begin(ctx context.Context, m matchstore.Started) error
	Finish(ctx context.Context, f matchstore.Finished) error
}

func storeKind(queue matchmaking.Queue) string {
	if queue == matchmaking.QueueRanked {
		return matchstore.KindRanked
	}
	return matchstore.KindCasual
}

func started(match matchmaking.Match, roster []room.Player, seed uint64, now time.Time) matchstore.Started {
	loadouts := map[rules.Seat]profile.Loadout{}
	for _, p := range roster {
		loadouts[p.Seat] = profile.Loadout{Avatar: p.Avatar, Frame: p.Frame, Character: p.Character, Emojis: p.Emoji}
	}
	s := matchstore.Started{ID: uuid.NewString(), StartedAt: now, Mode: match.Mode, Kind: storeKind(match.Queue), Region: match.Region, Seed: seed}
	for _, seat := range match.Seats {
		s.Participants = append(s.Participants, matchstore.Participant{Seat: seat.Seat, Account: seat.Account, Bot: seat.Bot, Loadout: loadouts[seat.Seat]})
	}
	return s
}

func finished(s matchstore.Started, record replay.Record, recorded bool, now time.Time) matchstore.Finished {
	f := matchstore.FinishedFrom(s, record, now)
	stats := map[rules.Seat]replay.SeatStats{}
	if recorded {
		stats = replay.NewTimeline(record).Stats()
	}
	outcome := matchstore.Outcome{Match: s.ID, Mode: s.Mode, Kind: s.Kind, Cancelled: f.Cancelled, EndedAt: now}
	for _, p := range s.Participants {
		if p.Bot {
			outcome.BotSeat = true
			continue
		}
		seat := stats[p.Seat]
		outcome.Players = append(outcome.Players, matchstore.PlayerOutcome{
			Account:      p.Account,
			Team:         int(p.Seat.Team()),
			Won:          f.Results[p.Seat] == matchstore.ResultWin,
			HandsWon:     int64(seat.HandsWon),
			Dominoes:     int64(seat.Dominoes),
			BoardPoints:  int64(seat.BoardPoints),
			DoubleChains: int64(seat.DoubleChains),
		})
	}
	f.Outbox = matchstore.MatchMessages(s.Kind, "match:"+s.ID, outcome)
	return f
}

func (h *Host) archived(r *room.Room, s matchstore.Started) {
	record, recorded := r.FinalRecord()
	h.finish(finished(s, record, recorded, h.clock.Now()))
}

func (h *Host) abandon(s matchstore.Started) {
	h.finish(matchstore.FinishedFrom(s, replay.Record{}, h.clock.Now()))
}

func (h *Host) finish(f matchstore.Finished) {
	if err := h.archive.Finish(context.Background(), f); err != nil {
		h.logger.Error("archiving a finished match failed", "match", f.ID, "error", err)
	}
}

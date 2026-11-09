package matchstart

import (
	"context"
	"log/slog"
	"math/rand/v2"

	"github.com/Alievs-corp/hypedomino/server/internal/bot"
	"github.com/Alievs-corp/hypedomino/server/internal/matchmaking"
	"github.com/Alievs-corp/hypedomino/server/internal/matchqueue"
	"github.com/Alievs-corp/hypedomino/server/internal/room"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

type Rooms interface {
	Open(options room.Options) (*room.Room, error)
}

type Seater interface {
	Place(id, account string, r *room.Room, seat rules.Seat) error
}

type Notifier interface {
	Found(ctx context.Context, ticket matchqueue.Ticket) error
}

type Queue interface {
	Join(ctx context.Context, ticket matchqueue.Ticket) error
}

type RosterFunc func(ctx context.Context, match matchmaking.Match) ([]room.Player, error)

type Host struct {
	rooms    Rooms
	seats    Seater
	notifier Notifier
	queue    Queue
	roster   RosterFunc
	config   func() matchmaking.Config
	clock    room.Clock
	random   *rand.Rand
	logger   *slog.Logger
}

func NewHost(rooms Rooms, seats Seater, notifier Notifier, queue Queue, roster RosterFunc, config func() matchmaking.Config, clock room.Clock, random *rand.Rand, logger *slog.Logger) *Host {
	return &Host{rooms: rooms, seats: seats, notifier: notifier, queue: queue, roster: roster, config: config, clock: clock, random: random, logger: logger}
}

func kind(queue matchmaking.Queue) room.Kind {
	if queue == matchmaking.QueueRanked {
		return room.KindRanked
	}
	return room.KindCasual
}

func (h *Host) Open(ctx context.Context, placement matchqueue.Placement) {
	match := placement.Match
	roster, err := h.roster(ctx, match)
	if err != nil {
		h.requeue(ctx, placement, err)
		return
	}
	r, err := h.rooms.Open(room.Options{Kind: kind(match.Queue), Mode: match.Mode, Config: rules.DefaultConfig(), Bot: bot.DefaultConfig(), BotLevel: h.config().BotLevel(match.Queue), Seed: h.random.Uint64(), Clock: h.clock, Roster: roster})
	if err != nil {
		h.requeue(ctx, placement, err)
		return
	}
	seats := map[string]rules.Seat{}
	for _, seat := range match.Seats {
		seats[seat.Account] = seat.Seat
	}
	for _, ticket := range placement.Tickets {
		if err := h.seats.Place(ticket.Session, ticket.Entry.Account, r, seats[ticket.Entry.Account]); err != nil {
			h.logger.Error("seating a matched player failed", "account", ticket.Entry.Account, "error", err)
			continue
		}
		if err := h.notifier.Found(ctx, ticket); err != nil {
			h.logger.Error("match found notice failed", "account", ticket.Entry.Account, "error", err)
		}
	}
}

func (h *Host) requeue(ctx context.Context, placement matchqueue.Placement, cause error) {
	h.logger.Error("opening a matched room failed", "error", cause)
	for _, ticket := range placement.Tickets {
		if err := h.queue.Join(ctx, ticket); err != nil {
			h.logger.Error("requeue failed", "account", ticket.Entry.Account, "error", err)
		}
	}
}

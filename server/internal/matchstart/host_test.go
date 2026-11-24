package matchstart_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/bot"
	"github.com/Alievs-corp/hypedomino/server/internal/matchmaking"
	"github.com/Alievs-corp/hypedomino/server/internal/matchqueue"
	"github.com/Alievs-corp/hypedomino/server/internal/matchstart"
	"github.com/Alievs-corp/hypedomino/server/internal/matchstore"
	"github.com/Alievs-corp/hypedomino/server/internal/penalties"
	"github.com/Alievs-corp/hypedomino/server/internal/room"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

type clock struct{}

func (clock) Now() time.Time                         { return time.Now() }
func (clock) After(d time.Duration) <-chan time.Time { return time.After(d) }

type fakes struct {
	options  room.Options
	openErr  error
	seated   map[string]rules.Seat
	found    []string
	requeued []string
	begun    []matchstore.Started
	finished []matchstore.Finished
}

func (f *fakes) Begin(ctx context.Context, m matchstore.Started) error {
	f.begun = append(f.begun, m)
	return nil
}

func (f *fakes) Finish(ctx context.Context, m matchstore.Finished) error {
	f.finished = append(f.finished, m)
	return nil
}

func (f *fakes) Open(options room.Options) (*room.Room, error) {
	f.options = options
	if f.openErr != nil {
		return nil, f.openErr
	}
	return room.New(options), nil
}

func (f *fakes) Place(id, account string, r *room.Room, seat rules.Seat) error {
	f.seated[id] = seat
	return nil
}

func (f *fakes) Found(ctx context.Context, ticket matchqueue.Ticket) error {
	f.found = append(f.found, ticket.Session)
	return nil
}

func (f *fakes) Join(ctx context.Context, ticket matchqueue.Ticket) error {
	f.requeued = append(f.requeued, ticket.Entry.Account)
	return nil
}

func placement() matchqueue.Placement {
	match := matchmaking.Match{Mode: rules.Mode2v2, Queue: matchmaking.QueueCasual, Seats: []matchmaking.Seat{{Seat: rules.SeatNorth, Account: "a"}, {Seat: rules.SeatEast, Bot: true}, {Seat: rules.SeatSouth, Account: "b"}, {Seat: rules.SeatWest, Bot: true}}}
	tickets := []matchqueue.Ticket{{Entry: matchmaking.Entry{Account: "a"}, Session: "sa"}, {Entry: matchmaking.Entry{Account: "b"}, Session: "sb"}}
	return matchqueue.Placement{Match: match, Tickets: tickets}
}

func host(f *fakes, roster matchstart.RosterFunc) *matchstart.Host {
	config := func() matchmaking.Config {
		c := matchmaking.DefaultConfig()
		c.CasualBotLevel = bot.Easy
		return c
	}
	return matchstart.NewHost(f, f, f, f, f, f, roster, config, clock{}, rand.New(rand.NewPCG(1, 2)), slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestAPlacementOpensARoomSeatsEveryPlayerAndAnnouncesIt(t *testing.T) {
	f := &fakes{seated: map[string]rules.Seat{}}
	roster := func(ctx context.Context, match matchmaking.Match) ([]room.Player, error) {
		return []room.Player{{Seat: rules.SeatNorth, Nickname: "A"}}, nil
	}
	host(f, roster).Open(context.Background(), placement())
	if f.options.Kind != room.KindCasual || f.options.Mode != rules.Mode2v2 || f.options.BotLevel != bot.Easy || len(f.options.Roster) != 1 {
		t.Fatalf("room options %+v", f.options)
	}
	if f.seated["sa"] != rules.SeatNorth || f.seated["sb"] != rules.SeatSouth || len(f.found) != 2 || len(f.requeued) != 0 {
		t.Fatalf("seated %v found %v requeued %v", f.seated, f.found, f.requeued)
	}
	if len(f.begun) != 1 || f.begun[0].Kind != matchstore.KindCasual || len(f.begun[0].Participants) != 4 || f.begun[0].Seed != f.options.Seed || len(f.finished) != 0 {
		t.Fatalf("begun %+v finished %+v", f.begun, f.finished)
	}
}

func TestARoomThatCannotOpenPutsThePlayersBackInTheQueue(t *testing.T) {
	f := &fakes{seated: map[string]rules.Seat{}, openErr: errors.New("draining")}
	roster := func(ctx context.Context, match matchmaking.Match) ([]room.Player, error) { return nil, nil }
	host(f, roster).Open(context.Background(), placement())
	if len(f.requeued) != 2 || len(f.seated) != 0 || len(f.found) != 0 {
		t.Fatalf("requeued %v seated %v found %v", f.requeued, f.seated, f.found)
	}
	if len(f.begun) != 1 || len(f.finished) != 1 || f.finished[0].ID != f.begun[0].ID || !f.finished[0].Cancelled || len(f.finished[0].Outbox) != 0 {
		t.Fatalf("begun %+v finished %+v", f.begun, f.finished)
	}
}

func (f *fakes) Config() (penalties.Config, int) {
	return penalties.DefaultConfig(), 0
}

func (f *fakes) MatchEnd(ctx context.Context, account string, config penalties.Config, queue matchmaking.Queue, matchID string, seat rules.Seat, report rules.PresenceReport, configVersion int, now time.Time) (penalties.Notice, error) {
	return penalties.Notice{}, nil
}

package matchqueue_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/cluster/redistest"
	"github.com/Alievs-corp/hypedomino/server/internal/matchmaking"
	"github.com/Alievs-corp/hypedomino/server/internal/matchqueue"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func TestAPlacementReachesItsHostAndTheNoticeReachesThePlayersNode(t *testing.T) {
	client, prefix := redistest.Client(t), redistest.Prefix(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	host := matchqueue.NewDispatcher(client, prefix, "host")
	origin := matchqueue.NewDispatcher(client, prefix, "origin")
	placed := make(chan matchqueue.Placement, 1)
	found := make(chan string, 1)
	go host.Listen(ctx, func(_ context.Context, p matchqueue.Placement) { placed <- p }, func(string) {})
	go origin.Listen(ctx, func(context.Context, matchqueue.Placement) {}, func(session string) { found <- session })
	ticket := matchqueue.Ticket{Entry: matchmaking.Entry{Account: "a", Mode: rules.Mode1v1}, Node: "origin", Session: "sa"}
	placement := matchqueue.Placement{Match: matchmaking.Match{Mode: rules.Mode1v1}, Tickets: []matchqueue.Ticket{ticket}}
	placeOnceHeard(t, ctx, host, "origin", placement)
	placeOnceHeard(t, ctx, origin, "host", placement)
	select {
	case p := <-placed:
		if len(p.Tickets) != 1 || p.Tickets[0].Session != "sa" {
			t.Fatalf("placement %+v", p)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no placement")
	}
	if err := host.Found(ctx, ticket); err != nil {
		t.Fatal(err)
	}
	select {
	case session := <-found:
		if session != "sa" {
			t.Fatalf("found %s", session)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no match found notice")
	}
	if err := host.Place(ctx, "gone", matchqueue.Placement{}); !errors.Is(err, matchqueue.ErrHostGone) {
		t.Fatalf("placement to a gone node %v", err)
	}
}

func placeOnceHeard(t *testing.T, ctx context.Context, from *matchqueue.Dispatcher, host string, placement matchqueue.Placement) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		err := from.Place(ctx, host, placement)
		if err == nil {
			return
		}
		if !errors.Is(err, matchqueue.ErrHostGone) || time.Now().After(deadline) {
			t.Fatalf("place on %s: %v", host, err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

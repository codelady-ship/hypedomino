package matchqueue_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/cluster/redistest"
	"github.com/Alievs-corp/hypedomino/server/internal/matchmaking"
	"github.com/Alievs-corp/hypedomino/server/internal/matchqueue"
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
	ticket := matchqueue.Ticket{Entry: matchmaking.Entry{Account: "a"}, Node: "origin", Session: "sa"}
	deadline := time.Now().Add(5 * time.Second)
	for {
		err := origin.Place(ctx, "host", matchqueue.Placement{Tickets: []matchqueue.Ticket{ticket}})
		if err == nil {
			break
		}
		if !errors.Is(err, matchqueue.ErrHostGone) || time.Now().After(deadline) {
			t.Fatalf("place %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if p := <-placed; len(p.Tickets) != 1 || p.Tickets[0].Session != "sa" {
		t.Fatalf("placement %+v", p)
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

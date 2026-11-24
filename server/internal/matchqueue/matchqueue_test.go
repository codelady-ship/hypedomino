package matchqueue_test

import (
	"context"
	"errors"
	"math/rand/v2"
	"os"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/cluster/redistest"
	"github.com/Alievs-corp/hypedomino/server/internal/matchmaking"
	"github.com/Alievs-corp/hypedomino/server/internal/matchqueue"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func TestMain(m *testing.M) {
	os.Exit(redistest.Main(m))
}

var now = time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)

func ticket(account string, mmr int) matchqueue.Ticket {
	return matchqueue.Ticket{Entry: matchmaking.Entry{Account: account, Mode: rules.Mode1v1, Queue: matchmaking.QueueRanked, Region: "eu", Mmr: mmr, EnqueuedAt: now, SearchFrom: now}, Node: "node-" + account, Session: "session-" + account}
}

func TestOnlyOneNodeLeadsARegionUntilItResigns(t *testing.T) {
	client, prefix := redistest.Client(t), redistest.Prefix(t)
	ctx := context.Background()
	first := matchqueue.NewLeader(client, prefix, "eu", "a", time.Minute)
	second := matchqueue.NewLeader(client, prefix, "eu", "b", time.Minute)
	other := matchqueue.NewLeader(client, prefix, "us", "b", time.Minute)
	for _, c := range []struct {
		leader *matchqueue.Leader
		want   bool
	}{{first, true}, {second, false}, {first, true}, {other, true}} {
		if held, err := c.leader.Hold(ctx); err != nil || held != c.want {
			t.Fatalf("hold %v, %v; want %v", held, err, c.want)
		}
	}
	if err := second.Resign(ctx); err != nil {
		t.Fatal(err)
	}
	if held, _ := second.Hold(ctx); held {
		t.Fatal("a resign by another node released the lock")
	}
	if err := first.Resign(ctx); err != nil {
		t.Fatal(err)
	}
	if held, err := second.Hold(ctx); err != nil || !held {
		t.Fatalf("after resign %v, %v", held, err)
	}
}

func TestATickTakesTheTicketsOfAFormedMatchAndAFailedPlacementReturnsThem(t *testing.T) {
	client, prefix := redistest.Client(t), redistest.Prefix(t)
	ctx := context.Background()
	store := matchqueue.NewStore(client, prefix)
	if err := store.Join(ctx, matchqueue.Ticket{Entry: matchmaking.Entry{Account: "x"}}); !errors.Is(err, matchqueue.ErrTicket) {
		t.Fatalf("incomplete ticket %v", err)
	}
	for _, tk := range []matchqueue.Ticket{ticket("a", 1200), ticket("b", 1250), ticket("far", 2400)} {
		if err := store.Join(ctx, tk); err != nil {
			t.Fatal(err)
		}
	}
	matcher := matchqueue.NewMatcher(store, matchmaking.DefaultConfig, rand.New(rand.NewPCG(1, 2)))
	placements, err := matcher.Tick(ctx, now)
	if err != nil || len(placements) != 1 || len(placements[0].Tickets) != 2 {
		t.Fatalf("placements %+v, %v", placements, err)
	}
	tickets, _, err := store.Tickets(ctx)
	if err != nil || len(tickets) != 1 || tickets["far"].Session != "session-far" {
		t.Fatalf("left %+v, %v", tickets, err)
	}
	if err := matcher.Return(ctx, placements[0]); err != nil {
		t.Fatal(err)
	}
	if left, err := store.Leave(ctx, "a"); err != nil || !left {
		t.Fatalf("leave %v, %v", left, err)
	}
	if again, err := matcher.Tick(ctx, now); err != nil || len(again) != 0 {
		t.Fatalf("after a leave %+v, %v", again, err)
	}
}

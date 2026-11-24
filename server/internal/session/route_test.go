package session

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/bot"
	"github.com/Alievs-corp/hypedomino/server/internal/room"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
	"github.com/Alievs-corp/hypedomino/server/internal/secrand"
)

type fakeRouter struct {
	mu       sync.Mutex
	claimed  []string
	released []string
	fail     error
}

func (r *fakeRouter) Claim(ctx context.Context, token string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.fail != nil {
		return r.fail
	}
	r.claimed = append(r.claimed, token)
	return nil
}

func (r *fakeRouter) Release(ctx context.Context, token string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.released = append(r.released, token)
	return nil
}

func (r *fakeRouter) counts() (int, int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.claimed), len(r.released)
}

func TestSeatingClaimsTheRouteAndLeavingReleasesIt(t *testing.T) {
	clock := &fakeClock{now: time.UnixMilli(1761480000000)}
	manager := New(DefaultConfig(), clock, secrand.Source{})
	router := &fakeRouter{}
	manager.Route(router)
	r := room.New(room.Options{Mode: rules.Mode1v1, Config: rules.DefaultConfig(), Bot: bot.DefaultConfig(), BotLevel: bot.Hard, Seed: 11, Clock: clock})
	ctx, cancel := context.WithCancel(context.Background())
	go r.Run(ctx)
	t.Cleanup(func() { cancel(); <-r.Done() })
	north := manager.Create("x")
	south := manager.Create("y")
	if err := manager.Seat(north, r, rules.SeatX); err != nil {
		t.Fatal(err)
	}
	if claimed, released := router.counts(); claimed != 1 || released != 0 || router.claimed[0] != north {
		t.Fatalf("after seating: %v %v", router.claimed, router.released)
	}
	router.fail = errors.New("redis down")
	if err := manager.Seat(south, r, rules.SeatY); err == nil || !errors.Is(err, router.fail) {
		t.Fatalf("seating without a route: %v", err)
	}
	if err := manager.Seat(south, r, rules.SeatY); !errors.Is(err, router.fail) {
		t.Fatalf("a failed seat left the session seated: %v", err)
	}
	router.fail = nil
	manager.Revoke(north)
	if claimed, released := router.counts(); claimed != 1 || released != 1 || router.released[0] != north {
		t.Fatalf("after revoke: %v %v", router.claimed, router.released)
	}
	manager.Revoke(south)
	if _, released := router.counts(); released != 1 {
		t.Fatalf("revoking an unseated session released a route: %v", router.released)
	}
}

func TestLeavingAfterTheGraceReleasesTheRoute(t *testing.T) {
	tab := newTable(t, rules.Mode1v1)
	router := &fakeRouter{}
	tab.manager.Route(router)
	south := tab.join(rules.SeatY)
	south.awaitReady()
	north := tab.join(rules.SeatX)
	north.awaitReady()
	north.socket.CloseNow()
	deadline := time.Now().Add(5 * time.Second)
	for tab.manager.Connected(tab.tokens[rules.SeatX]) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	tab.clock.advance(DefaultConfig().ReconnectGrace)
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, released := router.counts(); released == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, released := router.counts(); released != 1 || router.released[0] != tab.tokens[rules.SeatX] {
		t.Fatalf("released %v", router.released)
	}
}

func TestAFinishedMatchUnbindsTheSessionAndReleasesTheRoute(t *testing.T) {
	clock := &fakeClock{now: time.UnixMilli(1761480000000)}
	manager := New(DefaultConfig(), clock, secrand.Source{})
	router := &fakeRouter{}
	manager.Route(router)
	r := room.New(room.Options{Mode: rules.Mode1v1, Config: rules.DefaultConfig(), Bot: bot.DefaultConfig(), BotLevel: bot.Hard, Seed: 11, Clock: clock})
	ctx, cancel := context.WithCancel(context.Background())
	go r.Run(ctx)
	t.Cleanup(func() { cancel(); <-r.Done() })
	token := manager.Create("x")
	if manager.InMatch("x") {
		t.Fatal("in a match before being seated")
	}
	if err := manager.Seat(token, r, rules.SeatX); err != nil {
		t.Fatal(err)
	}
	if !manager.InMatch("x") {
		t.Fatal("not in a match after being seated")
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		select {
		case <-r.Finished():
		default:
			if time.Now().After(deadline) {
				t.Fatal("the bots did not finish the match")
			}
			clock.advance(rules.TurnLimit(rules.DefaultConfig()))
			time.Sleep(time.Millisecond)
			continue
		}
		break
	}
	for manager.InMatch("x") && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if manager.InMatch("x") {
		t.Fatal("still in a match after it finished")
	}
	if _, released := router.counts(); released != 1 || router.released[0] != token {
		t.Fatalf("released %v", router.released)
	}
	if err := manager.Seat(token, r, rules.SeatY); err != nil {
		t.Fatalf("seating again after the match: %v", err)
	}
}

func TestRenewReclaimsTheRoutesOfRoutedSessionsOnly(t *testing.T) {
	clock := &fakeClock{now: time.UnixMilli(1761480000000)}
	manager := New(DefaultConfig(), clock, secrand.Source{})
	if err := manager.Renew(context.Background()); err != nil {
		t.Fatalf("renewing without a router: %v", err)
	}
	router := &fakeRouter{}
	manager.Route(router)
	r := room.New(room.Options{Mode: rules.Mode1v1, Config: rules.DefaultConfig(), Bot: bot.DefaultConfig(), BotLevel: bot.Hard, Seed: 11, Clock: clock})
	ctx, cancel := context.WithCancel(context.Background())
	go r.Run(ctx)
	t.Cleanup(func() { cancel(); <-r.Done() })
	seated := manager.Create("x")
	manager.Create("y")
	if err := manager.Seat(seated, r, rules.SeatX); err != nil {
		t.Fatal(err)
	}
	router.claimed = nil
	if err := manager.Renew(context.Background()); err != nil {
		t.Fatal(err)
	}
	if claimed, released := router.counts(); claimed != 1 || released != 0 || router.claimed[0] != seated {
		t.Fatalf("renewed %v released %v, want only %s", router.claimed, router.released, seated)
	}
	router.fail = errors.New("redis down")
	if err := manager.Renew(context.Background()); !errors.Is(err, router.fail) {
		t.Fatalf("a failed renewal returned %v", err)
	}
	router.fail = nil
	manager.Revoke(seated)
	router.claimed = nil
	if err := manager.Renew(context.Background()); err != nil {
		t.Fatal(err)
	}
	if claimed, _ := router.counts(); claimed != 0 {
		t.Fatalf("renewed a revoked session: %v", router.claimed)
	}
}

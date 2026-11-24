package loadbot

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/botclient"
	"github.com/Alievs-corp/hypedomino/server/internal/matchstore"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

type fakeStores struct {
	mu       sync.Mutex
	started  []matchstore.Started
	finished []matchstore.Finished
	claimed  []string
	released []string
}

func (f *fakeStores) Begin(ctx context.Context, m matchstore.Started) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started = append(f.started, m)
	return nil
}

func (f *fakeStores) Finish(ctx context.Context, m matchstore.Finished) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.finished = append(f.finished, m)
	return nil
}

func (f *fakeStores) Claim(ctx context.Context, token string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.claimed = append(f.claimed, token)
	return nil
}

func (f *fakeStores) Release(ctx context.Context, token string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.released = append(f.released, token)
	return errors.New("route gone")
}

func (f *fakeStores) releasedCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.released)
}

func TestEveryRoomIsWrittenAndRoutedLikeAProductionMatch(t *testing.T) {
	target, handler := startServer(t, 30)
	fake := &fakeStores{}
	stores := NewStores(fake, fake)
	handler.Store(stores)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tokens := []string{Token(3, rules.Mode1v1, rules.SeatX), Token(3, rules.Mode1v1, rules.SeatY)}
	finished := make(chan error, 2)
	for _, token := range tokens {
		c, err := botclient.Dial(ctx, target, botclient.Options{Token: token, AppVersion: "1.0.0"})
		if err != nil {
			t.Fatal(err)
		}
		defer c.Drop()
		go func() {
			_, err := c.PlayUntil(botclient.Finished)
			finished <- err
		}()
	}
	for range tokens {
		if err := <-finished; err != nil {
			t.Fatal(err)
		}
	}
	for fake.releasedCount() < len(tokens) {
		if ctx.Err() != nil {
			t.Fatal("routes were never released")
		}
		time.Sleep(10 * time.Millisecond)
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.started) != 1 || len(fake.started[0].Participants) != 2 || !fake.started[0].Participants[0].Bot {
		t.Fatalf("started %+v", fake.started)
	}
	if len(fake.finished) != 1 {
		t.Fatalf("finished %d matches", len(fake.finished))
	}
	f := fake.finished[0]
	results := []string{f.Results[rules.SeatX], f.Results[rules.SeatY]}
	slices.Sort(results)
	if f.ID != fake.started[0].ID || !f.HasWinner || f.Cancelled || len(f.Replay) == 0 || !slices.Equal(results, []string{matchstore.ResultLoss, matchstore.ResultWin}) {
		t.Fatalf("finished %+v results %v", f, results)
	}
	if !slices.Equal(fake.claimed, tokens) || !slices.Equal(fake.released, tokens) {
		t.Fatalf("claimed %v released %v", fake.claimed, fake.released)
	}
	out := &strings.Builder{}
	stores.WriteTo(out)
	for _, line := range []string{"match_begin_seconds_count 1\n", "match_finish_seconds_count 1\n", "route_claim_seconds_count 2\n", "route_release_seconds_count 2\n", "hd_store_failures 2\n"} {
		if !strings.Contains(out.String(), line) {
			t.Fatalf("missing %q in\n%s", line, out.String())
		}
	}
}

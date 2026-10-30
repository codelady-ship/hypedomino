package rating_test

import (
	"context"
	"errors"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/matchmaking"
	"github.com/Alievs-corp/hypedomino/server/internal/rating"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func (f *fixture) enqueue(t *testing.T, pool *matchmaking.Pool, account string, mode rules.Mode, queue matchmaking.Queue, at time.Time) {
	t.Helper()
	q, err := f.store.Queue(context.Background(), account, mode)
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.Enqueue(matchmaking.Entry{Account: account, Mode: mode, Queue: queue, Region: "eu", Mmr: q.Mmr, Rank: q.Rank, Placed: q.Placed, EnqueuedAt: at}); err != nil {
		t.Fatal(err)
	}
}

func result(match matchmaking.Match, id string, winner int, at time.Time) rating.Result {
	var teams [2][]rating.Participant
	for team, members := range match.Teams() {
		for _, account := range members {
			teams[team] = append(teams[team], rating.Participant{Account: account})
		}
	}
	return rating.Result{MatchID: id, Mode: match.Mode, Teams: teams, Winner: winner, At: at}
}

func TestARankedQueueFormsFromRatingsAndOnlyRankedMatchesAreApplied(t *testing.T) {
	f := newFixture(t)
	gold, diamond, silver, fresh := f.account(t), f.account(t), f.account(t), f.account(t)
	f.set(t, gold, rules.Mode1v1, 1400, 30, rank(t, "Gold II", 40))
	f.set(t, diamond, rules.Mode1v1, 1420, 30, rank(t, "Diamond IV", 10))
	f.set(t, silver, rules.Mode1v1, 1380, 30, rank(t, "Silver I", 0))
	if q, err := f.store.Queue(context.Background(), fresh, rules.Mode1v1); err != nil || q.Placed || q.Mmr != 1200 {
		t.Fatalf("fresh queue entry %+v %v", q, err)
	}
	pool := matchmaking.NewPool(matchmaking.DefaultConfig(), rand.New(rand.NewPCG(1, 1)))
	f.enqueue(t, pool, gold, rules.Mode1v1, matchmaking.QueueRanked, now.Add(-time.Minute))
	f.enqueue(t, pool, diamond, rules.Mode1v1, matchmaking.QueueRanked, now.Add(-time.Minute))
	f.enqueue(t, pool, silver, rules.Mode1v1, matchmaking.QueueRanked, now.Add(-time.Minute))
	if err := pool.Enqueue(matchmaking.Entry{Account: gold, Mode: rules.Mode1v1, Queue: matchmaking.QueueRanked, Region: "eu", EnqueuedAt: now}); !errors.Is(err, matchmaking.ErrAlreadyQueued) {
		t.Fatalf("second entry of the same account: %v", err)
	}
	matches := pool.Tick(now)
	if len(matches) != 1 || !matches[0].Ranked() || pool.Len() != 1 || !pool.Queued(diamond) {
		t.Fatalf("ranked tick formed %+v, queued %d", matches, pool.Len())
	}
	applied, err := f.store.Apply(context.Background(), result(matches[0], "aaaaaaaa-1111-1111-1111-aaaaaaaaaaaa", 0, now))
	if err != nil || len(applied.Changes) != 2 {
		t.Fatalf("apply %+v %v", applied, err)
	}
	if r := f.rating(t, silver, rules.Mode1v1); r.Matches != 31 {
		t.Fatalf("silver after the ranked match %+v", r)
	}
	casual := matchmaking.NewPool(matchmaking.DefaultConfig(), rand.New(rand.NewPCG(2, 2)))
	f.enqueue(t, casual, gold, rules.Mode1v1, matchmaking.QueueCasual, now)
	f.enqueue(t, casual, diamond, rules.Mode1v1, matchmaking.QueueCasual, now)
	formed := casual.Tick(now)
	if len(formed) != 1 || formed[0].Ranked() {
		t.Fatalf("casual tick formed %+v", formed)
	}
	for _, match := range formed {
		if match.Ranked() {
			t.Fatal("a casual match reached the rating store")
		}
	}
	if r := f.rating(t, gold, rules.Mode1v1); r.Matches != 31 {
		t.Fatalf("gold after the casual match %+v", r)
	}
}

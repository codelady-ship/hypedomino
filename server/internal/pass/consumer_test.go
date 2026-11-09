package pass_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/matchstore"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func TestTheXpRelayAwardsEveryPlayerOfAFinishedMatchOnce(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	winner, loser, returned := f.account(t), f.account(t), f.account(t)
	payload, _ := json.Marshal(matchstore.Outcome{Match: "m1", Mode: rules.Mode1v1, Kind: matchstore.KindRanked, EndedAt: now, Players: []matchstore.PlayerOutcome{{Account: winner, Won: true}, {Account: loser, Offender: true}, {Account: returned, Offender: true, Returned: true}}})
	if _, err := f.pool.Exec(ctx, `insert into outbox (topic, key, payload) values ($1, 'match:m1', $2)`, matchstore.TopicXp, payload); err != nil {
		t.Fatal(err)
	}
	relay := matchstore.NewRelay(matchstore.New(f.pool), f.store, matchstore.TopicXp, 0, nil)
	for range 2 {
		if published, err := relay.Once(ctx); err != nil || published != 1 {
			t.Fatalf("published %d, %v", published, err)
		}
		if _, err := f.pool.Exec(ctx, `update outbox set published_at = null where topic = $1`, matchstore.TopicXp); err != nil {
			t.Fatal(err)
		}
	}
	if got := f.xp(t, winner); got != 100 {
		t.Fatalf("winner xp %d, want 100", got)
	}
	if got := f.count(t, `select count(*) from pass_progress where account_id = $1`, loser); got != 0 {
		t.Fatalf("offender progress rows %d", got)
	}
	if got := f.count(t, `select count(*) from pass_progress where account_id = $1`, returned); got != 1 {
		t.Fatalf("a returned offender keeps the match xp, progress rows %d", got)
	}
}

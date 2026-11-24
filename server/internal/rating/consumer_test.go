package rating_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/matchstore"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func TestTheRatingRelayAppliesARankedResultOnceAndSkipsACancelledMatch(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	winner, loser := f.account(t), f.account(t)
	f.set(t, winner, rules.Mode1v1, 1300, 30, rank(t, "Silver I", 40))
	f.set(t, loser, rules.Mode1v1, 1250, 30, rank(t, "Silver I", 0))
	for id, cancelled := range map[string]bool{"11111111-1111-1111-1111-111111111111": false, "22222222-2222-2222-2222-222222222222": true} {
		payload, _ := json.Marshal(matchstore.Outcome{Match: id, Mode: rules.Mode1v1, Kind: matchstore.KindRanked, Cancelled: cancelled, EndedAt: now, Players: []matchstore.PlayerOutcome{{Account: loser, Team: 1}, {Account: winner, Team: 0, Won: !cancelled}}})
		if _, err := f.pool.Exec(ctx, `insert into outbox (topic, key, payload) values ($1, $2, $3)`, matchstore.TopicRating, "match:"+id, payload); err != nil {
			t.Fatal(err)
		}
	}
	relay := matchstore.NewRelay(matchstore.New(f.pool), f.store, matchstore.TopicRating, 0, nil)
	for range 2 {
		if published, err := relay.Once(ctx); err != nil || published != 2 {
			t.Fatalf("published %d, %v", published, err)
		}
		if _, err := f.pool.Exec(ctx, `update outbox set published_at = null where topic = $1`, matchstore.TopicRating); err != nil {
			t.Fatal(err)
		}
	}
	if won, lost := f.rating(t, winner, rules.Mode1v1), f.rating(t, loser, rules.Mode1v1); won.Matches != 31 || won.MMR <= 1300 || lost.Matches != 31 || lost.MMR >= 1250 {
		t.Fatalf("winner %+v, loser %+v", won, lost)
	}
	var results int
	if err := f.pool.QueryRow(ctx, `select count(*) from rating_results`).Scan(&results); err != nil || results != 1 {
		t.Fatalf("results %d, %v", results, err)
	}
}

func TestTheRatingRelayUsesThePenaltiesInForceWhenTheMatchStarted(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	f.store.UseOffensePenalties(func() (int, int) { return 0, 0 })
	matches := map[string]*matchstore.OffensePenalties{"33333333-3333-3333-3333-333333333333": {RatingPenalty: 15, PartnerLossReductionPercent: 50}, "44444444-4444-4444-4444-444444444444": nil}
	offenders := map[string]string{}
	for id, penalties := range matches {
		winner, offender := f.account(t), f.account(t)
		f.set(t, winner, rules.Mode1v1, 1300, 30, rank(t, "Silver I", 40))
		f.set(t, offender, rules.Mode1v1, 1300, 30, rank(t, "Silver I", 40))
		offenders[id] = offender
		payload, _ := json.Marshal(matchstore.Outcome{Match: id, Mode: rules.Mode1v1, Kind: matchstore.KindRanked, EndedAt: now, Penalties: penalties, Players: []matchstore.PlayerOutcome{{Account: offender, Team: 1, Offender: true}, {Account: winner, Team: 0, Won: true}}})
		if _, err := f.pool.Exec(ctx, `insert into outbox (topic, key, payload) values ($1, $2, $3)`, matchstore.TopicRating, "match:"+id, payload); err != nil {
			t.Fatal(err)
		}
	}
	relay := matchstore.NewRelay(matchstore.New(f.pool), f.store, matchstore.TopicRating, 0, nil)
	if published, err := relay.Once(ctx); err != nil || published != 2 {
		t.Fatalf("published %d, %v", published, err)
	}
	delta := func(id string) int {
		var lp int
		if err := f.pool.QueryRow(ctx, `select lp_delta from rating_changes where match_id = $1 and account_id = $2`, id, offenders[id]).Scan(&lp); err != nil {
			t.Fatal(err)
		}
		return lp
	}
	if atStart, live := delta("33333333-3333-3333-3333-333333333333"), delta("44444444-4444-4444-4444-444444444444"); atStart != live-15 {
		t.Fatalf("lp delta with the penalty at start %d, with the live penalty %d", atStart, live)
	}
}

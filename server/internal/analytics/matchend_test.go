package analytics_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/analytics"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/matchstore"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

const (
	playerA = "0b6f3c1e-1a2b-4c3d-8e4f-5a6b7c8d9e01"
	playerB = "0b6f3c1e-1a2b-4c3d-8e4f-5a6b7c8d9e02"
	playerC = "0b6f3c1e-1a2b-4c3d-8e4f-5a6b7c8d9e03"
	playerD = "0b6f3c1e-1a2b-4c3d-8e4f-5a6b7c8d9e04"
)

func queueOutcome(t *testing.T, pool *pgxpool.Pool, outcome matchstore.Outcome) {
	t.Helper()
	payload, err := json.Marshal(outcome)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `insert into outbox (topic, key, payload) values ($1, $2, $3)`, matchstore.TopicAnalytics, "match:"+outcome.Match, payload); err != nil {
		t.Fatal(err)
	}
}

func matchEnds(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var count, identified int
	err := pool.QueryRow(context.Background(), `select count(*), count(*) filter (where analytics_id is not null or client_time_ms is not null or label <> '' or value <> 0 or received_at <> $2)
		from analytics_events where name = $1`, analytics.MatchEnd, now).Scan(&count, &identified)
	if err != nil || identified != 0 {
		t.Fatalf("match_end events: %d with an id, a device time, a label, a value or another time, %v", identified, err)
	}
	return count
}

func TestTheAnalyticsRelayWritesOneAnonymousMatchEndPerPlayerOnce(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	ctx := context.Background()
	queueOutcome(t, pool, matchstore.Outcome{Match: "m1", Mode: rules.Mode1v1, Kind: matchstore.KindRanked, EndedAt: now, Players: []matchstore.PlayerOutcome{{Account: playerA, Won: true}, {Account: playerB}}})
	queueOutcome(t, pool, matchstore.Outcome{Match: "m2", Mode: rules.Mode2v2, Kind: matchstore.KindCasual, BotSeat: true, Cancelled: true, EndedAt: now, Players: []matchstore.PlayerOutcome{{Account: playerA}, {Account: playerC}, {Account: playerD}}})
	relay := matchstore.NewRelay(matchstore.New(pool), analytics.NewStore(pool), matchstore.TopicAnalytics, 0, nil)
	for range 2 {
		if published, err := relay.Once(ctx); err != nil || published != 2 {
			t.Fatalf("published %d, %v", published, err)
		}
		if _, err := pool.Exec(ctx, `update outbox set published_at = null where topic = $1`, matchstore.TopicAnalytics); err != nil {
			t.Fatal(err)
		}
	}
	if got := matchEnds(t, pool); got != 5 {
		t.Fatalf("%d match_end events, want one per player of both matches", got)
	}
}

func TestTheAnalyticsRelayKeepsAnUnreadableOutcomePending(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `insert into outbox (topic, key, payload) values ($1, 'match:bad', '"not an outcome"')`, matchstore.TopicAnalytics); err != nil {
		t.Fatal(err)
	}
	relay := matchstore.NewRelay(matchstore.New(pool), analytics.NewStore(pool), matchstore.TopicAnalytics, 0, nil)
	if published, err := relay.Once(ctx); err == nil || published != 0 {
		t.Fatalf("published %d, %v", published, err)
	}
	if got := matchEnds(t, pool); got != 0 {
		t.Fatalf("%d match_end events from an unreadable outcome", got)
	}
}

func TestPurgeRawDropsServerEventsAndTheirSourcesBeforeTheCutoff(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	ctx := context.Background()
	store := analytics.NewStore(pool)
	queueOutcome(t, pool, matchstore.Outcome{Match: "m1", Mode: rules.Mode1v1, Kind: matchstore.KindRanked, EndedAt: now, Players: []matchstore.PlayerOutcome{{Account: playerA}, {Account: playerB}}})
	if _, err := matchstore.NewRelay(matchstore.New(pool), store, matchstore.TopicAnalytics, 0, nil).Once(ctx); err != nil {
		t.Fatal(err)
	}
	if deleted, err := store.PurgeRaw(ctx, now); err != nil || deleted != 0 {
		t.Fatalf("purge at the match end deleted %d, %v", deleted, err)
	}
	if deleted, err := store.PurgeRaw(ctx, now.Add(time.Second)); err != nil || deleted != 2 {
		t.Fatalf("purge after the match end deleted %d, %v", deleted, err)
	}
	var sources int
	if err := pool.QueryRow(ctx, `select count(*) from analytics_sources`).Scan(&sources); err != nil || sources != 0 {
		t.Fatalf("%d analytics sources kept, %v", sources, err)
	}
}

func TestTheAnalyticsRelayBindsMatchEndOnlyForAPlayerWithAnalyticsOn(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	ayla, bob := w.player(), w.stranger()
	w.sendWith(ayla, id, now)
	w.sendWith(bob, othersID, now)
	queueOutcome(t, w.pool, matchstore.Outcome{Match: "m1", Mode: rules.Mode1v1, Kind: matchstore.KindRanked, EndedAt: now, Players: []matchstore.PlayerOutcome{{Account: ayla, Won: true}, {Account: bob}}})
	if _, err := matchstore.NewRelay(matchstore.New(w.pool), analytics.NewStore(w.pool), matchstore.TopicAnalytics, 0, nil).Once(ctx); err != nil {
		t.Fatal(err)
	}
	var events, bound int
	var analyticsID string
	err := w.pool.QueryRow(ctx, `select count(*), count(analytics_id), coalesce(max(analytics_id::text), '') from analytics_events where name = $1`, analytics.MatchEnd).Scan(&events, &bound, &analyticsID)
	if err != nil || events != 2 || bound != 1 || analyticsID != id {
		t.Fatalf("%d match_end events, %d bound to %q, %v", events, bound, analyticsID, err)
	}
}

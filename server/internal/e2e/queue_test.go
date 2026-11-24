package e2e

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"math/rand/v2"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/Alievs-corp/hypedomino/server/internal/app"
	"github.com/Alievs-corp/hypedomino/server/internal/botclient"
	"github.com/Alievs-corp/hypedomino/server/internal/cluster/redistest"
	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/inventory"
	"github.com/Alievs-corp/hypedomino/server/internal/matchmaking"
	"github.com/Alievs-corp/hypedomino/server/internal/matchqueue"
	"github.com/Alievs-corp/hypedomino/server/internal/matchstart"
	"github.com/Alievs-corp/hypedomino/server/internal/matchstore"
	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
	"github.com/Alievs-corp/hypedomino/server/internal/penalties"
	"github.com/Alievs-corp/hypedomino/server/internal/profile"
	"github.com/Alievs-corp/hypedomino/server/internal/rating"
)

func TestMain(m *testing.M) {
	code := m.Run()
	dbtest.Stop()
	redistest.Stop()
	os.Exit(code)
}

const hostNode = "node-e2e"

func always() bool { return true }

func startMatchmaking(t *testing.T, ctx context.Context, pool *pgxpool.Pool, client *redis.Client, srv *server, clock frozenClock) (*matchqueue.Store, *penalties.Store) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	catalog, err := cosmetics.Load("../../../docs/cosmetics/examples/catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	prefix := redistest.Prefix(t)
	store := matchqueue.NewStore(client, prefix)
	records := penalties.NewStore(pool, func() (penalties.Config, int) { return penalties.DefaultConfig(), 0 })
	ratings := rating.NewStore(pool, rating.NewWatcher(pool, logger))
	srv.sessions.Queue(matchqueue.NewGate(store, hostNode, "", matchqueue.PenalizedEntry(records.Record, matchqueue.RatedEntry(ratings)), always, always, time.Now, logger))
	roster := matchstart.ProfileRoster(profile.NewStore(pool, profile.DefaultConfig()), inventory.NewStore(pool, catalog, inventory.DefaultConfig()).Freeze, ratings.Label, ratings.Showcase, matchstart.Muted(pool, time.Now))
	rooms := app.NewRooms()
	t.Cleanup(rooms.Stop)
	dispatcher := matchqueue.NewDispatcher(client, prefix, hostNode)
	host := matchstart.NewHost(rooms, srv.sessions, dispatcher, store, matchstore.New(pool), records, roster, matchmaking.DefaultConfig, clock, rand.New(rand.NewPCG(7, 8)), logger)
	go dispatcher.Listen(ctx, host.Open, srv.sessions.MatchFound)
	place := func(ctx context.Context, placement matchqueue.Placement) error {
		return dispatcher.Place(ctx, hostNode, placement)
	}
	matcher := matchqueue.NewMatcher(store, matchmaking.DefaultConfig, rand.New(rand.NewPCG(1, 2)))
	go matchqueue.Run(ctx, matchqueue.NewLeader(client, prefix, "", hostNode, time.Second), matcher, 20*time.Millisecond, time.Now, place, func(error) {})
	return store, records
}

func queuedPlayer(t *testing.T, ctx context.Context, pool *pgxpool.Pool, srv *server, nickname string) (string, *botclient.Client) {
	t.Helper()
	var account string
	if err := pool.QueryRow(ctx, `insert into accounts default values returning id`).Scan(&account); err != nil {
		t.Fatal(err)
	}
	if _, err := profile.NewStore(pool, profile.DefaultConfig()).SetNickname(ctx, account, nickname, time.Now()); err != nil {
		t.Fatal(err)
	}
	c, err := botclient.Dial(ctx, srv.url, botclient.Options{Token: srv.sessions.Create(account), AppVersion: "1.0.0", Platform: pb.Platform_PLATFORM_ANDROID})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	join := &pb.JoinQueueIntent{Mode: pb.Mode_MODE_1V1, Queue: pb.QueueKind_QUEUE_KIND_RANKED}
	if err := c.Send(&pb.Envelope{Payload: &pb.Envelope_JoinQueueIntent{JoinQueueIntent: join}}); err != nil {
		t.Fatal(err)
	}
	if status := nextQueueStatus(t, c); status != pb.QueueStatus_QUEUE_STATUS_SEARCHING {
		t.Fatalf("%s join answered %v", nickname, status)
	}
	return account, c
}

func nextQueueStatus(t *testing.T, c *botclient.Client) pb.QueueStatus {
	t.Helper()
	for {
		frame, err := c.Read()
		if err != nil {
			t.Fatal(err)
		}
		if state := frame.GetQueueState(); state != nil {
			return state.GetStatus()
		}
	}
}

func playFoundMatch(c *botclient.Client, finals chan<- *pb.MatchView, failures chan<- error) {
	for {
		frame, err := c.Read()
		if err != nil {
			failures <- err
			return
		}
		if frame.GetQueueState().GetStatus() == pb.QueueStatus_QUEUE_STATUS_MATCH_FOUND {
			break
		}
	}
	if err := c.Ready(); err != nil {
		failures <- err
		return
	}
	final, err := c.PlayUntil(botclient.Finished)
	if err != nil {
		failures <- err
		return
	}
	finals <- final
}

func archivedMatch(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (string, bool, bool) {
	t.Helper()
	for {
		var id, kind string
		var cancelled bool
		var winner *int16
		err := pool.QueryRow(ctx, `select id, kind, cancelled, winner_team from matches where ended_at is not null`).Scan(&id, &kind, &cancelled, &winner)
		if err == nil {
			return id, kind == matchstore.KindRanked && !cancelled, winner != nil
		}
		select {
		case <-ctx.Done():
			t.Fatalf("no finished match archived: %v", err)
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func TestTwoQueuedClientsPlayARankedMatchThatIsArchivedOnceWithItsOutbox(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	client := redistest.Client(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	clock := frozenClock{now: time.Now().UTC()}
	srv := startServer(t, clock, 11)
	t.Cleanup(srv.stop)
	store, records := startMatchmaking(t, ctx, pool, client, srv, clock)

	ayla, first := queuedPlayer(t, ctx, pool, srv, "Ayla")
	equipUnowned(t, ctx, pool, ayla, "avatar.tile_wizard")
	kamal, second := queuedPlayer(t, ctx, pool, srv, "Kamal")
	finals, failures := make(chan *pb.MatchView, 2), make(chan error, 2)
	go playFoundMatch(first, finals, failures)
	go playFoundMatch(second, finals, failures)
	for range 2 {
		select {
		case final := <-finals:
			var nicknames []string
			for _, entry := range final.GetRoster() {
				nicknames = append(nicknames, entry.GetNickname())
				if entry.GetNickname() == "Ayla" && entry.GetAvatar() != "avatar.starter" {
					t.Fatalf("Ayla shows the unowned avatar %q", entry.GetAvatar())
				}
			}
			if slices.Sort(nicknames); !slices.Equal(nicknames, []string{"Ayla", "Kamal"}) {
				t.Fatalf("roster %v", nicknames)
			}
		case err := <-failures:
			t.Fatal(err)
		case <-ctx.Done():
			t.Fatal("the matched clients did not finish")
		}
	}

	id, ranked, won := archivedMatch(t, ctx, pool)
	if !ranked || !won {
		t.Fatalf("match %s archived ranked %v with winner %v", id, ranked, won)
	}
	assertResults(t, ctx, pool, id)
	assertRecordedAvatar(t, ctx, pool, id, ayla, "avatar.starter")
	assertOutbox(t, ctx, pool, id, ayla, kamal)
	for _, account := range []string{ayla, kamal} {
		record, err := records.Record(ctx, account)
		if err != nil || len(record.Ranked.Completed) != 1 || record.Ranked.Completed[0].MatchID != id || len(record.Ranked.Offenses) != 0 {
			t.Fatalf("penalty record of %s: %+v, %v", account, record, err)
		}
	}
	if tickets, _, err := store.Tickets(ctx); err != nil || len(tickets) != 0 {
		t.Fatalf("tickets left %+v, %v", tickets, err)
	}
	assertConsumersApplyTheMatchOnce(t, ctx, pool, id, clock.now)
}

func equipUnowned(t *testing.T, ctx context.Context, pool *pgxpool.Pool, account, avatar string) {
	t.Helper()
	if _, err := pool.Exec(ctx, `update profiles set loadout = jsonb_set(loadout, '{avatar}', to_jsonb($2::text)) where account_id = $1`, account, avatar); err != nil {
		t.Fatal(err)
	}
}

func assertRecordedAvatar(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id, account, want string) {
	t.Helper()
	var got string
	if err := pool.QueryRow(ctx, `select loadout->>'avatar' from match_participants where match_id = $1 and account_id = $2`, id, account).Scan(&got); err != nil || got != want {
		t.Fatalf("recorded avatar of %s = %q, %v", account, got, err)
	}
}

func assertResults(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id string) {
	t.Helper()
	rows, err := pool.Query(ctx, `select result from match_participants where match_id = $1 order by result`, id)
	if err != nil {
		t.Fatal(err)
	}
	var results []string
	for rows.Next() {
		var result string
		if err := rows.Scan(&result); err != nil {
			t.Fatal(err)
		}
		results = append(results, result)
	}
	if !slices.Equal(results, []string{matchstore.ResultLoss, matchstore.ResultWin}) {
		t.Fatalf("results %v, %v", results, rows.Err())
	}
}

func assertOutbox(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id string, accounts ...string) {
	t.Helper()
	rows, err := pool.Query(ctx, `select topic, key, payload from outbox order by topic`)
	if err != nil {
		t.Fatal(err)
	}
	var topics []string
	for rows.Next() {
		var topic, key string
		var payload []byte
		if err := rows.Scan(&topic, &key, &payload); err != nil {
			t.Fatal(err)
		}
		topics = append(topics, topic)
		var outcome matchstore.Outcome
		if err := json.Unmarshal(payload, &outcome); err != nil || key != "match:"+id || outcome.Match != id || outcome.Cancelled || outcome.BotSeat {
			t.Fatalf("%s message %s: %s, %v", topic, key, payload, err)
		}
		assertPlayers(t, topic, outcome.Players, accounts)
	}
	want := []string{matchstore.TopicAnalytics, matchstore.TopicEconomy, matchstore.TopicMissions, matchstore.TopicRating, matchstore.TopicXp}
	if !slices.Equal(topics, want) {
		t.Fatalf("outbox topics %v, %v", topics, rows.Err())
	}
}

func assertPlayers(t *testing.T, topic string, players []matchstore.PlayerOutcome, accounts []string) {
	t.Helper()
	wins := 0
	for _, p := range players {
		if !slices.Contains(accounts, p.Account) || p.Offender || p.Returned {
			t.Fatalf("%s player %+v", topic, p)
		}
		if p.Won {
			wins++
		}
	}
	if len(players) != len(accounts) || wins != 1 {
		t.Fatalf("%s players %+v", topic, players)
	}
}

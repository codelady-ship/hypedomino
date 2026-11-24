package matchreward_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/matchreward"
	"github.com/Alievs-corp/hypedomino/server/internal/matchstore"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

func TestAPublishedConfigPaysTheNextMatchesUpToItsDailyCap(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	ctx := context.Background()
	config := matchreward.Default()
	config.Ranked1v1 = matchreward.Coins{Win: 50, Loss: 20}
	config.DailyMatchCoinCap = 60
	config.FirstWinBonusCoins = 7
	raw, _ := json.Marshal(config)
	version, err := snapshot.Publish(ctx, pool, matchreward.Domain, raw)
	if err != nil {
		t.Fatal(err)
	}
	watcher := matchreward.NewWatcher(pool, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if _, err := watcher.Load(ctx); err != nil {
		t.Fatal(err)
	}
	var winner, loser string
	for _, account := range []*string{&winner, &loser} {
		if err := pool.QueryRow(ctx, `insert into accounts (created_at) values ($1) returning id`, start.Add(-time.Hour)).Scan(account); err != nil {
			t.Fatal(err)
		}
	}
	for i, id := range []string{"m1", "m2"} {
		outcome := matchstore.Outcome{Match: id, Mode: rules.Mode1v1, Kind: matchstore.KindRanked, EndedAt: start.Add(time.Duration(i) * time.Minute), Players: []matchstore.PlayerOutcome{{Account: winner, Won: true}, {Account: loser}}}
		payload, _ := json.Marshal(outcome)
		if _, err := pool.Exec(ctx, `insert into outbox (topic, key, payload) values ($1, $2, $3)`, matchstore.TopicEconomy, "match:"+id, payload); err != nil {
			t.Fatal(err)
		}
	}
	relay := matchstore.NewRelay(matchstore.New(pool), matchreward.NewStore(pool, watcher, wallet.NewStore(pool)), matchstore.TopicEconomy, 0, nil)
	if published, err := relay.Once(ctx); err != nil || published != 2 {
		t.Fatalf("published %d, %v", published, err)
	}
	for account, want := range map[string]string{winner: "m1:match_reward:50 m1:first_win:7 m2:match_reward:10", loser: "m1:match_reward:20 m2:match_reward:20"} {
		var got string
		if err := pool.QueryRow(ctx, `select coalesce(string_agg(source_ref || ':' || reason || ':' || amount, ' ' order by id), '') from ledger_entries where account_id = $1 and config_version = $2`, account, version).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("ledger %q, want %q", got, want)
		}
	}
}

func TestTheEconomyRelayPaysEveryPlayerOfAFinishedMatchOnce(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	ctx := context.Background()
	store := matchreward.NewStore(pool, matchreward.NewWatcher(pool, slog.New(slog.NewTextHandler(io.Discard, nil))), wallet.NewStore(pool))
	var winner, loser string
	for _, account := range []*string{&winner, &loser} {
		if err := pool.QueryRow(ctx, `insert into accounts (created_at) values ($1) returning id`, start.Add(-time.Hour)).Scan(account); err != nil {
			t.Fatal(err)
		}
	}
	outcome := matchstore.Outcome{Match: "m1", Mode: rules.Mode1v1, Kind: matchstore.KindRanked, EndedAt: start, Players: []matchstore.PlayerOutcome{{Account: winner, Won: true}, {Account: loser}}}
	payload, _ := json.Marshal(outcome)
	for _, topic := range []string{matchstore.TopicEconomy, matchstore.TopicXp} {
		if _, err := pool.Exec(ctx, `insert into outbox (topic, key, payload) values ($1, 'match:m1', $2)`, topic, payload); err != nil {
			t.Fatal(err)
		}
	}
	relay := matchstore.NewRelay(matchstore.New(pool), store, matchstore.TopicEconomy, 0, nil)
	if published, err := relay.Once(ctx); err != nil || published != 1 {
		t.Fatalf("published %d, %v", published, err)
	}
	if _, err := pool.Exec(ctx, `update outbox set published_at = null where topic = $1`, matchstore.TopicEconomy); err != nil {
		t.Fatal(err)
	}
	if published, err := relay.Once(ctx); err != nil || published != 1 {
		t.Fatalf("redelivery published %d, %v", published, err)
	}
	for account, want := range map[string]int64{winner: 130, loser: 10} {
		var got int64
		if err := pool.QueryRow(ctx, `select coalesce(sum(amount), 0) from ledger_entries where account_id = $1`, account).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("account paid %d, want %d", got, want)
		}
	}
}

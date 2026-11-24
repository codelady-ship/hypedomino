package mission_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/matchstore"
	"github.com/Alievs-corp/hypedomino/server/internal/mission"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

func TestTheMissionsRelayAppliesAFinishedMatchToEveryPlayerOnce(t *testing.T) {
	f := newFixture(t, periodic("daily_play_1", mission.Daily, 1, mission.MatchCompleted, 1), periodic("weekly_win_9", mission.Weekly, 1, mission.MatchWon, 9))
	ctx := context.Background()
	winner, loser := f.account(t), f.account(t)
	payload, _ := json.Marshal(matchstore.Outcome{Match: "m1", Mode: rules.Mode1v1, Kind: matchstore.KindRanked, EndedAt: now, Players: []matchstore.PlayerOutcome{{Account: winner, Won: true}, {Account: loser}}})
	if _, err := f.pool.Exec(ctx, `insert into outbox (topic, key, payload) values ($1, 'match:m1', $2)`, matchstore.TopicMissions, payload); err != nil {
		t.Fatal(err)
	}
	relay := matchstore.NewRelay(matchstore.New(f.pool), f.store, matchstore.TopicMissions, 0, nil)
	for range 2 {
		if published, err := relay.Once(ctx); err != nil || published != 1 {
			t.Fatalf("published %d, %v", published, err)
		}
		if _, err := f.pool.Exec(ctx, `update outbox set published_at = null where topic = $1`, matchstore.TopicMissions); err != nil {
			t.Fatal(err)
		}
	}
	for _, account := range []string{winner, loser} {
		if balance, err := f.wallet.Balance(ctx, account, wallet.Coin); err != nil || balance != 30 {
			t.Fatalf("balance %d, %v, want the 30 slot coins once", balance, err)
		}
	}
}

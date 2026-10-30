package rating_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/rating"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func (f *fixture) play(t *testing.T, id string, winner, loser string, at time.Time, offender bool) {
	t.Helper()
	teams := [2][]rating.Participant{{{Account: winner}}, {{Account: loser, Offender: offender}}}
	if _, err := f.store.Apply(context.Background(), rating.Result{MatchID: id, Mode: rules.Mode1v1, Teams: teams, Winner: 0, At: at}); err != nil {
		t.Fatal(err)
	}
}

func kinds(signals []rating.Signal) string {
	out := make([]string, 0, len(signals))
	for _, s := range signals {
		out = append(out, fmt.Sprintf("%s:%d/%d", s.Kind, s.Wins, s.Matches))
	}
	return strings.Join(out, " ")
}

func TestFeedingAndWinTradingBetweenTheSamePlayersAreFlagged(t *testing.T) {
	f := newFixture(t)
	booster, feeder, honest := f.account(t), f.account(t), f.account(t)
	for _, account := range []string{booster, feeder, honest} {
		f.set(t, account, rules.Mode1v1, 1300, 30, rank(t, "Silver I", 40))
	}
	for i := 0; i < 5; i++ {
		f.play(t, fmt.Sprintf("a0000000-0000-0000-0000-0000000000%02d", i), booster, feeder, now.Add(time.Duration(i)*time.Hour), i == 4)
	}
	if signals, _ := f.store.Signals(context.Background(), feeder); len(signals) != 0 {
		t.Fatalf("flagged below the minimum: %v", kinds(signals))
	}
	avoid, err := f.store.Avoid(context.Background(), booster, rules.Mode1v1, now.Add(5*time.Hour))
	if err != nil || strings.Join(avoid, ",") != feeder {
		t.Fatalf("avoid %v %v", avoid, err)
	}
	if avoid, _ := f.store.Avoid(context.Background(), booster, rules.Mode1v1, now.Add(30*time.Hour)); len(avoid) != 0 {
		t.Fatalf("avoid outside the window %v", avoid)
	}
	if q, err := f.store.Queue(context.Background(), feeder, rules.Mode1v1, now.Add(5*time.Hour)); err != nil || strings.Join(q.Avoid, ",") != booster {
		t.Fatalf("queue entry %+v %v", q, err)
	}
	f.play(t, "a0000000-0000-0000-0000-000000000005", booster, feeder, now.Add(5*time.Hour), false)
	signals, err := f.store.Signals(context.Background(), feeder)
	if err != nil || kinds(signals) != "feeding:0/6 pair_frequency:0/6" || signals[0].Offenses != 1 || signals[0].Counterpart != booster {
		t.Fatalf("feeder signals %s %+v %v", kinds(signals), signals, err)
	}
	if signals, _ := f.store.Signals(context.Background(), booster); kinds(signals) != "feeding:6/6 pair_frequency:6/6" {
		t.Fatalf("booster signals %s", kinds(signals))
	}
	f.play(t, "a0000000-0000-0000-0000-000000000006", feeder, booster, now.Add(6*time.Hour), false)
	if signals, _ := f.store.Signals(context.Background(), feeder); kinds(signals) != "feeding:1/7 pair_frequency:1/7" {
		t.Fatalf("updated signals %s", kinds(signals))
	}
	var published int
	if err := f.pool.QueryRow(context.Background(), `select count(*) from outbox where topic = $1`, rating.SignalTopic).Scan(&published); err != nil || published != 8 {
		t.Fatalf("outbox %d %v", published, err)
	}
	f.play(t, "a0000000-0000-0000-0000-000000000007", honest, feeder, now.Add(7*time.Hour), false)
	if signals, _ := f.store.Signals(context.Background(), honest); len(signals) != 0 {
		t.Fatalf("honest flagged %s", kinds(signals))
	}
	trader, partner := f.account(t), f.account(t)
	f.set(t, trader, rules.Mode1v1, 1300, 30, rank(t, "Silver I", 40))
	f.set(t, partner, rules.Mode1v1, 1300, 30, rank(t, "Silver I", 40))
	for i := 0; i < 8; i++ {
		winner, loser := trader, partner
		if i%2 == 1 {
			winner, loser = partner, trader
		}
		f.play(t, fmt.Sprintf("b0000000-0000-0000-0000-0000000000%02d", i), winner, loser, now.Add(time.Duration(i)*time.Minute), false)
	}
	if signals, _ := f.store.Signals(context.Background(), trader); kinds(signals) != "pair_frequency:4/8 win_trade:4/8" {
		t.Fatalf("trader signals %s", kinds(signals))
	}
}

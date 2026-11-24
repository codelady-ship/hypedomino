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

func TestPartnersDrawnTogetherTooOftenAreFlagged(t *testing.T) {
	f := newFixture(t)
	sniper, friend, stranger := f.account(t), f.account(t), f.account(t)
	opponents := make([]string, 8)
	for i := range opponents {
		opponents[i] = f.account(t)
	}
	for _, account := range append([]string{sniper, friend, stranger}, opponents...) {
		f.set(t, account, rules.Mode2v2, 1300, 30, rank(t, "Silver I", 40))
	}
	play := func(id string, team [2]string, rivals [2]string, won bool, at time.Time) {
		t.Helper()
		teams := [2][]rating.Participant{{{Account: team[0]}, {Account: team[1]}}, {{Account: rivals[0]}, {Account: rivals[1]}}}
		winner := 1
		if won {
			winner = 0
		}
		if _, err := f.store.Apply(context.Background(), rating.Result{MatchID: id, Mode: rules.Mode2v2, Teams: teams, Winner: winner, At: at}); err != nil {
			t.Fatal(err)
		}
	}
	play("c0000000-0000-0000-0000-000000000000", [2]string{sniper, stranger}, [2]string{opponents[0], opponents[1]}, true, now)
	play("c0000000-0000-0000-0000-000000000001", [2]string{sniper, friend}, [2]string{opponents[2], opponents[3]}, true, now.Add(time.Hour))
	play("c0000000-0000-0000-0000-000000000002", [2]string{sniper, friend}, [2]string{opponents[4], opponents[5]}, false, now.Add(2*time.Hour))
	if signals, _ := f.store.Signals(context.Background(), sniper); len(signals) != 0 {
		t.Fatalf("flagged below the minimum: %s", kinds(signals))
	}
	play("c0000000-0000-0000-0000-000000000003", [2]string{opponents[6], opponents[7]}, [2]string{friend, sniper}, false, now.Add(3*time.Hour))
	for _, account := range []string{sniper, friend} {
		signals, err := f.store.Signals(context.Background(), account)
		if err != nil || kinds(signals) != "partner_frequency:2/3" || signals[0].Mode != "2v2" || signals[0].WindowHours != 168 {
			t.Fatalf("%s signals %s %+v %v", account, kinds(signals), signals, err)
		}
	}
	if signals, _ := f.store.Signals(context.Background(), sniper); signals[0].Counterpart != friend {
		t.Fatalf("counterpart %s", signals[0].Counterpart)
	}
	for _, account := range append([]string{stranger}, opponents...) {
		if signals, _ := f.store.Signals(context.Background(), account); len(signals) != 0 {
			t.Fatalf("%s flagged %s", account, kinds(signals))
		}
	}
	play("c0000000-0000-0000-0000-000000000004", [2]string{sniper, friend}, [2]string{opponents[0], opponents[2]}, true, now.Add(200*time.Hour))
	if signals, _ := f.store.Signals(context.Background(), friend); kinds(signals) != "partner_frequency:2/3" {
		t.Fatalf("matches outside the window counted: %s", kinds(signals))
	}
}

type measuredSeat struct {
	account         string
	team            int
	decisions, fits int
}

func (f *fixture) measured(t *testing.T, id string, at time.Time, seats [4]measuredSeat) {
	t.Helper()
	ctx := context.Background()
	if _, err := f.pool.Exec(ctx, `insert into matches (id, started_at, mode, kind, config_version, seed) values ($1, $2, '2v2', 'ranked', 1, 1)`, id, at); err != nil {
		t.Fatal(err)
	}
	for seat, s := range seats {
		if _, err := f.pool.Exec(ctx, `insert into match_participants (match_id, started_at, seat, team, account_id, partner_decisions, partner_fits) values ($1, $2, $3, $4, $5, $6, $7)`,
			id, at, seat, s.team, s.account, s.decisions, s.fits); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPartnerWhosePlacementsSuitTheirHandIsFlagged(t *testing.T) {
	f := newFixture(t)
	sniper, friend, stranger := f.account(t), f.account(t), f.account(t)
	opponents := make([]string, 8)
	for i := range opponents {
		opponents[i] = f.account(t)
	}
	for _, account := range append([]string{sniper, friend, stranger}, opponents...) {
		f.set(t, account, rules.Mode2v2, 1300, 30, rank(t, "Silver I", 40))
	}
	play := func(id string, partner string, rivals [2]string, sniperFit, partnerFit [2]int, won bool, at time.Time) {
		t.Helper()
		f.measured(t, id, at, [4]measuredSeat{
			{account: sniper, team: 0, decisions: sniperFit[0], fits: sniperFit[1]},
			{account: rivals[0], team: 1},
			{account: partner, team: 0, decisions: partnerFit[0], fits: partnerFit[1]},
			{account: rivals[1], team: 1},
		})
		teams := [2][]rating.Participant{{{Account: sniper}, {Account: partner}}, {{Account: rivals[0]}, {Account: rivals[1]}}}
		winner := 1
		if won {
			winner = 0
		}
		if _, err := f.store.Apply(context.Background(), rating.Result{MatchID: id, Mode: rules.Mode2v2, Teams: teams, Winner: winner, At: at}); err != nil {
			t.Fatal(err)
		}
	}
	play("d0000000-0000-0000-0000-000000000000", stranger, [2]string{opponents[0], opponents[1]}, [2]int{100, 50}, [2]int{100, 50}, true, now)
	play("d0000000-0000-0000-0000-000000000001", friend, [2]string{opponents[2], opponents[3]}, [2]int{30, 22}, [2]int{40, 20}, true, now.Add(time.Hour))
	play("d0000000-0000-0000-0000-000000000002", friend, [2]string{opponents[4], opponents[5]}, [2]int{30, 22}, [2]int{40, 20}, false, now.Add(2*time.Hour))
	play("d0000000-0000-0000-0000-000000000003", friend, [2]string{opponents[6], opponents[7]}, [2]int{39, 29}, [2]int{40, 20}, true, now.Add(3*time.Hour))
	if signals, _ := f.store.Signals(context.Background(), sniper); kinds(signals) != "partner_frequency:2/3" {
		t.Fatalf("flagged with 99 decisions: %s", kinds(signals))
	}
	play("d0000000-0000-0000-0000-000000000004", friend, [2]string{opponents[0], opponents[2]}, [2]int{1, 1}, [2]int{1, 0}, true, now.Add(4*time.Hour))
	signals, err := f.store.Signals(context.Background(), sniper)
	if err != nil || kinds(signals) != "partner_fit:3/4 partner_frequency:3/4" || signals[0].Counterpart != friend {
		t.Fatalf("sniper signals %s %+v %v", kinds(signals), signals, err)
	}
	if signals, _ := f.store.Signals(context.Background(), friend); kinds(signals) != "partner_frequency:3/4" {
		t.Fatalf("friend without other partners flagged: %s", kinds(signals))
	}
}

func TestPartnerFitNeedsTheMarginOverOtherPartners(t *testing.T) {
	f := newFixture(t)
	player, partner, other := f.account(t), f.account(t), f.account(t)
	opponents := make([]string, 6)
	for i := range opponents {
		opponents[i] = f.account(t)
	}
	for _, account := range append([]string{player, partner, other}, opponents...) {
		f.set(t, account, rules.Mode2v2, 1300, 30, rank(t, "Silver I", 40))
	}
	apply := func(id string, mate string, rivals [2]string, fit [2]int, at time.Time) {
		t.Helper()
		f.measured(t, id, at, [4]measuredSeat{{account: player, team: 0, decisions: fit[0], fits: fit[1]}, {account: rivals[0], team: 1}, {account: mate, team: 0}, {account: rivals[1], team: 1}})
		teams := [2][]rating.Participant{{{Account: player}, {Account: mate}}, {{Account: rivals[0]}, {Account: rivals[1]}}}
		if _, err := f.store.Apply(context.Background(), rating.Result{MatchID: id, Mode: rules.Mode2v2, Teams: teams, Winner: 0, At: at}); err != nil {
			t.Fatal(err)
		}
	}
	apply("e0000000-0000-0000-0000-000000000000", other, [2]string{opponents[0], opponents[1]}, [2]int{100, 50}, now)
	apply("e0000000-0000-0000-0000-000000000001", partner, [2]string{opponents[2], opponents[3]}, [2]int{50, 34}, now.Add(time.Hour))
	apply("e0000000-0000-0000-0000-000000000002", partner, [2]string{opponents[4], opponents[5]}, [2]int{50, 34}, now.Add(2*time.Hour))
	apply("e0000000-0000-0000-0000-000000000003", partner, [2]string{opponents[0], opponents[2]}, [2]int{0, 0}, now.Add(3*time.Hour))
	if signals, _ := f.store.Signals(context.Background(), player); kinds(signals) != "partner_frequency:3/3" {
		t.Fatalf("68 against 50 is under the margin: %s", kinds(signals))
	}
}

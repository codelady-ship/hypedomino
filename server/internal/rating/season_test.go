package rating_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/rating"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func TestSeasonsAreContiguousAndOnlyExtendLater(t *testing.T) {
	f := newFixture(t)
	end := now.Add(60 * 24 * time.Hour)
	for name, season := range map[string]rating.Season{
		"bad id":   {ID: "s2", Start: end, End: end.Add(time.Hour)},
		"reversed": {ID: "season.s2", Start: end, End: end},
		"gap":      {ID: "season.s2", Start: end.Add(time.Hour), End: end.Add(48 * time.Hour)},
		"overlap":  {ID: "season.s2", Start: now, End: end.Add(48 * time.Hour)},
	} {
		if _, err := f.store.CreateSeason(context.Background(), season, now); !errors.Is(err, rating.ErrSeason) && !errors.Is(err, rating.ErrSeasonGap) && !errors.Is(err, rating.ErrSeasonOverlap) {
			t.Errorf("%s: %v", name, err)
		}
	}
	second, err := f.store.CreateSeason(context.Background(), rating.Season{ID: "season.s2", Start: end, End: end.Add(90 * 24 * time.Hour)}, now)
	if err != nil || second.State(now) != rating.StateScheduled {
		t.Fatalf("second %+v %v", second, err)
	}
	if _, err := f.store.ExtendSeason(context.Background(), "season.test", end.Add(-time.Hour)); !errors.Is(err, rating.ErrSeasonEarlier) {
		t.Fatalf("earlier end: %v", err)
	}
	extended, err := f.store.ExtendSeason(context.Background(), "season.test", end.Add(24*time.Hour))
	if err != nil || !extended.End.Equal(end.Add(24*time.Hour)) {
		t.Fatalf("extended %+v %v", extended, err)
	}
	seasons, err := f.store.Seasons(context.Background())
	if err != nil || len(seasons) != 2 || !seasons[1].Start.Equal(end.Add(24*time.Hour)) || !seasons[1].End.Equal(end.Add(91*24*time.Hour)) {
		t.Fatalf("seasons %+v %v", seasons, err)
	}
	current, err := f.store.SeasonAt(context.Background(), now)
	if err != nil || current.ID != "season.test" || current.State(now) != rating.StateLive {
		t.Fatalf("current %+v %v", current, err)
	}
	if _, err := f.store.SeasonAt(context.Background(), now.Add(-40*24*time.Hour)); !errors.Is(err, rating.ErrNoSeason) {
		t.Fatalf("before the first season: %v", err)
	}
	if _, err := f.store.ExtendSeason(context.Background(), "season.missing", end); !errors.Is(err, rating.ErrSeasonUnknown) {
		t.Fatalf("unknown: %v", err)
	}
}

func TestSeasonEndSnapshotsResetsAndRunsOnce(t *testing.T) {
	f := newFixture(t)
	end := now.Add(60 * 24 * time.Hour)
	challenger := f.apex(t, 600, 40, now)
	master := f.apex(t, 100, 20, now)
	gold := f.account(t)
	f.set(t, gold, rules.Mode1v1, 1400, 50, rank(t, "Gold II", 50))
	bronze := f.account(t)
	f.set(t, bronze, rules.Mode1v1, 1000, 50, rank(t, "Bronze III", 10))
	iron := f.account(t)
	f.set(t, iron, rules.Mode1v1, 900, 50, rank(t, "Iron II", 0))
	other := f.account(t)
	f.set(t, other, rules.Mode2v2, 1400, 50, rank(t, "Gold II", 50))
	unplaced := f.account(t)
	if _, err := f.store.Apply(context.Background(), rating.Result{MatchID: "66666666-6666-6666-6666-666666666666", Mode: rules.Mode1v1, Teams: [2][]rating.Participant{{{Account: unplaced}}, {{Account: iron}}}, Winner: 0, At: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.CloseSeason(context.Background(), "season.test", rules.Mode1v1, now); !errors.Is(err, rating.ErrSeasonOpen) {
		t.Fatalf("closing a live season: %v", err)
	}
	closed, err := f.store.CloseSeason(context.Background(), "season.test", rules.Mode1v1, end)
	if err != nil || closed.Skipped || closed.Placed != 5 || closed.Next != end.Format("season.2006_01") || closed.Apex.Challenger != 1 || closed.Apex.Master != 1 {
		t.Fatalf("closed %+v %v", closed, err)
	}
	rows, err := f.pool.Query(context.Background(), `select account_id, position, rank_index, coalesce(apex_tier, ''), reset_index from season_results where season_id = 'season.test' and mode = '1v1' order by position`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	type row struct {
		account         string
		position, index int
		apex            string
		reset           int
	}
	var results []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.account, &r.position, &r.index, &r.apex, &r.reset); err != nil {
			t.Fatal(err)
		}
		results = append(results, r)
	}
	want := []row{{challenger, 1, 28, rating.ApexChallenger, 24}, {master, 2, 28, rating.ApexMaster, 24}, {gold, 3, 14, "", 10}, {bronze, 4, 5, "", 4}, {iron, 5, 1, "", 1}}
	if len(results) != len(want) {
		t.Fatalf("results %+v", results)
	}
	for i := range want {
		if results[i] != want[i] {
			t.Errorf("position %d: %+v, want %+v", i+1, results[i], want[i])
		}
	}
	for account, reset := range map[string]string{challenger: "Diamond IV", gold: "Silver II", bronze: "Bronze IV", iron: "Iron III"} {
		r := f.rating(t, account, rules.Mode1v1)
		if r.Placed || r.PlacementPlayed != 0 || r.Rank.LP != 0 || r.Wins != 0 || r.Tier == rating.TierChallenger || r.Season != closed.Next || r.Reset == nil || r.Reset.String() != reset || r.Rank.String() != reset {
			t.Errorf("%s after reset: %+v", account, r)
		}
	}
	if r := f.rating(t, unplaced, rules.Mode1v1); r.Placed || r.PlacementPlayed != 0 || r.Reset != nil || r.Season != closed.Next || r.MMR == 1200 {
		t.Errorf("unplaced after reset: %+v", r)
	}
	if r := f.rating(t, other, rules.Mode2v2); !r.Placed || r.Rank.String() != "Gold II" || r.Season != "" {
		t.Errorf("2v2 touched: %+v", r)
	}
	again, err := f.store.CloseSeason(context.Background(), "season.test", rules.Mode1v1, end.Add(time.Hour))
	if err != nil || !again.Skipped {
		t.Fatalf("second close %+v %v", again, err)
	}
	late, err := f.store.Apply(context.Background(), rating.Result{MatchID: "77777777-7777-7777-7777-777777777777", Mode: rules.Mode1v1, Teams: [2][]rating.Participant{{{Account: gold}}, {{Account: bronze}}}, Winner: 0, At: now.Add(time.Hour)})
	if err == nil || !errors.Is(err, rating.ErrSeasonTransition) {
		t.Fatalf("a match of the closed season against reset rows: %+v %v", late, err)
	}
	fresh, err := f.store.Apply(context.Background(), rating.Result{MatchID: "88888888-8888-8888-8888-888888888888", Mode: rules.Mode1v1, Teams: [2][]rating.Participant{{{Account: gold}}, {{Account: bronze}}}, Winner: 0, At: end.Add(time.Hour)})
	if err != nil || fresh.Late || fresh.Season != closed.Next || !fresh.Changes[0][0].Placement {
		t.Fatalf("first match of the next season %+v %v", fresh, err)
	}
	due, err := f.store.CloseDue(context.Background(), end.Add(2*time.Hour))
	if err != nil || len(due) != 1 || due[0].Placed != 1 {
		t.Fatalf("close due %+v %v", due, err)
	}
	seasons, err := f.store.Seasons(context.Background())
	if err != nil || len(seasons) != 2 || seasons[0].State(end) != rating.StateClosed || !seasons[1].End.Equal(end.AddDate(0, 3, 0)) {
		t.Fatalf("seasons %+v %v", seasons, err)
	}
}

func TestALateResultOfAClosedSeasonChangesMmrOnly(t *testing.T) {
	f := newFixture(t)
	end := now.Add(60 * 24 * time.Hour)
	a, b := f.account(t), f.account(t)
	f.set(t, a, rules.Mode2v2, 1300, 30, rank(t, "Silver I", 40))
	f.set(t, b, rules.Mode2v2, 1250, 30, rank(t, "Silver I", 0))
	c, d := f.account(t), f.account(t)
	f.set(t, c, rules.Mode2v2, 1300, 30, rank(t, "Silver I", 40))
	f.set(t, d, rules.Mode2v2, 1250, 30, rank(t, "Silver I", 0))
	if _, err := f.store.CloseSeason(context.Background(), "season.test", rules.Mode2v2, end); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(context.Background(), `update ratings set season_id = 'season.test', placed = true, rank_index = 11, lp = 40 where mode = '2v2'`); err != nil {
		t.Fatal(err)
	}
	applied, err := f.store.Apply(context.Background(), rating.Result{MatchID: "99999999-9999-9999-9999-999999999999", Mode: rules.Mode2v2, Teams: [2][]rating.Participant{{{Account: a}, {Account: b}}, {{Account: c}, {Account: d}}}, Winner: 0, At: now})
	if err != nil || !applied.Late || applied.Season != "season.test" {
		t.Fatalf("late %+v %v", applied, err)
	}
	r := f.rating(t, a, rules.Mode2v2)
	if r.MMR != 1310 || r.Rank.String() != "Silver I" || r.Rank.LP != 40 || r.Wins != 0 || applied.Changes[0][0].LPDelta != 0 {
		t.Errorf("late winner %+v %+v", r, applied.Changes[0][0])
	}
}

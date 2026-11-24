package rating_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/risk"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func (f *fixture) collusionSignals(t *testing.T, account string) []string {
	t.Helper()
	rows, err := f.pool.Query(context.Background(), `select format('%s/%s/%s', weight, evidence->>'signal', evidence->>'matches') from risk_signals where account_id = $1 and kind = $2 order by created_at`, account, risk.KindCollusion)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var signal string
		if err := rows.Scan(&signal); err != nil {
			t.Fatal(err)
		}
		out = append(out, signal)
	}
	return out
}

func TestFeedingReachesTheRiskScoreOncePerPairAndDay(t *testing.T) {
	f := newFixture(t)
	weight := risk.DefaultConfig().CollusionWeight
	f.store.SetCollusionWeight(weight)
	booster, feeder := f.account(t), f.account(t)
	for _, account := range []string{booster, feeder} {
		f.set(t, account, rules.Mode1v1, 1300, 30, rank(t, "Silver I", 40))
	}
	day := now.UTC().Truncate(24 * time.Hour)
	for i := 0; i < 7; i++ {
		f.play(t, fmt.Sprintf("c0000000-0000-0000-0000-0000000000%02d", i), booster, feeder, day.Add(time.Duration(i)*time.Minute), false)
	}
	want := fmt.Sprintf("%d/feeding/6", weight)
	if got := f.collusionSignals(t, feeder); len(got) != 1 || got[0] != want {
		t.Fatalf("feeder %v, want one %s and no pair_frequency", got, want)
	}
	if got := f.collusionSignals(t, booster); len(got) != 1 || got[0] != want {
		t.Fatalf("booster %v", got)
	}
	f.play(t, "c0000000-0000-0000-0000-000000000007", booster, feeder, day.Add(24*time.Hour), false)
	if got := f.collusionSignals(t, feeder); len(got) != 2 || !strings.HasPrefix(got[1], fmt.Sprintf("%d/feeding/", weight)) {
		t.Fatalf("next day %v", got)
	}
}

func TestCollusionStaysOutOfTheRiskScoreWithoutAWeight(t *testing.T) {
	f := newFixture(t)
	booster, feeder := f.account(t), f.account(t)
	for _, account := range []string{booster, feeder} {
		f.set(t, account, rules.Mode1v1, 1300, 30, rank(t, "Silver I", 40))
	}
	for i := 0; i < 6; i++ {
		f.play(t, fmt.Sprintf("d0000000-0000-0000-0000-0000000000%02d", i), booster, feeder, now.Add(time.Duration(i)*time.Minute), false)
	}
	if got := f.collusionSignals(t, feeder); len(got) != 0 {
		t.Fatalf("recorded %v", got)
	}
}

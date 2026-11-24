package rating_test

import (
	"context"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/rating"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func TestTheRosterLabelIsThePlacedRankOrTheApexTier(t *testing.T) {
	for _, c := range []struct {
		rating rating.Rating
		want   string
	}{
		{rating.Rating{Placed: false, Rank: rank(t, "Gold II", 10)}, ""},
		{rating.Rating{Placed: true, Rank: rank(t, "Gold II", 10)}, "Gold II"},
		{rating.Rating{Placed: true, Rank: rating.Rank{Index: rating.ApexIndex}, Tier: rating.TierGrandmaster}, rating.TierGrandmaster},
	} {
		if got := c.rating.Label(); got != c.want {
			t.Fatalf("%+v: %q, want %q", c.rating, got, c.want)
		}
	}
	f := newFixture(t)
	placed, fresh := f.account(t), f.account(t)
	f.set(t, placed, rules.Mode2v2, 1300, 30, rank(t, "Silver I", 40))
	for account, want := range map[string]string{placed: "Silver I", fresh: ""} {
		if got, err := f.store.Label(context.Background(), account, rules.Mode2v2); err != nil || got != want {
			t.Fatalf("label %q, %v, want %q", got, err, want)
		}
	}
}

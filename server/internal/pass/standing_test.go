package pass_test

import (
	"context"
	"testing"
	"time"
)

func TestStandingIsTheActiveSeasonLevelAndXp(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	account := f.account(t)
	if level, xp, err := f.store.Standing(ctx, account, now); err != nil || level != 1 || xp != 0 {
		t.Fatalf("before entering = %d, %d, %v", level, xp, err)
	}
	season, _, _, err := f.store.Enter(ctx, account, now)
	if err != nil {
		t.Fatal(err)
	}
	total := season.Curve.XpToReach(5) + 10
	if _, err := f.pool.Exec(ctx, `update pass_progress set xp_total = $1 where account_id = $2`, total, account); err != nil {
		t.Fatal(err)
	}
	if level, xp, err := f.store.Standing(ctx, account, now); err != nil || level != 5 || xp != total {
		t.Fatalf("in the season = %d, %d, %v", level, xp, err)
	}
	if level, xp, err := f.store.Standing(ctx, account, now.Add(81*24*time.Hour)); err != nil || level != 1 || xp != 0 {
		t.Fatalf("after the season = %d, %d, %v", level, xp, err)
	}
}

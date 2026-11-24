package pass_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/analytics"
	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/matchstore"
	"github.com/Alievs-corp/hypedomino/server/internal/pass"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

func win(account, id string, at time.Time) pass.Match {
	return pass.Match{ID: id, Account: account, Mode: rules.Mode1v1, Kind: matchstore.KindRanked, Won: true, EndedAt: at}
}

func (f *fixture) xp(t *testing.T, account string) int64 {
	t.Helper()
	return int64(f.count(t, `select xp_total from pass_progress where account_id = $1 and season_id = 'season_1'`, account))
}

func TestMatchXpCountsOnceAndStopsAtTheDailyCap(t *testing.T) {
	f := newFixture(t)
	account := f.account(t)
	for range 2 {
		if got, err := f.store.AwardMatch(context.Background(), win(account, "match-1", now)); err != nil || f.xp(t, account) != 100 {
			t.Fatalf("AwardMatch = %d, %v; xp %d", got, err, f.xp(t, account))
		}
	}
	for i := 2; i <= 20; i++ {
		if _, err := f.store.AwardMatch(context.Background(), win(account, fmt.Sprintf("match-%d", i), now)); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := f.store.AwardMatch(context.Background(), win(account, "match-21", now)); err != nil || got != 0 {
		t.Fatalf("match over the cap = %d, %v", got, err)
	}
	if got := f.xp(t, account); got != 2000 {
		t.Fatalf("xp = %d, want 2000", got)
	}
	if got, err := f.store.AwardMatch(context.Background(), win(account, "match-22", now.Add(24*time.Hour))); err != nil || got != 100 {
		t.Fatalf("next economy day = %d, %v", got, err)
	}
	if got := f.balance(t, account); got != 11*20 {
		t.Fatalf("balance at level 22 = %d, want %d", got, 11*20)
	}
	if got := f.count(t, `select count(*) from outbox where topic = $1 and key = $2`, pass.XpTopic, account+":match:match-1"); got != 1 {
		t.Fatalf("outbox events for match-1 = %d", got)
	}
	if got := f.count(t, `select count(*) from analytics_events where name = $1 and received_at = $2 and analytics_id is null and client_time_ms is null and label = '' and value = 0`, analytics.PassLevel, now); got != 20 {
		t.Fatalf("anonymous pass_level events on the first day = %d, want 20", got)
	}
	if got := f.count(t, `select count(*) from analytics_events where name = $1`, analytics.PassLevel); got != 21 {
		t.Fatalf("pass_level events = %d, want one per level from 1 to 22", got)
	}
}

func TestMatchXpOutsideASeasonIsDropped(t *testing.T) {
	f := newFixture(t)
	account := f.account(t)
	if got, err := f.store.AwardMatch(context.Background(), win(account, "match-late", now.Add(81*24*time.Hour))); err != nil || got != 0 {
		t.Fatalf("AwardMatch in a gap = %d, %v", got, err)
	}
	if got := f.count(t, `select count(*) from pass_xp_events where account_id = $1`, account); got != 0 {
		t.Fatalf("xp events = %d", got)
	}
}

func TestBundleXpGoesThroughTheRewardEngineOnce(t *testing.T) {
	f := newFixture(t)
	account := f.account(t)
	grant := reward.Grant{
		Account: account,
		Bundle:  reward.Bundle{ID: "bundle.test_xp", Revision: 1, PassXP: 450},
		Source:  cosmetics.Source{Type: "promo", Ref: "campaign.test"},
		Reason:  wallet.Promo,
		Key:     "promo:test:" + account,
	}
	for range 2 {
		report, err := f.engine.Apply(context.Background(), grant, now)
		if err != nil || report.PassXP != 450 {
			t.Fatalf("Apply = %+v, %v", report, err)
		}
	}
	if got := f.xp(t, account); got != 450 {
		t.Fatalf("xp = %d, want 450", got)
	}
	if got := f.balance(t, account); got != 3*20 {
		t.Fatalf("balance at level 5 = %d, want 60", got)
	}
}

func TestALatePremiumUnlockGrantsEveryPremiumLevelReached(t *testing.T) {
	f := newFixture(t)
	account := f.account(t)
	for i := range 5 {
		if _, err := f.store.AwardMatch(context.Background(), win(account, fmt.Sprintf("match-%d", i), now)); err != nil {
			t.Fatal(err)
		}
	}
	unlock := func(ref string) (int, error) {
		tx, err := f.pool.Begin(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(context.Background())
		if err := reward.LockIn(context.Background(), tx, account); err != nil {
			t.Fatal(err)
		}
		granted, err := f.store.UnlockPremiumIn(context.Background(), tx, account, "season_1", ref, now)
		if err != nil {
			return granted, err
		}
		return granted, tx.Commit(context.Background())
	}
	if granted, err := unlock("purchase:google:1"); err != nil || granted != 6 {
		t.Fatalf("unlock = %d, %v", granted, err)
	}
	if granted, err := unlock("purchase:google:1"); err != nil || granted != 0 {
		t.Fatalf("repeated unlock = %d, %v", granted, err)
	}
	if _, err := unlock("purchase:google:2"); !errors.Is(err, pass.ErrPremiumOwned) {
		t.Fatalf("second purchase = %v", err)
	}
	if got, crystals := f.balance(t, account), f.crystals(t, account); got != 3*20 || crystals != 6*2 {
		t.Fatalf("coins %d crystals %d, want %d and %d", got, crystals, 3*20, 6*2)
	}
	if got, err := f.store.AwardMatch(context.Background(), win(account, "match-premium", now)); err != nil || got != 100 {
		t.Fatalf("AwardMatch after premium = %d, %v", got, err)
	}
	if got, crystals := f.balance(t, account), f.crystals(t, account); got != 4*20 || crystals != 7*2 {
		t.Fatalf("at level 7 coins %d crystals %d, want %d and %d", got, crystals, 4*20, 7*2)
	}
}

func TestPremiumPlusAddsItsLevelsAndPaysTheLevelsPastTheLastOneInCrystals(t *testing.T) {
	f := newFixture(t)
	account := f.account(t)
	if _, _, _, err := f.store.Enter(context.Background(), account, now); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(context.Background(), `update pass_progress set xp_total = 2650 where account_id = $1`, account); err != nil {
		t.Fatal(err)
	}
	tx, err := f.pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	if _, err := f.store.UnlockPremiumIn(context.Background(), tx, account, "season_1", "purchase:google:plus", now); err != nil {
		t.Fatal(err)
	}
	crystals, err := f.store.AddPremiumPlusIn(context.Background(), tx, account, "season_1", "purchase:google:plus", now)
	if err != nil || crystals != 2*40 {
		t.Fatalf("AddPremiumPlusIn = %d, %v", crystals, err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := f.xp(t, account); got != 2950 {
		t.Fatalf("xp = %d, want 2950", got)
	}
}

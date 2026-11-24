package reward_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/analytics"
	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/inventory"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

var start = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

type fixture struct {
	pool      *pgxpool.Pool
	wallet    *wallet.Store
	inventory *inventory.Store
	medals    *fakeMedals
	pass      *fakePass
}

type fakeMedals struct {
	held    map[string]bool
	revoked []string
}

func (f *fakeMedals) GrantIn(_ context.Context, _ pgx.Tx, account, medal string, _ cosmetics.Source, _ time.Time) (bool, error) {
	key := account + ":" + medal
	if f.held[key] {
		return false, nil
	}
	f.held[key] = true
	return true, nil
}

func (f *fakeMedals) RevokeIn(_ context.Context, _ pgx.Tx, account, medal string, _ time.Time) error {
	delete(f.held, account+":"+medal)
	f.revoked = append(f.revoked, medal)
	return nil
}

type fakePass struct {
	active bool
	added  int
}

func (f *fakePass) AddIn(_ context.Context, _ pgx.Tx, _ string, xp int, _ cosmetics.Source, _ string, _ time.Time) (bool, error) {
	if !f.active {
		return false, nil
	}
	f.added += xp
	return true, nil
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	_, pool := dbtest.Fresh(t)
	catalog, err := cosmetics.Load("../../../docs/cosmetics/examples/catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	return &fixture{
		pool:      pool,
		wallet:    wallet.NewStore(pool),
		inventory: inventory.NewStore(pool, catalog, inventory.DefaultConfig()),
		medals:    &fakeMedals{held: map[string]bool{}},
		pass:      &fakePass{active: true},
	}
}

func (f *fixture) engine(medals reward.Medals, pass reward.Pass) *reward.Engine {
	return reward.NewEngine(f.pool, f.wallet, f.inventory, medals, pass, reward.DefaultConfig())
}

func (f *fixture) account(t *testing.T) string {
	t.Helper()
	var id string
	if err := f.pool.QueryRow(context.Background(), `insert into accounts default values returning id`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

var (
	spring = cosmetics.Source{Type: "event", Ref: "event.spring_cup"}
	fox    = reward.Bundle{ID: "bundle.spring_fox", Revision: 2, Coins: 300, Cosmetics: []string{"avatar.comet_fox", "emoji.victory_dance"}, Medals: []string{"medal.spring"}, PassXP: 40}
)

func TestAGrantWritesCoinsItemsMedalsAndXpOnceAndReportsIt(t *testing.T) {
	f := newFixture(t)
	engine := f.engine(f.medals, f.pass)
	ctx := context.Background()
	ayla := f.account(t)
	grant := reward.Grant{Account: ayla, Bundle: fox, Source: spring, Reason: wallet.EventReward, Key: "event:spring_cup:r1", ConfigVersion: 5}
	report, err := engine.Apply(ctx, grant, start)
	if err != nil || report.Repeated || report.ID == 0 || report.Coins != 300 || len(report.Cosmetics) != 2 || len(report.Duplicates) != 0 || len(report.Medals) != 1 || len(report.MedalsSkipped) != 0 || report.PassXP != 40 || report.PassXPDropped != 0 || !report.At.Equal(start) {
		t.Fatalf("report %+v %v", report, err)
	}
	if balance, err := f.wallet.Balance(ctx, ayla, wallet.Coin); err != nil || balance != 300 {
		t.Fatalf("balance %d %v", balance, err)
	}
	if owns, err := f.inventory.Owns(ctx, ayla, "emoji.victory_dance"); err != nil || !owns {
		t.Fatalf("owns %v %v", owns, err)
	}
	if f.pass.added != 40 || !f.medals.held[ayla+":medal.spring"] {
		t.Fatalf("pass %d medals %v", f.pass.added, f.medals.held)
	}
	again, err := engine.Apply(ctx, grant, start.Add(time.Hour))
	if err != nil || !again.Repeated || again.ID != report.ID || again.Coins != 300 {
		t.Fatalf("repeat %+v %v", again, err)
	}
	if balance, err := f.wallet.Balance(ctx, ayla, wallet.Coin); err != nil || balance != 300 {
		t.Fatalf("balance after the repeat %d %v", balance, err)
	}
	if f.pass.added != 40 {
		t.Fatalf("pass xp added twice: %d", f.pass.added)
	}
	loaded, err := engine.Get(ctx, ayla, grant.Key)
	if err != nil || loaded.ID != report.ID || loaded.BundleID != "bundle.spring_fox" || loaded.Revision != 2 || loaded.ConfigVersion != 5 || loaded.Revoked() {
		t.Fatalf("loaded %+v %v", loaded, err)
	}
	var topic, key string
	if err := f.pool.QueryRow(ctx, `select topic, key from outbox`).Scan(&topic, &key); err != nil || topic != reward.Topic || key != ayla+":"+grant.Key {
		t.Fatalf("outbox %s %s %v", topic, key, err)
	}
	if _, err := engine.Get(ctx, ayla, "nope"); !errors.Is(err, reward.ErrUnknownGrant) {
		t.Fatalf("unknown grant: %v", err)
	}
	if events, anonymous := rewardGrantEvents(t, f.pool, start, spring.Type); events != 1 || anonymous != 1 {
		t.Fatalf("%d reward_grant events, %d anonymous at the grant time labeled %q, want one for a repeated grant", events, anonymous, spring.Type)
	}
}

func rewardGrantEvents(t *testing.T, pool *pgxpool.Pool, at time.Time, label string) (int, int) {
	t.Helper()
	var events, anonymous int
	if err := pool.QueryRow(context.Background(), `select count(*), count(*) filter (where analytics_id is null and client_time_ms is null and label = $3 and value = 0 and received_at = $2)
		from analytics_events where name = $1`, analytics.RewardGrant, at, label).Scan(&events, &anonymous); err != nil {
		t.Fatal(err)
	}
	return events, anonymous
}

func TestARewardGrantEventIsLabeledWithTheGrantSourceTypeOnly(t *testing.T) {
	f := newFixture(t)
	engine := f.engine(f.medals, f.pass)
	ctx := context.Background()
	ayla := f.account(t)
	coins := reward.Bundle{ID: "bundle.coins_100", Revision: 1, Coins: 100}
	grants := map[string]reward.Grant{
		"promo": {Account: ayla, Bundle: coins, Source: cosmetics.Source{Type: "promo", Ref: "campaign.winter"}, Reason: wallet.Promo, Key: "promo:winter"},
		"event": {Account: ayla, Bundle: coins, Source: spring, LedgerSource: wallet.Source{Type: "admin", Ref: "admin.ayla"}, Reason: wallet.AdminAdjust, Key: "admin-grant:admin.ayla:r1"},
	}
	for label, grant := range grants {
		if _, err := engine.Apply(ctx, grant, start); err != nil {
			t.Fatalf("%s: %v", label, err)
		}
	}
	rows, err := f.pool.Query(ctx, `select label, count(*) from analytics_events where name = $1 group by label`, analytics.RewardGrant)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	labels := map[string]int{}
	for rows.Next() {
		var label string
		var count int
		if err := rows.Scan(&label, &count); err != nil {
			t.Fatal(err)
		}
		labels[label] = count
	}
	if err := rows.Err(); err != nil || len(labels) != 2 || labels["promo"] != 1 || labels["event"] != 1 {
		t.Fatalf("reward_grant labels %v %v, want one promo and one event", labels, err)
	}
}

func TestDuplicatesPayCompensationAndMissingServicesDropTheirParts(t *testing.T) {
	f := newFixture(t)
	engine := f.engine(nil, nil)
	ctx := context.Background()
	ayla := f.account(t)
	if _, err := f.inventory.Grant(ctx, ayla, "avatar.comet_fox", cosmetics.Source{Type: "shop", Ref: "offer.avatar_comet_fox"}, start); err != nil {
		t.Fatal(err)
	}
	report, err := engine.Apply(ctx, reward.Grant{Account: ayla, Bundle: fox, Source: spring, Reason: wallet.EventReward, Key: "event:spring_cup:r2", ConfigVersion: 5}, start)
	if err != nil {
		t.Fatal(err)
	}
	item, _ := f.inventory.Catalog().Item("avatar.comet_fox")
	compensation := reward.DefaultConfig().Compensation(item)
	if compensation == 0 || report.Duplicates["avatar.comet_fox"] != compensation || len(report.Cosmetics) != 1 || report.Cosmetics[0] != "emoji.victory_dance" {
		t.Fatalf("duplicate handling %+v (compensation %d)", report, compensation)
	}
	if len(report.Medals) != 0 || len(report.MedalsSkipped) != 1 || report.PassXP != 0 || report.PassXPDropped != 40 {
		t.Fatalf("medals and xp without services %+v", report)
	}
	if balance, err := f.wallet.Balance(ctx, ayla, wallet.Coin); err != nil || balance != 300+compensation {
		t.Fatalf("balance %d %v", balance, err)
	}
	history, err := f.wallet.History(ctx, ayla, 10)
	if err != nil || len(history) != 2 || history[0].Reason != wallet.DuplicateCompensation || history[0].Key != "event:spring_cup:r2:dup:avatar.comet_fox" {
		t.Fatalf("history %+v %v", history, err)
	}
}

func TestARefusedGrantWritesNothing(t *testing.T) {
	f := newFixture(t)
	engine := f.engine(f.medals, f.pass)
	ctx := context.Background()
	ayla := f.account(t)
	unlisted := reward.Bundle{ID: "bundle.bad", Revision: 1, Coins: 500, Cosmetics: []string{"avatar.comet_fox"}}
	if _, err := engine.Apply(ctx, reward.Grant{Account: ayla, Bundle: unlisted, Source: cosmetics.Source{Type: "mission", Ref: "mission.x"}, Reason: wallet.Mission, Key: "mission:x:1"}, start); !errors.Is(err, inventory.ErrSourceNotListed) {
		t.Fatalf("unlisted source: %v", err)
	}
	if balance, err := f.wallet.Balance(ctx, ayla, wallet.Coin); err != nil || balance != 0 {
		t.Fatalf("coins credited by a refused grant: %d %v", balance, err)
	}
	if _, err := engine.Get(ctx, ayla, "mission:x:1"); !errors.Is(err, reward.ErrUnknownGrant) {
		t.Fatalf("refused grant recorded: %v", err)
	}
	if events, _ := rewardGrantEvents(t, f.pool, start, "mission"); events != 0 {
		t.Fatalf("%d reward_grant events from a refused grant", events)
	}
	for name, grant := range map[string]reward.Grant{
		"bad bundle":     {Account: ayla, Bundle: reward.Bundle{ID: "x"}, Source: spring, Reason: wallet.EventReward, Key: "k"},
		"no key":         {Account: ayla, Bundle: fox, Source: spring, Reason: wallet.EventReward},
		"default source": {Account: ayla, Bundle: fox, Source: cosmetics.Source{Type: "default"}, Reason: wallet.EventReward, Key: "k"},
		"debit reason":   {Account: ayla, Bundle: fox, Source: spring, Reason: wallet.Rename, Key: "k"},
		"unknown reason": {Account: ayla, Bundle: fox, Source: spring, Reason: "gift", Key: "k"},
	} {
		if _, err := engine.Apply(ctx, grant, start); !errors.Is(err, reward.ErrInvalidGrant) && !errors.Is(err, reward.ErrInvalidBundle) {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

func TestRevokeTakesBackCoinsItemsAndMedalsOnce(t *testing.T) {
	f := newFixture(t)
	engine := f.engine(f.medals, f.pass)
	ctx := context.Background()
	ayla := f.account(t)
	grant := reward.Grant{Account: ayla, Bundle: fox, Source: spring, Reason: wallet.EventReward, Key: "event:spring_cup:r3", ConfigVersion: 5}
	if _, err := engine.Apply(ctx, grant, start); err != nil {
		t.Fatal(err)
	}
	if _, err := f.inventory.Equip(ctx, ayla, "emoji", []string{"emoji.victory_dance"}, start); !errors.Is(err, inventory.ErrNoProfile) {
		t.Fatalf("equip without a profile: %v", err)
	}
	revoked, err := engine.Revoke(ctx, ayla, grant.Key, wallet.AdminAdjust, "refund:google:t9", start.Add(time.Hour))
	if err != nil || !revoked.Revoked() || revoked.RevokeKey != "refund:google:t9" || !revoked.RevokedAt.Equal(start.Add(time.Hour)) {
		t.Fatalf("revoke %+v %v", revoked, err)
	}
	if balance, err := f.wallet.Balance(ctx, ayla, wallet.Coin); err != nil || balance != 0 {
		t.Fatalf("balance after revoke %d %v", balance, err)
	}
	if owns, err := f.inventory.Owns(ctx, ayla, "avatar.comet_fox"); err != nil || owns {
		t.Fatalf("item kept after revoke %v %v", owns, err)
	}
	if len(f.medals.revoked) != 1 || f.medals.held[ayla+":medal.spring"] {
		t.Fatalf("medals after revoke %v %v", f.medals.revoked, f.medals.held)
	}
	if _, err := engine.Revoke(ctx, ayla, grant.Key, wallet.AdminAdjust, "refund:google:t9", start.Add(2*time.Hour)); !errors.Is(err, reward.ErrAlreadyRevoke) {
		t.Fatalf("second revoke: %v", err)
	}
	if loaded, err := engine.Get(ctx, ayla, grant.Key); err != nil || !loaded.Revoked() {
		t.Fatalf("loaded after revoke %+v %v", loaded, err)
	}
	if _, err := f.wallet.Apply(ctx, wallet.Entry{Account: ayla, Amount: -1, Reason: wallet.Rename, Source: wallet.Source{Type: "rename", Ref: "r"}, Key: "spend:r"}, start); !errors.Is(err, wallet.ErrInsufficient) {
		t.Fatalf("balance went negative: %v", err)
	}
	spent := f.account(t)
	bought := cosmetics.Source{Type: "shop", Ref: "offer.crystals_tier1"}
	if _, err := engine.Apply(ctx, reward.Grant{Account: spent, Bundle: reward.Bundle{ID: "bundle.crystals_tier1", Revision: 1, Crystals: 100}, Source: bought, Reason: wallet.Purchase, Key: "purchase:google:t4"}, start); err != nil {
		t.Fatal(err)
	}
	crystalSpend := func(amount int64, key string) error {
		_, err := f.wallet.Apply(ctx, wallet.Entry{Account: spent, Currency: wallet.Crystal, Amount: amount, Reason: wallet.ShopPurchase, Source: wallet.Source{Type: "shop", Ref: "offer.x"}, Key: key}, start)
		return err
	}
	if err := crystalSpend(-100, "spend:r"); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Revoke(ctx, spent, "purchase:google:t4", wallet.ShopPurchase, "shop:revoke", start); !errors.Is(err, wallet.ErrInsufficient) {
		t.Fatalf("revoke of spent crystals with a plain debit: %v", err)
	}
	if loaded, err := engine.Get(ctx, spent, "purchase:google:t4"); err != nil || loaded.Revoked() {
		t.Fatalf("a refused revoke was recorded %+v %v", loaded, err)
	}
	if _, err := engine.Revoke(ctx, spent, "purchase:google:t4", wallet.RefundRevoke, "refund:google:t4", start); err != nil {
		t.Fatalf("refund revoke of spent crystals: %v", err)
	}
	if balance, err := f.wallet.Balance(ctx, spent, wallet.Crystal); err != nil || balance != -100 {
		t.Fatalf("crystals after a refund revoke of spent crystals %d %v", balance, err)
	}
	if err := crystalSpend(-1, "spend:r2"); !errors.Is(err, wallet.ErrInsufficient) {
		t.Fatalf("spend from a negative balance: %v", err)
	}
	if _, err := engine.Apply(ctx, reward.Grant{Account: spent, Bundle: reward.Bundle{ID: "bundle.crystals_tier2", Revision: 1, Crystals: 150}, Source: bought, Reason: wallet.Purchase, Key: "purchase:google:t5"}, start); err != nil {
		t.Fatal(err)
	}
	if balance, _ := f.wallet.Balance(ctx, spent, wallet.Crystal); balance != 50 {
		t.Fatalf("crystals after buying back %d", balance)
	}
}

func TestAReinstatedGrantComesBackAndCanBeRevokedAgain(t *testing.T) {
	f := newFixture(t)
	engine := f.engine(f.medals, f.pass)
	ctx := context.Background()
	ayla := f.account(t)
	grant := reward.Grant{Account: ayla, Bundle: fox, Source: spring, Reason: wallet.EventReward, Key: "event:spring_cup:r5", ConfigVersion: 5}
	granted, err := engine.Apply(ctx, grant, start)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Revoke(ctx, ayla, grant.Key, wallet.AdminAdjust, "refund:google:t11", start.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	reinstated, err := engine.ReinstateIn(ctx, tx, ayla, grant.Key, wallet.AdminAdjust, "reversal:google:t11", start.Add(2*time.Hour))
	if err != nil || reinstated.Revoked() {
		t.Fatalf("reinstate %+v %v", reinstated, err)
	}
	if again, err := engine.ReinstateIn(ctx, tx, ayla, grant.Key, wallet.AdminAdjust, "reversal:google:t11:again", start.Add(2*time.Hour)); err != nil || again.Revoked() {
		t.Fatalf("second reinstate %+v %v", again, err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if balance, err := f.wallet.Balance(ctx, ayla, wallet.Coin); err != nil || balance != granted.Coins {
		t.Fatalf("balance after reinstate %d %v, want %d", balance, err, granted.Coins)
	}
	if owns, err := f.inventory.Owns(ctx, ayla, "avatar.comet_fox"); err != nil || !owns {
		t.Fatalf("item after reinstate %v %v", owns, err)
	}
	if !f.medals.held[ayla+":medal.spring"] {
		t.Fatalf("medal after reinstate %v", f.medals.held)
	}
	if _, err := engine.Revoke(ctx, ayla, grant.Key, wallet.AdminAdjust, "refund:google:t12", start.Add(3*time.Hour)); err != nil {
		t.Fatalf("revoke after reinstate: %v", err)
	}
	if balance, err := f.wallet.Balance(ctx, ayla, wallet.Coin); err != nil || balance != 0 {
		t.Fatalf("balance after the second revoke %d %v", balance, err)
	}
}

func TestCrystalsAreGrantedRevokedAndReinstatedInTheirOwnBalance(t *testing.T) {
	f := newFixture(t)
	engine := f.engine(f.medals, f.pass)
	ctx := context.Background()
	ayla := f.account(t)
	store := cosmetics.Source{Type: "shop", Ref: "offer.crystals_tier2"}
	pack := reward.Bundle{ID: "bundle.crystals_tier2", Revision: 1, Crystals: 550}
	if _, err := f.wallet.Apply(ctx, wallet.Entry{Account: ayla, Amount: 70, Reason: wallet.MatchReward, Source: wallet.Source{Type: "match", Ref: "m1"}, Key: "match:m1"}, start); err != nil {
		t.Fatal(err)
	}
	report, err := engine.Apply(ctx, reward.Grant{Account: ayla, Bundle: pack, Source: store, Reason: wallet.Purchase, Key: "purchase:google:t1"}, start)
	if err != nil || report.Crystals != 550 || report.Coins != 0 {
		t.Fatalf("crystal grant %+v %v", report, err)
	}
	balances := func() (int64, int64) {
		coins, err := f.wallet.Balance(ctx, ayla, wallet.Coin)
		if err != nil {
			t.Fatal(err)
		}
		crystals, err := f.wallet.Balance(ctx, ayla, wallet.Crystal)
		if err != nil {
			t.Fatal(err)
		}
		return coins, crystals
	}
	if coins, crystals := balances(); coins != 70 || crystals != 550 {
		t.Fatalf("after the grant coins %d crystals %d", coins, crystals)
	}
	if _, err := engine.Revoke(ctx, ayla, "purchase:google:t1", wallet.RefundRevoke, "refund:google:t1", start); err != nil {
		t.Fatal(err)
	}
	if coins, crystals := balances(); coins != 70 || crystals != 0 {
		t.Fatalf("after the refund coins %d crystals %d", coins, crystals)
	}
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err := engine.ReinstateIn(ctx, tx, ayla, "purchase:google:t1", wallet.RefundReversal, "refund_reversal:google:t1", start); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if coins, crystals := balances(); coins != 70 || crystals != 550 {
		t.Fatalf("after the reversal coins %d crystals %d", coins, crystals)
	}
}

func TestAFreeSourceCannotGrantCrystals(t *testing.T) {
	f := newFixture(t)
	engine := f.engine(f.medals, f.pass)
	ayla := f.account(t)
	crystals := reward.Bundle{ID: "bundle.free_crystals", Revision: 1, Crystals: 50}
	for _, reason := range []wallet.Reason{wallet.MatchReward, wallet.DailyLogin, wallet.Mission, wallet.RankReward, wallet.EventReward, wallet.Promo} {
		if _, err := engine.Apply(context.Background(), reward.Grant{Account: ayla, Bundle: crystals, Source: spring, Reason: reason, Key: "k:" + string(reason)}, start); !errors.Is(err, reward.ErrInvalidGrant) {
			t.Errorf("%s granted crystals: %v", reason, err)
		}
	}
	if balance, err := f.wallet.Balance(context.Background(), ayla, wallet.Crystal); err != nil || balance != 0 {
		t.Fatalf("crystal balance %d %v", balance, err)
	}
}

func TestARefundNeverTouchesCoins(t *testing.T) {
	f := newFixture(t)
	engine := f.engine(f.medals, f.pass)
	ctx := context.Background()
	ayla := f.account(t)
	grant := reward.Grant{Account: ayla, Bundle: fox, Source: spring, Reason: wallet.EventReward, Key: "event:spring_cup:r6"}
	if _, err := engine.Apply(ctx, grant, start); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Revoke(ctx, ayla, grant.Key, wallet.RefundRevoke, "refund:google:t13", start); err != nil {
		t.Fatal(err)
	}
	if balance, err := f.wallet.Balance(ctx, ayla, wallet.Coin); err != nil || balance != fox.Coins {
		t.Fatalf("coins after a refund %d %v", balance, err)
	}
	if owns, err := f.inventory.Owns(ctx, ayla, "avatar.comet_fox"); err != nil || owns {
		t.Fatalf("item kept after a refund %v %v", owns, err)
	}
}

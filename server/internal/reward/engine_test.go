package reward_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

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

func (f *fakePass) AddIn(_ context.Context, _ pgx.Tx, _ string, xp int, _ time.Time) (bool, error) {
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
	fox    = reward.Bundle{ID: "bundle.spring_fox", Revision: 2, Coins: 300, Cosmetics: []string{"avatar.comet_fox", "action.victory_dance"}, Medals: []string{"medal.spring"}, PassXP: 40}
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
	if balance, err := f.wallet.Balance(ctx, ayla); err != nil || balance != 300 {
		t.Fatalf("balance %d %v", balance, err)
	}
	if owns, err := f.inventory.Owns(ctx, ayla, "action.victory_dance"); err != nil || !owns {
		t.Fatalf("owns %v %v", owns, err)
	}
	if f.pass.added != 40 || !f.medals.held[ayla+":medal.spring"] {
		t.Fatalf("pass %d medals %v", f.pass.added, f.medals.held)
	}
	again, err := engine.Apply(ctx, grant, start.Add(time.Hour))
	if err != nil || !again.Repeated || again.ID != report.ID || again.Coins != 300 {
		t.Fatalf("repeat %+v %v", again, err)
	}
	if balance, err := f.wallet.Balance(ctx, ayla); err != nil || balance != 300 {
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
	if compensation == 0 || report.Duplicates["avatar.comet_fox"] != compensation || len(report.Cosmetics) != 1 || report.Cosmetics[0] != "action.victory_dance" {
		t.Fatalf("duplicate handling %+v (compensation %d)", report, compensation)
	}
	if len(report.Medals) != 0 || len(report.MedalsSkipped) != 1 || report.PassXP != 0 || report.PassXPDropped != 40 {
		t.Fatalf("medals and xp without services %+v", report)
	}
	if balance, err := f.wallet.Balance(ctx, ayla); err != nil || balance != 300+compensation {
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
	if balance, err := f.wallet.Balance(ctx, ayla); err != nil || balance != 0 {
		t.Fatalf("coins credited by a refused grant: %d %v", balance, err)
	}
	if _, err := engine.Get(ctx, ayla, "mission:x:1"); !errors.Is(err, reward.ErrUnknownGrant) {
		t.Fatalf("refused grant recorded: %v", err)
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
	if _, err := f.inventory.Equip(ctx, ayla, "action", []string{"action.victory_dance"}, start); !errors.Is(err, inventory.ErrNoProfile) {
		t.Fatalf("equip without a profile: %v", err)
	}
	revoked, err := engine.Revoke(ctx, ayla, grant.Key, wallet.RefundRevoke, "refund:google:t9", start.Add(time.Hour))
	if err != nil || !revoked.Revoked() || revoked.RevokeKey != "refund:google:t9" || !revoked.RevokedAt.Equal(start.Add(time.Hour)) {
		t.Fatalf("revoke %+v %v", revoked, err)
	}
	if balance, err := f.wallet.Balance(ctx, ayla); err != nil || balance != 0 {
		t.Fatalf("balance after revoke %d %v", balance, err)
	}
	if owns, err := f.inventory.Owns(ctx, ayla, "avatar.comet_fox"); err != nil || owns {
		t.Fatalf("item kept after revoke %v %v", owns, err)
	}
	if len(f.medals.revoked) != 1 || f.medals.held[ayla+":medal.spring"] {
		t.Fatalf("medals after revoke %v %v", f.medals.revoked, f.medals.held)
	}
	if _, err := engine.Revoke(ctx, ayla, grant.Key, wallet.RefundRevoke, "refund:google:t9", start.Add(2*time.Hour)); !errors.Is(err, reward.ErrAlreadyRevoke) {
		t.Fatalf("second revoke: %v", err)
	}
	if loaded, err := engine.Get(ctx, ayla, grant.Key); err != nil || !loaded.Revoked() {
		t.Fatalf("loaded after revoke %+v %v", loaded, err)
	}
	if _, err := f.wallet.Apply(ctx, wallet.Entry{Account: ayla, Amount: -1, Reason: wallet.Rename, Source: wallet.Source{Type: "rename", Ref: "r"}, Key: "spend:r"}, start); !errors.Is(err, wallet.ErrInsufficient) {
		t.Fatalf("balance went negative: %v", err)
	}
	spent := f.account(t)
	if _, err := engine.Apply(ctx, reward.Grant{Account: spent, Bundle: reward.Bundle{ID: "bundle.coins", Revision: 1, Coins: 100}, Source: spring, Reason: wallet.EventReward, Key: "event:spring_cup:r4"}, start); err != nil {
		t.Fatal(err)
	}
	if _, err := f.wallet.Apply(ctx, wallet.Entry{Account: spent, Amount: -100, Reason: wallet.Rename, Source: wallet.Source{Type: "rename", Ref: "r"}, Key: "spend:r"}, start); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Revoke(ctx, spent, "event:spring_cup:r4", wallet.Rename, "rename:revoke", start); !errors.Is(err, wallet.ErrInsufficient) {
		t.Fatalf("revoke of spent coins with a plain debit: %v", err)
	}
	if loaded, err := engine.Get(ctx, spent, "event:spring_cup:r4"); err != nil || loaded.Revoked() {
		t.Fatalf("a refused revoke was recorded %+v %v", loaded, err)
	}
	if _, err := engine.Revoke(ctx, spent, "event:spring_cup:r4", wallet.RefundRevoke, "refund:google:t10", start); err != nil {
		t.Fatalf("refund revoke of spent coins: %v", err)
	}
	if balance, err := f.wallet.Balance(ctx, spent); err != nil || balance != -100 {
		t.Fatalf("balance after a refund revoke of spent coins %d %v", balance, err)
	}
	if _, err := f.wallet.Apply(ctx, wallet.Entry{Account: spent, Amount: -1, Reason: wallet.Rename, Source: wallet.Source{Type: "rename", Ref: "r"}, Key: "spend:r2"}, start); !errors.Is(err, wallet.ErrInsufficient) {
		t.Fatalf("spend from a negative balance: %v", err)
	}
	if _, err := f.wallet.Apply(ctx, wallet.Entry{Account: spent, Amount: 150, Reason: wallet.MatchReward, Source: wallet.Source{Type: "match", Ref: "m"}, Key: "earn:m"}, start); err != nil {
		t.Fatal(err)
	}
	if balance, _ := f.wallet.Balance(ctx, spent); balance != 50 {
		t.Fatalf("balance after earning back %d", balance)
	}
}

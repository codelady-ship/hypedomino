package promo_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/inventory"
	"github.com/Alievs-corp/hypedomino/server/internal/promo"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

var (
	now      = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	coins    = reward.Bundle{ID: "bundle.promo_coins", Revision: 1, Coins: 300}
	bigCoins = reward.Bundle{ID: "bundle.promo_big", Revision: 1, Coins: 2001}
	nite     = reward.Bundle{ID: "bundle.promo_nite", Revision: 1, Cosmetics: []string{"table.night_market"}}
	unlisted = reward.Bundle{ID: "bundle.promo_unlisted", Revision: 1, Cosmetics: []string{"avatar.comet_fox"}}
)

type fixture struct {
	pool      *pgxpool.Pool
	wallet    *wallet.Store
	inventory *inventory.Store
	engine    *reward.Engine
	store     *promo.Store
}

func newFixture(t *testing.T, config promo.Config) *fixture {
	t.Helper()
	_, pool := dbtest.Fresh(t)
	catalog, err := cosmetics.Load("../../../docs/cosmetics/examples/catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	hasher, err := promo.NewHasher("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{pool: pool, wallet: wallet.NewStore(pool)}
	f.inventory = inventory.NewStore(pool, catalog, inventory.DefaultConfig())
	f.engine = reward.NewEngine(pool, f.wallet, f.inventory, nil, nil, reward.DefaultConfig())
	f.store = promo.NewStore(pool, f.engine, catalog, hasher, config)
	return f
}

func (f *fixture) account(t *testing.T) string {
	t.Helper()
	var id string
	if err := f.pool.QueryRow(context.Background(), `insert into accounts (created_at) values ($1) returning id`, now.Add(-time.Hour)).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func definition(id, kind string, bundle reward.Bundle) promo.Definition {
	return promo.Definition{ID: id, Campaign: "campaign.launch_week", Kind: kind, Bundle: bundle, Start: now.Add(-time.Hour), End: now.Add(24 * time.Hour), Creator: "admin.alice"}
}

func (f *fixture) create(t *testing.T, d promo.Definition) promo.Created {
	t.Helper()
	created, err := f.store.Create(context.Background(), d, now)
	if err != nil {
		t.Fatalf("create %s: %v", d.ID, err)
	}
	return created
}

func (f *fixture) redemptions(t *testing.T, id string) int {
	t.Helper()
	var count int
	if err := f.pool.QueryRow(context.Background(), `select count(*) from promo_redemptions where promo_id = $1`, id).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func TestDefinitionsAreValidated(t *testing.T) {
	f := newFixture(t, promo.DefaultConfig())
	cases := map[string]func(*promo.Definition){
		"bad id":              func(d *promo.Definition) { d.ID = "code.x" },
		"bad campaign":        func(d *promo.Definition) { d.Campaign = "promo.x" },
		"bad kind":            func(d *promo.Definition) { d.Kind = "forever" },
		"no creator":          func(d *promo.Definition) { d.Creator = "" },
		"reversed window":     func(d *promo.Definition) { d.End = d.Start },
		"window too long":     func(d *promo.Definition) { d.End = d.Start.Add(91 * 24 * time.Hour) },
		"multi without cap":   func(d *promo.Definition) { d.Cap = 0 },
		"single with cap":     func(d *promo.Definition) { d.Kind = promo.SingleUse },
		"single chosen code":  func(d *promo.Definition) { d.Kind, d.Cap, d.Code = promo.SingleUse, 0, "HYPE2345" },
		"multi with accounts": func(d *promo.Definition) { d.Accounts = []string{"a"} },
		"duplicate accounts":  func(d *promo.Definition) { d.Kind, d.Cap, d.Accounts = promo.Targeted, 0, []string{"a", "a"} },
		"medal":               func(d *promo.Definition) { d.Bundle.Medals = []string{"medal.first"} },
		"crystals":            func(d *promo.Definition) { d.Bundle.Crystals = 10 },
		"empty bundle":        func(d *promo.Definition) { d.Bundle = reward.Bundle{ID: "bundle.empty", Revision: 1} },
		"look-alike in code":  func(d *promo.Definition) { d.Code = "WELC0ME" },
		"unknown account": func(d *promo.Definition) {
			d.Kind, d.Cap, d.Accounts = promo.Targeted, 0, []string{"00000000-0000-0000-0000-000000000000"}
		},
	}
	for name, mutate := range cases {
		d := definition("promo.valid", promo.MultiUse, coins)
		d.Cap = 5
		mutate(&d)
		if _, err := f.store.Create(context.Background(), d, now); !errors.Is(err, promo.ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := f.store.Batch(context.Background(), definition("promo.batch", promo.MultiUse, coins), 3, now); !errors.Is(err, promo.ErrInvalid) {
		t.Errorf("batch of multi_use: %v", err)
	}
}

func TestAMultiUseCodeGrantsOncePerAccountUpToItsCap(t *testing.T) {
	f := newFixture(t, promo.DefaultConfig())
	d := definition("promo.welcome", promo.MultiUse, coins)
	d.Cap, d.Code = 2, "hype-2345"
	created := f.create(t, d)
	if created.Plaintext != "HYPE2345" || created.Code.State != promo.StateActive || created.Code.Cap != 2 {
		t.Fatalf("created %+v", created)
	}
	if _, err := f.store.Create(context.Background(), d, now); !errors.Is(err, promo.ErrDuplicate) {
		t.Fatalf("duplicate: %v", err)
	}
	a, b, c := f.account(t), f.account(t), f.account(t)
	redemption, err := f.store.Redeem(context.Background(), a, " hype 2345 ", now)
	if err != nil {
		t.Fatal(err)
	}
	if redemption.Promo != "promo.welcome" || redemption.Campaign != "campaign.launch_week" || redemption.Grant.Coins != 300 || redemption.Grant.Key != "promo:promo.welcome:"+a || redemption.PassPremium {
		t.Fatalf("redemption %+v", redemption)
	}
	if balance, _ := f.wallet.Balance(context.Background(), a, wallet.Coin); balance != 300 {
		t.Fatalf("balance %d", balance)
	}
	if _, err := f.store.Redeem(context.Background(), a, "HYPE2345", now.Add(time.Minute)); !errors.Is(err, promo.ErrAlreadyRedeemed) {
		t.Fatalf("second redemption: %v", err)
	}
	if _, err := f.store.Redeem(context.Background(), b, "HYPE2345", now); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.Redeem(context.Background(), c, "HYPE2345", now); !errors.Is(err, promo.ErrExhausted) {
		t.Fatalf("over the cap: %v", err)
	}
	if _, err := f.store.Redeem(context.Background(), c, "NXSUCHCDE", now); !errors.Is(err, promo.ErrUnknown) {
		t.Fatalf("unknown: %v", err)
	}
	code, err := f.store.Get(context.Background(), "promo.welcome")
	if err != nil || code.Redemptions != 2 {
		t.Fatalf("get %+v %v", code, err)
	}
}

func TestParallelRedemptionsNeverExceedTheCap(t *testing.T) {
	f := newFixture(t, promo.Config{CodeLength: 12, MaxValidityDays: 90, AttemptsPerHour: 100, FailedAttemptsBeforeLock: 100, LockMinutes: 1, CoinsApprovalThreshold: 2000, LockSignalWeight: 5, SpikePercent: 100, SpikeWindowMinutes: 1})
	d := definition("promo.rush", promo.MultiUse, coins)
	d.Cap = 3
	created := f.create(t, d)
	accounts := make([]string, 12)
	for i := range accounts {
		accounts[i] = f.account(t)
	}
	var wg sync.WaitGroup
	results := make([]error, len(accounts))
	for i, account := range accounts {
		wg.Add(2)
		for j := 0; j < 2; j++ {
			go func(slot int, account string) {
				defer wg.Done()
				_, err := f.store.Redeem(context.Background(), account, created.Plaintext, now)
				if err == nil {
					results[slot] = nil
					return
				}
				if !errors.Is(err, promo.ErrExhausted) && !errors.Is(err, promo.ErrAlreadyRedeemed) {
					results[slot] = err
				}
			}(i, account)
		}
		wg.Wait()
	}
	for i, err := range results {
		if err != nil {
			t.Errorf("account %d: %v", i, err)
		}
	}
	if got := f.redemptions(t, "promo.rush"); got != 3 {
		t.Fatalf("redemptions %d", got)
	}
	var total int64
	if err := f.pool.QueryRow(context.Background(), `select coalesce(sum(amount), 0) from ledger_entries where reason = 'promo'`).Scan(&total); err != nil {
		t.Fatal(err)
	}
	if total != 900 {
		t.Fatalf("coins granted %d", total)
	}
}

func TestParallelAccountsRaceForTheLastSlots(t *testing.T) {
	f := newFixture(t, promo.Config{CodeLength: 12, MaxValidityDays: 90, AttemptsPerHour: 100, FailedAttemptsBeforeLock: 100, LockMinutes: 1, CoinsApprovalThreshold: 2000, LockSignalWeight: 5, SpikePercent: 100, SpikeWindowMinutes: 1})
	d := definition("promo.race", promo.MultiUse, coins)
	d.Cap = 4
	created := f.create(t, d)
	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for i := 0; i < 16; i++ {
		account := f.account(t)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := f.store.Redeem(context.Background(), account, created.Plaintext, now); err != nil && !errors.Is(err, promo.ErrExhausted) {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	if got := f.redemptions(t, "promo.race"); got != 4 {
		t.Fatalf("redemptions %d", got)
	}
}

func TestParallelAttemptsOfOneAccountStayWithinTheHourlyLimit(t *testing.T) {
	f := newFixture(t, promo.Config{CodeLength: 12, MaxValidityDays: 90, AttemptsPerHour: 3, FailedAttemptsBeforeLock: 100, LockMinutes: 30, CoinsApprovalThreshold: 2000, LockSignalWeight: 5, SpikePercent: 100, SpikeWindowMinutes: 1})
	a := f.account(t)
	var wg sync.WaitGroup
	results := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := f.store.Redeem(context.Background(), a, "WRXNGCDE", now)
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	unknown, limited := 0, 0
	for err := range results {
		switch {
		case errors.Is(err, promo.ErrUnknown):
			unknown++
		case errors.Is(err, promo.ErrRateLimited):
			limited++
		default:
			t.Errorf("unexpected result %v", err)
		}
	}
	if unknown != 3 || limited != 9 {
		t.Fatalf("unknown %d, rate limited %d", unknown, limited)
	}
}

func TestABatchGeneratesUniqueSingleUseCodes(t *testing.T) {
	f := newFixture(t, promo.DefaultConfig())
	created, err := f.store.Batch(context.Background(), definition("promo.launch", promo.SingleUse, coins), 12, now)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for i, c := range created {
		if c.Code.ID != fmt.Sprintf("promo.launch_%02d", i+1) || c.Code.Cap != 1 || c.Code.State != promo.StateActive || len(c.Plaintext) != 12 || seen[c.Plaintext] {
			t.Fatalf("code %d: %+v %s", i, c.Code, c.Plaintext)
		}
		seen[c.Plaintext] = true
	}
	a, b := f.account(t), f.account(t)
	if _, err := f.store.Redeem(context.Background(), a, strings.ToLower(created[0].Plaintext), now); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.Redeem(context.Background(), b, created[0].Plaintext, now); !errors.Is(err, promo.ErrExhausted) {
		t.Fatalf("single_use reused: %v", err)
	}
	if _, err := f.store.Redeem(context.Background(), a, created[1].Plaintext, now); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.Batch(context.Background(), definition("promo.launch", promo.SingleUse, coins), 12, now); !errors.Is(err, promo.ErrDuplicate) {
		t.Fatalf("duplicate batch: %v", err)
	}
}

func TestATargetedCodeIsRedeemableByListedAccountsOnly(t *testing.T) {
	f := newFixture(t, promo.DefaultConfig())
	a, b, c := f.account(t), f.account(t), f.account(t)
	d := definition("promo.vip", promo.Targeted, coins)
	d.Accounts = []string{a, b}
	created := f.create(t, d)
	if created.Code.Cap != 2 {
		t.Fatalf("cap %d", created.Code.Cap)
	}
	if _, err := f.store.Redeem(context.Background(), c, created.Plaintext, now); !errors.Is(err, promo.ErrNotEligible) {
		t.Fatalf("unlisted: %v", err)
	}
	for _, account := range []string{a, b} {
		if _, err := f.store.Redeem(context.Background(), account, created.Plaintext, now); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLargeOrMythicBundlesNeedASecondAdmin(t *testing.T) {
	f := newFixture(t, promo.DefaultConfig())
	a := f.account(t)
	for _, bundle := range []reward.Bundle{bigCoins, nite} {
		d := definition("promo."+strings.TrimPrefix(bundle.ID, "bundle.promo_"), promo.MultiUse, bundle)
		d.Cap = 10
		created := f.create(t, d)
		if created.Code.State != promo.StatePending {
			t.Fatalf("%s state %s", d.ID, created.Code.State)
		}
		if _, err := f.store.Redeem(context.Background(), a, created.Plaintext, now); !errors.Is(err, promo.ErrNotActive) {
			t.Fatalf("pending redeemed: %v", err)
		}
		if _, err := f.store.Approve(context.Background(), d.ID, "admin.alice", now); !errors.Is(err, promo.ErrApprover) {
			t.Fatalf("self approval: %v", err)
		}
		code, err := f.store.Approve(context.Background(), d.ID, "admin.bob", now)
		if err != nil || code.State != promo.StateActive || code.Approver != "admin.bob" {
			t.Fatalf("approve %+v %v", code, err)
		}
		if _, err := f.store.Approve(context.Background(), d.ID, "admin.bob", now); !errors.Is(err, promo.ErrNotActive) {
			t.Fatalf("second approval: %v", err)
		}
		if _, err := f.store.Redeem(context.Background(), a, created.Plaintext, now); err != nil {
			t.Fatal(err)
		}
	}
	if owns, _ := f.inventory.Owns(context.Background(), a, "table.night_market"); !owns {
		t.Fatal("mythic table not granted")
	}
	if _, err := f.store.Approve(context.Background(), "promo.missing", "admin.bob", now); !errors.Is(err, promo.ErrUnknown) {
		t.Fatalf("approve unknown: %v", err)
	}
}

func TestDisabledAndOutOfWindowCodesAreRefused(t *testing.T) {
	f := newFixture(t, promo.DefaultConfig())
	a := f.account(t)
	d := definition("promo.window", promo.MultiUse, coins)
	d.Cap = 10
	created := f.create(t, d)
	if _, err := f.store.Redeem(context.Background(), a, created.Plaintext, now.Add(-2*time.Hour)); !errors.Is(err, promo.ErrNotStarted) {
		t.Fatalf("before start: %v", err)
	}
	if _, err := f.store.Redeem(context.Background(), a, created.Plaintext, now.Add(24*time.Hour)); !errors.Is(err, promo.ErrExpired) {
		t.Fatalf("after end: %v", err)
	}
	code, err := f.store.Disable(context.Background(), "promo.window", now)
	if err != nil || code.State != promo.StateDisabled {
		t.Fatalf("disable %+v %v", code, err)
	}
	if _, err := f.store.Redeem(context.Background(), a, created.Plaintext, now); !errors.Is(err, promo.ErrDisabled) {
		t.Fatalf("disabled: %v", err)
	}
	if balance, _ := f.wallet.Balance(context.Background(), a, wallet.Coin); balance != 0 {
		t.Fatalf("balance %d", balance)
	}
}

func TestFailedAttemptsLockAndAttemptsAreRateLimited(t *testing.T) {
	f := newFixture(t, promo.Config{CodeLength: 12, MaxValidityDays: 90, AttemptsPerHour: 4, FailedAttemptsBeforeLock: 2, LockMinutes: 30, CoinsApprovalThreshold: 2000, LockSignalWeight: 5, SpikePercent: 50, SpikeWindowMinutes: 60})
	a := f.account(t)
	d := definition("promo.limits", promo.MultiUse, coins)
	d.Cap = 10
	created := f.create(t, d)
	for i := 0; i < 2; i++ {
		if _, err := f.store.Redeem(context.Background(), a, "WRXNGCDE", now); !errors.Is(err, promo.ErrUnknown) {
			t.Fatalf("attempt %d: %v", i, err)
		}
	}
	if _, err := f.store.Redeem(context.Background(), a, created.Plaintext, now.Add(time.Minute)); !errors.Is(err, promo.ErrLocked) {
		t.Fatalf("locked: %v", err)
	}
	if _, err := f.store.Redeem(context.Background(), a, "WRXNGCDE", now.Add(time.Minute)); !errors.Is(err, promo.ErrLocked) {
		t.Fatalf("locked failure: %v", err)
	}
	if kind, weight, failed := f.lockSignal(t, a); kind != 1 || weight != 5 || failed != 2 {
		t.Fatalf("lock signals %d, weight %d, failed %d", kind, weight, failed)
	}
	unlocked := now.Add(31 * time.Minute)
	if _, err := f.store.Redeem(context.Background(), a, created.Plaintext, unlocked); err != nil {
		t.Fatalf("after the lock: %v", err)
	}
	if _, err := f.store.Redeem(context.Background(), a, created.Plaintext, unlocked); !errors.Is(err, promo.ErrAlreadyRedeemed) {
		t.Fatalf("fourth attempt: %v", err)
	}
	if _, err := f.store.Redeem(context.Background(), a, created.Plaintext, unlocked.Add(time.Minute)); !errors.Is(err, promo.ErrRateLimited) {
		t.Fatalf("fifth attempt in the hour: %v", err)
	}
	if _, err := f.store.Redeem(context.Background(), a, created.Plaintext, now.Add(2*time.Hour)); !errors.Is(err, promo.ErrAlreadyRedeemed) {
		t.Fatalf("next hour: %v", err)
	}
	if _, err := f.store.Redeem(context.Background(), f.account(t), "bad!", now); !errors.Is(err, promo.ErrCodeFormat) {
		t.Fatalf("format: %v", err)
	}
}

func TestAnUnlistedSourceWritesNothing(t *testing.T) {
	f := newFixture(t, promo.DefaultConfig())
	a := f.account(t)
	d := definition("promo.unlisted", promo.MultiUse, unlisted)
	d.Cap = 10
	created := f.create(t, d)
	if _, err := f.store.Redeem(context.Background(), a, created.Plaintext, now); !errors.Is(err, inventory.ErrSourceNotListed) {
		t.Fatalf("unlisted source: %v", err)
	}
	if got := f.redemptions(t, "promo.unlisted"); got != 0 {
		t.Fatalf("redemptions %d", got)
	}
	if owns, _ := f.inventory.Owns(context.Background(), a, "avatar.comet_fox"); owns {
		t.Fatal("item granted")
	}
}

func TestSupportListsTheAccountsOfATargetedCode(t *testing.T) {
	f := newFixture(t, promo.DefaultConfig())
	ctx := context.Background()
	a, b, c := f.account(t), f.account(t), f.account(t)
	created := f.create(t, definition("promo.vip", promo.Targeted, coins))
	if created.Code.Cap != 0 {
		t.Fatalf("cap before listing %d", created.Code.Cap)
	}
	if _, err := f.store.Redeem(ctx, a, created.Plaintext, now); !errors.Is(err, promo.ErrNotEligible) {
		t.Fatalf("redeem before listing: %v", err)
	}
	change, err := f.store.SetTargets(ctx, "promo.vip", []string{a, b})
	if err != nil || len(change.Added) != 2 || len(change.Removed) != 0 || change.Count != 2 {
		t.Fatalf("first list %+v %v", change, err)
	}
	if _, err := f.store.Redeem(ctx, a, created.Plaintext, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.SetTargets(ctx, "promo.vip", []string{b}); !errors.Is(err, promo.ErrInvalid) {
		t.Fatalf("dropping a redeemer: %v", err)
	}
	change, err = f.store.SetTargets(ctx, "promo.vip", []string{a, c})
	if err != nil || !slices.Equal(change.Added, []string{c}) || !slices.Equal(change.Removed, []string{b}) {
		t.Fatalf("second list %+v %v", change, err)
	}
	code, _ := f.store.Get(ctx, "promo.vip")
	targets, _ := f.store.Targets(ctx, "promo.vip")
	redeemed, _ := f.store.Redemptions(ctx, "promo.vip")
	if code.Cap != 2 || len(targets) != 2 || !slices.Contains(targets, c) || len(redeemed) != 1 || redeemed[0].Account != a {
		t.Fatalf("code %+v targets %v redeemed %v", code, targets, redeemed)
	}
	for name, accounts := range map[string][]string{"unknown": {"00000000-0000-0000-0000-000000000000"}, "not an id": {"kamal"}, "repeated": {a, a}} {
		if _, err := f.store.SetTargets(ctx, "promo.vip", accounts); !errors.Is(err, promo.ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	f.create(t, func() promo.Definition { d := definition("promo.open", promo.MultiUse, coins); d.Cap = 5; return d }())
	if _, err := f.store.SetTargets(ctx, "promo.open", []string{a}); !errors.Is(err, promo.ErrInvalid) {
		t.Fatalf("multi_use: %v", err)
	}
	if _, err := f.store.SetTargets(ctx, "promo.none", []string{a}); !errors.Is(err, promo.ErrUnknown) {
		t.Fatalf("unknown code: %v", err)
	}
	f.store.Disable(ctx, "promo.vip", now)
	if _, err := f.store.SetTargets(ctx, "promo.vip", []string{a}); !errors.Is(err, promo.ErrDisabled) {
		t.Fatalf("disabled: %v", err)
	}
}

func TestATemplateStaysADraftOutsideTheCodes(t *testing.T) {
	f := newFixture(t, promo.DefaultConfig())
	ctx := context.Background()
	welcome := promo.Template{ID: "promo.welcome", Campaign: "campaign.launch", Kind: promo.MultiUse, Bundle: &nite, Creator: "leyla"}

	for name, broken := range map[string]promo.Template{
		"id":       {ID: "welcome", Campaign: "campaign.launch", Kind: promo.MultiUse, Creator: "leyla"},
		"campaign": {ID: "promo.welcome", Campaign: "launch", Kind: promo.MultiUse, Creator: "leyla"},
		"kind":     {ID: "promo.welcome", Campaign: "campaign.launch", Kind: "gift", Creator: "leyla"},
		"creator":  {ID: "promo.welcome", Campaign: "campaign.launch", Kind: promo.MultiUse},
		"medal":    {ID: "promo.welcome", Campaign: "campaign.launch", Kind: promo.MultiUse, Creator: "leyla", Bundle: &reward.Bundle{ID: "bundle.medal", Revision: 1, Medals: []string{"medal.first_win"}}},
		"bundle":   {ID: "promo.welcome", Campaign: "campaign.launch", Kind: promo.MultiUse, Creator: "leyla", Bundle: &reward.Bundle{ID: "bundle.empty", Revision: 1}},
	} {
		if _, err := f.store.CreateTemplate(ctx, broken, now); !errors.Is(err, promo.ErrInvalid) {
			t.Fatalf("%s: %v", name, err)
		}
	}

	created, err := f.store.CreateTemplate(ctx, welcome, now)
	if err != nil || !created.Template || created.State != promo.StateDraft {
		t.Fatalf("create %+v %v", created, err)
	}
	if _, err := f.store.CreateTemplate(ctx, welcome, now); !errors.Is(err, promo.ErrDuplicate) {
		t.Fatalf("duplicate %v", err)
	}
	compensation := promo.Template{ID: "promo.compensation", Campaign: "campaign.compensation", Kind: promo.SingleUse, Creator: "leyla"}
	if _, err := f.store.CreateTemplate(ctx, compensation, now); err != nil {
		t.Fatal(err)
	}

	later := now.Add(time.Hour)
	before, after, err := f.store.EditTemplate(ctx, promo.Template{ID: "promo.welcome", Campaign: "campaign.launch_week", Kind: promo.MultiUse, Bundle: &coins, Creator: "vusal"}, later)
	if err != nil || before.Campaign != "campaign.launch" || after.Campaign != "campaign.launch_week" || after.Creator != "leyla" || !after.Updated.Equal(later) {
		t.Fatalf("edit %+v %+v %v", before, after, err)
	}
	if _, _, err := f.store.EditTemplate(ctx, promo.Template{ID: "promo.missing", Campaign: "campaign.launch", Kind: promo.MultiUse}, later); !errors.Is(err, promo.ErrUnknown) {
		t.Fatalf("unknown %v", err)
	}

	templates, err := f.store.Templates(ctx)
	if err != nil || len(templates) != 2 || templates[0].ID != "promo.compensation" || templates[0].Bundle != nil || templates[1].Bundle.Coins != coins.Coins || templates[1].State != promo.StateDraft {
		t.Fatalf("templates %+v %v", templates, err)
	}
	if codes, err := f.store.List(ctx, promo.ListQuery{Limit: 10}); err != nil || len(codes) != 0 {
		t.Fatalf("codes %v %v", codes, err)
	}
}

func (f *fixture) lockSignal(t *testing.T, account string) (int, int, int) {
	t.Helper()
	var count, weight, failed int
	if err := f.pool.QueryRow(context.Background(), `select count(*), coalesce(max(weight), 0), coalesce(max((evidence->>'failed')::int), 0) from risk_signals where account_id = $1 and kind = $2`, account, promo.SignalLock).Scan(&count, &weight, &failed); err != nil {
		t.Fatal(err)
	}
	return count, weight, failed
}

func TestAMultiUseCodeIsMarkedOnceWhenHalfItsCapGoesWithinTheWindow(t *testing.T) {
	f := newFixture(t, promo.DefaultConfig())
	d := definition("promo.leaky", promo.MultiUse, coins)
	d.Cap, d.Code = 4, "WAVE2345"
	f.create(t, d)
	slow := definition("promo.steady", promo.MultiUse, coins)
	slow.Cap, slow.Code = 4, "STEADY23"
	f.create(t, slow)
	for i, want := range []bool{false, true, false} {
		redemption, err := f.store.Redeem(context.Background(), f.account(t), "WAVE2345", now.Add(time.Duration(i)*time.Minute))
		if err != nil || redemption.Spiked != want {
			t.Fatalf("redemption %d: spiked %v, want %v, %v", i+1, redemption.Spiked, want, err)
		}
	}
	code, err := f.store.Get(context.Background(), "promo.leaky")
	if err != nil || code.SpikeAt == nil || !code.SpikeAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("leaky code %+v %v", code, err)
	}
	for i := range 3 {
		redemption, err := f.store.Redeem(context.Background(), f.account(t), "STEADY23", now.Add(time.Duration(i)*61*time.Minute))
		if err != nil || redemption.Spiked {
			t.Fatalf("steady redemption %d: spiked %v, %v", i+1, redemption.Spiked, err)
		}
	}
	if code, err := f.store.Get(context.Background(), "promo.steady"); err != nil || code.SpikeAt != nil {
		t.Fatalf("steady code %+v %v", code, err)
	}
}

func TestSingleUseCodesAreNeverMarkedAsSpikes(t *testing.T) {
	f := newFixture(t, promo.DefaultConfig())
	created := f.create(t, definition("promo.once", promo.SingleUse, coins))
	redemption, err := f.store.Redeem(context.Background(), f.account(t), created.Plaintext, now)
	if err != nil || redemption.Spiked {
		t.Fatalf("single use: spiked %v, %v", redemption.Spiked, err)
	}
}

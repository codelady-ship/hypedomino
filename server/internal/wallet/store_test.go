package wallet_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

var start = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

func account(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `insert into accounts default values returning id`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

var match = wallet.Source{Type: "match", Ref: "m1"}

func TestEntriesAreIdempotentPerAccountAndKey(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := wallet.NewStore(pool)
	ctx := context.Background()
	ayla, ben := account(t, pool), account(t, pool)
	if balance, err := store.Balance(ctx, ayla, wallet.Coin); err != nil || balance != 0 {
		t.Fatalf("balance before any entry %d %v", balance, err)
	}
	first, err := store.Apply(ctx, wallet.Entry{Account: ayla, Amount: 120, Reason: wallet.MatchReward, Source: match, Key: "match:m1", ConfigVersion: 3}, start)
	if err != nil || first.Repeated || first.BalanceAfter != 120 || first.ID == 0 || !first.At.Equal(start) || first.ConfigVersion != 3 {
		t.Fatalf("first entry %+v %v", first, err)
	}
	again, err := store.Apply(ctx, wallet.Entry{Account: ayla, Amount: 999, Reason: wallet.MatchReward, Source: match, Key: "match:m1", ConfigVersion: 4}, start.Add(time.Hour))
	if err != nil || !again.Repeated || again.ID != first.ID || again.Amount != 120 || again.BalanceAfter != 120 {
		t.Fatalf("repeated entry %+v %v", again, err)
	}
	if balance, err := store.Balance(ctx, ayla, wallet.Coin); err != nil || balance != 120 {
		t.Fatalf("balance after the repeat %d %v", balance, err)
	}
	other, err := store.Apply(ctx, wallet.Entry{Account: ben, Amount: 50, Reason: wallet.MatchReward, Source: match, Key: "match:m1"}, start)
	if err != nil || other.Repeated || other.BalanceAfter != 50 {
		t.Fatalf("same key on another account %+v %v", other, err)
	}
	history, err := store.History(ctx, ayla, 10)
	if err != nil || len(history) != 1 || history[0].ID != first.ID {
		t.Fatalf("history %+v %v", history, err)
	}
}

func TestEntriesAreValidatedAndSpendsNeverGoNegative(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := wallet.NewStore(pool)
	ctx := context.Background()
	ayla := account(t, pool)
	for name, entry := range map[string]struct {
		entry wallet.Entry
		err   error
	}{
		"no key":         {wallet.Entry{Account: ayla, Amount: 10, Reason: wallet.MatchReward, Source: match}, wallet.ErrInvalid},
		"no source":      {wallet.Entry{Account: ayla, Amount: 10, Reason: wallet.MatchReward, Key: "k"}, wallet.ErrInvalid},
		"zero":           {wallet.Entry{Account: ayla, Amount: 0, Reason: wallet.MatchReward, Source: match, Key: "k"}, wallet.ErrInvalid},
		"reason":         {wallet.Entry{Account: ayla, Amount: 10, Reason: "gift", Source: match, Key: "k"}, wallet.ErrUnknownReason},
		"credit sign":    {wallet.Entry{Account: ayla, Amount: -10, Reason: wallet.MatchReward, Source: match, Key: "k"}, wallet.ErrWrongSign},
		"debit sign":     {wallet.Entry{Account: ayla, Amount: 10, Reason: wallet.Rename, Source: match, Key: "k"}, wallet.ErrWrongSign},
		"overspend":      {wallet.Entry{Account: ayla, Amount: -1, Reason: wallet.Rename, Source: match, Key: "spend:1"}, wallet.ErrInsufficient},
		"admin negative": {wallet.Entry{Account: ayla, Amount: -5, Reason: wallet.AdminAdjust, Source: wallet.Source{Type: "admin", Ref: "a1"}, Key: "admin:a1"}, wallet.ErrInsufficient},
	} {
		if _, err := store.Apply(ctx, entry.entry, start); !errors.Is(err, entry.err) {
			t.Fatalf("%s: %v, want %v", name, err, entry.err)
		}
	}
	if history, err := store.History(ctx, ayla, 10); err != nil || len(history) != 0 {
		t.Fatalf("refused entries were written: %+v %v", history, err)
	}
	if _, err := store.Apply(ctx, wallet.Entry{Account: ayla, Amount: 1000, Reason: wallet.Promo, Source: wallet.Source{Type: "promo", Ref: "c1"}, Key: "promo:c1"}, start); err != nil {
		t.Fatal(err)
	}
	spent, err := store.Apply(ctx, wallet.Entry{Account: ayla, Amount: -1000, Reason: wallet.Rename, Source: wallet.Source{Type: "rename", Ref: "r1"}, Key: "spend:r1"}, start)
	if err != nil || spent.BalanceAfter != 0 {
		t.Fatalf("spend to zero %+v %v", spent, err)
	}
	if adjusted, err := store.Apply(ctx, wallet.Entry{Account: ayla, Amount: 7, Reason: wallet.AdminAdjust, Source: wallet.Source{Type: "admin", Ref: "a2"}, Key: "admin:a2"}, start); err != nil || adjusted.BalanceAfter != 7 {
		t.Fatalf("positive adjust %+v %v", adjusted, err)
	}
	if adjusted, err := store.Apply(ctx, wallet.Entry{Account: ayla, Amount: -7, Reason: wallet.AdminAdjust, Source: wallet.Source{Type: "admin", Ref: "a3"}, Key: "admin:a3"}, start); err != nil || adjusted.BalanceAfter != 0 {
		t.Fatalf("negative adjust %+v %v", adjusted, err)
	}
}

func TestParallelSpendsNeverOverdraw(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := wallet.NewStore(pool)
	ctx := context.Background()
	ayla := account(t, pool)
	if _, err := store.Apply(ctx, wallet.Entry{Account: ayla, Amount: 500, Reason: wallet.Purchase, Source: wallet.Source{Type: "store", Ref: "t1"}, Key: "purchase:google:t1"}, start); err != nil {
		t.Fatal(err)
	}
	const attempts = 12
	var wg sync.WaitGroup
	errs := make([]error, attempts)
	for i := range attempts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = store.Apply(ctx, wallet.Entry{Account: ayla, Currency: wallet.Crystal, Amount: -100, Reason: wallet.ShopPurchase, Source: wallet.Source{Type: "offer", Ref: "o1"}, Key: fmt.Sprintf("spend:%d", i)}, start)
		}()
	}
	wg.Wait()
	won := 0
	for _, err := range errs {
		switch {
		case err == nil:
			won++
		case !errors.Is(err, wallet.ErrInsufficient):
			t.Fatalf("unexpected error %v", err)
		}
	}
	balance, err := store.Balance(ctx, ayla, wallet.Crystal)
	if err != nil || won != 5 || balance != 0 {
		t.Fatalf("%d spends won, balance %d %v", won, balance, err)
	}
	history, err := store.History(ctx, ayla, 100)
	if err != nil || len(history) != 6 || history[0].BalanceAfter != 0 || history[5].BalanceAfter != 500 {
		t.Fatalf("history %+v %v", history, err)
	}
}

func TestAPurchaseTransactionIsGrantedToOneAccountOnly(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := wallet.NewStore(pool)
	ctx := context.Background()
	ayla, ben := account(t, pool), account(t, pool)
	entry := wallet.Entry{Account: ayla, Amount: 500, Reason: wallet.Purchase, Source: wallet.Source{Type: "store", Ref: "t1"}, Key: "purchase:google:t1"}
	if _, err := store.Apply(ctx, entry, start); err != nil {
		t.Fatal(err)
	}
	entry.Account = ben
	if _, err := store.Apply(ctx, entry, start); !errors.Is(err, wallet.ErrPurchaseReuse) {
		t.Fatalf("purchase reused on another account: %v", err)
	}
	if balance, err := store.Balance(ctx, ben, wallet.Coin); err != nil || balance != 0 {
		t.Fatalf("ben's balance %d %v", balance, err)
	}
}

func TestAuditFindsTamperedBalancesAndReconcileRestoresThem(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := wallet.NewStore(pool)
	ctx := context.Background()
	ayla, ben := account(t, pool), account(t, pool)
	for _, id := range []string{ayla, ben} {
		if _, err := store.Apply(ctx, wallet.Entry{Account: id, Amount: 300, Reason: wallet.DailyLogin, Source: wallet.Source{Type: "daily", Ref: "d1"}, Key: "daily_login:1"}, start); err != nil {
			t.Fatal(err)
		}
	}
	if mismatches, err := store.Audit(ctx); err != nil || len(mismatches) != 0 {
		t.Fatalf("audit of consistent wallets %+v %v", mismatches, err)
	}
	if _, err := pool.Exec(ctx, `update wallets set balance = 9999 where account_id = $1 and currency = 'coin'`, ben); err != nil {
		t.Fatal(err)
	}
	mismatches, err := store.Audit(ctx)
	if err != nil || len(mismatches) != 1 || mismatches[0] != (wallet.Mismatch{Account: ben, Currency: wallet.Coin, Stored: 9999, Computed: 300}) {
		t.Fatalf("audit %+v %v", mismatches, err)
	}
	if m, fixed, err := store.Reconcile(ctx, ayla, wallet.Coin, start); err != nil || fixed || m.Stored != 300 {
		t.Fatalf("reconcile of a consistent wallet %+v %v %v", m, fixed, err)
	}
	if m, fixed, err := store.Reconcile(ctx, ben, wallet.Coin, start); err != nil || !fixed || m.Stored != 9999 || m.Computed != 300 {
		t.Fatalf("reconcile %+v %v %v", m, fixed, err)
	}
	if balance, err := store.Balance(ctx, ben, wallet.Coin); err != nil || balance != 300 {
		t.Fatalf("balance after reconcile %d %v", balance, err)
	}
}

func TestEveryReasonMovesTheCurrencyOfTheLedgerTable(t *testing.T) {
	for _, c := range []struct {
		reason          wallet.Reason
		coins, crystals bool
	}{
		{wallet.MatchReward, true, false},
		{wallet.FirstWin, true, false},
		{wallet.DailyLogin, true, false},
		{wallet.Mission, true, false},
		{wallet.RankReward, true, false},
		{wallet.EventReward, true, false},
		{wallet.DuplicateCompensation, true, false},
		{wallet.Promo, true, false},
		{wallet.Rename, true, false},
		{wallet.PassReward, true, true},
		{wallet.ShopPurchase, true, true},
		{wallet.PassLevelPurchase, true, true},
		{wallet.AdminAdjust, true, true},
		{wallet.Purchase, false, true},
		{wallet.FirstPurchaseBonus, false, true},
		{wallet.RefundRevoke, false, true},
		{wallet.RefundReversal, false, true},
	} {
		if wallet.Moves(c.reason, wallet.Coin) != c.coins || wallet.Moves(c.reason, wallet.Crystal) != c.crystals {
			t.Errorf("%s: coins %v crystals %v", c.reason, wallet.Moves(c.reason, wallet.Coin), wallet.Moves(c.reason, wallet.Crystal))
		}
	}
	if wallet.Moves("gift", wallet.Coin) || wallet.Moves(wallet.MatchReward, "gem") {
		t.Fatal("unknown reason or currency moves something")
	}
}

func TestAFreeSourceNeverGrantsCrystals(t *testing.T) {
	for _, reason := range []wallet.Reason{wallet.MatchReward, wallet.FirstWin, wallet.DailyLogin, wallet.Mission, wallet.RankReward, wallet.EventReward, wallet.DuplicateCompensation, wallet.Promo} {
		entry := wallet.Entry{Account: "a", Currency: wallet.Crystal, Amount: 10, Reason: reason, Source: match, Key: "k"}
		if err := entry.Validate(); !errors.Is(err, wallet.ErrWrongCurrency) {
			t.Errorf("%s in crystals: %v", reason, err)
		}
	}
	if err := (wallet.Entry{Account: "a", Currency: "gem", Amount: 10, Reason: wallet.Purchase, Source: match, Key: "k"}).Validate(); !errors.Is(err, wallet.ErrWrongCurrency) {
		t.Fatalf("unknown currency %v", err)
	}
}

func TestCoinsAndCrystalsAreSeparateBalances(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := wallet.NewStore(pool)
	ctx := context.Background()
	ayla := account(t, pool)
	bought := wallet.Source{Type: "store", Ref: "t1"}
	if _, err := store.Apply(ctx, wallet.Entry{Account: ayla, Amount: 200, Reason: wallet.MatchReward, Source: match, Key: "match:m1"}, start); err != nil {
		t.Fatal(err)
	}
	crystals, err := store.Apply(ctx, wallet.Entry{Account: ayla, Currency: wallet.Crystal, Amount: 550, Reason: wallet.Purchase, Source: bought, Key: "purchase:google:t1"}, start)
	if err != nil || crystals.Currency != wallet.Crystal || crystals.BalanceAfter != 550 {
		t.Fatalf("crystal purchase %+v %v", crystals, err)
	}
	if _, err := store.Apply(ctx, wallet.Entry{Account: ayla, Currency: wallet.Crystal, Amount: -600, Reason: wallet.ShopPurchase, Source: wallet.Source{Type: "offer", Ref: "o1"}, Key: "spend:r1"}, start); !errors.Is(err, wallet.ErrInsufficient) {
		t.Fatalf("crystal spend above the crystal balance with coins left: %v", err)
	}
	spent, err := store.Apply(ctx, wallet.Entry{Account: ayla, Currency: wallet.Crystal, Amount: -500, Reason: wallet.ShopPurchase, Source: wallet.Source{Type: "offer", Ref: "o1"}, Key: "spend:r1"}, start)
	if err != nil || spent.BalanceAfter != 50 {
		t.Fatalf("crystal spend %+v %v", spent, err)
	}
	coins, coinsErr := store.Balance(ctx, ayla, wallet.Coin)
	left, leftErr := store.Balance(ctx, ayla, wallet.Crystal)
	if coinsErr != nil || leftErr != nil || coins != 200 || left != 50 {
		t.Fatalf("balances coins %d crystals %d %v %v", coins, left, coinsErr, leftErr)
	}
	revoked, err := store.Apply(ctx, wallet.Entry{Account: ayla, Currency: wallet.Crystal, Amount: -550, Reason: wallet.RefundRevoke, Source: bought, Key: "refund:google:t1"}, start)
	if err != nil || revoked.BalanceAfter != -500 {
		t.Fatalf("refund below zero %+v %v", revoked, err)
	}
	if _, err := store.Apply(ctx, wallet.Entry{Account: ayla, Currency: wallet.Crystal, Amount: -1, Reason: wallet.PassLevelPurchase, Source: wallet.Source{Type: "pass", Ref: "s1"}, Key: "spend:r2"}, start); !errors.Is(err, wallet.ErrInsufficient) {
		t.Fatalf("crystal spend while negative: %v", err)
	}
	if spentCoins, err := store.Apply(ctx, wallet.Entry{Account: ayla, Amount: -200, Reason: wallet.Rename, Source: wallet.Source{Type: "rename", Ref: "r3"}, Key: "spend:r3"}, start); err != nil || spentCoins.BalanceAfter != 0 {
		t.Fatalf("coin spend while crystals are negative %+v %v", spentCoins, err)
	}
	if mismatches, err := store.Audit(ctx); err != nil || len(mismatches) != 0 {
		t.Fatalf("audit %+v %v", mismatches, err)
	}
}

func TestOneKeyIsIdempotentPerCurrency(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := wallet.NewStore(pool)
	ctx := context.Background()
	ayla := account(t, pool)
	adjust := wallet.Source{Type: "admin", Ref: "a1"}
	coins, err := store.Apply(ctx, wallet.Entry{Account: ayla, Amount: 40, Reason: wallet.AdminAdjust, Source: adjust, Key: "admin:a1"}, start)
	if err != nil || coins.Currency != wallet.Coin {
		t.Fatalf("coin adjust %+v %v", coins, err)
	}
	crystals, err := store.Apply(ctx, wallet.Entry{Account: ayla, Currency: wallet.Crystal, Amount: 9, Reason: wallet.AdminAdjust, Source: adjust, Key: "admin:a1"}, start)
	if err != nil || crystals.Repeated || crystals.BalanceAfter != 9 {
		t.Fatalf("crystal adjust with the same key %+v %v", crystals, err)
	}
	again, err := store.Apply(ctx, wallet.Entry{Account: ayla, Currency: wallet.Crystal, Amount: 99, Reason: wallet.AdminAdjust, Source: adjust, Key: "admin:a1"}, start)
	if err != nil || !again.Repeated || again.ID != crystals.ID {
		t.Fatalf("repeated crystal adjust %+v %v", again, err)
	}
	if _, err := pool.Exec(ctx, `update wallets set balance = 1 where account_id = $1 and currency = 'crystal'`, ayla); err != nil {
		t.Fatal(err)
	}
	mismatches, err := store.Audit(ctx)
	if err != nil || len(mismatches) != 1 || mismatches[0] != (wallet.Mismatch{Account: ayla, Currency: wallet.Crystal, Stored: 1, Computed: 9}) {
		t.Fatalf("audit %+v %v", mismatches, err)
	}
	if _, fixed, err := store.Reconcile(ctx, ayla, wallet.Crystal, start); err != nil || !fixed {
		t.Fatalf("reconcile crystals %v %v", fixed, err)
	}
	if balance, err := store.Balance(ctx, ayla, wallet.Coin); err != nil || balance != 40 {
		t.Fatalf("coins touched by the crystal reconcile %d %v", balance, err)
	}
}

func TestMoneyNeverCreatesCoins(t *testing.T) {
	for _, reason := range []wallet.Reason{wallet.Purchase, wallet.FirstPurchaseBonus, wallet.RefundReversal} {
		coins := wallet.Entry{Account: "a", Currency: wallet.Coin, Amount: 100, Reason: reason, Source: match, Key: "k"}
		if err := coins.Validate(); !errors.Is(err, wallet.ErrWrongCurrency) {
			t.Errorf("%s in coins: %v", reason, err)
		}
		if err := (wallet.Entry{Account: "a", Amount: 100, Reason: reason, Source: match, Key: "k"}).Validate(); err != nil {
			t.Errorf("%s without a currency takes crystals: %v", reason, err)
		}
	}
	refund := wallet.Entry{Account: "a", Currency: wallet.Coin, Amount: -100, Reason: wallet.RefundRevoke, Source: match, Key: "k"}
	if err := refund.Validate(); !errors.Is(err, wallet.ErrWrongCurrency) {
		t.Fatalf("a refund debits coins: %v", err)
	}
}

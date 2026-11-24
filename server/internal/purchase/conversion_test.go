package purchase_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/purchase"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/shop"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

func legacyCoinPack(t *testing.T, f *fixture, account, transaction, product string, coins int64, state string) {
	t.Helper()
	ctx := context.Background()
	key := "purchase:google:" + transaction
	if _, err := f.pool.Exec(ctx, `insert into purchases (platform, transaction_id, account_id, offer_id, product_id, environment, state, grant_key, purchased_at, granted_at)
		values ('google', $1, $2, $3, $4, 'production', $5, $6, $7, $7)`, transaction, account, "offer."+product, product, state, key, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	var balance int64
	if err := f.pool.QueryRow(ctx, `insert into wallets (account_id, currency, balance, updated_at) values ($1, 'coin', $2, $3)
		on conflict (account_id, currency) do update set balance = wallets.balance + $2 returning balance`, account, coins, now).Scan(&balance); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `insert into ledger_entries (account_id, currency, amount, reason, source_type, source_ref, idempotency_key, config_version, balance_after, created_at)
		values ($1, 'coin', $2, 'purchase', 'shop', $3, $4, 1, $5, $6)`, account, coins, "offer."+product, key, balance, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	report := reward.Report{Account: account, BundleID: "bundle." + product, Revision: 1, Source: cosmetics.Source{Type: shop.SourceType, Ref: "offer." + product}, Reason: wallet.Purchase, Key: key, ConfigVersion: 1, Coins: coins, Cosmetics: []string{}, Duplicates: map[string]int64{}, Medals: []string{}, MedalsSkipped: []string{}, At: now.Add(-time.Hour)}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `insert into reward_grants (account_id, bundle_id, revision, source_type, source_ref, idempotency_key, config_version, report, created_at)
		values ($1, $2, 1, 'shop', $3, $4, 1, $5, $6)`, account, report.BundleID, report.Source.Ref, key, encoded, report.At); err != nil {
		t.Fatal(err)
	}
}

func catalog(t *testing.T, _ *fixture) *shop.Catalog {
	t.Helper()
	raw, err := os.ReadFile("../../../docs/economy/examples/shop.json")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := shop.Parse(raw, shop.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func TestCoinPacksConvertOnceToTheCrystalsOfTheirTier(t *testing.T) {
	f := newFixture(t, purchase.DefaultConfig())
	ctx := context.Background()
	ayla, bora := f.account(t), f.account(t)
	legacyCoinPack(t, f, ayla, "tok-old-1", "coins_tier1", 500, purchase.RecordGranted)
	legacyCoinPack(t, f, ayla, "tok-old-2", "coins_tier2", 2750, purchase.RecordGranted)
	legacyCoinPack(t, f, bora, "tok-old-3", "coins_tier2", 2750, purchase.RecordGranted)
	legacyCoinPack(t, f, bora, "tok-old-4", "coins_tier1", 500, purchase.RecordRefunded)
	if _, err := f.wallet.Apply(ctx, wallet.Entry{Account: bora, Amount: -3000, Reason: wallet.ShopPurchase, Source: wallet.Source{Type: "shop", Ref: "offer.x"}, Key: "spend:1"}, now); err != nil {
		t.Fatal(err)
	}
	converted, err := purchase.ConvertCoinPacks(ctx, f.pool, f.wallet, catalog(t, f), now)
	if err != nil || len(converted) != 3 {
		t.Fatalf("converted %+v %v", converted, err)
	}
	if c := converted[2]; c.Account != bora || c.CoinsDebited != 250 || c.Crystals != 550 {
		t.Fatalf("bora's conversion %+v", c)
	}
	for account, want := range map[string][2]int64{ayla: {0, 100 + 550}, bora: {0, 550}} {
		if coins, crystals := balances(t, f, account); coins != want[0] || crystals != want[1] {
			t.Fatalf("%s coins %d crystals %d, want %v", account, coins, crystals, want)
		}
	}
	again, err := purchase.ConvertCoinPacks(ctx, f.pool, f.wallet, catalog(t, f), now.Add(time.Hour))
	if err != nil || len(again) != 0 {
		t.Fatalf("second run %+v %v", again, err)
	}
	if _, err := f.wallet.Apply(ctx, wallet.Entry{Account: bora, Amount: 400, Reason: wallet.MatchReward, Source: wallet.Source{Type: "match", Ref: "m1"}, Key: "match:m1"}, now); err != nil {
		t.Fatal(err)
	}
	if again, err := purchase.ConvertCoinPacks(ctx, f.pool, f.wallet, catalog(t, f), now.Add(2*time.Hour)); err != nil || len(again) != 0 {
		t.Fatalf("run after earning coins %+v %v", again, err)
	}
	if coins, _ := balances(t, f, bora); coins != 400 {
		t.Fatalf("earned coins taken by a later run: %d", coins)
	}
	if mismatches, err := f.wallet.Audit(ctx); err != nil || len(mismatches) != 0 {
		t.Fatalf("audit after the conversion %+v %v", mismatches, err)
	}
}

func TestARefundAfterTheConversionTakesTheCrystalsBackAndLeavesCoins(t *testing.T) {
	f := newFixture(t, purchase.DefaultConfig())
	ctx := context.Background()
	ayla := f.account(t)
	legacyCoinPack(t, f, ayla, "tok-old", "coins_tier2", 2750, purchase.RecordGranted)
	if _, err := purchase.ConvertCoinPacks(ctx, f.pool, f.wallet, catalog(t, f), now); err != nil {
		t.Fatal(err)
	}
	if _, err := f.wallet.Apply(ctx, wallet.Entry{Account: ayla, Amount: 70, Reason: wallet.MatchReward, Source: wallet.Source{Type: "match", Ref: "m1"}, Key: "match:m1"}, now); err != nil {
		t.Fatal(err)
	}
	refund := purchase.Notice{ID: "r-old", Kind: purchase.KindRefund, TransactionID: "tok-old"}
	if handled, err := f.store.Notice(ctx, purchase.PlatformGoogle, refund, now.Add(time.Hour)); err != nil || !handled {
		t.Fatalf("refund %v %v", handled, err)
	}
	if coins, crystals := balances(t, f, ayla); coins != 70 || crystals != 0 {
		t.Fatalf("after the refund coins %d crystals %d, want 70 and 0", coins, crystals)
	}
}

func TestAConversionWithoutACrystalPackOfTheTierChangesNothing(t *testing.T) {
	f := newFixture(t, purchase.DefaultConfig())
	ctx := context.Background()
	ayla := f.account(t)
	legacyCoinPack(t, f, ayla, "tok-old-1", "coins_tier1", 500, purchase.RecordGranted)
	legacyCoinPack(t, f, ayla, "tok-old-9", "coins_tier9", 99000, purchase.RecordGranted)
	if _, err := purchase.ConvertCoinPacks(ctx, f.pool, f.wallet, catalog(t, f), now); !errors.Is(err, purchase.ErrNoCrystalPack) {
		t.Fatalf("missing tier: %v", err)
	}
	if coins, crystals := balances(t, f, ayla); coins != 99500 || crystals != 0 {
		t.Fatalf("coins %d crystals %d after a refused run", coins, crystals)
	}
}

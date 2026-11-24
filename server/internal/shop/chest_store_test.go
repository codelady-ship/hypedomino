package shop_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/shop"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

type fixedRolls struct {
	mu     sync.Mutex
	values []uint64
}

func (r *fixedRolls) Uint64() uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	value := r.values[0]
	r.values = r.values[1:]
	return value
}

const chest = "offer.chest_avatar"

func chestOffer(t *testing.T, f *fixture, account string) shop.OfferView {
	t.Helper()
	view, err := f.store.View(context.Background(), account, now)
	if err != nil {
		t.Fatal(err)
	}
	chests := section(t, view, shop.SectionChests)
	if len(chests.Offers) != 1 || chests.Offers[0].ID != chest {
		t.Fatalf("chests %+v", chests)
	}
	return chests.Offers[0]
}

func TestAChestGrantsOnlyUnownedItemsUntilItsPoolIsOwned(t *testing.T) {
	f := newFixture(t, true)
	ctx := context.Background()
	account := f.account(t, 1500)
	f.store.WithRandom(&fixedRolls{values: []uint64{0, 0, 0, 0, 0}})
	before := chestOffer(t, f, account)
	if !before.Purchasable || before.PriceCoins != 500 || len(before.Cosmetics) != 2 || len(before.Odds) != 2 || before.Odds[0] != (shop.ChestOdds{Rarity: "common", Percent: 85}) || before.Odds[1] != (shop.ChestOdds{Rarity: "rare", Percent: 15}) || before.OpensToPity != 10 || before.Limit != 0 {
		t.Fatalf("chest before opening %+v", before)
	}
	first, err := f.store.Buy(ctx, account, chest, "c1", now)
	if err != nil || first.Balance != 1000 || first.Currency != wallet.Coin || first.Chest == nil || *first.Chest != (shop.ChestOpen{Item: "avatar.paper_kite", Rarity: "common", OpensToPity: 9}) {
		t.Fatalf("first open %+v %+v %v", first, first.Chest, err)
	}
	if first.Grant.Source.Type != shop.SourceChest || first.Grant.Source.Ref != chest || first.Grant.Key != "chest:c1" || len(first.Grant.Cosmetics) != 1 || first.Grant.BundleID != "bundle.chest_avatar" {
		t.Fatalf("first grant %+v", first.Grant)
	}
	again, err := f.store.Buy(ctx, account, chest, "c1", now)
	if err != nil || !again.Repeated || again.Balance != 1000 || *again.Chest != *first.Chest {
		t.Fatalf("repeated open %+v %v", again, err)
	}
	second, err := f.store.Buy(ctx, account, chest, "c2", now)
	if err != nil || second.Balance != 500 || *second.Chest != (shop.ChestOpen{Item: "avatar.moon_owl", Rarity: "rare", OpensToPity: 10}) {
		t.Fatalf("second open %+v %v", second, err)
	}
	if _, err := f.store.Buy(ctx, account, chest, "c3", now); !errors.Is(err, shop.ErrOwned) {
		t.Fatalf("open of a fully owned chest: %v", err)
	}
	after := chestOffer(t, f, account)
	if after.Purchasable || after.Reason != shop.ErrOwned.Error() || len(after.Owned) != 2 || len(after.Compensation) != 0 {
		t.Fatalf("chest after its pool is owned %+v", after)
	}
	if coins, err := f.wallet.Balance(ctx, account, wallet.Coin); err != nil || coins != 500 {
		t.Fatalf("balance %d %v", coins, err)
	}
	var opens, pity int
	var reason string
	if err := f.pool.QueryRow(ctx, `select count(*), sum(pity_after) from chest_opens where account_id = $1`, account).Scan(&opens, &pity); err != nil || opens != 2 || pity != 1 {
		t.Fatalf("chest opens %d pity %d %v", opens, pity, err)
	}
	if err := f.pool.QueryRow(ctx, `select reason from ledger_entries where account_id = $1 and idempotency_key = 'spend:c1'`, account).Scan(&reason); err != nil || reason != string(wallet.ChestPurchase) {
		t.Fatalf("spend reason %q %v", reason, err)
	}
}

func TestThePityCarriesOverBetweenOpensAndGivesARare(t *testing.T) {
	f := newFixture(t, true)
	ctx := context.Background()
	account := f.account(t, 500)
	if _, err := f.pool.Exec(ctx, `insert into chest_opens (account_id, offer_id, idempotency_key, roll, pity_before, pity_after, rarity, item_id, config_version, created_at)
		values ($1, 'offer.chest_older', 'chest:old', 3, 8, 9, 'common', 'avatar.old_common', 0, $2)`, account, now); err != nil {
		t.Fatal(err)
	}
	if offer := chestOffer(t, f, account); offer.OpensToPity != 1 {
		t.Fatalf("opens to pity %d", offer.OpensToPity)
	}
	f.store.WithRandom(&fixedRolls{values: []uint64{0, 0}})
	opened, err := f.store.Buy(ctx, account, chest, "c1", now)
	if err != nil || opened.Chest.Item != "avatar.moon_owl" || opened.Chest.OpensToPity != 10 {
		t.Fatalf("pity open %+v %v", opened, err)
	}
	var roll, before int
	if err := f.pool.QueryRow(ctx, `select roll, pity_before from chest_opens where idempotency_key = 'chest:c1'`).Scan(&roll, &before); err != nil || roll != 0 || before != 9 {
		t.Fatalf("recorded roll %d pity %d %v", roll, before, err)
	}
}

func TestAChestRefusesARequestIDSpentByAnotherAction(t *testing.T) {
	f := newFixture(t, true)
	ctx := context.Background()
	account := f.account(t, 1000)
	if _, err := f.wallet.Apply(ctx, wallet.Entry{Account: account, Amount: -40, Reason: wallet.PassLevelPurchase, Source: wallet.Source{Type: "pass", Ref: "pass.season_1"}, Key: "spend:r1"}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.Buy(ctx, account, chest, "r1", now); !errors.Is(err, shop.ErrRequest) {
		t.Fatalf("reused request id: %v", err)
	}
	if coins, _ := f.wallet.Balance(ctx, account, wallet.Coin); coins != 960 {
		t.Fatalf("balance %d after a refused open", coins)
	}
}

func TestParallelChestOpensNeverOverdraw(t *testing.T) {
	f := newFixture(t, true)
	ctx := context.Background()
	account := f.account(t, 700)
	var wg sync.WaitGroup
	results := make(chan error, 8)
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := f.store.Buy(ctx, account, chest, fmt.Sprintf("p%d", i), now)
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	opened := 0
	for err := range results {
		switch {
		case err == nil:
			opened++
		case !errors.Is(err, wallet.ErrInsufficient):
			t.Fatalf("parallel open: %v", err)
		}
	}
	coins, err := f.wallet.Balance(ctx, account, wallet.Coin)
	if err != nil || opened != 1 || coins != 200 {
		t.Fatalf("%d opened, balance %d %v", opened, coins, err)
	}
	if mismatches, err := f.wallet.Audit(ctx); err != nil || len(mismatches) != 0 {
		t.Fatalf("audit %+v %v", mismatches, err)
	}
}

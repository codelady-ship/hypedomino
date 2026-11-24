package shop_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/inventory"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/shop"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

var (
	now  = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	seed = wallet.Source{Type: "promo", Ref: "promo.test"}
)

type fixture struct {
	pool      *pgxpool.Pool
	wallet    *wallet.Store
	inventory *inventory.Store
	engine    *reward.Engine
	watcher   *snapshot.Watcher[*shop.Catalog]
	store     *shop.Store
}

func newFixture(t testing.TB, seedFile bool) *fixture {
	t.Helper()
	_, pool := dbtest.Fresh(t)
	catalog, err := cosmetics.Load("../../../docs/cosmetics/examples/catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{pool: pool, wallet: wallet.NewStore(pool)}
	f.inventory = inventory.NewStore(pool, catalog, inventory.DefaultConfig())
	f.engine = reward.NewEngine(pool, f.wallet, f.inventory, nil, nil, reward.DefaultConfig())
	f.watcher = snapshot.New(pool, shop.Domain, shop.Parser(shop.DefaultConfig()), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if seedFile {
		if err := f.watcher.SeedFile("../../../docs/economy/examples/shop.json"); err != nil {
			t.Fatal(err)
		}
	}
	f.store = shop.NewStore(pool, f.wallet, f.inventory, f.engine, f.watcher, shop.DefaultConfig())
	return f
}

func (f *fixture) account(t testing.TB, coins int64) string {
	t.Helper()
	var id string
	if err := f.pool.QueryRow(context.Background(), `insert into accounts (created_at) values ($1) returning id`, now.Add(-time.Hour)).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if coins > 0 {
		if _, err := f.wallet.Apply(context.Background(), wallet.Entry{Account: id, Amount: coins, Reason: wallet.Promo, Source: seed, Key: "seed"}, now); err != nil {
			t.Fatal(err)
		}
	}
	return id
}

func section(t *testing.T, view shop.View, name string) shop.SectionView {
	t.Helper()
	for _, s := range view.Sections {
		if s.Section == name {
			return s
		}
	}
	t.Fatalf("section %s missing from %+v", name, view)
	return shop.SectionView{}
}

func discountedTable(t *testing.T) *shop.Catalog {
	t.Helper()
	raw, err := os.ReadFile("../../../docs/economy/examples/shop.json")
	if err != nil {
		t.Fatal(err)
	}
	named := `"nameKey": "shop.offer.table_night_market"`
	catalog, err := shop.Parse([]byte(strings.Replace(string(raw), named, named+`, "referencePriceCoins": 3500`, 1)), shop.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func TestTheViewShowsEverySectionForTheAccount(t *testing.T) {
	f := newFixture(t, false)
	f.watcher.Seed(discountedTable(t), 0)
	ctx := context.Background()
	account := f.account(t, 0)
	view, err := f.store.View(ctx, account, now)
	if err != nil {
		t.Fatal(err)
	}
	if view.Version != 0 || len(view.Sections) != len(shop.Sections) {
		t.Fatalf("view %+v", view)
	}
	daily := section(t, view, shop.SectionDaily)
	if daily.EndsAt != "2026-10-31T00:00:00Z" || len(daily.Offers) != 1 || daily.Offers[0].ID != "offer.avatar_comet_fox" || !daily.Offers[0].Purchasable || daily.Offers[0].PriceCoins != 400 || daily.Offers[0].Limit != 1 || daily.Offers[0].Bought != 0 {
		t.Fatalf("daily %+v", daily)
	}
	weekly := section(t, view, shop.SectionWeekly)
	if weekly.EndsAt != "2026-11-02T00:00:00Z" || len(weekly.Offers) != 1 || weekly.Offers[0].ReferencePriceCoins != 3500 || weekly.Offers[0].EndsAt != "2027-01-01T00:00:00Z" {
		t.Fatalf("weekly %+v", weekly)
	}
	featured := section(t, view, shop.SectionFeatured)
	if len(featured.Offers) != 1 || featured.Offers[0].ProductID != "offer_starter" || !featured.Offers[0].Purchasable || featured.Offers[0].Crystals != 300 {
		t.Fatalf("featured %+v", featured)
	}
	if packs := section(t, view, shop.SectionCrystals); len(packs.Offers) != 5 || packs.Offers[4].ProductID != "crystals_tier5" || packs.Offers[4].Crystals != 7000 || packs.Offers[4].Limit != 0 {
		t.Fatalf("crystal packs %+v", packs)
	}
	if pass := section(t, view, shop.SectionPass); len(pass.Offers) != 1 || pass.Offers[0].Purchasable || pass.Offers[0].Reason != "offer.unsupported" {
		t.Fatalf("pass %+v", pass)
	}
	if event := section(t, view, shop.SectionEvent); len(event.Offers) != 0 {
		t.Fatalf("event %+v", event)
	}
	if _, err := f.store.View(ctx, account, time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
}

func TestACoinPurchaseIsAtomicIdempotentAndLimited(t *testing.T) {
	f := newFixture(t, true)
	ctx := context.Background()
	account := f.account(t, 5000)
	first, err := f.store.Buy(ctx, account, "offer.avatar_comet_fox", "r1", now)
	if err != nil {
		t.Fatal(err)
	}
	if first.Repeated || first.Currency != wallet.Coin || first.Balance != 4600 || first.Offer != "offer.avatar_comet_fox" || len(first.Grant.Cosmetics) != 1 || first.Grant.Cosmetics[0] != "avatar.comet_fox" || first.Grant.Reason != wallet.ShopPurchase {
		t.Fatalf("first purchase %+v", first)
	}
	if owned, err := f.inventory.Owns(ctx, account, "avatar.comet_fox"); err != nil || !owned {
		t.Fatalf("owned after purchase %v %v", owned, err)
	}
	repeat, err := f.store.Buy(ctx, account, "offer.table_night_market", "r1", now)
	if err != nil {
		t.Fatal(err)
	}
	if !repeat.Repeated || repeat.Balance != 4600 || repeat.Grant.ID != first.Grant.ID || repeat.Offer != "offer.avatar_comet_fox" {
		t.Fatalf("repeated purchase %+v", repeat)
	}
	if _, err := f.store.Buy(ctx, account, "offer.avatar_comet_fox", "r2", now); !errors.Is(err, shop.ErrLimit) {
		t.Fatalf("second purchase of a limited offer: %v", err)
	}
	if balance, _ := f.wallet.Balance(ctx, account, wallet.Coin); balance != 4600 {
		t.Fatalf("balance after refusals %d", balance)
	}
	view, err := f.store.View(ctx, account, now)
	if err != nil {
		t.Fatal(err)
	}
	fox := section(t, view, shop.SectionDaily).Offers[0]
	if fox.Purchasable || fox.Reason != "offer.limit_reached" || fox.Bought != 1 || len(fox.Owned) != 1 {
		t.Fatalf("fox after purchase %+v", fox)
	}
	starter := section(t, view, shop.SectionFeatured).Offers[0]
	if !starter.Purchasable || len(starter.Owned) != 1 || starter.Compensation["avatar.comet_fox"] != 300 {
		t.Fatalf("starter after purchase %+v", starter)
	}
	history, err := f.wallet.History(ctx, account, 10)
	if err != nil || len(history) != 2 || history[0].Amount != -400 || history[0].Key != "shop:r1:spend" || history[0].Reason != wallet.ShopPurchase {
		t.Fatalf("history %+v %v", history, err)
	}
}

func TestARefusedPurchaseWritesNothing(t *testing.T) {
	f := newFixture(t, true)
	ctx := context.Background()
	poor := f.account(t, 100)
	if _, err := f.store.Buy(ctx, poor, "offer.table_night_market", "r1", now); !errors.Is(err, wallet.ErrInsufficient) {
		t.Fatalf("purchase without coins: %v", err)
	}
	if owned, _ := f.inventory.Owns(ctx, poor, "table.night_market"); owned {
		t.Fatal("item granted without payment")
	}
	if _, err := f.engine.Get(ctx, poor, "shop:r1"); !errors.Is(err, reward.ErrUnknownGrant) {
		t.Fatalf("grant after a refused purchase: %v", err)
	}
	if balance, _ := f.wallet.Balance(ctx, poor, wallet.Coin); balance != 100 {
		t.Fatalf("balance %d", balance)
	}
	cases := map[string]struct {
		offer   string
		request string
		at      time.Time
		want    error
	}{
		"unknown":    {"offer.missing", "r2", now, shop.ErrUnknownOffer},
		"no request": {"offer.avatar_comet_fox", "", now, shop.ErrRequest},
		"long":       {"offer.avatar_comet_fox", strings.Repeat("x", shop.MaxRequestID+1), now, shop.ErrRequest},
		"closed":     {"offer.avatar_comet_fox", "r3", time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), shop.ErrUnavailable},
		"store only": {"offer.starter", "r4", now, shop.ErrStoreOnly},
		"pass":       {"offer.pass_premium", "r5", now, shop.ErrStoreOnly},
	}
	for name, c := range cases {
		if _, err := f.store.Buy(ctx, poor, c.offer, c.request, c.at); !errors.Is(err, c.want) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if _, err := f.store.Begin(ctx, poor, "offer.pass_premium", now); !errors.Is(err, shop.ErrUnsupported) {
		t.Fatalf("pass checkout: %v", err)
	}
}

func TestAStoreOfferStartsACheckoutForEligibleAccountsOnly(t *testing.T) {
	f := newFixture(t, true)
	ctx := context.Background()
	account := f.account(t, 0)
	checkout, err := f.store.Begin(ctx, account, "offer.starter", now)
	if err != nil {
		t.Fatal(err)
	}
	if checkout.Offer != "offer.starter" || checkout.ProductID != "offer_starter" || checkout.AccountToken != shop.AccountToken(account) || checkout.AccountToken == account || len(checkout.AccountToken) != 36 {
		t.Fatalf("checkout %+v", checkout)
	}
	if _, err := f.store.Begin(ctx, account, "offer.avatar_comet_fox", now); !errors.Is(err, shop.ErrInGameOnly) {
		t.Fatalf("checkout of a coin offer: %v", err)
	}
	if _, err := f.store.Begin(ctx, account, "offer.starter", now.Add(169*time.Hour)); !errors.Is(err, shop.ErrNotEligible) {
		t.Fatalf("checkout after the starter window: %v", err)
	}
	if _, err := f.wallet.Apply(ctx, wallet.Entry{Account: account, Amount: 500, Reason: wallet.Purchase, Source: wallet.Source{Type: "purchase", Ref: "crystals_tier1"}, Key: "tx-1"}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.Begin(ctx, account, "offer.starter", now); !errors.Is(err, shop.ErrNotEligible) {
		t.Fatalf("checkout after a purchase: %v", err)
	}
	view, err := f.store.View(ctx, account, now)
	if err != nil {
		t.Fatal(err)
	}
	if starter := section(t, view, shop.SectionFeatured).Offers[0]; starter.Purchasable || starter.Reason != "offer.not_eligible" {
		t.Fatalf("starter after a purchase %+v", starter)
	}
	if _, err := f.store.Begin(ctx, account, "offer.crystals_tier1", now); err != nil {
		t.Fatalf("coin pack checkout: %v", err)
	}
}

func TestAPublishedCatalogIsVisibleWithoutARestart(t *testing.T) {
	f := newFixture(t, false)
	ctx := context.Background()
	account := f.account(t, 1000)
	if _, err := f.store.View(ctx, account, now); !errors.Is(err, shop.ErrNoCatalog) {
		t.Fatalf("view before any catalog: %v", err)
	}
	if _, err := f.store.Buy(ctx, account, "offer.avatar_comet_fox", "r1", now); !errors.Is(err, shop.ErrNoCatalog) {
		t.Fatalf("purchase before any catalog: %v", err)
	}
	raw, err := os.ReadFile("../../../docs/economy/examples/shop.json")
	if err != nil {
		t.Fatal(err)
	}
	version, err := snapshot.Publish(ctx, f.pool, shop.Domain, raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.watcher.Load(ctx); err != nil {
		t.Fatal(err)
	}
	view, err := f.store.View(ctx, account, now)
	if err != nil || view.Version != version || section(t, view, shop.SectionDaily).Offers[0].PriceCoins != 400 {
		t.Fatalf("view after the first publish %+v %v", view, err)
	}
	cheaper := strings.Replace(string(raw), `"priceCoins": 400`, `"priceCoins": 250`, 1)
	next, err := snapshot.Publish(ctx, f.pool, shop.Domain, []byte(cheaper))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.watcher.Load(ctx); err != nil {
		t.Fatal(err)
	}
	view, err = f.store.View(ctx, account, now)
	if err != nil || view.Version != next || section(t, view, shop.SectionDaily).Offers[0].PriceCoins != 250 {
		t.Fatalf("view after the second publish %+v %v", view, err)
	}
	purchase, err := f.store.Buy(ctx, account, "offer.avatar_comet_fox", "r1", now)
	if err != nil || purchase.Balance != 750 || purchase.Grant.ConfigVersion != int(next) {
		t.Fatalf("purchase at the new price %+v %v", purchase, err)
	}
	if _, err := snapshot.Publish(ctx, f.pool, shop.Domain, []byte(`{"schema":1,"offers":[{"id":"bad"}]}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.watcher.Load(ctx); !errors.Is(err, shop.ErrInvalid) {
		t.Fatalf("invalid publish: %v", err)
	}
	if view, err := f.store.View(ctx, account, now); err != nil || view.Version != next {
		t.Fatalf("view after an invalid publish %+v %v", view, err)
	}
}

func TestTheViewShowsEachAccountItsVariantOfAnOffer(t *testing.T) {
	f := newFixture(t, false)
	raw, err := os.ReadFile("../../../docs/economy/examples/shop.json")
	if err != nil {
		t.Fatal(err)
	}
	named := `"nameKey": "shop.offer.crystals_tier5"`
	catalog, err := shop.Parse([]byte(strings.Replace(string(raw), named, named+`, "variants": [{"id": "top", "nameKey": "shop.offer.crystals_tier5_top", "position": 1}]`, 1)), shop.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	f.watcher.Seed(catalog, 0)
	offer, _ := catalog.Offer("offer.crystals_tier5")
	seen := map[bool]bool{}
	for attempt := 0; len(seen) < 2; attempt++ {
		if attempt == 64 {
			t.Fatalf("64 accounts saw only %v", seen)
		}
		account := f.account(t, 0)
		_, inVariant := offer.VariantFor(account)
		view, err := f.store.View(context.Background(), account, now)
		if err != nil {
			t.Fatal(err)
		}
		packs := section(t, view, shop.SectionCrystals).Offers
		want := shop.OfferView{ID: "offer.crystals_tier5", NameKey: "shop.offer.crystals_tier5"}
		at := 4
		if inVariant {
			want.NameKey, want.Variant, at = "shop.offer.crystals_tier5_top", "top", 0
		}
		if len(packs) != 5 || packs[at].ID != want.ID || packs[at].NameKey != want.NameKey || packs[at].Variant != want.Variant || packs[at].Crystals != 7000 || packs[at].ProductID != "crystals_tier5" {
			t.Fatalf("variant %v packs %+v", inVariant, packs)
		}
		for _, other := range view.Sections {
			for _, shown := range other.Offers {
				if shown.ID != want.ID && shown.Variant != "" {
					t.Fatalf("offer %s without variants shows variant %q", shown.ID, shown.Variant)
				}
			}
		}
		seen[inVariant] = true
	}
}

func TestACrystalPricedOfferSpendsCrystalsOnly(t *testing.T) {
	f := newFixture(t, false)
	ctx := context.Background()
	raw, err := os.ReadFile("../../../docs/economy/examples/shop.json")
	if err != nil {
		t.Fatal(err)
	}
	crystalPriced := strings.Replace(string(raw), `"priceCoins": 400`, `"priceCrystals": 80`, 1)
	if _, err := snapshot.Publish(ctx, f.pool, shop.Domain, []byte(crystalPriced)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.watcher.Load(ctx); err != nil {
		t.Fatal(err)
	}
	account := f.account(t, 5000)
	if _, err := f.store.Buy(ctx, account, "offer.avatar_comet_fox", "r1", now); !errors.Is(err, wallet.ErrInsufficient) {
		t.Fatalf("crystal offer bought with coins only: %v", err)
	}
	if _, err := f.wallet.Apply(ctx, wallet.Entry{Account: account, Currency: wallet.Crystal, Amount: 100, Reason: wallet.Purchase, Source: wallet.Source{Type: "store", Ref: "t1"}, Key: "purchase:google:t1"}, now); err != nil {
		t.Fatal(err)
	}
	bought, err := f.store.Buy(ctx, account, "offer.avatar_comet_fox", "r2", now)
	if err != nil || bought.Currency != wallet.Crystal || bought.Balance != 20 || len(bought.Grant.Cosmetics) != 1 {
		t.Fatalf("crystal purchase %+v %v", bought, err)
	}
	if coins, _ := f.wallet.Balance(ctx, account, wallet.Coin); coins != 5000 {
		t.Fatalf("coins after a crystal purchase %d", coins)
	}
	view, err := f.store.View(ctx, account, now)
	if err != nil || section(t, view, shop.SectionDaily).Offers[0].PriceCrystals != 80 {
		t.Fatalf("view of the crystal price %+v %v", view, err)
	}
}

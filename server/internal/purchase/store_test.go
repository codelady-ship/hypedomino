package purchase_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/inventory"
	"github.com/Alievs-corp/hypedomino/server/internal/purchase"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/shop"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

var now = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

type fakeValidator struct {
	receipts map[string]purchase.Receipt
	errs     map[string]error
	finished []string
	stuck    bool
}

func (f *fakeValidator) Verify(_ context.Context, product, token string) (purchase.Receipt, error) {
	if err, ok := f.errs[token]; ok {
		return purchase.Receipt{}, err
	}
	receipt, ok := f.receipts[token]
	if !ok {
		return purchase.Receipt{}, purchase.ErrNotFound
	}
	if receipt.ProductID == "" {
		receipt.ProductID = product
	}
	receipt.TransactionID = token
	return receipt, nil
}

func (f *fakeValidator) Finish(_ context.Context, receipt purchase.Receipt) error {
	f.finished = append(f.finished, receipt.TransactionID)
	if f.stuck {
		return purchase.ErrStore
	}
	return nil
}

type fixture struct {
	pool      *pgxpool.Pool
	wallet    *wallet.Store
	inventory *inventory.Store
	engine    *reward.Engine
	google    *fakeValidator
	shop      *shop.Store
	store     *purchase.Store
}

func newFixture(t *testing.T, config purchase.Config) *fixture {
	t.Helper()
	_, pool := dbtest.Fresh(t)
	catalog, err := cosmetics.Load("../../../docs/cosmetics/examples/catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	f := &fixture{pool: pool, wallet: wallet.NewStore(pool), google: &fakeValidator{receipts: map[string]purchase.Receipt{}, errs: map[string]error{}}}
	f.inventory = inventory.NewStore(pool, catalog, inventory.DefaultConfig())
	f.engine = reward.NewEngine(pool, f.wallet, f.inventory, nil, nil, reward.DefaultConfig())
	offers := snapshot.New(pool, shop.Domain, shop.Parser(shop.DefaultConfig()), logger)
	if err := offers.SeedFile("../../../docs/economy/examples/shop.json"); err != nil {
		t.Fatal(err)
	}
	f.shop = shop.NewStore(pool, f.wallet, f.inventory, f.engine, offers, shop.DefaultConfig())
	f.store = purchase.NewStore(pool, f.wallet, f.engine, offers, map[purchase.Platform]purchase.Validator{purchase.PlatformGoogle: f.google}, config, logger)
	f.shop.Guard(f.store)
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

func (f *fixture) receipt(token, product, account string, state purchase.State) {
	f.google.receipts[token] = purchase.Receipt{Platform: purchase.PlatformGoogle, OrderID: "GPA." + token, ProductID: product, AccountToken: shop.AccountToken(account), State: state, Environment: purchase.EnvironmentProduction, PurchasedAt: now}
}

func (f *fixture) state(t *testing.T, token string) string {
	t.Helper()
	var state string
	if err := f.pool.QueryRow(context.Background(), `select state from purchases where platform = 'google' and transaction_id = $1`, token).Scan(&state); err != nil {
		t.Fatal(err)
	}
	return state
}

func (f *fixture) events(t *testing.T, name string) int {
	t.Helper()
	var count int
	if err := f.pool.QueryRow(context.Background(), `select count(*) from outbox where topic = $1 and payload->>'event' = $2`, purchase.Topic, name).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func TestAValidatedPurchaseIsGrantedOnceWithTheFirstPurchaseBonus(t *testing.T) {
	f := newFixture(t, purchase.DefaultConfig())
	ctx := context.Background()
	ayla := f.account(t)
	f.receipt("tok-1", "offer_starter", ayla, purchase.StatePurchased)
	result, err := f.store.Complete(ctx, ayla, "offer.starter", purchase.PlatformGoogle, "tok-1", now)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != purchase.RecordGranted || result.Repeated || !result.Finished || result.Balance != 2000 || result.Grant == nil || result.Grant.Coins != 1500 || result.Grant.Reason != wallet.Purchase || result.Bonus == nil || result.Bonus.Coins != 500 || len(result.Bonus.Cosmetics) != 1 {
		t.Fatalf("result %+v", result)
	}
	for _, id := range []string{"avatar.comet_fox", "frame.season_1_top"} {
		if owned, err := f.inventory.Owns(ctx, ayla, id); err != nil || !owned {
			t.Fatalf("%s owned %v %v", id, owned, err)
		}
	}
	if f.state(t, "tok-1") != purchase.RecordGranted || f.events(t, purchase.EventGranted) != 1 {
		t.Fatal("purchase not recorded as granted with its event")
	}
	repeat, err := f.store.Complete(ctx, ayla, "offer.starter", purchase.PlatformGoogle, "tok-1", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if !repeat.Repeated || repeat.Balance != 2000 || repeat.Grant.ID != result.Grant.ID || repeat.Bonus.ID != result.Bonus.ID || !repeat.Finished {
		t.Fatalf("repeat %+v", repeat)
	}
	f.receipt("tok-2", "coins_tier1", ayla, purchase.StatePurchased)
	second, err := f.store.Complete(ctx, ayla, "offer.coins_tier1", purchase.PlatformGoogle, "tok-2", now)
	if err != nil {
		t.Fatal(err)
	}
	if second.Bonus != nil || second.Balance != 2500 || second.Grant.Coins != 500 {
		t.Fatalf("second purchase %+v", second)
	}
	if len(f.google.finished) != 3 {
		t.Fatalf("finished %v", f.google.finished)
	}
	history, err := f.wallet.History(ctx, ayla, 10)
	if err != nil || len(history) != 3 || history[0].Key != "purchase:google:tok-2" || history[1].Key != "purchase:google:tok-1:first" {
		t.Fatalf("history %+v %v", history, err)
	}
	view, err := f.shop.View(ctx, ayla, now)
	if err != nil {
		t.Fatal(err)
	}
	if starter := view.Sections[0].Offers[0]; starter.Purchasable || starter.Reason != "offer.limit_reached" {
		t.Fatalf("starter after purchase %+v", starter)
	}
	f.google.stuck = true
	f.receipt("tok-3", "coins_tier2", ayla, purchase.StatePurchased)
	if third, err := f.store.Complete(ctx, ayla, "offer.coins_tier2", purchase.PlatformGoogle, "tok-3", now); err != nil || third.Finished || third.State != purchase.RecordGranted {
		t.Fatalf("purchase with a stuck consume %+v %v", third, err)
	}
}

func TestARefusedPurchaseGrantsNothing(t *testing.T) {
	f := newFixture(t, purchase.DefaultConfig())
	ctx := context.Background()
	ayla, bora := f.account(t), f.account(t)
	f.receipt("tok-1", "offer_starter", ayla, purchase.StatePurchased)
	if _, err := f.store.Complete(ctx, ayla, "offer.starter", purchase.PlatformGoogle, "tok-1", now); err != nil {
		t.Fatal(err)
	}
	f.receipt("tok-other", "coins_tier1", ayla, purchase.StatePurchased)
	f.receipt("tok-1", "offer_starter", bora, purchase.StatePurchased)
	f.receipt("tok-wrong-product", "coins_tier2", bora, purchase.StatePurchased)
	f.receipt("tok-cancelled", "coins_tier1", bora, purchase.StateCancelled)
	f.google.errs["tok-down"] = purchase.ErrStore
	f.google.errs["tok-sandbox"] = purchase.ErrEnvironment
	cases := map[string]struct {
		offer    string
		platform purchase.Platform
		token    string
		want     error
	}{
		"reused by another account": {"offer.starter", purchase.PlatformGoogle, "tok-1", purchase.ErrReused},
		"account mismatch":          {"offer.coins_tier1", purchase.PlatformGoogle, "tok-other", purchase.ErrAccount},
		"product mismatch":          {"offer.coins_tier1", purchase.PlatformGoogle, "tok-wrong-product", purchase.ErrProduct},
		"cancelled":                 {"offer.coins_tier1", purchase.PlatformGoogle, "tok-cancelled", purchase.ErrCancelled},
		"unknown receipt":           {"offer.coins_tier1", purchase.PlatformGoogle, "tok-nope", purchase.ErrNotFound},
		"store down":                {"offer.coins_tier1", purchase.PlatformGoogle, "tok-down", purchase.ErrStore},
		"sandbox":                   {"offer.coins_tier1", purchase.PlatformGoogle, "tok-sandbox", purchase.ErrEnvironment},
		"platform":                  {"offer.coins_tier1", purchase.PlatformApple, "tok-other", purchase.ErrPlatform},
		"unknown offer":             {"offer.missing", purchase.PlatformGoogle, "tok-other", shop.ErrUnknownOffer},
		"coin offer":                {"offer.avatar_comet_fox", purchase.PlatformGoogle, "tok-other", shop.ErrCoinsOnly},
		"pass offer":                {"offer.pass_premium", purchase.PlatformGoogle, "tok-other", shop.ErrUnsupported},
	}
	for name, c := range cases {
		if _, err := f.store.Complete(ctx, bora, c.offer, c.platform, c.token, now); !errors.Is(err, c.want) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	var rows int
	if err := f.pool.QueryRow(ctx, `select count(*) from purchases where account_id = $1`, bora).Scan(&rows); err != nil || rows != 0 {
		t.Fatalf("rows for the refused account %d %v", rows, err)
	}
	if balance, _ := f.wallet.Balance(ctx, bora); balance != 0 {
		t.Fatalf("balance %d", balance)
	}
	if f.events(t, purchase.EventReuse) != 1 || f.events(t, purchase.EventAccountMatch) != 1 {
		t.Fatal("hard evidence was not published")
	}
}

func TestAPendingPurchaseIsGrantedByTheStoreNotification(t *testing.T) {
	f := newFixture(t, purchase.DefaultConfig())
	ctx := context.Background()
	ayla := f.account(t)
	f.receipt("tok-p", "coins_tier1", ayla, purchase.StatePending)
	result, err := f.store.Complete(ctx, ayla, "offer.coins_tier1", purchase.PlatformGoogle, "tok-p", now)
	if err != nil || result.State != purchase.RecordPending || result.Grant != nil {
		t.Fatalf("pending result %+v %v", result, err)
	}
	if again, err := f.store.Complete(ctx, ayla, "offer.coins_tier1", purchase.PlatformGoogle, "tok-p", now); err != nil || again.State != purchase.RecordPending {
		t.Fatalf("pending again %+v %v", again, err)
	}
	if balance, _ := f.wallet.Balance(ctx, ayla); balance != 0 || f.state(t, "tok-p") != purchase.RecordPending || len(f.google.finished) != 0 {
		t.Fatalf("pending purchase granted: balance %d state %s", balance, f.state(t, "tok-p"))
	}
	notice := purchase.Notice{ID: "m-1", Kind: purchase.KindPurchased, TransactionID: "tok-p", ProductID: "coins_tier1"}
	if handled, err := f.store.Notice(ctx, purchase.PlatformGoogle, notice, now); err != nil || handled {
		t.Fatalf("notice while the store still says pending %v %v", handled, err)
	}
	f.receipt("tok-p", "coins_tier1", ayla, purchase.StatePurchased)
	notice.ID = "m-2"
	if handled, err := f.store.Notice(ctx, purchase.PlatformGoogle, notice, now); err != nil || !handled {
		t.Fatalf("purchased notice %v %v", handled, err)
	}
	if balance, _ := f.wallet.Balance(ctx, ayla); balance != 1000 || f.state(t, "tok-p") != purchase.RecordGranted {
		t.Fatalf("after the notice: balance %d state %s", balance, f.state(t, "tok-p"))
	}
	if handled, err := f.store.Notice(ctx, purchase.PlatformGoogle, notice, now); err != nil || handled {
		t.Fatalf("repeated notice %v %v", handled, err)
	}
	if handled, err := f.store.Notice(ctx, purchase.PlatformGoogle, purchase.Notice{ID: "m-3", Kind: purchase.KindRefund, TransactionID: "tok-unknown"}, now); err != nil || handled {
		t.Fatalf("notice for an unknown purchase %v %v", handled, err)
	}
	if _, err := f.store.Notice(ctx, purchase.PlatformGoogle, purchase.Notice{Kind: purchase.KindRefund, TransactionID: "tok-p"}, now); !errors.Is(err, purchase.ErrMalformed) {
		t.Fatalf("notice without an id: %v", err)
	}
	f.receipt("tok-c", "coins_tier1", ayla, purchase.StatePending)
	if _, err := f.store.Complete(ctx, ayla, "offer.coins_tier1", purchase.PlatformGoogle, "tok-c", now); err != nil {
		t.Fatal(err)
	}
	f.receipt("tok-c", "coins_tier1", ayla, purchase.StateCancelled)
	if handled, err := f.store.Notice(ctx, purchase.PlatformGoogle, purchase.Notice{ID: "m-4", Kind: purchase.KindPurchased, TransactionID: "tok-c"}, now); err != nil || !handled || f.state(t, "tok-c") != purchase.RecordCancelled {
		t.Fatalf("cancelled pending purchase %v %v %s", handled, err, f.state(t, "tok-c"))
	}
	if completed, err := f.store.Complete(ctx, ayla, "offer.coins_tier1", purchase.PlatformGoogle, "tok-p", now); err != nil || !completed.Repeated || completed.Balance != 1000 || completed.Bonus == nil {
		t.Fatalf("complete after the notice granted %+v %v", completed, err)
	}
}

func TestARefundRevokesTheGoodsAndAReversalRestoresThem(t *testing.T) {
	f := newFixture(t, purchase.Config{RefundsBeforePurchaseBlock: 2, RefundWindowDays: 30})
	ctx := context.Background()
	ayla := f.account(t)
	f.receipt("tok-1", "offer_starter", ayla, purchase.StatePurchased)
	if _, err := f.store.Complete(ctx, ayla, "offer.starter", purchase.PlatformGoogle, "tok-1", now); err != nil {
		t.Fatal(err)
	}
	if _, err := f.wallet.Apply(ctx, wallet.Entry{Account: ayla, Amount: -1900, Reason: wallet.ShopPurchase, Source: wallet.Source{Type: "shop", Ref: "offer.x"}, Key: "spend:1"}, now); err != nil {
		t.Fatal(err)
	}
	refund := purchase.Notice{ID: "r-1", Kind: purchase.KindRefund, TransactionID: "tok-1"}
	if handled, err := f.store.Notice(ctx, purchase.PlatformGoogle, refund, now.Add(time.Hour)); err != nil || !handled {
		t.Fatalf("refund %v %v", handled, err)
	}
	if balance, _ := f.wallet.Balance(ctx, ayla); balance != -1900 {
		t.Fatalf("balance after the refund %d", balance)
	}
	for _, id := range []string{"avatar.comet_fox", "frame.season_1_top"} {
		if owned, _ := f.inventory.Owns(ctx, ayla, id); owned {
			t.Fatalf("%s still owned after the refund", id)
		}
	}
	if f.state(t, "tok-1") != purchase.RecordRefunded || f.events(t, purchase.EventRefunded) != 1 {
		t.Fatal("refund not recorded")
	}
	if handled, err := f.store.Notice(ctx, purchase.PlatformGoogle, purchase.Notice{ID: "r-2", Kind: purchase.KindRefund, TransactionID: "tok-1"}, now); err != nil || handled {
		t.Fatalf("second refund of the same purchase %v %v", handled, err)
	}
	if _, err := f.wallet.Apply(ctx, wallet.Entry{Account: ayla, Amount: -1, Reason: wallet.ShopPurchase, Source: wallet.Source{Type: "shop", Ref: "offer.x"}, Key: "spend:2"}, now); !errors.Is(err, wallet.ErrInsufficient) {
		t.Fatalf("spend from a negative balance: %v", err)
	}
	if blocked, err := f.store.Blocked(ctx, ayla, now.Add(2*time.Hour)); err != nil || blocked {
		t.Fatalf("blocked after one refund %v %v", blocked, err)
	}
	reversal := purchase.Notice{ID: "v-1", Kind: purchase.KindReversal, TransactionID: "tok-1"}
	if handled, err := f.store.Notice(ctx, purchase.PlatformGoogle, reversal, now.Add(2*time.Hour)); err != nil || !handled {
		t.Fatalf("reversal %v %v", handled, err)
	}
	if balance, _ := f.wallet.Balance(ctx, ayla); balance != -400 {
		t.Fatalf("balance after the reversal %d", balance)
	}
	if owned, _ := f.inventory.Owns(ctx, ayla, "avatar.comet_fox"); !owned || f.state(t, "tok-1") != purchase.RecordGranted || f.events(t, purchase.EventReversed) != 1 {
		t.Fatal("reversal did not restore the purchase")
	}
	if completed, err := f.store.Complete(ctx, ayla, "offer.starter", purchase.PlatformGoogle, "tok-1", now); err != nil || !completed.Repeated || completed.Grant.Key != "purchase:google:tok-1:reversal:v-1" {
		t.Fatalf("complete after the reversal %+v %v", completed, err)
	}
	f.receipt("tok-2", "coins_tier1", ayla, purchase.StatePurchased)
	f.receipt("tok-3", "coins_tier1", ayla, purchase.StatePurchased)
	for _, token := range []string{"tok-2", "tok-3"} {
		if _, err := f.store.Complete(ctx, ayla, "offer.coins_tier1", purchase.PlatformGoogle, token, now); err != nil {
			t.Fatal(err)
		}
		if _, err := f.store.Notice(ctx, purchase.PlatformGoogle, purchase.Notice{ID: "r-" + token, Kind: purchase.KindRefund, TransactionID: token}, now.Add(3*time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	if blocked, err := f.store.Blocked(ctx, ayla, now.Add(4*time.Hour)); err != nil || !blocked {
		t.Fatalf("blocked after two refunds %v %v", blocked, err)
	}
	f.receipt("tok-4", "coins_tier1", ayla, purchase.StatePurchased)
	if _, err := f.store.Complete(ctx, ayla, "offer.coins_tier1", purchase.PlatformGoogle, "tok-4", now.Add(4*time.Hour)); !errors.Is(err, purchase.ErrBlocked) {
		t.Fatalf("purchase while blocked: %v", err)
	}
	if _, err := f.shop.Begin(ctx, ayla, "offer.coins_tier1", now.Add(4*time.Hour)); !errors.Is(err, shop.ErrBlocked) {
		t.Fatalf("checkout while blocked: %v", err)
	}
	view, err := f.shop.View(ctx, ayla, now.Add(4*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if pack := view.Sections[5].Offers[0]; pack.Purchasable || pack.Reason != "purchase.blocked" {
		t.Fatalf("coin pack while blocked %+v", pack)
	}
	if blocked, err := f.store.Blocked(ctx, ayla, now.Add(31*24*time.Hour)); err != nil || blocked {
		t.Fatalf("blocked after the window %v %v", blocked, err)
	}
}

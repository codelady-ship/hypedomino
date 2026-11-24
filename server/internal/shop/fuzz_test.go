package shop_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/shop"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

func FuzzTamperedBuyRequestsNeverRaiseABalance(f *testing.F) {
	for _, seed := range []string{
		`{"offer":"offer.avatar_comet_fox","request_id":"r1"}`,
		`{"offer":"offer.avatar_comet_fox","request_id":"r2","price_coins":0,"coins":100000}`,
		`{"offer":"offer.chest_avatar","request_id":"r3","balance":999999,"price":-500}`,
		`{"offer":"offer.chest_avatar","request_id":"r3"}`,
		`{"offer":"offer.table_night_market","request_id":"r4","currency":"crystal","price_crystals":1}`,
		`{"offer":"offer.crystals_tier5","request_id":"r5","crystals":7000}`,
		`{"offer":"offer.starter","request_id":"r6","productId":"","priceCoins":1}`,
		`{"offer":"offer.missing","request_id":"r7","reward":{"coins":5}}`,
		`{"offer":"offer.chest_avatar","request_id":"r8","bundle":{"cosmetics":["table.night_market"]}}`,
		`{"offer":"offer.avatar_comet_fox","request_id":""}`,
		`{"offer":["offer.chest_avatar"],"request_id":7}`,
		`{"offer":"offer.chest_avatar","request_id":"r9","offer":"offer.table_night_market"}`,
		`not json`,
	} {
		f.Add(seed)
	}
	fx := newFixture(f, true)
	ctx := context.Background()
	account := fx.account(f, 5000)
	if _, err := fx.wallet.Apply(ctx, wallet.Entry{Account: account, Currency: wallet.Crystal, Amount: 100, Reason: wallet.Purchase, Source: wallet.Source{Type: "purchase", Ref: "seed"}, Key: "purchase:seed"}, now); err != nil {
		f.Fatal(err)
	}
	handler := shop.NewHandler(fakeVerifier{}, fx.store, slog.New(slog.NewTextHandler(io.Discard, nil)), func() time.Time { return now })
	catalogPrices := map[int64]bool{0: true, 400: true, 500: true, 3000: true}
	balances := func(t *testing.T) (int64, int64) {
		coins, err := fx.wallet.Balance(ctx, account, wallet.Coin)
		if err != nil {
			t.Fatal(err)
		}
		crystals, err := fx.wallet.Balance(ctx, account, wallet.Crystal)
		if err != nil {
			t.Fatal(err)
		}
		return coins, crystals
	}
	f.Fuzz(func(t *testing.T, body string) {
		coins, crystals := balances(t)
		request := httptest.NewRequest(http.MethodPost, "/shop/buy", strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer access:"+account)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		coinsAfter, crystalsAfter := balances(t)
		if coinsAfter > coins || crystalsAfter != crystals || !catalogPrices[coins-coinsAfter] {
			t.Fatalf("%s moved coins %d to %d and crystals %d to %d", body, coins, coinsAfter, crystals, crystalsAfter)
		}
		if recorder.Code == http.StatusOK {
			var purchase shop.Purchase
			if err := json.Unmarshal(recorder.Body.Bytes(), &purchase); err != nil || purchase.Balance != coinsAfter {
				t.Fatalf("%s answered balance %d, stored %d: %v", body, purchase.Balance, coinsAfter, err)
			}
		}
		if mismatches, err := fx.wallet.Audit(ctx); err != nil || len(mismatches) != 0 {
			t.Fatalf("%s left the ledger out of balance: %+v %v", body, mismatches, err)
		}
	})
}

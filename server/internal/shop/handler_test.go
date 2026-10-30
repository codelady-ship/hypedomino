package shop_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/inventory"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/shop"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

type fakeVerifier struct{}

func (fakeVerifier) Verify(access string) (token.Claims, error) {
	switch {
	case access == "expired":
		return token.Claims{}, token.ErrExpired
	case strings.HasPrefix(access, "access:"):
		return token.Claims{Account: strings.TrimPrefix(access, "access:")}, nil
	}
	return token.Claims{}, token.ErrSignature
}

type fakeShop struct {
	calls []string
}

func (f *fakeShop) View(_ context.Context, account string, at time.Time) (shop.View, error) {
	f.calls = append(f.calls, "view "+account+" "+at.Format(time.RFC3339))
	if account == "broken" {
		return shop.View{}, errors.New("database down")
	}
	if account == "empty" {
		return shop.View{}, shop.ErrNoCatalog
	}
	return shop.View{Version: 7, Sections: []shop.SectionView{{Section: shop.SectionCoins, Offers: []shop.OfferView{{ID: "offer.coins_tier1", ProductID: "coins_tier1", Purchasable: true}}}}}, nil
}

func (f *fakeShop) Buy(_ context.Context, account, offer, requestID string, _ time.Time) (shop.Purchase, error) {
	f.calls = append(f.calls, "buy "+account+" "+offer+" "+requestID)
	switch offer {
	case "offer.missing":
		return shop.Purchase{}, shop.ErrUnknownOffer
	case "offer.closed":
		return shop.Purchase{}, shop.ErrUnavailable
	case "offer.limited":
		return shop.Purchase{}, shop.ErrLimit
	case "offer.owned":
		return shop.Purchase{}, shop.ErrOwned
	case "offer.store":
		return shop.Purchase{}, shop.ErrStoreOnly
	case "offer.pass":
		return shop.Purchase{}, shop.ErrUnsupported
	case "offer.young":
		return shop.Purchase{}, shop.ErrNotEligible
	case "offer.pricey":
		return shop.Purchase{}, wallet.ErrInsufficient
	case "offer.unlisted":
		return shop.Purchase{}, inventory.ErrSourceNotListed
	case "offer.broken":
		return shop.Purchase{}, errors.New("database down")
	}
	if requestID == "" {
		return shop.Purchase{}, shop.ErrRequest
	}
	return shop.Purchase{Offer: offer, Balance: 600, Grant: reward.Report{ID: 3, Cosmetics: []string{"avatar.comet_fox"}}, Repeated: requestID == "again"}, nil
}

func (f *fakeShop) Begin(_ context.Context, account, offer string, _ time.Time) (shop.Checkout, error) {
	f.calls = append(f.calls, "begin "+account+" "+offer)
	if offer == "offer.coins" {
		return shop.Checkout{}, shop.ErrCoinsOnly
	}
	return shop.Checkout{Offer: offer, ProductID: "offer_starter", AccountToken: shop.AccountToken(account)}, nil
}

func serve(t *testing.T, handler http.Handler, method, path, access, body string) (int, map[string]any) {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if access != "" {
		request.Header.Set("Authorization", "Bearer "+access)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	var decoded map[string]any
	if strings.HasPrefix(recorder.Header().Get("Content-Type"), "application/json") {
		if err := json.Unmarshal(recorder.Body.Bytes(), &decoded); err != nil {
			t.Fatalf("%s %s: %v: %s", method, path, err, recorder.Body.String())
		}
	}
	if recorder.Code != http.StatusNotFound && recorder.Code != http.StatusMethodNotAllowed && recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("%s %s: cache control %q", method, path, recorder.Header().Get("Cache-Control"))
	}
	return recorder.Code, decoded
}

func TestTheHandlerAnswersViewPurchaseAndCheckout(t *testing.T) {
	fake := &fakeShop{}
	at := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	handler := shop.NewHandler(fakeVerifier{}, fake, slog.New(slog.NewTextHandler(io.Discard, nil)), func() time.Time { return at })
	if code, body := serve(t, handler, http.MethodGet, "/shop", "access:a1", ""); code != http.StatusOK || body["version"].(float64) != 7 || len(body["sections"].([]any)) != 1 {
		t.Fatalf("view %d %v", code, body)
	}
	if code, body := serve(t, handler, http.MethodGet, "/shop", "", ""); code != http.StatusUnauthorized || body["error"] != "access_rejected" {
		t.Fatalf("view without a token %d %v", code, body)
	}
	if code, body := serve(t, handler, http.MethodGet, "/shop", "expired", ""); code != http.StatusUnauthorized || body["error"] != "access_expired" {
		t.Fatalf("view with an expired token %d %v", code, body)
	}
	if code, body := serve(t, handler, http.MethodGet, "/shop", "access:empty", ""); code != http.StatusServiceUnavailable || body["error"] != "shop.unavailable" {
		t.Fatalf("view without a catalog %d %v", code, body)
	}
	if code, body := serve(t, handler, http.MethodGet, "/shop", "access:broken", ""); code != http.StatusInternalServerError || body["error"] != "internal" {
		t.Fatalf("broken view %d %v", code, body)
	}
	code, body := serve(t, handler, http.MethodPost, "/shop/buy", "access:a1", `{"offer":"offer.avatar_comet_fox","request_id":"r1"}`)
	if code != http.StatusOK || body["offer"] != "offer.avatar_comet_fox" || body["balance"].(float64) != 600 || body["repeated"] != false || body["grant"].(map[string]any)["id"].(float64) != 3 {
		t.Fatalf("purchase %d %v", code, body)
	}
	if code, body := serve(t, handler, http.MethodPost, "/shop/buy", "access:a1", `{"offer":"offer.avatar_comet_fox","request_id":"again"}`); code != http.StatusOK || body["repeated"] != true {
		t.Fatalf("repeated purchase %d %v", code, body)
	}
	if code, body := serve(t, handler, http.MethodPost, "/shop/buy", "access:a1", `{"offer":`); code != http.StatusBadRequest || body["error"] != "malformed" {
		t.Fatalf("malformed purchase %d %v", code, body)
	}
	if code, body := serve(t, handler, http.MethodPost, "/shop/buy", "access:a1", `{"offer":"offer.x","request_id":"`+strings.Repeat("x", shop.MaxBodyBytes)+`"}`); code != http.StatusBadRequest || body["error"] != "malformed" {
		t.Fatalf("oversized purchase %d %v", code, body)
	}
	refusals := []struct {
		offer  string
		status int
		err    string
	}{
		{"offer.missing", http.StatusNotFound, "offer.unknown"},
		{"offer.closed", http.StatusConflict, "offer.unavailable"},
		{"offer.limited", http.StatusConflict, "offer.limit_reached"},
		{"offer.owned", http.StatusConflict, "offer.owned"},
		{"offer.store", http.StatusConflict, "offer.store_only"},
		{"offer.pass", http.StatusConflict, "offer.unsupported"},
		{"offer.young", http.StatusForbidden, "offer.not_eligible"},
		{"offer.pricey", http.StatusPaymentRequired, "wallet.insufficient"},
		{"offer.unlisted", http.StatusConflict, "item.source_not_listed"},
		{"offer.broken", http.StatusInternalServerError, "internal"},
	}
	for _, r := range refusals {
		if code, body := serve(t, handler, http.MethodPost, "/shop/buy", "access:a1", `{"offer":"`+r.offer+`","request_id":"r"}`); code != r.status || body["error"] != r.err {
			t.Fatalf("%s: %d %v", r.offer, code, body)
		}
	}
	if code, body := serve(t, handler, http.MethodPost, "/shop/buy", "access:a1", `{"offer":"offer.ok"}`); code != http.StatusBadRequest || body["error"] != "request.invalid" {
		t.Fatalf("purchase without a request id %d %v", code, body)
	}
	code, body = serve(t, handler, http.MethodPost, "/shop/begin", "access:a1", `{"offer":"offer.starter"}`)
	if code != http.StatusOK || body["product_id"] != "offer_starter" || body["account_token"] != shop.AccountToken("a1") {
		t.Fatalf("checkout %d %v", code, body)
	}
	if code, body := serve(t, handler, http.MethodPost, "/shop/begin", "access:a1", `{"offer":"offer.coins"}`); code != http.StatusConflict || body["error"] != "offer.coins_only" {
		t.Fatalf("checkout of a coin offer %d %v", code, body)
	}
	if code, _ := serve(t, handler, http.MethodGet, "/shop/buy", "access:a1", ""); code != http.StatusMethodNotAllowed {
		t.Fatalf("GET buy %d", code)
	}
	if fake.calls[0] != "view a1 2026-10-30T12:00:00Z" || fake.calls[len(fake.calls)-1] != "begin a1 offer.coins" {
		t.Fatalf("calls %v", fake.calls)
	}
}

package wallet_test

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

type fakeWallet struct{ limit int }

func (f *fakeWallet) Balance(_ context.Context, account string, currency wallet.Currency) (int64, error) {
	if account == "broken" {
		return 0, errors.New("database down")
	}
	if currency == wallet.Crystal {
		return 75, nil
	}
	return 320, nil
}

func (f *fakeWallet) History(_ context.Context, account string, limit int) ([]wallet.Applied, error) {
	f.limit = limit
	return []wallet.Applied{
		{ID: 2, Account: account, Currency: wallet.Coin, Amount: -180, Reason: wallet.ShopPurchase, Source: wallet.Source{Type: "offer", Ref: "offer.comet_fox"}, Key: "spend:r1", BalanceAfter: 320, At: start},
		{ID: 1, Account: account, Currency: wallet.Crystal, Amount: 500, Reason: wallet.Purchase, Source: wallet.Source{Type: "store", Ref: "t1"}, Key: "purchase:google:t1", BalanceAfter: 500, At: start},
	}, nil
}

func call(t *testing.T, handler http.Handler, method, bearer string) (int, map[string]any) {
	t.Helper()
	server := httptest.NewServer(handler)
	defer server.Close()
	request, _ := http.NewRequest(method, server.URL+"/wallet", nil)
	if bearer != "" {
		request.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, decoded
}

func TestTheWalletRouteShowsBothBalancesAndRecentEntries(t *testing.T) {
	fake := &fakeWallet{}
	handler := wallet.NewHandler(fakeVerifier{}, fake, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if status, body := call(t, handler, "GET", ""); status != http.StatusUnauthorized || body["error"] != "access_rejected" {
		t.Fatalf("without a token %d %v", status, body)
	}
	if status, body := call(t, handler, "GET", "expired"); status != http.StatusUnauthorized || body["error"] != "access_expired" {
		t.Fatalf("with an expired token %d %v", status, body)
	}
	if status, body := call(t, handler, "POST", "access:ayla"); status != http.StatusMethodNotAllowed || body["error"] != "method_not_allowed" {
		t.Fatalf("post %d %v", status, body)
	}
	if status, body := call(t, handler, "GET", "access:broken"); status != http.StatusInternalServerError || body["error"] != "internal" {
		t.Fatalf("failing store %d %v", status, body)
	}
	status, body := call(t, handler, "GET", "access:ayla")
	if status != http.StatusOK || body["coins"] != float64(320) || body["crystals"] != float64(75) || fake.limit != wallet.HistoryPage {
		t.Fatalf("wallet %d %v limit %d", status, body, fake.limit)
	}
	entries := body["entries"].([]any)
	latest := entries[0].(map[string]any)
	if len(entries) != 2 || latest["id"] != float64(2) || latest["currency"] != "coin" || entries[1].(map[string]any)["currency"] != "crystal" || latest["amount"] != float64(-180) || latest["reason"] != "shop_purchase" || latest["source_ref"] != "offer.comet_fox" || latest["balance_after"] != float64(320) || latest["at"] != "2026-10-30T12:00:00Z" {
		t.Fatalf("entries %v", entries)
	}
}

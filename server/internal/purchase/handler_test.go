package purchase_test

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

	"github.com/Alievs-corp/hypedomino/server/internal/purchase"
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

type fakePurchases struct {
	calls   []string
	notices []purchase.Notice
}

var refusals = map[string]error{
	"offer.malformed": purchase.ErrMalformed, "offer.signature": purchase.ErrSignature, "offer.unknown_receipt": purchase.ErrNotFound,
	"offer.account": purchase.ErrAccount, "offer.reused": purchase.ErrReused, "offer.blocked": purchase.ErrBlocked,
	"offer.missing": shop.ErrUnknownOffer, "offer.platform": purchase.ErrPlatform, "offer.product": purchase.ErrProduct,
	"offer.cancelled": purchase.ErrCancelled, "offer.sandbox": purchase.ErrEnvironment, "offer.app": purchase.ErrApp,
	"offer.coins": shop.ErrCoinsOnly, "offer.pass": shop.ErrUnsupported, "offer.ledger": wallet.ErrPurchaseReuse,
	"offer.down": purchase.ErrStore, "offer.closed": shop.ErrNoCatalog, "offer.broken": errors.New("database down"),
}

func (f *fakePurchases) Complete(_ context.Context, account, offer string, platform purchase.Platform, tok string, _ time.Time) (purchase.Result, error) {
	f.calls = append(f.calls, account+" "+offer+" "+string(platform)+" "+tok)
	if err, ok := refusals[offer]; ok {
		return purchase.Result{}, err
	}
	return purchase.Result{Platform: platform, TransactionID: tok, Offer: offer, State: purchase.RecordGranted, Balance: 2000, Grant: &reward.Report{ID: 9, Coins: 1500}, Finished: true}, nil
}

func (f *fakePurchases) Notice(_ context.Context, _ purchase.Platform, notice purchase.Notice, _ time.Time) (bool, error) {
	f.notices = append(f.notices, notice)
	if notice.ID == "m-broken" {
		return false, errors.New("database down")
	}
	return notice.Kind == purchase.KindRefund, nil
}

type fakeNotifier struct{}

func (fakeNotifier) Notice(body []byte) (purchase.Notice, error) {
	switch string(body) {
	case "bad-signature":
		return purchase.Notice{}, purchase.ErrSignature
	case "other-app":
		return purchase.Notice{}, purchase.ErrApp
	case "sandbox":
		return purchase.Notice{}, purchase.ErrEnvironment
	case "garbage":
		return purchase.Notice{}, purchase.ErrMalformed
	case "broken":
		return purchase.Notice{ID: "m-broken", Kind: purchase.KindRefund, TransactionID: "t"}, nil
	}
	return purchase.Notice{ID: "m-1", Kind: purchase.KindRefund, TransactionID: "t"}, nil
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
	return recorder.Code, decoded
}

func TestTheHandlerCompletesPurchasesAndTakesStoreNotifications(t *testing.T) {
	fake := &fakePurchases{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := purchase.NewHandler(fakeVerifier{}, fake, fakeNotifier{}, fakeNotifier{}, "push-secret", logger, func() time.Time { return now })
	code, body := serve(t, handler, http.MethodPost, "/shop/complete", "access:a1", `{"offer":"offer.starter","platform":"google","token":"tok-1"}`)
	if code != http.StatusOK || body["state"] != "granted" || body["balance"].(float64) != 2000 || body["finished"] != true || body["grant"].(map[string]any)["coins"].(float64) != 1500 {
		t.Fatalf("complete %d %v", code, body)
	}
	if code, body := serve(t, handler, http.MethodPost, "/shop/complete", "", `{"offer":"offer.starter","platform":"google","token":"tok-1"}`); code != http.StatusUnauthorized || body["error"] != "access_rejected" {
		t.Fatalf("complete without a token %d %v", code, body)
	}
	if code, body := serve(t, handler, http.MethodPost, "/shop/complete", "expired", `{}`); code != http.StatusUnauthorized || body["error"] != "access_expired" {
		t.Fatalf("complete with an expired token %d %v", code, body)
	}
	for name, payload := range map[string]string{"not json": `{`, "no token": `{"offer":"offer.starter","platform":"google"}`, "huge": `{"token":"` + strings.Repeat("x", purchase.MaxBodyBytes) + `"}`} {
		if code, body := serve(t, handler, http.MethodPost, "/shop/complete", "access:a1", payload); code != http.StatusBadRequest || body["error"] != "malformed" {
			t.Fatalf("%s: %d %v", name, code, body)
		}
	}
	statuses := map[string]int{
		"offer.malformed": 400, "offer.signature": 400, "offer.unknown_receipt": 400,
		"offer.account": 403, "offer.reused": 403, "offer.blocked": 403, "offer.missing": 404,
		"offer.platform": 409, "offer.product": 409, "offer.cancelled": 409, "offer.sandbox": 409, "offer.app": 409, "offer.coins": 409, "offer.pass": 409, "offer.ledger": 409,
		"offer.down": 502, "offer.closed": 503, "offer.broken": 500,
	}
	for offer, want := range statuses {
		code, body := serve(t, handler, http.MethodPost, "/shop/complete", "access:a1", `{"offer":"`+offer+`","platform":"google","token":"t"}`)
		wantError := refusals[offer].Error()
		if want == 500 {
			wantError = "internal"
		}
		if code != want || body["error"] != wantError {
			t.Fatalf("%s: %d %v", offer, code, body)
		}
	}
	if code, body := serve(t, handler, http.MethodPost, "/store/google/notifications", "", "x"); code != http.StatusUnauthorized || body["error"] != "notification_rejected" {
		t.Fatalf("google notice without the secret %d %v", code, body)
	}
	if code, _ := serve(t, handler, http.MethodPost, "/store/google/notifications?token=wrong", "", "x"); code != http.StatusUnauthorized {
		t.Fatalf("google notice with a wrong secret %d", code)
	}
	if code, body := serve(t, handler, http.MethodPost, "/store/google/notifications?token=push-secret", "", "ok"); code != http.StatusOK || body["handled"] != true {
		t.Fatalf("google notice %d %v", code, body)
	}
	for payload, want := range map[string]int{"bad-signature": 401, "other-app": 401, "sandbox": 200, "garbage": 400, "broken": 500} {
		if code, _ := serve(t, handler, http.MethodPost, "/store/google/notifications?token=push-secret", "", payload); code != want {
			t.Fatalf("google notice %s: %d", payload, code)
		}
	}
	if code, body := serve(t, handler, http.MethodPost, "/store/apple/notifications", "", `{"signedPayload":"ok"}`); code != http.StatusOK || body["handled"] != true {
		t.Fatalf("apple notice %d %v", code, body)
	}
	if code, body := serve(t, handler, http.MethodPost, "/store/apple/notifications", "", `{"signedPayload":"bad-signature"}`); code != http.StatusUnauthorized || body["error"] != "notification_rejected" {
		t.Fatalf("apple notice with a bad signature %d %v", code, body)
	}
	if code, _ := serve(t, handler, http.MethodPost, "/store/apple/notifications", "", `{}`); code != http.StatusBadRequest {
		t.Fatalf("apple notice without a payload %d", code)
	}
	if len(fake.notices) != 3 || fake.notices[0].ID != "m-1" {
		t.Fatalf("notices %+v", fake.notices)
	}
	if fake.calls[0] != "a1 offer.starter google tok-1" {
		t.Fatalf("calls %v", fake.calls)
	}
	none := purchase.NewHandler(fakeVerifier{}, fake, nil, nil, "", logger, func() time.Time { return now })
	if code, _ := serve(t, none, http.MethodPost, "/store/google/notifications?token=", "", "ok"); code != http.StatusNotFound {
		t.Fatalf("google notice without a store %d", code)
	}
	if code, _ := serve(t, none, http.MethodPost, "/store/apple/notifications", "", `{"signedPayload":"ok"}`); code != http.StatusNotFound {
		t.Fatalf("apple notice without a store %d", code)
	}
	secretless := purchase.NewHandler(fakeVerifier{}, fake, fakeNotifier{}, nil, "", logger, func() time.Time { return now })
	if code, _ := serve(t, secretless, http.MethodPost, "/store/google/notifications?token=", "", "ok"); code != http.StatusUnauthorized {
		t.Fatalf("google notice without a configured secret %d", code)
	}
}

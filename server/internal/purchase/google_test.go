package purchase

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/googleauth/googleauthtest"
)

type playServer struct {
	*googleauthtest.Server
	mu       sync.Mutex
	consumed []string
}

func newPlayServer(t *testing.T) *playServer {
	t.Helper()
	p := &playServer{Server: googleauthtest.New(t, googleScope)}
	p.Mux.HandleFunc("GET /androidpublisher/v3/applications/com.example.hypedomino/purchases/products/{product}/tokens/{token}", p.purchase)
	p.Mux.HandleFunc("POST /androidpublisher/v3/applications/com.example.hypedomino/purchases/products/{product}/tokens/{token}", p.consume)
	return p
}

func (p *playServer) purchase(w http.ResponseWriter, r *http.Request) {
	if !googleauthtest.Authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	bodies := map[string]string{
		"tok-ok":        `{"purchaseState":0,"consumptionState":0,"orderId":"GPA.1234","purchaseTimeMillis":"` + millis() + `","obfuscatedExternalAccountId":"0000000A-0000-4000-8000-00000000000B","productId":"crystals_tier1","acknowledgementState":0}`,
		"tok-pending":   `{"purchaseState":2,"orderId":"GPA.1235","purchaseTimeMillis":"` + millis() + `","obfuscatedExternalAccountId":"a"}`,
		"tok-cancelled": `{"purchaseState":1,"orderId":"GPA.1236","purchaseTimeMillis":"` + millis() + `","obfuscatedExternalAccountId":"a"}`,
		"tok-test":      `{"purchaseState":0,"orderId":"","purchaseTimeMillis":"` + millis() + `","obfuscatedExternalAccountId":"a","purchaseType":0}`,
		"tok-weird":     `{"purchaseState":7}`,
		"tok-html":      `<html>`,
	}
	token := r.PathValue("token")
	switch token {
	case "tok-gone":
		http.Error(w, `{"error":{"code":410}}`, http.StatusGone)
	case "tok-broken":
		http.Error(w, "boom", http.StatusInternalServerError)
	default:
		body, ok := bodies[token]
		if !ok || r.PathValue("product") != "crystals_tier1" {
			http.Error(w, `{"error":{"code":404}}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	}
}

func (p *playServer) consume(w http.ResponseWriter, r *http.Request) {
	token, ok := strings.CutSuffix(r.PathValue("token"), ":consume")
	if !ok || !googleauthtest.Authorized(r) {
		http.Error(w, "bad consume", http.StatusBadRequest)
		return
	}
	p.mu.Lock()
	p.consumed = append(p.consumed, r.PathValue("product")+"/"+token)
	p.mu.Unlock()
	if token == "tok-stuck" {
		http.Error(w, "boom", http.StatusInternalServerError)
		return
	}
	w.Write([]byte(`{}`))
}

func millis() string {
	return "1793448000000"
}

func newGoogle(t *testing.T, p *playServer, allowTest bool) *Google {
	t.Helper()
	g, err := NewGoogle(GoogleConfig{PackageName: "com.example.hypedomino", ServiceAccountFile: p.File, AllowTest: allowTest, Endpoint: p.URL, Now: func() time.Time { return testNow }})
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestGooglePurchasesAreLookedUpWithAServiceAccountToken(t *testing.T) {
	play := newPlayServer(t)
	g := newGoogle(t, play, false)
	ctx := context.Background()
	receipt, err := g.Verify(ctx, "crystals_tier1", "tok-ok")
	if err != nil {
		t.Fatal(err)
	}
	want := Receipt{Platform: PlatformGoogle, TransactionID: "tok-ok", OrderID: "GPA.1234", ProductID: "crystals_tier1", AccountToken: "0000000a-0000-4000-8000-00000000000b", State: StatePurchased, Environment: EnvironmentProduction, PurchasedAt: time.UnixMilli(1793448000000).UTC()}
	if receipt != want {
		t.Fatalf("receipt %+v", receipt)
	}
	if receipt, err := g.Verify(ctx, "crystals_tier1", "tok-pending"); err != nil || receipt.State != StatePending {
		t.Fatalf("pending %+v %v", receipt, err)
	}
	if receipt, err := g.Verify(ctx, "crystals_tier1", "tok-cancelled"); err != nil || receipt.State != StateCancelled {
		t.Fatalf("cancelled %+v %v", receipt, err)
	}
	if _, err := g.Verify(ctx, "crystals_tier1", "tok-test"); !errors.Is(err, ErrEnvironment) {
		t.Fatalf("test purchase in production: %v", err)
	}
	if receipt, err := newGoogle(t, play, true).Verify(ctx, "crystals_tier1", "tok-test"); err != nil || receipt.Environment != EnvironmentSandbox || receipt.ProductID != "crystals_tier1" {
		t.Fatalf("test purchase when allowed %+v %v", receipt, err)
	}
	for name, c := range map[string]struct {
		product, token string
		want           error
	}{
		"gone":          {"crystals_tier1", "tok-gone", ErrNotFound},
		"unknown":       {"crystals_tier1", "tok-nope", ErrNotFound},
		"other product": {"crystals_tier2", "tok-ok", ErrNotFound},
		"broken":        {"crystals_tier1", "tok-broken", ErrStore},
		"weird state":   {"crystals_tier1", "tok-weird", ErrStore},
		"html":          {"crystals_tier1", "tok-html", ErrStore},
		"empty":         {"crystals_tier1", "", ErrMalformed},
		"slash":         {"crystals_tier1", "a/b", ErrMalformed},
		"no product":    {"", "tok-ok", ErrMalformed},
	} {
		if _, err := g.Verify(ctx, c.product, c.token); !errors.Is(err, c.want) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if err := g.Finish(ctx, receipt); err != nil {
		t.Fatal(err)
	}
	if err := g.Finish(ctx, Receipt{ProductID: "crystals_tier1", TransactionID: "tok-stuck"}); !errors.Is(err, ErrStore) {
		t.Fatalf("stuck consume: %v", err)
	}
	play.mu.Lock()
	defer play.mu.Unlock()
	if play.Tokens() != 2 || len(play.consumed) != 2 || play.consumed[0] != "crystals_tier1/tok-ok" {
		t.Fatalf("tokens %d consumed %v", play.Tokens(), play.consumed)
	}
}

func TestGoogleClientNeedsAUsableServiceAccount(t *testing.T) {
	dir := t.TempDir()
	broken := filepath.Join(dir, "broken.json")
	os.WriteFile(broken, []byte(`{"client_email":"a","private_key":"nope","token_uri":"https://x"}`), 0o600)
	cases := map[string]GoogleConfig{
		"no package": {ServiceAccountFile: broken},
		"no file":    {PackageName: "com.example.hypedomino"},
		"missing":    {PackageName: "com.example.hypedomino", ServiceAccountFile: filepath.Join(dir, "missing.json")},
		"broken key": {PackageName: "com.example.hypedomino", ServiceAccountFile: broken},
	}
	for name, config := range cases {
		if _, err := NewGoogle(config); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
	play := newPlayServer(t)
	play.Close()
	g := newGoogle(t, play, false)
	if _, err := g.Verify(context.Background(), "crystals_tier1", "tok-ok"); !errors.Is(err, ErrStore) {
		t.Fatalf("store down: %v", err)
	}
}

func push(t *testing.T, id string, data map[string]any) []byte {
	t.Helper()
	encoded, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]any{"message": map[string]any{"data": base64.StdEncoding.EncodeToString(encoded), "messageId": id}, "subscription": "projects/x/subscriptions/y"})
	return body
}

func TestGoogleDeveloperNotificationsAreDecoded(t *testing.T) {
	g := newGoogle(t, newPlayServer(t), false)
	purchased := push(t, "m-1", map[string]any{"version": "1.0", "packageName": "com.example.hypedomino", "eventTimeMillis": "1", "oneTimeProductNotification": map[string]any{"version": "1.0", "notificationType": 1, "purchaseToken": "tok-ok", "sku": "crystals_tier1"}})
	if notice, err := g.Notice(purchased); err != nil || notice != (Notice{ID: "m-1", Kind: KindPurchased, TransactionID: "tok-ok", ProductID: "crystals_tier1", Environment: EnvironmentProduction}) {
		t.Fatalf("purchased %+v %v", notice, err)
	}
	cancelled := push(t, "m-2", map[string]any{"packageName": "com.example.hypedomino", "oneTimeProductNotification": map[string]any{"notificationType": 2, "purchaseToken": "tok-ok", "sku": "crystals_tier1"}})
	if notice, err := g.Notice(cancelled); err != nil || notice.Kind != KindOther || notice.TransactionID != "tok-ok" {
		t.Fatalf("cancelled %+v %v", notice, err)
	}
	voided := push(t, "m-3", map[string]any{"packageName": "com.example.hypedomino", "voidedPurchaseNotification": map[string]any{"purchaseToken": "tok-ok", "orderId": "GPA.1234", "productType": 1, "refundType": 1}})
	if notice, err := g.Notice(voided); err != nil || notice.Kind != KindRefund || notice.TransactionID != "tok-ok" {
		t.Fatalf("voided %+v %v", notice, err)
	}
	test := push(t, "m-4", map[string]any{"packageName": "com.example.hypedomino", "testNotification": map[string]any{"version": "1.0"}})
	if notice, err := g.Notice(test); err != nil || notice.Kind != KindOther || notice.TransactionID != "" {
		t.Fatalf("test %+v %v", notice, err)
	}
	other := push(t, "m-5", map[string]any{"packageName": "com.example.other", "voidedPurchaseNotification": map[string]any{"purchaseToken": "tok-ok"}})
	if _, err := g.Notice(other); !errors.Is(err, ErrApp) {
		t.Fatalf("other package: %v", err)
	}
	for name, body := range map[string][]byte{"empty": []byte(``), "no id": []byte(`{"message":{"data":"e30="}}`), "bad base64": []byte(`{"message":{"data":"!!","messageId":"m"}}`), "not json": []byte(`{"message":{"data":"e30","messageId":"m"}}`)} {
		if _, err := g.Notice(body); !errors.Is(err, ErrMalformed) {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

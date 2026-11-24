package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/promo"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/shop"
)

const playAccessToken = "play-access-redteam"

type fakePlayStore struct {
	server         *httptest.Server
	serviceAccount string
	mu             sync.Mutex
	sold           map[string]map[string]any
	consumed       map[string]int
}

func newFakePlayStore(t *testing.T) *fakePlayStore {
	t.Helper()
	s := &fakePlayStore{sold: map[string]map[string]any{}, consumed: map[string]int{}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"access_token": playAccessToken, "expires_in": 3600})
	})
	mux.HandleFunc("/androidpublisher/v3/applications/{pkg}/purchases/products/{product}/tokens/{token}", s.purchase)
	s.server = httptest.NewServer(mux)
	t.Cleanup(s.server.Close)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	account, err := json.Marshal(map[string]string{
		"client_email": "redteam@fake-play.iam.gserviceaccount.com",
		"private_key":  string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})),
		"token_uri":    s.server.URL + "/token",
	})
	if err != nil {
		t.Fatal(err)
	}
	s.serviceAccount = filepath.Join(t.TempDir(), "play-service-account.json")
	if err := os.WriteFile(s.serviceAccount, account, 0o600); err != nil {
		t.Fatal(err)
	}
	return s
}

func (s *fakePlayStore) purchase(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer "+playAccessToken || r.PathValue("pkg") != redTeamPackage {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	purchaseToken, consume := strings.CutSuffix(r.PathValue("token"), ":consume")
	s.mu.Lock()
	defer s.mu.Unlock()
	sold, ok := s.sold[purchaseToken]
	switch {
	case !ok:
		w.WriteHeader(http.StatusNotFound)
	case consume && r.Method == http.MethodPost:
		s.consumed[purchaseToken]++
		w.WriteHeader(http.StatusNoContent)
	case r.Method == http.MethodGet:
		json.NewEncoder(w).Encode(sold)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *fakePlayStore) sell(purchaseToken, product, account string, test bool) {
	sold := map[string]any{
		"purchaseState":               0,
		"orderId":                     "GPA.redteam-" + purchaseToken,
		"purchaseTimeMillis":          strconv.FormatInt(time.Now().UnixMilli(), 10),
		"obfuscatedExternalAccountId": shop.AccountToken(account),
		"productId":                   product,
	}
	if test {
		sold["purchaseType"] = 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sold[purchaseToken] = sold
}

func (s *fakePlayStore) cancel(purchaseToken string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sold[purchaseToken]["purchaseState"] = 1
}

func complete(offer, platform, receipt string) map[string]any {
	return map[string]any{"offer": offer, "platform": platform, "token": receipt}
}

func TestRedTeamReceiptsGrantOnlyWhatTheStoreSold(t *testing.T) {
	h := startRedTeam(t)

	t.Run("a receipt the store never issued", func(t *testing.T) {
		a := h.attacker(t)
		before := h.economy(t, a)
		if r := h.post(t, a, "/shop/complete", complete("offer.crystals_tier5", "google", "forged-token-1")); r.status != http.StatusBadRequest || r.code() != "receipt.unknown" {
			t.Fatalf("forged receipt answered %d %v", r.status, r.body)
		}
		h.unchanged(t, a, before, "a forged receipt")
		h.awaitSignal(t, a, "fake_receipt")
	})

	t.Run("a receipt bought by another account", func(t *testing.T) {
		victim, a := h.attacker(t), h.attacker(t)
		h.play.sell("victim-token-1", "crystals_tier5", victim.account, false)
		before := h.economy(t, a)
		if r := h.post(t, a, "/shop/complete", complete("offer.crystals_tier5", "google", "victim-token-1")); r.status != http.StatusForbidden || r.code() != "receipt.account" {
			t.Fatalf("a receipt of another account answered %d %v", r.status, r.body)
		}
		h.unchanged(t, a, before, "a receipt of another account")
		h.awaitSignal(t, a, "fake_receipt")
		if r := h.post(t, victim, "/shop/complete", complete("offer.crystals_tier5", "google", "victim-token-1")); r.status != http.StatusOK {
			t.Fatalf("the buyer's own receipt answered %d %v", r.status, r.body)
		}
		if r := h.post(t, a, "/shop/complete", complete("offer.crystals_tier5", "google", "victim-token-1")); r.status != http.StatusForbidden {
			t.Fatalf("a granted receipt replayed by another account answered %d %v", r.status, r.body)
		}
		h.unchanged(t, a, before, "a granted receipt replayed by another account")
	})

	t.Run("one receipt submitted again and again", func(t *testing.T) {
		a := h.attacker(t)
		h.play.sell("own-token-1", "crystals_tier1", a.account, false)
		var after string
		for attempt := range 4 {
			r := h.post(t, a, "/shop/complete", complete("offer.crystals_tier1", "google", "own-token-1"))
			if r.status != http.StatusOK {
				t.Fatalf("attempt %d answered %d %v", attempt, r.status, r.body)
			}
			if attempt == 0 {
				after = h.economy(t, a)
				continue
			}
			h.unchanged(t, a, after, "a repeated receipt")
		}
		if got := h.purchaseGrants(t, a); got != 1 {
			t.Fatalf("four submissions of one receipt granted %d purchases, want 1", got)
		}
	})

	t.Run("a cheap receipt submitted for an expensive offer", func(t *testing.T) {
		a := h.attacker(t)
		h.play.sell("cheap-token-1", "crystals_tier1", a.account, false)
		before := h.economy(t, a)
		if r := h.post(t, a, "/shop/complete", complete("offer.crystals_tier5", "google", "cheap-token-1")); r.status != http.StatusConflict || r.code() != "receipt.product" {
			t.Fatalf("a swapped product answered %d %v", r.status, r.body)
		}
		h.unchanged(t, a, before, "a swapped product")
	})

	t.Run("a test purchase in production", func(t *testing.T) {
		a := h.attacker(t)
		h.play.sell("test-token-1", "crystals_tier5", a.account, true)
		before := h.economy(t, a)
		if r := h.post(t, a, "/shop/complete", complete("offer.crystals_tier5", "google", "test-token-1")); r.status != http.StatusConflict || r.code() != "receipt.environment" {
			t.Fatalf("a test purchase answered %d %v", r.status, r.body)
		}
		h.unchanged(t, a, before, "a test purchase")
		h.awaitSignal(t, a, "fake_receipt")
	})

	t.Run("a cancelled purchase", func(t *testing.T) {
		a := h.attacker(t)
		h.play.sell("cancelled-token-1", "crystals_tier5", a.account, false)
		h.play.cancel("cancelled-token-1")
		before := h.economy(t, a)
		if r := h.post(t, a, "/shop/complete", complete("offer.crystals_tier5", "google", "cancelled-token-1")); r.status != http.StatusConflict || r.code() != "receipt.cancelled" {
			t.Fatalf("a cancelled purchase answered %d %v", r.status, r.body)
		}
		h.unchanged(t, a, before, "a cancelled purchase")
	})

	t.Run("a signed transaction from a forged certificate chain", func(t *testing.T) {
		a := h.attacker(t)
		before := h.economy(t, a)
		forged := forgedAppleTransaction(t, map[string]any{
			"transactionId": "forged-apple-1", "originalTransactionId": "forged-apple-1", "bundleId": redTeamBundle, "productId": "crystals_tier5",
			"purchaseDate": time.Now().UnixMilli(), "environment": "Production", "appAccountToken": shop.AccountToken(a.account),
		})
		if r := h.post(t, a, "/shop/complete", complete("offer.crystals_tier5", "apple", forged)); r.status != http.StatusBadRequest || r.code() != "receipt.signature" {
			t.Fatalf("a forged App Store transaction answered %d %v", r.status, r.body)
		}
		if r := h.post(t, a, "/shop/complete", complete("offer.crystals_tier5", "apple", "e30.e30.c2ln")); r.status != http.StatusBadRequest || r.code() != "receipt.malformed" {
			t.Fatalf("an unsigned App Store transaction answered %d %v", r.status, r.body)
		}
		h.unchanged(t, a, before, "a forged App Store transaction")
		h.awaitSignal(t, a, "fake_receipt")
	})

	t.Run("a forged store notification", func(t *testing.T) {
		buyer := h.attacker(t)
		h.play.sell("refund-token-1", "crystals_tier2", buyer.account, false)
		if r := h.post(t, buyer, "/shop/complete", complete("offer.crystals_tier2", "google", "refund-token-1")); r.status != http.StatusOK {
			t.Fatalf("purchase answered %d %v", r.status, r.body)
		}
		before := h.economy(t, buyer)
		voided, err := json.Marshal(map[string]any{"packageName": redTeamPackage, "voidedPurchaseNotification": map[string]any{"purchaseToken": "refund-token-1", "orderId": "GPA.redteam-refund-token-1"}})
		if err != nil {
			t.Fatal(err)
		}
		push, err := json.Marshal(map[string]any{"message": map[string]any{"messageId": "forged-1", "data": base64.StdEncoding.EncodeToString(voided)}})
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range []string{"/store/google/notifications", "/store/google/notifications?token=guessed-secret"} {
			if r := h.call(t, nil, http.MethodPost, path, push, ""); r.status != http.StatusUnauthorized {
				t.Fatalf("%s answered %d %v", path, r.status, r.body)
			}
		}
		forged := forgedAppleTransaction(t, map[string]any{"notificationType": "REFUND", "notificationUUID": "forged-2", "data": map[string]any{"bundleId": redTeamBundle, "environment": "Production"}})
		body, err := json.Marshal(map[string]string{"signedPayload": forged})
		if err != nil {
			t.Fatal(err)
		}
		if r := h.call(t, nil, http.MethodPost, "/store/apple/notifications", body, ""); r.status != http.StatusUnauthorized {
			t.Fatalf("a forged App Store notification answered %d %v", r.status, r.body)
		}
		h.unchanged(t, buyer, before, "a forged store notification")
	})
}

func (h *redTeam) purchaseGrants(t *testing.T, a *attacker) int {
	t.Helper()
	var grants int
	if err := h.pool.QueryRow(context.Background(), `select count(*) from ledger_entries where account_id = $1 and reason = 'purchase'`, a.account).Scan(&grants); err != nil {
		t.Fatal(err)
	}
	return grants
}

var appleReceiptSigning = asn1.ObjectIdentifier{1, 2, 840, 113635, 100, 6, 11, 1}

func forgedAppleTransaction(t *testing.T, payload map[string]any) string {
	t.Helper()
	rootKey, root := certificate(t, nil, nil, true, nil)
	_, intermediate := certificate(t, root, rootKey, true, nil)
	leafKey, leaf := certificate(t, root, rootKey, false, []pkix.Extension{{Id: appleReceiptSigning, Value: []byte{5, 0}}})
	chain := []string{base64.StdEncoding.EncodeToString(leaf.Raw), base64.StdEncoding.EncodeToString(intermediate.Raw), base64.StdEncoding.EncodeToString(root.Raw)}
	header, err := json.Marshal(map[string]any{"alg": "ES256", "x5c": chain})
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	signingInput := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(body)
	digest := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.Reader, leafKey, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	signature := make([]byte, 64)
	r.FillBytes(signature[:32])
	s.FillBytes(signature[32:])
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func certificate(t *testing.T, parent *x509.Certificate, parentKey *ecdsa.PrivateKey, ca bool, extensions []pkix.Extension) (*ecdsa.PrivateKey, *x509.Certificate) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	serial, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: fmt.Sprintf("Apple Root CA - G3 %d", serial)},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  ca,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtraExtensions:       extensions,
	}
	if parent == nil {
		parent, parentKey = template, key
	}
	der, err := x509.CreateCertificate(rand.Reader, template, parent, &key.PublicKey, parentKey)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return key, cert
}

func TestRedTeamPromoCodesResistGuessingAndReuse(t *testing.T) {
	h := startRedTeam(t)
	hasher, err := promo.NewHasher(h.promoKey)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	codes := promo.NewStore(h.pool, nil, nil, hasher, promo.DefaultConfig())
	created, err := codes.Create(context.Background(), promo.Definition{
		ID: "promo.redteam_launch", Campaign: "campaign.redteam", Kind: promo.MultiUse, Cap: 100, Creator: "admin.redteam",
		Bundle: reward.Bundle{ID: "bundle.redteam_promo", Revision: 1, Coins: 100}, Start: now.Add(-time.Hour), End: now.Add(24 * time.Hour),
	}, now)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("one code redeemed twice", func(t *testing.T) {
		a := h.attacker(t)
		if r := h.post(t, a, "/promo/redeem", map[string]any{"code": created.Plaintext}); r.status != http.StatusOK {
			t.Fatalf("first redemption answered %d %v", r.status, r.body)
		}
		after := h.economy(t, a)
		if r := h.post(t, a, "/promo/redeem", map[string]any{"code": created.Plaintext}); r.status == http.StatusOK {
			t.Fatalf("a second redemption answered %d %v", r.status, r.body)
		}
		h.unchanged(t, a, after, "a second redemption of one code")
	})

	t.Run("guessing codes", func(t *testing.T) {
		a := h.attacker(t)
		before := h.economy(t, a)
		var answers []string
		for range 12 {
			guess, err := promo.Generate(promo.DefaultConfig().CodeLength)
			if err != nil {
				t.Fatal(err)
			}
			r := h.post(t, a, "/promo/redeem", map[string]any{"code": guess})
			answers = append(answers, fmt.Sprintf("%d %s", r.status, r.code()))
			if r.status == http.StatusOK {
				t.Fatalf("guess %s was redeemed: %v", guess, r.body)
			}
		}
		if !strings.Contains(strings.Join(answers, ","), "429 rate_limited") {
			t.Fatalf("twelve guesses in a row were never rate limited: %v", answers)
		}
		h.awaitSignal(t, a, "promo_lock")
		if _, err := codes.Redeem(context.Background(), a.account, created.Plaintext, time.Now()); !errors.Is(err, promo.ErrLocked) {
			t.Fatalf("a locked account redeeming the real code got %v", err)
		}
		h.unchanged(t, a, before, "guessing promo codes")
	})
}

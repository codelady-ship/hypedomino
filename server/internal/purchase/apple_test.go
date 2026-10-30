package purchase

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"
)

var testNow = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

type testChain struct {
	roots *x509.CertPool
	leaf  *ecdsa.PrivateKey
	x5c   []string
}

func certificate(t *testing.T, template, parent *x509.Certificate, key any, parentKey any) *x509.Certificate {
	t.Helper()
	der, err := x509.CreateCertificate(rand.Reader, template, parent, key, parentKey)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return cert
}

func newChain(t *testing.T, signing bool) testChain {
	t.Helper()
	rootKey, _ := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	interKey, _ := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	leafKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	rootTemplate := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Test Root CA"}, NotBefore: testNow.Add(-time.Hour), NotAfter: testNow.Add(24 * time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}
	root := certificate(t, rootTemplate, rootTemplate, &rootKey.PublicKey, rootKey)
	interTemplate := &x509.Certificate{SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: "Test Intermediate"}, NotBefore: testNow.Add(-time.Hour), NotAfter: testNow.Add(24 * time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}
	inter := certificate(t, interTemplate, root, &interKey.PublicKey, rootKey)
	leafTemplate := &x509.Certificate{SerialNumber: big.NewInt(3), Subject: pkix.Name{CommonName: "Test Receipt Signer"}, NotBefore: testNow.Add(-time.Hour), NotAfter: testNow.Add(24 * time.Hour), KeyUsage: x509.KeyUsageDigitalSignature}
	if signing {
		leafTemplate.ExtraExtensions = []pkix.Extension{{Id: appleReceiptSigning, Value: []byte{0x05, 0x00}}}
	}
	leaf := certificate(t, leafTemplate, inter, &leafKey.PublicKey, interKey)
	roots := x509.NewCertPool()
	roots.AddCert(root)
	x5c := []string{base64.StdEncoding.EncodeToString(leaf.Raw), base64.StdEncoding.EncodeToString(inter.Raw), base64.StdEncoding.EncodeToString(root.Raw)}
	return testChain{roots: roots, leaf: leafKey, x5c: x5c}
}

func (c testChain) sign(t *testing.T, alg string, payload any) string {
	t.Helper()
	header, _ := json.Marshal(map[string]any{"alg": alg, "x5c": c.x5c})
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(body)
	digest := sha256.Sum256([]byte(input))
	r, s, err := ecdsa.Sign(rand.Reader, c.leaf, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	signature := make([]byte, 64)
	r.FillBytes(signature[:32])
	s.FillBytes(signature[32:])
	return input + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func transaction(environment string) map[string]any {
	return map[string]any{
		"transactionId": "2000000123", "originalTransactionId": "2000000100", "bundleId": "com.example.hypedomino",
		"productId": "offer_starter", "purchaseDate": testNow.UnixMilli(), "environment": environment,
		"appAccountToken": "5F0D6A2E-3C7B-4E1A-9D8F-2B6C4A1E7F30", "type": "Consumable",
	}
}

func newApple(t *testing.T, chain testChain, sandbox bool) *Apple {
	t.Helper()
	apple, err := NewApple(AppleConfig{BundleID: "com.example.hypedomino", AllowSandbox: sandbox, Roots: chain.roots, Now: func() time.Time { return testNow }})
	if err != nil {
		t.Fatal(err)
	}
	return apple
}

func TestAppleTransactionsVerifyAgainstTheChain(t *testing.T) {
	chain := newChain(t, true)
	apple := newApple(t, chain, false)
	receipt, err := apple.Verify(context.Background(), "", chain.sign(t, "ES256", transaction("Production")))
	if err != nil {
		t.Fatal(err)
	}
	want := Receipt{Platform: PlatformApple, TransactionID: "2000000123", OrderID: "2000000100", ProductID: "offer_starter", AccountToken: "5f0d6a2e-3c7b-4e1a-9d8f-2b6c4a1e7f30", State: StatePurchased, Environment: EnvironmentProduction, PurchasedAt: testNow}
	if receipt != want {
		t.Fatalf("receipt %+v", receipt)
	}
	if err := apple.Finish(context.Background(), receipt); err != nil {
		t.Fatal(err)
	}
	revoked := transaction("Production")
	revoked["revocationDate"] = testNow.Add(time.Hour).UnixMilli()
	if receipt, err := apple.Verify(context.Background(), "", chain.sign(t, "ES256", revoked)); err != nil || receipt.State != StateCancelled {
		t.Fatalf("revoked transaction %+v %v", receipt, err)
	}
	if _, err := apple.Verify(context.Background(), "", chain.sign(t, "ES256", transaction("Sandbox"))); !errors.Is(err, ErrEnvironment) {
		t.Fatalf("sandbox in production: %v", err)
	}
	if receipt, err := newApple(t, chain, true).Verify(context.Background(), "", chain.sign(t, "ES256", transaction("Sandbox"))); err != nil || receipt.Environment != EnvironmentSandbox {
		t.Fatalf("sandbox when allowed %+v %v", receipt, err)
	}
	if _, err := NewApple(AppleConfig{BundleID: "com.example.hypedomino"}); err != nil {
		t.Fatalf("embedded apple root: %v", err)
	}
	if _, err := NewApple(AppleConfig{}); err == nil {
		t.Fatal("no bundle id accepted")
	}
}

func TestAppleTransactionsAreRefusedWhenAnythingIsOff(t *testing.T) {
	chain := newChain(t, true)
	apple := newApple(t, chain, false)
	valid := chain.sign(t, "ES256", transaction("Production"))
	segments := strings.Split(valid, ".")
	otherBundle := transaction("Production")
	otherBundle["bundleId"] = "com.example.other"
	noID := transaction("Production")
	delete(noID, "transactionId")
	weird := transaction("Staging")
	tampered := segments[0] + "." + base64.RawURLEncoding.EncodeToString([]byte(`{"transactionId":"1","productId":"x","bundleId":"com.example.hypedomino","environment":"Production"}`)) + "." + segments[2]
	cases := map[string]struct {
		token string
		want  error
	}{
		"bundle":         {chain.sign(t, "ES256", otherBundle), ErrApp},
		"no id":          {chain.sign(t, "ES256", noID), ErrMalformed},
		"environment":    {chain.sign(t, "ES256", weird), ErrMalformed},
		"tampered":       {tampered, ErrSignature},
		"algorithm":      {chain.sign(t, "RS256", transaction("Production")), ErrMalformed},
		"segments":       {segments[0] + "." + segments[1], ErrMalformed},
		"not signing":    {newChain(t, false).sign(t, "ES256", transaction("Production")), ErrSignature},
		"unknown root":   {newChain(t, true).sign(t, "ES256", transaction("Production")), ErrSignature},
		"short sig":      {segments[0] + "." + segments[1] + ".AAAA", ErrMalformed},
		"garbage header": {"!!." + segments[1] + "." + segments[2], ErrMalformed},
	}
	for name, c := range cases {
		if _, err := apple.Verify(context.Background(), "", c.token); !errors.Is(err, c.want) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	expired, err := NewApple(AppleConfig{BundleID: "com.example.hypedomino", Roots: chain.roots, Now: func() time.Time { return testNow.Add(48 * time.Hour) }})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := expired.Verify(context.Background(), "", valid); !errors.Is(err, ErrSignature) {
		t.Fatalf("expired chain: %v", err)
	}
}

func TestAppleNotificationsCarryTheSignedTransaction(t *testing.T) {
	chain := newChain(t, true)
	apple := newApple(t, chain, false)
	notify := func(signed string) (Notice, error) { return apple.Notice([]byte(signed)) }
	inner := chain.sign(t, "ES256", transaction("Production"))
	notification := func(kind, environment, bundle string) map[string]any {
		return map[string]any{"notificationType": kind, "notificationUUID": "n-1", "data": map[string]any{"bundleId": bundle, "environment": environment, "signedTransactionInfo": inner}}
	}
	notice, err := notify(chain.sign(t, "ES256", notification("REFUND", "Production", "com.example.hypedomino")))
	if err != nil || notice != (Notice{ID: "n-1", Kind: KindRefund, TransactionID: "2000000123", ProductID: "offer_starter", Environment: EnvironmentProduction}) {
		t.Fatalf("refund notice %+v %v", notice, err)
	}
	if notice, err := notify(chain.sign(t, "ES256", notification("REFUND_REVERSED", "Production", "com.example.hypedomino"))); err != nil || notice.Kind != KindReversal {
		t.Fatalf("reversal notice %+v %v", notice, err)
	}
	if notice, err := notify(chain.sign(t, "ES256", notification("CONSUMPTION_REQUEST", "Production", "com.example.hypedomino"))); err != nil || notice.Kind != KindOther {
		t.Fatalf("other notice %+v %v", notice, err)
	}
	if _, err := notify(chain.sign(t, "ES256", notification("REFUND", "Production", "com.example.other"))); !errors.Is(err, ErrApp) {
		t.Fatalf("other bundle: %v", err)
	}
	if _, err := notify(chain.sign(t, "ES256", notification("REFUND", "Sandbox", "com.example.hypedomino"))); !errors.Is(err, ErrEnvironment) {
		t.Fatalf("sandbox notice: %v", err)
	}
	if _, err := notify(chain.sign(t, "ES256", map[string]any{"notificationType": "TEST"})); !errors.Is(err, ErrMalformed) {
		t.Fatalf("notice without uuid: %v", err)
	}
	bare := chain.sign(t, "ES256", map[string]any{"notificationType": "TEST", "notificationUUID": "n-2", "data": map[string]any{"bundleId": "com.example.hypedomino", "environment": "Production"}})
	if notice, err := notify(bare); err != nil || notice.Kind != KindOther || notice.TransactionID != "" {
		t.Fatalf("test notice %+v %v", notice, err)
	}
	if _, err := notify(newChain(t, true).sign(t, "ES256", notification("REFUND", "Production", "com.example.hypedomino"))); !errors.Is(err, ErrSignature) {
		t.Fatalf("notice from an unknown chain: %v", err)
	}
}

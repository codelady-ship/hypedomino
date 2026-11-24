package integrity

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

type stubVerifier struct{}

func (stubVerifier) Verify(access string) (token.Claims, error) {
	switch access {
	case "good":
		return token.Claims{Account: "acc-1", Session: "sid-1"}, nil
	case "old":
		return token.Claims{}, token.ErrExpired
	}
	return token.Claims{}, token.ErrSignature
}

func TestNonceEndpointBindsTheNonceToTheCaller(t *testing.T) {
	clock := newTestClock()
	store := NewMemoryNonceStore(clock)
	nonces, err := NewNonces(NonceConfig{TTL: 5 * time.Minute}, store, clock, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(stubVerifier{}, nonces, slog.New(slog.NewTextHandler(io.Discard, nil)))
	call := func(access, body string) (int, map[string]any) {
		r := httptest.NewRequest(http.MethodPost, "/integrity/nonce", strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer "+access)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		var decoded map[string]any
		json.Unmarshal(w.Body.Bytes(), &decoded)
		return w.Code, decoded
	}
	for _, purpose := range []Purpose{PurposeSessionStart, PurposePurchase, PurposeRankedMatch, PurposeCasualMatch} {
		status, body := call("good", `{"purpose":"`+purpose.String()+`"}`)
		if status != http.StatusOK || body["expires_in"] != float64(300) {
			t.Fatalf("%v: %d %v", purpose, status, body)
		}
		stored, found, _ := store.Take(t.Context(), body["nonce"].(string))
		if want := (Binding{Account: "acc-1", Session: "sid-1", Purpose: purpose}); !found || stored.Binding != want || !stored.ExpiresAt.Equal(clock.Now().Add(5*time.Minute)) {
			t.Fatalf("%v stored %+v %v", purpose, stored, found)
		}
	}
	refusals := []struct {
		access, body string
		status       int
		code         string
	}{
		{"bad", `{"purpose":"purchase"}`, http.StatusUnauthorized, "access_rejected"},
		{"old", `{"purpose":"purchase"}`, http.StatusUnauthorized, "access_expired"},
		{"good", `{"purpose":"invalid"}`, http.StatusBadRequest, "unknown_purpose"},
		{"good", `{}`, http.StatusBadRequest, "unknown_purpose"},
		{"good", `not json`, http.StatusBadRequest, "bad_request"},
		{"good", `{"purpose":"` + strings.Repeat("x", MaxBodyBytes) + `"}`, http.StatusBadRequest, "bad_request"},
	}
	for _, c := range refusals {
		if status, body := call(c.access, c.body); status != c.status || body["error"] != c.code {
			t.Fatalf("%s %.40s: %d %v", c.access, c.body, status, body)
		}
	}
}

type fixtureVerifier struct{}

func (fixtureVerifier) Verify(access string) (token.Claims, error) {
	if access != "good" {
		return token.Claims{}, token.ErrSignature
	}
	binding := sessionBinding()
	return token.Claims{Account: binding.Account, Session: binding.Session}, nil
}

type savedKeys struct {
	keys map[string]AttestedKey
	err  error
}

func (s *savedKeys) Save(_ context.Context, account string, key AttestedKey, at time.Time) error {
	if s.err != nil {
		return s.err
	}
	if _, exists := s.keys[string(key.KeyID)]; exists {
		return ErrKeyExists
	}
	s.keys[string(key.KeyID)] = key
	return nil
}

func TestAttestEndpointStoresOnlyVerifiedKeys(t *testing.T) {
	fixture := newAttestFixture(t, false)
	keys := &savedKeys{keys: map[string]AttestedKey{}}
	handler := NewHandler(fixtureVerifier{}, fixture.nonces, slog.New(slog.NewTextHandler(io.Discard, nil)))
	handler.EnableAppAttest(fixture.verifier, keys, fixture.clock)
	keyID := base64.StdEncoding.EncodeToString(keyIDOf(t, fixture.credential))
	call := func(access string, body any) (int, string) {
		encoded, _ := json.Marshal(body)
		r := httptest.NewRequest(http.MethodPost, "/integrity/attest", bytes.NewReader(encoded))
		r.Header.Set("Authorization", "Bearer "+access)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		var decoded errorResponse
		json.Unmarshal(w.Body.Bytes(), &decoded)
		return w.Code, decoded.Error
	}
	attestBody := func(nonce string) map[string]string {
		return map[string]string{"nonce": nonce, "key_id": keyID, "attestation": base64.StdEncoding.EncodeToString(fixture.attestation(t, nonce, nil))}
	}
	nonce := fixture.issueNonce(t, sessionBinding())
	if status, code := call("good", attestBody(nonce)); status != http.StatusNoContent || code != "" {
		t.Fatalf("attest: %d %s", status, code)
	}
	stored, ok := keys.keys[string(keyIDOf(t, fixture.credential))]
	if !ok || len(stored.PublicKey) != 65 || string(stored.Receipt) != "receipt" || stored.Counter != 0 {
		t.Fatalf("stored %+v", stored)
	}
	purchase := sessionBinding()
	purchase.Purpose = PurposePurchase
	keys.err = errors.New("database down")
	downNonce := fixture.issueNonce(t, sessionBinding())
	cases := []struct {
		name   string
		access string
		body   any
		status int
		code   string
	}{
		{"again", "good", attestBody(fixture.issueNonce(t, sessionBinding())), http.StatusServiceUnavailable, "unavailable"},
		{"replayed nonce", "good", attestBody(nonce), http.StatusForbidden, "attestation_failed"},
		{"purchase nonce", "good", attestBody(fixture.issueNonce(t, purchase)), http.StatusForbidden, "attestation_failed"},
		{"no token", "bad", attestBody(fixture.issueNonce(t, sessionBinding())), http.StatusUnauthorized, "access_rejected"},
		{"not base64", "good", map[string]string{"nonce": "n", "key_id": "%%", "attestation": "AA=="}, http.StatusBadRequest, "bad_request"},
		{"empty attestation", "good", map[string]string{"nonce": "n", "key_id": keyID}, http.StatusBadRequest, "bad_request"},
		{"not json", "good", "nope", http.StatusBadRequest, "bad_request"},
	}
	for _, c := range cases {
		if status, code := call(c.access, c.body); status != c.status || code != c.code {
			t.Fatalf("%s: %d %s", c.name, status, code)
		}
	}
	keys.err = nil
	if status, code := call("good", attestBody(downNonce)); status != http.StatusConflict || code != "key_exists" {
		t.Fatalf("same key twice: %d %s", status, code)
	}
}

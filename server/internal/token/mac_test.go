package token

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fixedKeys struct{ signer *Signer }

func (k fixedKeys) Verify(access string) (Claims, error) {
	return k.signer.Verify(access, start)
}

func (k fixedKeys) RequestKey(claims Claims) ([]byte, error) {
	return k.signer.RequestKey(claims)
}

type failureLog struct{ accounts []string }

func (f *failureLog) Observe(ctx context.Context, account string) {
	f.accounts = append(f.accounts, account)
}

func TestRequestKeyIsPerSessionAndFollowsTheSigningKey(t *testing.T) {
	old := signer(t, keySpec("k1", 'a'))
	rotated := signer(t, keySpec("k2", 'b')+","+keySpec("k1", 'a'))
	first := claimsAt(start)
	other := claimsAt(start)
	other.Session = "sid-2"
	key := func(s *Signer, claims Claims) []byte {
		t.Helper()
		access, err := s.Sign(claims)
		if err != nil {
			t.Fatal(err)
		}
		verified, err := s.Verify(access, start)
		if err != nil {
			t.Fatal(err)
		}
		got, err := rotated.RequestKey(verified)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	oldKey := key(old, first)
	if len(oldKey) != 32 || !bytes.Equal(oldKey, requestKey(old.sign.Secret, "sid-1")) {
		t.Fatalf("old key %x", oldKey)
	}
	if bytes.Equal(oldKey, key(old, other)) {
		t.Fatal("two sessions share a request key")
	}
	if bytes.Equal(oldKey, key(rotated, first)) {
		t.Fatal("rotated signing key kept the request key")
	}
	if _, err := rotated.RequestKey(Claims{KeyID: "gone", Session: "sid-1"}); err != ErrUnknownKey {
		t.Fatalf("unknown kid: %v", err)
	}
}

func TestCriticalWritesNeedTheRequestMac(t *testing.T) {
	s := signer(t, keySpec("k1", 'a'))
	keys := fixedKeys{signer: s}
	access, _ := s.Sign(claimsAt(start))
	otherClaims := claimsAt(start)
	otherClaims.Session = "sid-2"
	otherAccess, _ := s.Sign(otherClaims)
	key := requestKey(s.sign.Secret, "sid-1")
	expired, _ := s.Sign(claimsAt(start.Add(-time.Hour)))
	body := []byte(`{"slot":"tile","items":["t1"]}`)
	cases := []struct {
		name     string
		method   string
		target   string
		access   string
		mac      string
		reached  bool
		status   int
		failures int
	}{
		{"signed write", http.MethodPut, "/inventory/equip", access, SignRequest(key, http.MethodPut, "/inventory/equip", access, body), true, http.StatusNoContent, 0},
		{"signed with query", http.MethodPost, "/shop/buy?x=1", access, SignRequest(key, http.MethodPost, "/shop/buy?x=1", access, body), true, http.StatusNoContent, 0},
		{"reads need no mac", http.MethodGet, "/inventory", access, "", true, http.StatusNoContent, 0},
		{"no access token goes to the handler", http.MethodPut, "/inventory/equip", "", "", true, http.StatusNoContent, 0},
		{"expired access token goes to the handler", http.MethodPut, "/inventory/equip", expired, "", true, http.StatusNoContent, 0},
		{"missing mac", http.MethodPut, "/inventory/equip", access, "", false, http.StatusUnauthorized, 1},
		{"not base64", http.MethodPut, "/inventory/equip", access, "%%%", false, http.StatusUnauthorized, 1},
		{"other path", http.MethodPut, "/inventory/equip", access, SignRequest(key, http.MethodPut, "/shop/buy", access, body), false, http.StatusUnauthorized, 1},
		{"other method", http.MethodPut, "/inventory/equip", access, SignRequest(key, http.MethodPost, "/inventory/equip", access, body), false, http.StatusUnauthorized, 1},
		{"other body", http.MethodPut, "/inventory/equip", access, SignRequest(key, http.MethodPut, "/inventory/equip", access, []byte(`{}`)), false, http.StatusUnauthorized, 1},
		{"other access token", http.MethodPut, "/inventory/equip", otherAccess, SignRequest(key, http.MethodPut, "/inventory/equip", access, body), false, http.StatusUnauthorized, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			failures := &failureLog{}
			var seen []byte
			reached := false
			handler := RequireMac(keys, failures, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				reached = true
				seen, _ = io.ReadAll(r.Body)
				w.WriteHeader(http.StatusNoContent)
			}))
			r := httptest.NewRequest(c.method, c.target, bytes.NewReader(body))
			if c.access != "" {
				r.Header.Set("Authorization", "Bearer "+c.access)
			}
			if c.mac != "" {
				r.Header.Set(MacHeader, c.mac)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if reached != c.reached || w.Code != c.status || len(failures.accounts) != c.failures {
				t.Fatalf("reached %v status %d failures %v body %s", reached, w.Code, failures.accounts, w.Body)
			}
			if c.reached && !bytes.Equal(seen, body) {
				t.Fatalf("handler read %q", seen)
			}
			if c.failures > 0 && (failures.accounts[0] != "acc-1" || w.Body.String() != "{\"error\":\"bad_mac\"}\n") {
				t.Fatalf("failure %v %s", failures.accounts, w.Body)
			}
		})
	}
}

func TestOversizedSignedBodiesAreRefusedUncounted(t *testing.T) {
	s := signer(t, keySpec("k1", 'a'))
	access, _ := s.Sign(claimsAt(start))
	failures := &failureLog{}
	handler := RequireMac(fixedKeys{signer: s}, failures, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler reached")
	}))
	r := httptest.NewRequest(http.MethodPost, "/shop/buy", bytes.NewReader(make([]byte, MaxMacBodyBytes+1)))
	r.Header.Set("Authorization", "Bearer "+access)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusRequestEntityTooLarge || len(failures.accounts) != 0 {
		t.Fatalf("status %d failures %v", w.Code, failures.accounts)
	}
}

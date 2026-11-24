package integrity

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"sync"
	"testing"
	"time"
)

var testStart = time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)

type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func newTestClock() *testClock {
	return &testClock{now: testStart}
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) Advance(by time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(by)
}

func newTestNonces(t *testing.T, clock *testClock) (*Nonces, *MemoryNonceStore) {
	t.Helper()
	store := NewMemoryNonceStore(clock)
	nonces, err := NewNonces(NonceConfig{TTL: 5 * time.Minute}, store, clock, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return nonces, store
}

func sessionBinding() Binding {
	return Binding{Account: "account-1", Session: "session-1", Purpose: PurposeSessionStart}
}

func TestNewNoncesRejectsInvalidConfig(t *testing.T) {
	clock := newTestClock()
	store := NewMemoryNonceStore(clock)
	cases := map[string]struct {
		config NonceConfig
		store  NonceStore
	}{
		"zero ttl":     {NonceConfig{}, store},
		"negative ttl": {NonceConfig{TTL: -time.Second}, store},
		"no store":     {NonceConfig{TTL: time.Minute}, nil},
	}
	for name, tc := range cases {
		if _, err := NewNonces(tc.config, tc.store, clock, rand.Reader); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestIssueRejectsInvalidBinding(t *testing.T) {
	nonces, _ := newTestNonces(t, newTestClock())
	for name, binding := range map[string]Binding{
		"no account":      {Session: "s", Purpose: PurposePurchase},
		"no purpose":      {Account: "a", Session: "s"},
		"unknown purpose": {Account: "a", Session: "s", Purpose: PurposeCasualMatch + 1},
	} {
		if _, err := nonces.Issue(context.Background(), binding); !errors.Is(err, ErrInvalidBinding) {
			t.Errorf("%s: err = %v, want ErrInvalidBinding", name, err)
		}
	}
}

func TestIssueFailsWhenRandomSourceFails(t *testing.T) {
	clock := newTestClock()
	nonces, err := NewNonces(NonceConfig{TTL: time.Minute}, NewMemoryNonceStore(clock), clock, bytes.NewReader([]byte{1, 2, 3}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := nonces.Issue(context.Background(), sessionBinding()); err == nil {
		t.Fatal("short random source produced a nonce")
	}
}

func TestIssuedNoncesAreUniqueAndExpireAfterTheTTL(t *testing.T) {
	clock := newTestClock()
	nonces, _ := newTestNonces(t, clock)
	seen := map[string]bool{}
	for range 100 {
		nonce, err := nonces.Issue(context.Background(), sessionBinding())
		if err != nil {
			t.Fatal(err)
		}
		if seen[nonce.Value] {
			t.Fatalf("nonce %q issued twice", nonce.Value)
		}
		seen[nonce.Value] = true
		if !nonce.ExpiresAt.Equal(testStart.Add(5 * time.Minute)) {
			t.Fatalf("expires at %v", nonce.ExpiresAt)
		}
		if len(nonce.Value) < 16 {
			t.Fatalf("nonce %q is shorter than 16 characters", nonce.Value)
		}
	}
}

func TestRedeemIsSingleUseAndBound(t *testing.T) {
	cases := []struct {
		name    string
		advance time.Duration
		redeem  func(issued Nonce) (string, Binding)
		want    Reason
	}{
		{"matching", 0, func(n Nonce) (string, Binding) { return n.Value, n.Binding }, 0},
		{"just before expiry", 5*time.Minute - time.Millisecond, func(n Nonce) (string, Binding) { return n.Value, n.Binding }, 0},
		{"expired", 5 * time.Minute, func(n Nonce) (string, Binding) { return n.Value, n.Binding }, ReasonNonceExpired},
		{"empty", 0, func(n Nonce) (string, Binding) { return "", n.Binding }, ReasonNonceMissing},
		{"never issued", 0, func(n Nonce) (string, Binding) { return n.Value + "x", n.Binding }, ReasonNonceUnknown},
		{"other account", 0, func(n Nonce) (string, Binding) {
			b := n.Binding
			b.Account = "account-2"
			return n.Value, b
		}, ReasonNonceBindingMismatch},
		{"other session", 0, func(n Nonce) (string, Binding) {
			b := n.Binding
			b.Session = "session-2"
			return n.Value, b
		}, ReasonNonceBindingMismatch},
		{"other purpose", 0, func(n Nonce) (string, Binding) {
			b := n.Binding
			b.Purpose = PurposePurchase
			return n.Value, b
		}, ReasonNonceBindingMismatch},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clock := newTestClock()
			nonces, _ := newTestNonces(t, clock)
			issued, err := nonces.Issue(context.Background(), sessionBinding())
			if err != nil {
				t.Fatal(err)
			}
			clock.Advance(tc.advance)
			value, binding := tc.redeem(issued)
			got, err := nonces.redeem(context.Background(), value, binding)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("redeem = %v, want %v", got, tc.want)
			}
			again, err := nonces.redeem(context.Background(), issued.Value, issued.Binding)
			if err != nil {
				t.Fatal(err)
			}
			if tc.want != ReasonNonceMissing && tc.want != ReasonNonceUnknown && again != ReasonNonceUnknown {
				t.Fatalf("second redeem = %v, want nonce_unknown", again)
			}
		})
	}
}

func TestMemoryStoreDropsExpiredNonces(t *testing.T) {
	clock := newTestClock()
	nonces, store := newTestNonces(t, clock)
	for range 10 {
		if _, err := nonces.Issue(context.Background(), sessionBinding()); err != nil {
			t.Fatal(err)
		}
	}
	clock.Advance(5 * time.Minute)
	if _, err := nonces.Issue(context.Background(), sessionBinding()); err != nil {
		t.Fatal(err)
	}
	if len(store.nonces) != 1 || len(store.queue) != 1 {
		t.Fatalf("store holds %d nonces and %d queued, want 1 and 1", len(store.nonces), len(store.queue))
	}
}

func TestMemoryStoreRejectsDuplicateValues(t *testing.T) {
	store := NewMemoryNonceStore(newTestClock())
	nonce := Nonce{Value: "v", Binding: sessionBinding(), ExpiresAt: testStart.Add(time.Minute)}
	if err := store.Put(context.Background(), nonce); err != nil {
		t.Fatal(err)
	}
	if err := store.Put(context.Background(), nonce); !errors.Is(err, ErrNonceExists) {
		t.Fatalf("err = %v, want ErrNonceExists", err)
	}
}

func TestMemoryStoreIsSafeForConcurrentRedeem(t *testing.T) {
	clock := newTestClock()
	nonces, _ := newTestNonces(t, clock)
	issued, err := nonces.Issue(context.Background(), sessionBinding())
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	accepted := 0
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reason, err := nonces.redeem(context.Background(), issued.Value, issued.Binding)
			if err == nil && reason == 0 {
				mu.Lock()
				accepted++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if accepted != 1 {
		t.Fatalf("nonce accepted %d times, want 1", accepted)
	}
}

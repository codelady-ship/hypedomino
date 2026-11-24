package integrity

import (
	"context"
	"crypto/rand"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/cluster/redistest"
)

func TestMain(m *testing.M) {
	os.Exit(redistest.Main(m))
}

func TestRedisNoncesAreSingleUseBoundAndExpire(t *testing.T) {
	client := redistest.Client(t)
	ctx := context.Background()
	clock := newTestClock()
	store := NewRedisNonceStore(client, "hd:test:"+t.Name()+":", clock)
	nonces, err := NewNonces(NonceConfig{TTL: 2 * time.Second}, store, clock, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	issued, err := nonces.Issue(ctx, sessionBinding())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put(ctx, issued); !errors.Is(err, ErrNonceExists) {
		t.Fatalf("second put: %v", err)
	}
	other := sessionBinding()
	other.Purpose = PurposePurchase
	if reason, err := nonces.redeem(ctx, issued.Value, other); reason != ReasonNonceBindingMismatch || err != nil {
		t.Fatalf("other purpose: %v %v", reason, err)
	}
	if reason, err := nonces.redeem(ctx, issued.Value, sessionBinding()); reason != ReasonNonceUnknown || err != nil {
		t.Fatalf("redeemed twice: %v %v", reason, err)
	}
	fresh, err := nonces.Issue(ctx, sessionBinding())
	if err != nil {
		t.Fatal(err)
	}
	if reason, err := nonces.redeem(ctx, fresh.Value, sessionBinding()); reason != 0 || err != nil {
		t.Fatalf("fresh nonce: %v %v", reason, err)
	}
	late, err := nonces.Issue(ctx, sessionBinding())
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(2500 * time.Millisecond)
	if reason, err := nonces.redeem(ctx, late.Value, sessionBinding()); reason != ReasonNonceUnknown || err != nil {
		t.Fatalf("expired nonce: %v %v", reason, err)
	}
}

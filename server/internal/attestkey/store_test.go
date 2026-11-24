package attestkey_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/attestkey"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/integrity"
)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

func TestAttestedKeysBelongToOneAccountAndOnlyCountUp(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	ctx := context.Background()
	var ayla, bora string
	for _, id := range []*string{&ayla, &bora} {
		if err := pool.QueryRow(ctx, `insert into accounts (created_at) values (now()) returning id`).Scan(id); err != nil {
			t.Fatal(err)
		}
	}
	store := attestkey.NewStore(pool)
	key := integrity.AttestedKey{KeyID: []byte("key-1"), PublicKey: []byte("public"), Receipt: []byte("receipt"), Development: false}
	at := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	if err := store.Save(ctx, ayla, key, at); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, bora, key, at); !errors.Is(err, attestkey.ErrKeyExists) {
		t.Fatalf("same key for another account: %v", err)
	}
	got, err := store.Key(ctx, ayla, key.KeyID)
	if err != nil || !bytes.Equal(got.PublicKey, key.PublicKey) || !bytes.Equal(got.Receipt, key.Receipt) || got.Counter != 0 || got.Development {
		t.Fatalf("key %+v %v", got, err)
	}
	if _, err := store.Key(ctx, bora, key.KeyID); !errors.Is(err, attestkey.ErrKeyNotFound) {
		t.Fatalf("another account read the key: %v", err)
	}
	if err := store.Advance(ctx, ayla, key.KeyID, 3); err != nil {
		t.Fatal(err)
	}
	for _, counter := range []uint32{3, 2} {
		if err := store.Advance(ctx, ayla, key.KeyID, counter); !errors.Is(err, attestkey.ErrStaleCounter) {
			t.Fatalf("counter %d: %v", counter, err)
		}
	}
	if err := store.Advance(ctx, bora, key.KeyID, 9); !errors.Is(err, attestkey.ErrStaleCounter) {
		t.Fatalf("another account advanced the key: %v", err)
	}
	if got, _ := store.Key(ctx, ayla, key.KeyID); got.Counter != 3 {
		t.Fatalf("counter %d", got.Counter)
	}
}

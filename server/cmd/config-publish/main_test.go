package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/shop"
)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

func TestPublishValidatesAndStoresTheSnapshot(t *testing.T) {
	url, pool := dbtest.Fresh(t)
	ctx := context.Background()
	var out bytes.Buffer
	if err := run(ctx, url, shop.Domain, "../../../docs/economy/examples/shop.json", &out); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), "shop version ") {
		t.Fatalf("output %q", out.String())
	}
	var count int
	if err := pool.QueryRow(ctx, `select count(*) from config_snapshots where domain = 'shop'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("stored snapshots %d %v", count, err)
	}
	broken := filepath.Join(t.TempDir(), "broken.json")
	if err := os.WriteFile(broken, []byte(`{"schema":1,"offers":[{"id":"x"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run(ctx, url, shop.Domain, broken, &out); !errors.Is(err, shop.ErrInvalid) {
		t.Fatalf("invalid catalog: %v", err)
	}
	if err := run(ctx, url, "weather", "../../../docs/economy/examples/shop.json", &out); !errors.Is(err, errDomain) {
		t.Fatalf("unknown domain: %v", err)
	}
	if err := run(ctx, url, shop.Domain, filepath.Join(t.TempDir(), "missing.json"), &out); err == nil {
		t.Fatal("missing file published")
	}
	if err := pool.QueryRow(ctx, `select count(*) from config_snapshots`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("snapshots after refusals %d %v", count, err)
	}
}

package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/contentpublish"
)

func TestDryRunPrintsThePlanAndNeedsNoStoreCredentials(t *testing.T) {
	dir := t.TempDir()
	for _, file := range []string{"catalog_2.0.bin", "a_0123456789abcdef0123456789abcdef.bundle"} {
		if err := os.WriteFile(filepath.Join(dir, file), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	target := contentpublish.Target{Environment: "dev", Platform: "iOS", CatalogVersion: "2.0", StoreURL: "http://localhost:9000/content-dev"}
	var out bytes.Buffer
	if err := run(context.Background(), target, dir, true, func(string) string { return "" }, &out); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[1], "addressables/iOS/catalog_2.0.bin") {
		t.Fatalf("output %q", out.String())
	}
}

func TestARealRunWithoutCredentialsFails(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "catalog_2.0.bin"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	target := contentpublish.Target{Environment: "dev", Platform: "iOS", CatalogVersion: "2.0", StoreURL: "http://localhost:9000/content-dev"}
	if err := run(context.Background(), target, dir, false, func(string) string { return "" }, &bytes.Buffer{}); err == nil {
		t.Fatal("expected a config error")
	}
}

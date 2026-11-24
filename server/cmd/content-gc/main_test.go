package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/contentpublish"
	"github.com/Alievs-corp/hypedomino/server/internal/objectstore"
)

const (
	liveBundle   = "addressables/Android/live_0123456789abcdef0123456789abcdef.bundle"
	orphanBundle = "addressables/Android/orphan_0123456789abcdef0123456789abcdef.bundle"
	liveCatalog  = "addressables/Android/catalog_1.4.0-10.bin"
	oldCatalog   = "addressables/Android/catalog_1.2.0-3.bin"
)

type bucket struct {
	bodies  map[string][]byte
	deleted []string
}

func (b *bucket) List(_ context.Context, prefix string, now time.Time) ([]objectstore.Entry, error) {
	var entries []objectstore.Entry
	for key := range b.bodies {
		if strings.HasPrefix(key, prefix) {
			entries = append(entries, objectstore.Entry{Key: key, LastModified: now.Add(-48 * time.Hour)})
		}
	}
	return entries, nil
}

func (b *bucket) Open(_ context.Context, key string, _ time.Time) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(b.bodies[key])), nil
}

func (b *bucket) Delete(_ context.Context, key string, _ time.Time) error {
	b.deleted = append(b.deleted, key)
	return nil
}

func catalogNaming(key string) []byte {
	name := strings.TrimPrefix(key, "addressables/Android/")
	return append(binary.LittleEndian.AppendUint32(nil, uint32(len(name))), name...)
}

func newBucket() *bucket {
	return &bucket{bodies: map[string][]byte{
		liveCatalog:  catalogNaming(liveBundle),
		oldCatalog:   catalogNaming(orphanBundle),
		liveBundle:   nil,
		orphanBundle: nil,
	}}
}

func retention() contentpublish.Retention {
	return contentpublish.Retention{
		Target:        contentpublish.Target{Environment: "dev", Platform: "Android", CatalogVersion: "1.4.0-10", StoreURL: "http://localhost:9000/content-dev"},
		MinAppVersion: "1.4.0",
		Grace:         defaultGrace,
	}
}

func TestADryRunReportsAndDeletesNothing(t *testing.T) {
	store := newBucket()
	var out bytes.Buffer
	if err := run(context.Background(), store, retention(), false, &out, time.Now()); err != nil {
		t.Fatal(err)
	}
	if len(store.deleted) != 0 {
		t.Fatalf("a dry run deleted %v", store.deleted)
	}
	for _, want := range []string{
		"catalog unsupported  " + oldCatalog,
		"keep referenced  " + liveBundle,
		"unreferenced  " + orphanBundle,
		"catalogs: 1 in scope, 1 unsupported; bundles: 1 referenced, 0 young, 1 unreferenced, 0 referenced but missing",
		"dry run: 1 bundles would be deleted",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output lacks %q:\n%s", want, out.String())
		}
	}
}

func TestDeleteRemovesOnlyTheUnreferencedBundles(t *testing.T) {
	store := newBucket()
	var out bytes.Buffer
	if err := run(context.Background(), store, retention(), true, &out, time.Now()); err != nil {
		t.Fatal(err)
	}
	if len(store.deleted) != 1 || store.deleted[0] != orphanBundle {
		t.Fatalf("deleted %v", store.deleted)
	}
	if !strings.Contains(out.String(), "deleted 1 of 1 unreferenced bundles") {
		t.Fatalf("output %s", out.String())
	}
}

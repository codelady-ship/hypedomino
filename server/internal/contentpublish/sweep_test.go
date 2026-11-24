package contentpublish_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"slices"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/contentpublish"
	"github.com/Alievs-corp/hypedomino/server/internal/objectstore"
)

var sweepNow = time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)

type memoryObject struct {
	body     []byte
	modified time.Time
	broken   bool
}

type memoryBucket struct {
	objects map[string]memoryObject
	deleted []string
}

func (m *memoryBucket) List(_ context.Context, prefix string, _ time.Time) ([]objectstore.Entry, error) {
	var entries []objectstore.Entry
	for key, object := range m.objects {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			entries = append(entries, objectstore.Entry{Key: key, LastModified: object.modified})
		}
	}
	return entries, nil
}

func (m *memoryBucket) Open(_ context.Context, key string, _ time.Time) (io.ReadCloser, error) {
	object, ok := m.objects[key]
	if !ok {
		return nil, objectstore.ErrNotFound
	}
	if object.broken {
		return nil, objectstore.ErrUnavailable
	}
	return io.NopCloser(bytes.NewReader(object.body)), nil
}

func (m *memoryBucket) Delete(_ context.Context, key string, _ time.Time) error {
	m.deleted = append(m.deleted, key)
	delete(m.objects, key)
	return nil
}

func bundle(name string) string {
	return name + "_" + hash + ".bundle"
}

func androidKey(file string) string {
	return contentpublish.Key("Android", file)
}

func lengthPrefixed(text string) []byte {
	out := binary.LittleEndian.AppendUint32(nil, uint32(len(text)))
	return append(out, text...)
}

func binaryCatalog(bundles ...string) []byte {
	out := []byte{0x07, 0x00, 0x00, 0x00, 0xff}
	out = append(out, lengthPrefixed("{HypeDomino.Content.ContentUrls.RemoteLoadPath}")...)
	for _, name := range bundles {
		out = append(out, lengthPrefixed(name)...)
		out = append(out, 0x2a, 0x00, 0x00, 0x00)
	}
	return out
}

func old(body []byte) memoryObject {
	return memoryObject{body: body, modified: sweepNow.Add(-30 * 24 * time.Hour)}
}

func retention(active string) contentpublish.Retention {
	return contentpublish.Retention{Target: target(active), MinAppVersion: "1.4.0", Grace: 24 * time.Hour}
}

func TestSweepKeepsWhatASupportedCatalogNeeds(t *testing.T) {
	cases := []struct {
		name         string
		objects      map[string]memoryObject
		active       string
		referenced   []string
		young        []string
		unreferenced []string
		missing      []string
		unsupported  []string
	}{
		{
			name: "a bundle shared with an old supported catalog is kept",
			objects: map[string]memoryObject{
				androidKey("catalog_1.4.0-10.bin"): old(binaryCatalog(bundle("shared"), bundle("first"))),
				androidKey("catalog_1.5.0-20.bin"): old(binaryCatalog(bundle("shared"), bundle("second"))),
				androidKey(bundle("shared")):       old(nil),
				androidKey(bundle("first")):        old(nil),
				androidKey(bundle("second")):       old(nil),
			},
			active:     "1.5.0-20",
			referenced: []string{androidKey(bundle("first")), androidKey(bundle("second")), androidKey(bundle("shared"))},
		},
		{
			name: "a bundle only unsupported catalogs name is deleted and no catalog file is",
			objects: map[string]memoryObject{
				androidKey("catalog_1.2.0-3.bin"):  old(binaryCatalog(bundle("retired"), bundle("shared"))),
				androidKey("catalog_1.2.0-3.hash"): old(nil),
				androidKey("catalog_1.3.9-8.bin"):  old(binaryCatalog(bundle("retired"))),
				androidKey("catalog_1.4.0-10.bin"): old(binaryCatalog(bundle("shared"))),
				androidKey(bundle("retired")):      old(nil),
				androidKey(bundle("shared")):       old(nil),
			},
			active:       "1.4.0-10",
			referenced:   []string{androidKey(bundle("shared"))},
			unreferenced: []string{androidKey(bundle("retired"))},
			unsupported:  []string{androidKey("catalog_1.2.0-3.bin"), androidKey("catalog_1.3.9-8.bin")},
		},
		{
			name: "an unreferenced bundle younger than the grace period is kept",
			objects: map[string]memoryObject{
				androidKey("catalog_1.4.0-10.bin"): old(binaryCatalog(bundle("live"))),
				androidKey(bundle("live")):         old(nil),
				androidKey(bundle("uploading")):    {modified: sweepNow.Add(-time.Hour)},
				androidKey(bundle("orphan")):       old(nil),
			},
			active:       "1.4.0-10",
			referenced:   []string{androidKey(bundle("live"))},
			young:        []string{androidKey(bundle("uploading"))},
			unreferenced: []string{androidKey(bundle("orphan"))},
		},
		{
			name: "the active catalog is in scope even below the minimum app version",
			objects: map[string]memoryObject{
				androidKey("catalog_1.3.0-9.bin"): old(binaryCatalog(bundle("rolled-back"))),
				androidKey(bundle("rolled-back")): old(nil),
			},
			active:     "1.3.0-9",
			referenced: []string{androidKey(bundle("rolled-back"))},
		},
		{
			name: "a catalog version without an app version is kept in scope",
			objects: map[string]memoryObject{
				androidKey("catalog_1.4.0-10.bin"): old(binaryCatalog(bundle("live"))),
				androidKey("catalog_hotfix.bin"):   old(binaryCatalog(bundle("manual"))),
				androidKey(bundle("live")):         old(nil),
				androidKey(bundle("manual")):       old(nil),
			},
			active:     "1.4.0-10",
			referenced: []string{androidKey(bundle("live")), androidKey(bundle("manual"))},
		},
		{
			name: "a referenced bundle absent from the store is reported",
			objects: map[string]memoryObject{
				androidKey("catalog_1.4.0-10.bin"):                  old(binaryCatalog(bundle("live"), bundle("lost"))),
				androidKey(bundle("live")):                          old(nil),
				contentpublish.Key("iOS", bundle("other-platform")): old(nil),
			},
			active:     "1.4.0-10",
			referenced: []string{androidKey(bundle("live"))},
			missing:    []string{androidKey(bundle("lost"))},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bucket := &memoryBucket{objects: c.objects}
			sweep, err := contentpublish.PlanSweep(context.Background(), bucket, retention(c.active), sweepNow)
			if err != nil {
				t.Fatal(err)
			}
			if len(bucket.deleted) != 0 {
				t.Fatalf("planning deleted %v", bucket.deleted)
			}
			var unsupported []string
			for _, catalog := range sweep.Catalogs {
				if !catalog.Supported {
					unsupported = append(unsupported, catalog.Key)
				}
			}
			for label, pair := range map[string][2][]string{
				"referenced":   {sweep.Referenced, c.referenced},
				"young":        {sweep.Young, c.young},
				"unreferenced": {sweep.Unreferenced, c.unreferenced},
				"missing":      {sweep.Missing, c.missing},
				"unsupported":  {unsupported, c.unsupported},
			} {
				if !slices.Equal(pair[0], pair[1]) {
					t.Errorf("%s %v, want %v", label, pair[0], pair[1])
				}
			}
			deleted, err := contentpublish.DeleteUnreferenced(context.Background(), bucket, sweep, sweepNow)
			if err != nil || deleted != len(c.unreferenced) || !slices.Equal(bucket.deleted, c.unreferenced) {
				t.Fatalf("deleted %d %v %v", deleted, bucket.deleted, err)
			}
		})
	}
}

func TestSweepRefusesToGuess(t *testing.T) {
	live := map[string]memoryObject{
		androidKey("catalog_1.4.0-10.bin"): old(binaryCatalog(bundle("live"))),
		androidKey(bundle("live")):         old(nil),
		androidKey(bundle("orphan")):       old(nil),
	}
	with := func(key string, object memoryObject) map[string]memoryObject {
		objects := map[string]memoryObject{key: object}
		for k, v := range live {
			objects[k] = v
		}
		return objects
	}
	cases := []struct {
		name      string
		objects   map[string]memoryObject
		retention contentpublish.Retention
		want      error
	}{
		{"an unreadable catalog in scope", with(androidKey("catalog_1.5.0-1.bin"), memoryObject{broken: true}), retention("1.4.0-10"), contentpublish.ErrCatalog},
		{"a catalog in scope that names no bundle", with(androidKey("catalog_1.5.0-1.bin"), old([]byte("not a catalog"))), retention("1.4.0-10"), contentpublish.ErrCatalog},
		{"a catalog file with an invalid version", with(androidKey("catalog_.bin"), old(nil)), retention("1.4.0-10"), contentpublish.ErrCatalog},
		{"an active catalog that is not in the store", live, retention("1.6.0-1"), contentpublish.ErrCatalog},
		{"a store url of another environment", live, contentpublish.Retention{Target: contentpublish.Target{Environment: "production", Platform: "Android", CatalogVersion: "1.4.0-10", StoreURL: "https://store.example/content-staging"}, MinAppVersion: "1.4.0", Grace: time.Hour}, contentpublish.ErrEnvironment},
		{"a minimum app version that is not a version", live, contentpublish.Retention{Target: target("1.4.0-10"), MinAppVersion: "latest", Grace: time.Hour}, contentpublish.ErrRetention},
		{"no grace period", live, contentpublish.Retention{Target: target("1.4.0-10"), MinAppVersion: "1.4.0"}, contentpublish.ErrRetention},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bucket := &memoryBucket{objects: c.objects}
			if _, err := contentpublish.PlanSweep(context.Background(), bucket, c.retention, sweepNow); !errors.Is(err, c.want) {
				t.Fatalf("error %v, want %v", err, c.want)
			}
			if len(bucket.deleted) != 0 {
				t.Fatalf("deleted %v", bucket.deleted)
			}
		})
	}
}

func TestBundlesInCatalogReadsLengthPrefixedNamesOnly(t *testing.T) {
	catalog := append([]byte("RemoteLoadPathgarbage"), lengthPrefixed(bundle("tiles_assets_all"))...)
	catalog = append(catalog, lengthPrefixed(bundle("tiles_assets_all"))...)
	catalog = append(catalog, []byte("x"+bundle("unprefixed"))...)
	catalog = append(catalog, lengthPrefixed(bundle("avatar.fox"))...)
	got := contentpublish.BundlesInCatalog(catalog)
	want := []string{bundle("avatar.fox"), bundle("tiles_assets_all")}
	if !slices.Equal(got, want) {
		t.Fatalf("bundles %v, want %v", got, want)
	}
}

package contentpublish_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/contentpublish"
	"github.com/Alievs-corp/hypedomino/server/internal/objectstore"
)

const hash = "0123456789abcdef0123456789abcdef"

type fakeStore struct {
	mutex   sync.Mutex
	bodies  map[string][]byte
	headers map[string]http.Header
	puts    []string
}

func newFakeStore() (*fakeStore, *objectstore.Store) {
	fake := &fakeStore{bodies: map[string][]byte{}, headers: map[string]http.Header{}}
	server := httptest.NewServer(fake)
	store, err := objectstore.New(objectstore.Config{BaseURL: server.URL + "/content-staging", Region: "auto", AccessKey: "a", SecretKey: "s", URLTTL: time.Minute})
	if err != nil {
		panic(err)
	}
	return fake, store
}

func (f *fakeStore) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	key := strings.TrimPrefix(r.URL.Path, "/content-staging/")
	switch r.Method {
	case http.MethodPut:
		body, _ := io.ReadAll(r.Body)
		f.bodies[key] = body
		f.headers[key] = r.Header.Clone()
		f.puts = append(f.puts, key)
	case http.MethodGet:
		body, ok := f.bodies[key]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write(body)
	}
}

func buildFolder(t *testing.T, version string, files ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, file := range files {
		if err := os.WriteFile(filepath.Join(dir, file), []byte("content of "+file), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func target(version string) contentpublish.Target {
	return contentpublish.Target{Environment: "staging", Platform: "Android", CatalogVersion: version, StoreURL: "https://store.example/content-staging"}
}

func TestPlanSetsCachePolicyAndPutsTheCatalogLast(t *testing.T) {
	dir := buildFolder(t, "1.0", "catalog_1.0.bin", "catalog_1.0.hash", "tiles_assets_all_"+hash+".bundle", "a_assets_all_"+hash+".bundle")
	plan, err := contentpublish.Plan(dir, target("1.0"))
	if err != nil {
		t.Fatal(err)
	}
	if len(plan) != 4 || plan[3].File != "catalog_1.0.bin" {
		t.Fatalf("plan %+v", plan)
	}
	for _, upload := range plan {
		bundle := strings.HasSuffix(upload.File, ".bundle")
		if bundle && upload.CacheControl != contentpublish.ImmutableCacheControl {
			t.Errorf("%s is %s", upload.File, upload.CacheControl)
		}
		if !bundle && upload.CacheControl != contentpublish.ShortCacheControl {
			t.Errorf("%s is %s", upload.File, upload.CacheControl)
		}
		if upload.Key != "addressables/Android/"+upload.File {
			t.Errorf("key %s", upload.Key)
		}
	}
}

func TestPlanRefusesABadBuildOrTarget(t *testing.T) {
	cases := map[string]struct {
		files  []string
		target contentpublish.Target
		want   error
	}{
		"bundle without a hash":      {[]string{"catalog_1.0.bin", "tiles.bundle"}, target("1.0"), contentpublish.ErrLayout},
		"catalog missing":            {[]string{"a_" + hash + ".bundle"}, target("1.0"), contentpublish.ErrLayout},
		"another catalog":            {[]string{"catalog_1.0.bin", "catalog_0.9.bin"}, target("1.0"), contentpublish.ErrLayout},
		"unknown file":               {[]string{"catalog_1.0.bin", "notes.txt"}, target("1.0"), contentpublish.ErrLayout},
		"bad platform":               {[]string{"catalog_1.0.bin"}, contentpublish.Target{Environment: "staging", Platform: "Web", CatalogVersion: "1.0", StoreURL: "https://x/content-staging"}, contentpublish.ErrLayout},
		"bad version":                {[]string{"catalog_1.0.bin"}, target("../1.0"), contentpublish.ErrLayout},
		"unknown environment":        {[]string{"catalog_1.0.bin"}, contentpublish.Target{Environment: "qa", Platform: "iOS", CatalogVersion: "1.0", StoreURL: "https://x/qa"}, contentpublish.ErrEnvironment},
		"store of another env":       {[]string{"catalog_1.0.bin"}, contentpublish.Target{Environment: "staging", Platform: "iOS", CatalogVersion: "1.0", StoreURL: "https://x/content-production"}, contentpublish.ErrEnvironment},
		"store names no environment": {[]string{"catalog_1.0.bin"}, contentpublish.Target{Environment: "staging", Platform: "iOS", CatalogVersion: "1.0", StoreURL: "https://x/content"}, contentpublish.ErrEnvironment},
	}
	for name, c := range cases {
		dir := buildFolder(t, "1.0", c.files...)
		if _, err := contentpublish.Plan(dir, c.target); !errors.Is(err, c.want) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestAStoreURLMustNameItsOwnEnvironmentOnly(t *testing.T) {
	cases := []struct {
		environment, url string
		want             error
	}{
		{"staging", "https://store.example/hypedomino-content-staging", nil},
		{"production", "https://hypedomino-content-PRODUCTION.store.example", nil},
		{"staging", "https://store.example/hypedomino-content-production", contentpublish.ErrEnvironment},
		{"production", "https://store.example/hypedomino-content", contentpublish.ErrEnvironment},
		{"staging", "https://staging.store.example/hypedomino-content-dev", contentpublish.ErrEnvironment},
	}
	for _, c := range cases {
		if err := contentpublish.CheckStoreURL(c.environment, c.url); !errors.Is(err, c.want) {
			t.Errorf("%s %s: %v, want %v", c.environment, c.url, err, c.want)
		}
	}
}

func TestPublishKeepsOlderCatalogsWorkingAndRefusesARepublish(t *testing.T) {
	ctx := context.Background()
	fake, store := newFakeStore()
	now := time.Now()
	shared := "shared_assets_all_" + hash + ".bundle"
	added := "new_assets_all_" + "fedcba9876543210fedcba9876543210" + ".bundle"

	first := buildFolder(t, "1.0", "catalog_1.0.bin", shared)
	plan, err := contentpublish.Plan(first, target("1.0"))
	if err != nil {
		t.Fatal(err)
	}
	if sent, err := contentpublish.Publish(ctx, store, first, plan, now); err != nil || sent != 2 {
		t.Fatalf("first %d %v", sent, err)
	}
	if last := fake.puts[len(fake.puts)-1]; last != "addressables/Android/catalog_1.0.bin" {
		t.Fatalf("the catalog was not last: %v", fake.puts)
	}
	if got := fake.headers["addressables/Android/"+shared].Get("Cache-Control"); got != contentpublish.ImmutableCacheControl {
		t.Fatalf("bundle cache control %q", got)
	}
	if got := fake.headers["addressables/Android/catalog_1.0.bin"].Get("Cache-Control"); got != contentpublish.ShortCacheControl {
		t.Fatalf("catalog cache control %q", got)
	}

	second := buildFolder(t, "1.1", "catalog_1.1.bin", shared, added)
	plan, err = contentpublish.Plan(second, target("1.1"))
	if err != nil {
		t.Fatal(err)
	}
	if sent, err := contentpublish.Publish(ctx, store, second, plan, now); err != nil || sent != 2 {
		t.Fatalf("second %d %v", sent, err)
	}
	for _, key := range []string{"addressables/Android/catalog_1.0.bin", "addressables/Android/catalog_1.1.bin", "addressables/Android/" + shared, "addressables/Android/" + added} {
		if _, ok := fake.bodies[key]; !ok {
			t.Errorf("%s is missing", key)
		}
	}
	if string(fake.bodies["addressables/Android/catalog_1.0.bin"]) != "content of catalog_1.0.bin" {
		t.Error("the old catalog changed")
	}

	puts := len(fake.puts)
	if _, err := contentpublish.Publish(ctx, store, second, plan, now); !errors.Is(err, contentpublish.ErrAlreadyLive) || len(fake.puts) != puts {
		t.Fatalf("republish %v, %d new puts", err, len(fake.puts)-puts)
	}
}

type failingOpen struct {
	*objectstore.Store
	failKey string
	puts    int
}

func (f *failingOpen) Open(ctx context.Context, key string, now time.Time) (io.ReadCloser, error) {
	if strings.HasSuffix(key, f.failKey) {
		return nil, objectstore.ErrUnavailable
	}
	return f.Store.Open(ctx, key, now)
}

func (f *failingOpen) PutObject(ctx context.Context, key string, object objectstore.Object, now time.Time) error {
	f.puts++
	return f.Store.PutObject(ctx, key, object, now)
}

func TestPublishStopsWhenTheStoreCannotAnswerAnExistenceCheck(t *testing.T) {
	dir := buildFolder(t, "1.0", "catalog_1.0.bin", "a_"+hash+".bundle")
	plan, err := contentpublish.Plan(dir, target("1.0"))
	if err != nil {
		t.Fatal(err)
	}
	for name, failKey := range map[string]string{"catalog": "catalog_1.0.bin", "bundle": "a_" + hash + ".bundle"} {
		_, store := newFakeStore()
		wrapped := &failingOpen{Store: store, failKey: failKey}
		sent, err := contentpublish.Publish(context.Background(), wrapped, dir, plan, time.Now())
		if !errors.Is(err, objectstore.ErrUnavailable) || errors.Is(err, contentpublish.ErrAlreadyLive) || sent != 0 || wrapped.puts != 0 {
			t.Errorf("%s: sent %d puts %d err %v", name, sent, wrapped.puts, err)
		}
	}
}

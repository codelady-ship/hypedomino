package deletionlog_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/account"
	"github.com/Alievs-corp/hypedomino/server/internal/deletionlog"
	"github.com/Alievs-corp/hypedomino/server/internal/objectstore"
)

type bucket struct {
	objects map[string][]byte
	types   map[string]string
}

func newBucket() *bucket {
	return &bucket{objects: map[string][]byte{}, types: map[string]string{}}
}

func (b *bucket) Put(_ context.Context, key, contentType string, body []byte, _ time.Time) error {
	b.objects[key] = append([]byte(nil), body...)
	b.types[key] = contentType
	return nil
}

func (b *bucket) List(_ context.Context, prefix string, _ time.Time) ([]objectstore.Entry, error) {
	var entries []objectstore.Entry
	for key := range b.objects {
		if strings.HasPrefix(key, prefix) {
			entries = append(entries, objectstore.Entry{Key: key})
		}
	}
	return entries, nil
}

func (b *bucket) Open(_ context.Context, key string, _ time.Time) (io.ReadCloser, error) {
	body, ok := b.objects[key]
	if !ok {
		return nil, objectstore.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(body)), nil
}

func (b *bucket) Delete(_ context.Context, key string, _ time.Time) error {
	delete(b.objects, key)
	return nil
}

var (
	day   = time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	early = account.Deletion{Account: "11111111-1111-4111-8111-111111111111", Tombstone: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", At: day.Add(time.Hour + 123456*time.Microsecond)}
	late  = account.Deletion{Account: "22222222-2222-4222-8222-222222222222", Tombstone: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", At: day.Add(26 * time.Hour)}
)

func TestAppendWritesOneObjectPerDeletionWithIdsAndTimeOnly(t *testing.T) {
	store := newBucket()
	log := deletionlog.New(store, time.Now)
	ctx := context.Background()
	for range 2 {
		if err := log.Append(ctx, early); err != nil {
			t.Fatal(err)
		}
	}
	key := "deletions/20261101T010000.123456Z_11111111-1111-4111-8111-111111111111.json"
	if len(store.objects) != 1 || deletionlog.Key(early) != key {
		t.Fatalf("objects %v, key %s", store.objects, deletionlog.Key(early))
	}
	want := `{"account_id":"11111111-1111-4111-8111-111111111111","tombstone_id":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","deleted_at":"2026-11-01T01:00:00.123456Z"}`
	if got := string(store.objects[key]); got != want || store.types[key] != "application/json" {
		t.Fatalf("body %s (%s), want %s", got, store.types[key], want)
	}
}

func TestSinceReadsEntriesFromTheRestorePointInOrder(t *testing.T) {
	store := newBucket()
	log := deletionlog.New(store, time.Now)
	ctx := context.Background()
	for _, d := range []account.Deletion{late, early} {
		if err := log.Append(ctx, d); err != nil {
			t.Fatal(err)
		}
	}
	all, err := log.Since(ctx, day)
	if err != nil || len(all) != 2 || all[0] != early || all[1] != late {
		t.Fatalf("since the day: %+v %v", all, err)
	}
	fromLate, err := log.Since(ctx, late.At)
	if err != nil || len(fromLate) != 1 || fromLate[0] != late {
		t.Fatalf("since the late entry: %+v %v", fromLate, err)
	}
	none, err := log.Since(ctx, late.At.Add(time.Microsecond))
	if err != nil || len(none) != 0 {
		t.Fatalf("after the last entry: %+v %v", none, err)
	}
}

func TestPruneDeletesEntriesOlderThanTheRetention(t *testing.T) {
	store := newBucket()
	log := deletionlog.New(store, time.Now)
	ctx := context.Background()
	for _, d := range []account.Deletion{early, late} {
		if err := log.Append(ctx, d); err != nil {
			t.Fatal(err)
		}
	}
	if pruned, err := log.Prune(ctx, late.At); err != nil || pruned != 1 {
		t.Fatalf("prune: %d %v", pruned, err)
	}
	if _, kept := store.objects[deletionlog.Key(late)]; !kept || len(store.objects) != 1 {
		t.Fatalf("objects after the prune: %v", store.objects)
	}
}

func TestMalformedEntriesStopTheReplay(t *testing.T) {
	ctx := context.Background()
	for name, objects := range map[string]map[string]string{
		"key without a time":  {"deletions/11111111-1111-4111-8111-111111111111.json": "{}"},
		"body of another key": {deletionlog.Key(early): `{"account_id":"22222222-2222-4222-8222-222222222222","tombstone_id":"bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb","deleted_at":"2026-11-01T01:00:00.123456Z"}`},
		"no tombstone":        {deletionlog.Key(early): `{"account_id":"11111111-1111-4111-8111-111111111111","deleted_at":"2026-11-01T01:00:00.123456Z"}`},
		"not json":            {deletionlog.Key(early): `deleted`},
	} {
		store := newBucket()
		for key, body := range objects {
			store.objects[key] = []byte(body)
		}
		if _, err := deletionlog.New(store, time.Now).Since(ctx, day); !errors.Is(err, deletionlog.ErrMalformed) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

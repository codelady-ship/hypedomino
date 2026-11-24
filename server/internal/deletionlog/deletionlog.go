package deletionlog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/account"
	"github.com/Alievs-corp/hypedomino/server/internal/objectstore"
)

const (
	Prefix      = "deletions/"
	keyTime     = "20060102T150405.000000Z"
	contentType = "application/json"
)

var ErrMalformed = errors.New("deletionlog: malformed entry")

type Bucket interface {
	Put(ctx context.Context, key, contentType string, body []byte, now time.Time) error
	List(ctx context.Context, prefix string, now time.Time) ([]objectstore.Entry, error)
	Open(ctx context.Context, key string, now time.Time) (io.ReadCloser, error)
	Delete(ctx context.Context, key string, now time.Time) error
}

type Log struct {
	bucket Bucket
	now    func() time.Time
}

type entry struct {
	Account   string    `json:"account_id"`
	Tombstone string    `json:"tombstone_id"`
	DeletedAt time.Time `json:"deleted_at"`
}

func New(bucket Bucket, now func() time.Time) *Log {
	return &Log{bucket: bucket, now: now}
}

func Key(d account.Deletion) string {
	return Prefix + d.At.UTC().Format(keyTime) + "_" + d.Account + ".json"
}

func (l *Log) Append(ctx context.Context, d account.Deletion) error {
	body, err := json.Marshal(entry{Account: d.Account, Tombstone: d.Tombstone, DeletedAt: d.At.UTC()})
	if err != nil {
		return err
	}
	return l.bucket.Put(ctx, Key(d), contentType, body, l.now())
}

func (l *Log) Since(ctx context.Context, since time.Time) ([]account.Deletion, error) {
	keys, err := l.keys(ctx, func(at time.Time) bool { return !at.Before(since) })
	if err != nil {
		return nil, err
	}
	deletions := make([]account.Deletion, 0, len(keys))
	for _, key := range keys {
		d, err := l.read(ctx, key)
		if err != nil {
			return nil, err
		}
		deletions = append(deletions, d)
	}
	return deletions, nil
}

func (l *Log) Prune(ctx context.Context, before time.Time) (int, error) {
	keys, err := l.keys(ctx, func(at time.Time) bool { return at.Before(before) })
	if err != nil {
		return 0, err
	}
	for pruned, key := range keys {
		if err := l.bucket.Delete(ctx, key, l.now()); err != nil {
			return pruned, fmt.Errorf("prune %s: %w", key, err)
		}
	}
	return len(keys), nil
}

func (l *Log) keys(ctx context.Context, wanted func(time.Time) bool) ([]string, error) {
	entries, err := l.bucket.List(ctx, Prefix, l.now())
	if err != nil {
		return nil, err
	}
	var keys []string
	for _, e := range entries {
		at, err := deletedAt(e.Key)
		if err != nil {
			return nil, err
		}
		if wanted(at) {
			keys = append(keys, e.Key)
		}
	}
	sort.Strings(keys)
	return keys, nil
}

func (l *Log) read(ctx context.Context, key string) (account.Deletion, error) {
	body, err := l.bucket.Open(ctx, key, l.now())
	if err != nil {
		return account.Deletion{}, fmt.Errorf("read %s: %w", key, err)
	}
	defer body.Close()
	var e entry
	if err := json.NewDecoder(body).Decode(&e); err != nil {
		return account.Deletion{}, fmt.Errorf("%w: %s: %v", ErrMalformed, key, err)
	}
	d := account.Deletion{Account: e.Account, Tombstone: e.Tombstone, At: e.DeletedAt.UTC()}
	if d.Account == "" || d.Tombstone == "" || Key(d) != key {
		return account.Deletion{}, fmt.Errorf("%w: %s does not match its body", ErrMalformed, key)
	}
	return d, nil
}

func deletedAt(key string) (time.Time, error) {
	stamp, _, found := strings.Cut(strings.TrimPrefix(key, Prefix), "_")
	if !found {
		return time.Time{}, fmt.Errorf("%w: key %s", ErrMalformed, key)
	}
	at, err := time.Parse(keyTime, stamp)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: key %s: %v", ErrMalformed, key, err)
	}
	return at, nil
}

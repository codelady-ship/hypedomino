package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/account"
	"github.com/Alievs-corp/hypedomino/server/internal/analytics"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/deletionlog"
	"github.com/Alievs-corp/hypedomino/server/internal/idtoken"
	"github.com/Alievs-corp/hypedomino/server/internal/objectstore"
)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

type bucket struct {
	objects map[string][]byte
	failPut error
}

func (b *bucket) Put(_ context.Context, key, _ string, body []byte, _ time.Time) error {
	if b.failPut != nil {
		return b.failPut
	}
	b.objects[key] = append([]byte(nil), body...)
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
	return io.NopCloser(bytes.NewReader(b.objects[key])), nil
}

func (b *bucket) Delete(_ context.Context, key string, _ time.Time) error {
	delete(b.objects, key)
	return nil
}

func signedUp(t *testing.T, accounts *account.Store, pool *pgxpool.Pool, subject, nickname string) string {
	t.Helper()
	ctx := context.Background()
	a, _, err := accounts.SignIn(ctx, idtoken.Identity{Provider: idtoken.ProviderGoogle, Subject: subject, Email: subject + "@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `insert into profiles (account_id, nickname, nickname_key) values ($1, $2, lower($2))`, a.ID, nickname); err != nil {
		t.Fatal(err)
	}
	return a.ID
}

func state(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var s string
	err := pool.QueryRow(context.Background(), `select concat_ws(' | ',
		(select string_agg(id::text || ':' || state || ':' || coalesce(deleted_at::text, ''), ',' order by id) from accounts),
		(select string_agg(provider || ':' || subject || ':' || account_id::text, ',' order by subject) from auth_identities),
		(select string_agg(nickname_key || ':' || account_id::text, ',' order by nickname_key) from profiles),
		(select string_agg(nickname_key || ':' || account_id::text, ',' order by nickname_key) from nickname_history),
		(select count(*)::text from deletion_log_pending))`).Scan(&s)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestRunErasesDueAccountsLogsThemAndPrunesOldEntries(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	accounts := account.New(pool)
	ctx := context.Background()
	ayla := signedUp(t, accounts, pool, "g-ayla", "Ayla")
	now := time.Date(2026, 11, 14, 3, 0, 0, 0, time.UTC)
	if _, err := accounts.RequestDeletion(ctx, ayla, now.Add(-31*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	stale := deletionlog.Key(account.Deletion{Account: "11111111-1111-4111-8111-111111111111", At: now.Add(-36 * 24 * time.Hour)})
	store := &bucket{objects: map[string][]byte{stale: nil}, failPut: errors.New("bucket unreachable")}
	log := deletionlog.New(store, func() time.Time { return now })
	events := analytics.NewStore(pool)
	ben := signedUp(t, accounts, pool, "g-ben", "Ben")
	for i, received := range []time.Time{now.Add(-defaultAnalyticsRetention - time.Hour), now.Add(-time.Hour)} {
		batch := analytics.Batch{AnalyticsID: "6f1c2a7e-3b4d-4e5f-8a9b-0c1d2e3f4a5b", Events: []analytics.Event{{Name: "app_open", ClientTimeMs: int64(i + 1)}}}
		if err := events.Record(ctx, ben, batch, received); err != nil {
			t.Fatal(err)
		}
	}
	config := settings{grace: defaultGrace, retention: defaultRetention, analyticsRetention: defaultAnalyticsRetention}
	var out bytes.Buffer
	if err := run(ctx, accounts, log, events, config, now, &out); err == nil || !strings.Contains(out.String(), "0 deletion(s) erased and logged") || !strings.Contains(out.String(), "1 raw analytics events older than 9480h0m0s deleted") {
		t.Fatalf("run with an unreachable log: %v %q", err, out.String())
	}
	var kept int64
	if err := pool.QueryRow(ctx, `select coalesce(sum(client_time_ms), 0) from analytics_events`).Scan(&kept); err != nil || kept != 2 {
		t.Fatalf("analytics events kept after the run: %d %v", kept, err)
	}
	if pending, err := accounts.UnloggedDeletions(ctx); err != nil || len(pending) != 1 || pending[0].Account != ayla {
		t.Fatalf("pending after the failed write: %+v %v", pending, err)
	}
	store.failPut = nil
	out.Reset()
	if err := run(ctx, accounts, log, events, config, now, &out); err != nil {
		t.Fatalf("run: %v %q", err, out.String())
	}
	if !strings.Contains(out.String(), "1 deletion(s) erased and logged") || !strings.Contains(out.String(), "1 log entries older than 840h0m0s pruned") || !strings.Contains(out.String(), "0 raw analytics events") {
		t.Fatalf("run output %q", out.String())
	}
	logged, err := log.Since(ctx, now.Add(-time.Hour))
	if err != nil || len(logged) != 1 || logged[0].Account != ayla || !logged[0].At.Equal(now) {
		t.Fatalf("log after the run: %+v %v", logged, err)
	}
	if _, kept := store.objects[stale]; kept || len(store.objects) != 1 {
		t.Fatalf("objects after the run: %v", store.objects)
	}
}

func TestReplayReappliesLoggedDeletionsAndASecondReplayChangesNothing(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	accounts := account.New(pool)
	ctx := context.Background()
	ayla := signedUp(t, accounts, pool, "g-ayla", "Ayla")
	signedUp(t, accounts, pool, "g-ben", "Ben")
	var tombstone, absent string
	if err := pool.QueryRow(ctx, `select gen_random_uuid()::text, gen_random_uuid()::text`).Scan(&tombstone, &absent); err != nil {
		t.Fatal(err)
	}
	restorePoint := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	log := deletionlog.New(&bucket{objects: map[string][]byte{}}, time.Now)
	for _, d := range []account.Deletion{
		{Account: absent, Tombstone: absent, At: restorePoint.Add(-time.Hour)},
		{Account: ayla, Tombstone: tombstone, At: restorePoint.Add(time.Hour)},
	} {
		if err := log.Append(ctx, d); err != nil {
			t.Fatal(err)
		}
	}
	before := state(t, pool)
	var out bytes.Buffer
	if err := replay(ctx, accounts, log, restorePoint, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "re-applied "+ayla+" as tombstone "+tombstone) || !strings.Contains(out.String(), "1 of 1 logged deletion(s)") {
		t.Fatalf("first replay output %q", out.String())
	}
	once := state(t, pool)
	if once == before || strings.Contains(once, ayla) || strings.Contains(once, "g-ayla") || !strings.Contains(once, "ayla:"+tombstone) {
		t.Fatalf("state after the first replay: %s", once)
	}
	out.Reset()
	if err := replay(ctx, accounts, log, restorePoint.Add(-2*time.Hour), &out); err != nil {
		t.Fatal(err)
	}
	if twice := state(t, pool); twice != once || !strings.Contains(out.String(), "0 of 2 logged deletion(s)") {
		t.Fatalf("second replay changed the state or reported work: %s\n%s\n%q", once, twice, out.String())
	}
}

func TestBadInvocationsAreUsageErrors(t *testing.T) {
	env := map[string]string{
		"HD_DATABASE_URL":            "postgres://localhost/hd",
		"HD_DELETION_LOG_URL":        "https://store.example/hd-deletion-log",
		"HD_DELETION_LOG_REGION":     "eu-central",
		"HD_DELETION_LOG_ACCESS_KEY": "key",
		"HD_DELETION_LOG_SECRET_KEY": "secret",
	}
	with := func(name, value string) func(string) string {
		return func(key string) string {
			if key == name {
				return value
			}
			return env[key]
		}
	}
	for name, c := range map[string]struct {
		args   []string
		getenv func(string) string
	}{
		"no command":                {nil, with("", "")},
		"unknown command":           {[]string{"purge"}, with("", "")},
		"run with arguments":        {[]string{"run", "now"}, with("", "")},
		"replay without since":      {[]string{"replay"}, with("", "")},
		"replay with a bad time":    {[]string{"replay", "-since", "yesterday"}, with("", "")},
		"no database":               {[]string{"run"}, with("HD_DATABASE_URL", "")},
		"no log store":              {[]string{"run"}, with("HD_DELETION_LOG_URL", "")},
		"no log secret":             {[]string{"replay", "-since", "2026-11-10T12:00:00Z"}, with("HD_DELETION_LOG_SECRET_KEY", "")},
		"bad grace":                 {[]string{"run"}, with("HD_ACCOUNT_DELETION_GRACE", "-1h")},
		"bad retention":             {[]string{"run"}, with("HD_DELETION_LOG_RETENTION", "35 days")},
		"bad analytics retention":   {[]string{"run"}, with("HD_ANALYTICS_RAW_RETENTION", "0s")},
		"short ban-evasion key":     {[]string{"run"}, with("HD_BAN_EVASION_KEY", strings.Repeat("k", 16))},
		"bad ban-evasion retention": {[]string{"run"}, with("HD_PERMANENT_BAN_EVASION_RETENTION", "-1h")},
	} {
		if err := dispatch(context.Background(), c.args, c.getenv, io.Discard, time.Now); !errors.Is(err, errUsage) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestTheBanEvasionHashIsOffWithoutAKeyAndKeepsPermanentBansForTheRetention(t *testing.T) {
	env := map[string]string{"HD_DATABASE_URL": "postgres://localhost/hd", "HD_DELETION_LOG_URL": "https://store.example/hd-deletion-log"}
	off, err := settingsFrom(func(name string) string { return env[name] })
	if err != nil || off.banEvasion != nil || off.banEvasionRetention != 26280*time.Hour {
		t.Fatalf("without a key: %+v %v", off, err)
	}
	env["HD_BAN_EVASION_KEY"] = "0123456789abcdef0123456789abcdef"
	env["HD_PERMANENT_BAN_EVASION_RETENTION"] = "720h"
	on, err := settingsFrom(func(name string) string { return env[name] })
	if err != nil || on.banEvasion == nil || on.banEvasion.PermanentRetention != 720*time.Hour {
		t.Fatalf("with a key: %+v %v", on, err)
	}
}

func TestRunHashesABannedAccountAndPurgesExpiredHashes(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	hasher, err := account.NewIdentityHasher("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	accounts := account.New(pool)
	accounts.EnableBanEvasion(account.BanEvasion{Hasher: hasher, PermanentRetention: defaultBanEvasionRetention})
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	cheater := signedUp(t, accounts, pool, "g-cheater", "Cheater")
	if _, err := pool.Exec(ctx, `update accounts set state = 'banned' where id = $1`, cheater); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `insert into account_sanctions (account_id, kind, category, starts_at) values ($1, 'permanent_ban', 'cheating', $2)`, cheater, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := accounts.RequestDeletion(ctx, cheater, now.Add(-31*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `insert into ban_evasion_hashes (identity_hash, category, ban_ends_at, expires_at) values ($1, 'other', $2, $2)`, hasher.Hash("google", "g-gone"), now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	log := deletionlog.New(&bucket{objects: map[string][]byte{}}, func() time.Time { return now })
	config := settings{grace: defaultGrace, retention: defaultRetention, analyticsRetention: defaultAnalyticsRetention}
	var out bytes.Buffer
	if err := run(ctx, accounts, log, analytics.NewStore(pool), config, now, &out); err != nil {
		t.Fatalf("run: %v %q", err, out.String())
	}
	if !strings.Contains(out.String(), "1 deletion(s) erased and logged") || !strings.Contains(out.String(), "1 expired ban-evasion hash(es) deleted") {
		t.Fatalf("run output %q", out.String())
	}
	var expires time.Time
	if err := pool.QueryRow(ctx, `select expires_at from ban_evasion_hashes where identity_hash = $1 and ban_ends_at is null`, hasher.Hash("google", "g-cheater")).Scan(&expires); err != nil || !expires.Equal(now.Add(defaultBanEvasionRetention)) {
		t.Fatalf("hash of the deleted cheater: %v %v", expires, err)
	}
	out.Reset()
	if err := run(ctx, accounts, log, analytics.NewStore(pool), config, now, &out); err != nil || !strings.Contains(out.String(), "0 expired ban-evasion hash(es) deleted") {
		t.Fatalf("second run: %v %q", err, out.String())
	}
}

package account_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/account"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/idtoken"
)

type recordedLog struct {
	failure  error
	appended []account.Deletion
}

func (l *recordedLog) Append(_ context.Context, d account.Deletion) error {
	if l.failure != nil {
		return l.failure
	}
	l.appended = append(l.appended, d)
	return nil
}

func signedUp(t *testing.T, store *account.Store, pool *pgxpool.Pool, subject, nickname string) account.Account {
	t.Helper()
	ctx := context.Background()
	a, _, err := store.SignIn(ctx, idtoken.Identity{Provider: idtoken.ProviderGoogle, Subject: subject, Email: subject + "@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `insert into profiles (account_id, nickname, nickname_key) values ($1, $2, lower($2))`, a.ID, nickname); err != nil {
		t.Fatal(err)
	}
	return a
}

func count(t *testing.T, pool *pgxpool.Pool, query string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), query, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestDeletionWaitsForTheGraceAndCanBeCancelled(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := account.New(pool)
	ctx := context.Background()
	ayla := signedUp(t, store, pool, "g-ayla", "Ayla")
	identity := idtoken.Identity{Provider: idtoken.ProviderGoogle, Subject: "g-ayla"}
	requestedAt := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `insert into push_devices (token, account_id, platform, registered_at) values ('push-1', $1, 'android', now())`, ayla.ID); err != nil {
		t.Fatal(err)
	}
	if at, err := store.RequestDeletion(ctx, ayla.ID, requestedAt); err != nil || !at.Equal(requestedAt) {
		t.Fatalf("request: %v %v", at, err)
	}
	if n := count(t, pool, `select count(*) from push_devices where account_id = $1`, ayla.ID); n != 0 {
		t.Fatalf("push devices after the request: %d", n)
	}
	pending, err := store.Lookup(ctx, identity)
	if !errors.Is(err, account.ErrDeletionPending) || pending.ID != ayla.ID || pending.DeletionAt == nil || !pending.DeletionAt.Equal(requestedAt) {
		t.Fatalf("lookup while pending: %+v %v", pending, err)
	}
	if deleted, err := store.DeleteDue(ctx, &recordedLog{}, requestedAt.Add(-time.Second), requestedAt); err != nil || deleted != 0 {
		t.Fatalf("delete before the grace: %d %v", deleted, err)
	}
	if err := store.CancelDeletion(ctx, ayla.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.CancelDeletion(ctx, ayla.ID); !errors.Is(err, account.ErrNoDeletionPending) {
		t.Fatalf("second cancel: %v", err)
	}
	if restored, _, err := store.SignIn(ctx, identity); err != nil || restored.ID != ayla.ID || restored.DeletionAt != nil {
		t.Fatalf("sign-in after cancel: %+v %v", restored, err)
	}
}

func TestBannedAccountsCannotRequestDeletion(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := account.New(pool)
	ctx := context.Background()
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	temporary := signedUp(t, store, pool, "g-temp", "Temp")
	if _, err := pool.Exec(ctx, `update accounts set banned_until = $2 where id = $1`, temporary.ID, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	permanent := signedUp(t, store, pool, "g-perm", "Perm")
	if _, err := pool.Exec(ctx, `update accounts set state = $2 where id = $1`, permanent.ID, account.StateBanned); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{temporary.ID, permanent.ID} {
		if _, err := store.RequestDeletion(ctx, id, now); !errors.Is(err, account.ErrBanned) {
			t.Fatalf("%s: %v", id, err)
		}
	}
}

func TestDeletionErasesPersonalDataAndKeepsSharedHistoryUnderATombstone(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := account.New(pool)
	ctx := context.Background()
	ayla := signedUp(t, store, pool, "g-ayla", "Ayla")
	ben := signedUp(t, store, pool, "g-ben", "Ben")
	if _, err := pool.Exec(ctx, `insert into friendships (account_id, friend_id, created_at) values ($1, $2, now()), ($2, $1, now())`, ayla.ID, ben.ID); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`insert into nickname_history (account_id, nickname, nickname_key) values ($1, 'Old', 'old')`,
		`insert into purchases (platform, transaction_id, account_id, offer_id, product_id, environment, state, purchased_at) values ('google', 'tx-1', $1, 'offer', 'product', 'production', 'granted', now())`,
		`insert into analytics_ids (analytics_id, account_id, first_seen_at) values ('6f1c2a7e-3b4d-4e5f-8a9b-0c1d2e3f4a5b', $1, now())`,
		`insert into analytics_events (analytics_id, name, label, value, client_time_ms, received_at) select analytics_id, 'app_open', '', 0, 1762300000000, now() from analytics_ids where account_id = $1`,
	} {
		if _, err := pool.Exec(ctx, statement, ayla.ID); err != nil {
			t.Fatalf("%s: %v", statement, err)
		}
	}
	requestedAt := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	if _, err := store.RequestDeletion(ctx, ayla.ID, requestedAt); err != nil {
		t.Fatal(err)
	}
	now := requestedAt.Add(30 * 24 * time.Hour)
	log := &recordedLog{}
	if deleted, err := store.DeleteDue(ctx, log, now.Add(-30*24*time.Hour), now); err != nil || deleted != 1 {
		t.Fatalf("delete due: %d %v", deleted, err)
	}
	if deleted, err := store.DeleteDue(ctx, log, now, now); err != nil || deleted != 0 {
		t.Fatalf("second run: %d %v", deleted, err)
	}
	if len(log.appended) != 1 || log.appended[0].Account != ayla.ID || !log.appended[0].At.Equal(now) {
		t.Fatalf("deletion log: %+v", log.appended)
	}
	if n := count(t, pool, `select count(*) from accounts where id = $1 and state = 'deleted' and deleted_at = $2`, log.appended[0].Tombstone, now); n != 1 {
		t.Fatalf("logged tombstone %s is not the deleted account row", log.appended[0].Tombstone)
	}
	checks := map[string]int{
		`select count(*) from accounts where id in ($1, $2) and state = 'active'`:                                                                                   1,
		`select count(*) from auth_identities where subject = 'g-ayla' or email like 'g-ayla%' or account_id = $1 or $2 = ''`:                                       0,
		`select count(*) from profiles where nickname = 'Ayla' or account_id = $1 or $2 = ''`:                                                                       0,
		`select count(*) from nickname_history where nickname = 'Old' or account_id = $1 or $2 = ''`:                                                                0,
		`select count(*) from friendships where account_id in ($1, $2)`:                                                                                             0,
		`select count(*) from analytics_ids where analytics_id = '6f1c2a7e-3b4d-4e5f-8a9b-0c1d2e3f4a5b' or $1 = '' or $2 = ''`:                                      0,
		`select count(*) from analytics_events where analytics_id = '6f1c2a7e-3b4d-4e5f-8a9b-0c1d2e3f4a5b' or $1 = '' or $2 = ''`:                                   0,
		`select count(*) from purchases p join accounts a on a.id = p.account_id where a.state = 'deleted' and a.deleted_at is not null and a.id not in ($1, $2)`:   1,
		`select count(*) from nickname_history h join accounts a on a.id = h.account_id where h.nickname = 'Ayla' and a.state = 'deleted' and a.id not in ($1, $2)`: 1,
	}
	for query, want := range checks {
		if got := count(t, pool, query, ayla.ID, ben.ID); got != want {
			t.Fatalf("%s: %d, want %d", query, got, want)
		}
	}
	fresh, created, err := store.SignIn(ctx, idtoken.Identity{Provider: idtoken.ProviderGoogle, Subject: "g-ayla"})
	if err != nil || !created || fresh.ID == ayla.ID {
		t.Fatalf("sign-in after deletion: %+v %v %v", fresh, created, err)
	}
}

func dueForDeletion(t *testing.T, store *account.Store, id string, requestedAt time.Time) {
	t.Helper()
	if _, err := store.RequestDeletion(context.Background(), id, requestedAt); err != nil {
		t.Fatal(err)
	}
}

func TestDeletionIsNotCompleteUntilTheLogWriteSucceeds(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := account.New(pool)
	ctx := context.Background()
	ayla := signedUp(t, store, pool, "g-ayla", "Ayla")
	requestedAt := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	dueForDeletion(t, store, ayla.ID, requestedAt)
	now := requestedAt.Add(31 * 24 * time.Hour)
	unreachable := errors.New("log store unreachable")
	if completed, err := store.DeleteDue(ctx, &recordedLog{failure: unreachable}, requestedAt, now); !errors.Is(err, unreachable) || completed != 0 {
		t.Fatalf("delete with a failing log: %d %v", completed, err)
	}
	if n := count(t, pool, `select count(*) from accounts where id = $1`, ayla.ID); n != 0 {
		t.Fatalf("account rows after the erase: %d", n)
	}
	pending, err := store.UnloggedDeletions(ctx)
	if err != nil || len(pending) != 1 || pending[0].Account != ayla.ID || pending[0].Tombstone == "" || !pending[0].At.Equal(now) {
		t.Fatalf("unlogged deletions after the failed write: %+v %v", pending, err)
	}
	log := &recordedLog{}
	if completed, err := store.DeleteDue(ctx, log, now, now); err != nil || completed != 1 {
		t.Fatalf("retry: %d %v", completed, err)
	}
	if len(log.appended) != 1 || log.appended[0] != pending[0] {
		t.Fatalf("retried entry %+v, want %+v", log.appended, pending[0])
	}
	if completed, err := store.LogDeletions(ctx, log); err != nil || completed != 0 || len(log.appended) != 1 {
		t.Fatalf("log again: %d %v %+v", completed, err, log.appended)
	}
	if pending, err := store.UnloggedDeletions(ctx); err != nil || len(pending) != 0 {
		t.Fatalf("unlogged deletions after the retry: %+v %v", pending, err)
	}
}

func TestReapplyErasesARestoredAccountUnderTheLoggedTombstoneOnce(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := account.New(pool)
	ctx := context.Background()
	ayla := signedUp(t, store, pool, "g-ayla", "Ayla")
	ben := signedUp(t, store, pool, "g-ben", "Ben")
	var tombstone, stranger string
	if err := pool.QueryRow(ctx, `select gen_random_uuid()::text, gen_random_uuid()::text`).Scan(&tombstone, &stranger); err != nil {
		t.Fatal(err)
	}
	deletedAt := time.Date(2026, 11, 2, 9, 30, 0, 0, time.UTC)
	logged := account.Deletion{Account: ayla.ID, Tombstone: tombstone, At: deletedAt}
	if reapplied, err := store.Reapply(ctx, logged); err != nil || !reapplied {
		t.Fatalf("first replay: %v %v", reapplied, err)
	}
	checks := []struct {
		query string
		args  []any
		want  int
	}{
		{`select count(*) from accounts where id = $1`, []any{ayla.ID}, 0},
		{`select count(*) from auth_identities where subject = 'g-ayla' or email = 'g-ayla@example.com' or account_id = $1`, []any{ayla.ID}, 0},
		{`select count(*) from profiles where nickname_key = 'ayla' or account_id = $1`, []any{ayla.ID}, 0},
		{`select count(*) from accounts where id = $1 and state = 'deleted' and deleted_at = $2`, []any{tombstone, deletedAt}, 1},
		{`select count(*) from nickname_history where nickname_key = 'ayla' and account_id = $1 and replaced_at = $2`, []any{tombstone, deletedAt}, 1},
		{`select count(*) from nickname_history h left join accounts a on a.id = h.account_id where h.nickname_key = 'ayla' and a.state is distinct from 'deleted'`, nil, 0},
		{`select count(*) from deletion_log_pending`, nil, 0},
	}
	assertCounts := func(stage string) {
		t.Helper()
		for _, c := range checks {
			if got := count(t, pool, c.query, c.args...); got != c.want {
				t.Fatalf("%s: %s: %d, want %d", stage, c.query, got, c.want)
			}
		}
	}
	assertCounts("first replay")
	if reapplied, err := store.Reapply(ctx, logged); err != nil || reapplied {
		t.Fatalf("second replay: %v %v", reapplied, err)
	}
	assertCounts("second replay")
	if reapplied, err := store.Reapply(ctx, account.Deletion{Account: stranger, Tombstone: tombstone, At: deletedAt}); err != nil || reapplied {
		t.Fatalf("replay of an account the database never had: %v %v", reapplied, err)
	}
	if reapplied, err := store.Reapply(ctx, account.Deletion{Account: ben.ID, At: deletedAt}); err == nil || reapplied {
		t.Fatalf("replay without a tombstone: %v %v", reapplied, err)
	}
	if _, created, err := store.SignIn(ctx, idtoken.Identity{Provider: idtoken.ProviderGoogle, Subject: "g-ben"}); err != nil || created {
		t.Fatalf("ben after a refused replay: created %v %v", created, err)
	}
}

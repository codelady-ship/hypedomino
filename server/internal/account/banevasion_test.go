package account_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/account"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/idtoken"
	"github.com/Alievs-corp/hypedomino/server/internal/sanction"
)

var banEvasionKey = strings.Repeat("k", 32)

const permanentBanEvasionRetention = 1095 * 24 * time.Hour

func banEvasion(t *testing.T) account.BanEvasion {
	t.Helper()
	hasher, err := account.NewIdentityHasher(banEvasionKey)
	if err != nil {
		t.Fatal(err)
	}
	return account.BanEvasion{Hasher: hasher, PermanentRetention: permanentBanEvasionRetention}
}

func trackingStore(t *testing.T, pool *pgxpool.Pool) *account.Store {
	t.Helper()
	store := account.New(pool)
	store.EnableBanEvasion(banEvasion(t))
	return store
}

func ban(t *testing.T, pool *pgxpool.Pool, id string, kind sanction.Kind, category sanction.Category, now time.Time, ends *time.Time) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	update, value := `update accounts set banned_until = $2 where id = $1`, any(ends)
	if kind == sanction.PermanentBan {
		update, value = `update accounts set state = $2 where id = $1`, account.StateBanned
	}
	if _, err := tx.Exec(ctx, update, id, value); err != nil {
		t.Fatal(err)
	}
	if _, err := sanction.Impose(ctx, tx, id, kind, category, now, ends); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func google(subject string) idtoken.Identity {
	return idtoken.Identity{Provider: idtoken.ProviderGoogle, Subject: subject}
}

func TestTheIdentityHashIsKeyedAndSeparatesProviderFromSubject(t *testing.T) {
	if _, err := account.NewIdentityHasher("0123456789abcdef0123456789abcde"); !errors.Is(err, account.ErrBanEvasionKey) {
		t.Fatalf("a 31-byte key: %v", err)
	}
	hasher := banEvasion(t).Hasher
	other, err := account.NewIdentityHasher("fedcba9876543210fedcba9876543210")
	if err != nil {
		t.Fatal(err)
	}
	hash := hasher.Hash("google", "g-1")
	if len(hash) != 32 || !bytes.Equal(hash, hasher.Hash("google", "g-1")) {
		t.Fatalf("hash is not a stable 32-byte digest: %x", hash)
	}
	for name, different := range map[string][]byte{
		"another key":      other.Hash("google", "g-1"),
		"another provider": hasher.Hash("apple", "g-1"),
		"another subject":  hasher.Hash("google", "g-2"),
		"a moved boundary": hasher.Hash("googleg", "-1"),
	} {
		if bytes.Equal(hash, different) {
			t.Errorf("%s gives the same hash", name)
		}
	}
}

func TestBannedAccountsMayRequestDeletionOnceTheHashIsTrackedAndCancelKeepsTheBan(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := trackingStore(t, pool)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	ends := now.Add(24 * time.Hour)
	temporary := signedUp(t, store, pool, "g-temp", "Temp")
	ban(t, pool, temporary.ID, sanction.TemporaryBan, sanction.Cheating, now, &ends)
	permanent := signedUp(t, store, pool, "g-perm", "Perm")
	ban(t, pool, permanent.ID, sanction.PermanentBan, sanction.Collusion, now, nil)
	for subject, id := range map[string]string{"g-temp": temporary.ID, "g-perm": permanent.ID} {
		if found, err := store.Lookup(ctx, google(subject)); err != nil || found.ID != id {
			t.Fatalf("lookup of banned %s: %+v %v", subject, found, err)
		}
		if _, err := store.RequestDeletion(ctx, id, now); err != nil {
			t.Fatalf("request of banned %s: %v", subject, err)
		}
		if _, err := store.Lookup(ctx, google(subject)); !errors.Is(err, account.ErrDeletionPending) {
			t.Fatalf("lookup of pending %s: %v", subject, err)
		}
		if err := store.CancelDeletion(ctx, id); err != nil {
			t.Fatal(err)
		}
		if _, _, err := store.SignIn(ctx, google(subject)); !errors.Is(err, account.ErrBanned) {
			t.Fatalf("sign-in of %s after cancel: %v", subject, err)
		}
	}
	if n := count(t, pool, `select count(*) from accounts where id = $1 and state = $2`, permanent.ID, account.StateBanned); n != 1 {
		t.Fatal("cancel did not restore the permanent ban state")
	}
}

func TestDeletingABannedAccountLeavesOnlyAKeyedHashUntilTheBanWouldEnd(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := trackingStore(t, pool)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	ends := now.Add(48 * time.Hour)
	temporary := signedUp(t, store, pool, "g-temp", "Temp")
	ban(t, pool, temporary.ID, sanction.TemporaryBan, sanction.Cheating, now.Add(-time.Hour), &ends)
	permanent := signedUp(t, store, pool, "g-perm", "Perm")
	ban(t, pool, permanent.ID, sanction.PermanentBan, sanction.Collusion, now.Add(-time.Hour), nil)
	if _, _, err := store.SignIn(ctx, idtoken.Identity{Provider: idtoken.ProviderApple, Subject: "a-perm"}); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `update auth_identities set account_id = $1 where subject = 'a-perm'`, permanent.ID); err != nil {
		t.Fatal(err)
	}
	clean := signedUp(t, store, pool, "g-clean", "Clean")
	lapsedEnd := now.Add(-time.Minute)
	lapsed := signedUp(t, store, pool, "g-lapsed", "Lapsed")
	ban(t, pool, lapsed.ID, sanction.TemporaryBan, sanction.Other, now.Add(-time.Hour), &lapsedEnd)
	lifted := signedUp(t, store, pool, "g-lifted", "Lifted")
	ban(t, pool, lifted.ID, sanction.PermanentBan, sanction.Other, now.Add(-time.Hour), nil)
	if _, err := sanction.LiftByID(ctx, pool, liftable(t, pool, lifted.ID), now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `update accounts set state = $2 where id = $1`, lifted.ID, account.StateActive); err != nil {
		t.Fatal(err)
	}
	requestedAt := now.Add(-time.Second)
	for _, id := range []string{temporary.ID, permanent.ID, clean.ID, lapsed.ID, lifted.ID} {
		dueForDeletion(t, store, id, requestedAt)
	}
	if deleted, err := store.DeleteDue(ctx, &recordedLog{}, requestedAt, now); err != nil || deleted != 5 {
		t.Fatalf("delete due: %d %v", deleted, err)
	}
	hasher := banEvasion(t).Hasher
	want := []struct {
		provider, subject, category string
		banEndsAt                   *time.Time
		expiresAt                   time.Time
	}{
		{"google", "g-temp", "cheating", &ends, ends},
		{"google", "g-perm", "collusion", nil, now.Add(permanentBanEvasionRetention)},
		{"apple", "a-perm", "collusion", nil, now.Add(permanentBanEvasionRetention)},
	}
	if n := count(t, pool, `select count(*) from ban_evasion_hashes`); n != len(want) {
		t.Fatalf("ban-evasion hashes: %d, want %d", n, len(want))
	}
	for _, w := range want {
		var category string
		var banEndsAt *time.Time
		var expiresAt time.Time
		if err := pool.QueryRow(ctx, `select category, ban_ends_at, expires_at from ban_evasion_hashes where identity_hash = $1`, hasher.Hash(w.provider, w.subject)).Scan(&category, &banEndsAt, &expiresAt); err != nil {
			t.Fatalf("%s %s: %v", w.provider, w.subject, err)
		}
		if category != w.category || (banEndsAt == nil) != (w.banEndsAt == nil) || (banEndsAt != nil && !banEndsAt.Equal(*w.banEndsAt)) || !expiresAt.Equal(w.expiresAt) {
			t.Fatalf("%s %s: %s %v %v", w.provider, w.subject, category, banEndsAt, expiresAt)
		}
	}
	if n := count(t, pool, `select count(*) from auth_identities where subject in ('g-temp', 'g-perm', 'a-perm')`); n != 0 {
		t.Fatalf("auth identities left after deletion: %d", n)
	}
}

func liftable(t *testing.T, pool *pgxpool.Pool, id string) int64 {
	t.Helper()
	var sanctionID int64
	if err := pool.QueryRow(context.Background(), `select id from account_sanctions where account_id = $1`, id).Scan(&sanctionID); err != nil {
		t.Fatal(err)
	}
	return sanctionID
}

func deletedWhileBanned(t *testing.T, store *account.Store, pool *pgxpool.Pool, subject string, kind sanction.Kind, ends *time.Time) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	a := signedUp(t, store, pool, subject, subject)
	ban(t, pool, a.ID, kind, sanction.Cheating, now.Add(-time.Hour), ends)
	dueForDeletion(t, store, a.ID, now.Add(-time.Minute))
	if deleted, err := store.DeleteDue(context.Background(), &recordedLog{}, now, now); err != nil || deleted != 1 {
		t.Fatalf("delete %s: %d %v", subject, deleted, err)
	}
}

func TestANewAccountWithTheSameIdentityInheritsTheBan(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := trackingStore(t, pool)
	ctx := context.Background()
	ends := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Microsecond)
	deletedWhileBanned(t, store, pool, "g-temp", sanction.TemporaryBan, &ends)
	deletedWhileBanned(t, store, pool, "g-perm", sanction.PermanentBan, nil)

	again, created, err := store.SignIn(ctx, google("g-temp"))
	var banned *account.BannedError
	if !created || !errors.As(err, &banned) || banned.Account != again.ID || banned.Category != "cheating" || banned.SanctionID == 0 || banned.Until == nil || !banned.Until.Equal(ends) {
		t.Fatalf("new account after a temporary ban: %+v %v %v", again, created, err)
	}
	if running, ok, err := sanction.Current(ctx, pool, again.ID, time.Now(), sanction.TemporaryBan); err != nil || !ok || running.EndsAt == nil || !running.EndsAt.Equal(ends) || running.Category != sanction.Cheating {
		t.Fatalf("inherited temporary sanction: %+v %v %v", running, ok, err)
	}
	if _, created, err := store.SignIn(ctx, google("g-temp")); created || !errors.Is(err, account.ErrBanned) {
		t.Fatalf("second sign-in: %v %v", created, err)
	}

	again, created, err = store.SignIn(ctx, google("g-perm"))
	if !created || !errors.As(err, &banned) || banned.Until != nil || banned.Category != "cheating" || again.State != account.StateBanned {
		t.Fatalf("new account after a permanent ban: %+v %v %v", again, created, err)
	}
	if _, ok, err := sanction.Current(ctx, pool, again.ID, time.Now(), sanction.PermanentBan); err != nil || !ok {
		t.Fatalf("inherited permanent sanction: %v %v", ok, err)
	}

	if fresh, created, err := store.SignIn(ctx, google("g-other")); err != nil || !created || fresh.State != account.StateActive {
		t.Fatalf("an unrelated identity: %+v %v %v", fresh, created, err)
	}
	untracked := account.New(pool)
	if _, err := pool.Exec(ctx, `delete from auth_identities where subject = 'g-perm'`); err != nil {
		t.Fatal(err)
	}
	if fresh, created, err := untracked.SignIn(ctx, google("g-perm")); err != nil || !created || fresh.State != account.StateActive {
		t.Fatalf("sign-in without the key set: %+v %v %v", fresh, created, err)
	}
}

func TestAnExpiredHashIsIgnoredAndPurgedIdempotently(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := trackingStore(t, pool)
	ctx := context.Background()
	ends := time.Now().UTC().Add(time.Hour).Truncate(time.Microsecond)
	deletedWhileBanned(t, store, pool, "g-temp", sanction.TemporaryBan, &ends)
	deletedWhileBanned(t, store, pool, "g-perm", sanction.PermanentBan, nil)
	if _, err := pool.Exec(ctx, `update ban_evasion_hashes set ban_ends_at = now() - interval '1 second', expires_at = now() - interval '1 second' where identity_hash = $1`, banEvasion(t).Hasher.Hash("google", "g-temp")); err != nil {
		t.Fatal(err)
	}
	if fresh, created, err := store.SignIn(ctx, google("g-temp")); err != nil || !created || fresh.State != account.StateActive {
		t.Fatalf("sign-in after the hash expired: %+v %v %v", fresh, created, err)
	}
	if purged, err := store.PurgeBanEvasion(ctx, time.Now()); err != nil || purged != 1 {
		t.Fatalf("purge: %d %v", purged, err)
	}
	if purged, err := store.PurgeBanEvasion(ctx, time.Now()); err != nil || purged != 0 {
		t.Fatalf("second purge: %d %v", purged, err)
	}
	if n := count(t, pool, `select count(*) from ban_evasion_hashes`); n != 1 {
		t.Fatalf("hashes after the purge: %d", n)
	}
	if purged, err := store.PurgeBanEvasion(ctx, time.Now().Add(permanentBanEvasionRetention+time.Hour)); err != nil || purged != 1 {
		t.Fatalf("purge after the permanent retention: %d %v", purged, err)
	}
}

func TestWithoutTheKeyBannedAccountsAreStillRefusedAndLeaveNoHash(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := account.New(pool)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	ends := now.Add(time.Hour)
	banned := signedUp(t, store, pool, "g-banned", "Banned")
	ban(t, pool, banned.ID, sanction.TemporaryBan, sanction.Cheating, now, &ends)
	if _, err := store.Lookup(ctx, google("g-banned")); !errors.Is(err, account.ErrBanned) {
		t.Fatalf("lookup: %v", err)
	}
	if _, err := store.RequestDeletion(ctx, banned.ID, now); !errors.Is(err, account.ErrBanned) {
		t.Fatalf("request: %v", err)
	}
	duringGrace := signedUp(t, store, pool, "g-grace", "Grace")
	dueForDeletion(t, store, duringGrace.ID, now.Add(-time.Minute))
	ban(t, pool, duringGrace.ID, sanction.TemporaryBan, sanction.Cheating, now, &ends)
	if deleted, err := store.DeleteDue(ctx, &recordedLog{}, now, now); err != nil || deleted != 1 {
		t.Fatalf("delete: %d %v", deleted, err)
	}
	if n := count(t, pool, `select count(*) from ban_evasion_hashes`); n != 0 {
		t.Fatalf("hashes written without the key: %d", n)
	}
}

func TestAPermanentBanWithoutARetentionStopsTheErase(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := account.New(pool)
	store.EnableBanEvasion(account.BanEvasion{Hasher: banEvasion(t).Hasher})
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	banned := signedUp(t, store, pool, "g-perm", "Perm")
	ban(t, pool, banned.ID, sanction.PermanentBan, sanction.Cheating, now.Add(-time.Hour), nil)
	dueForDeletion(t, store, banned.ID, now.Add(-time.Minute))
	if _, err := store.DeleteDue(ctx, &recordedLog{}, now, now); !errors.Is(err, account.ErrPermanentBanEvasionDisabled) {
		t.Fatalf("delete without a retention: %v", err)
	}
	if n := count(t, pool, `select count(*) from accounts where id = $1`, banned.ID); n != 1 {
		t.Fatal("the account was erased without its hash")
	}
}

func TestAnInheritedBanConsumesTheHashSoALiftedBanStaysLiftedAfterADeletion(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := trackingStore(t, pool)
	ctx := context.Background()
	deletedWhileBanned(t, store, pool, "g-perm", sanction.PermanentBan, nil)
	heir, created, err := store.SignIn(ctx, google("g-perm"))
	if !created || !errors.Is(err, account.ErrBanned) {
		t.Fatalf("sign-in that inherits the ban: %+v %v %v", heir, created, err)
	}
	if n := count(t, pool, `select count(*) from ban_evasion_hashes`); n != 0 {
		t.Fatalf("hashes left after the inheritance: %d", n)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	if lifted, err := sanction.LiftByID(ctx, pool, liftable(t, pool, heir.ID), now); err != nil || !lifted {
		t.Fatalf("lift of the inherited ban: %v %v", lifted, err)
	}
	if _, err := pool.Exec(ctx, `update accounts set state = $2 where id = $1`, heir.ID, account.StateActive); err != nil {
		t.Fatal(err)
	}
	dueForDeletion(t, store, heir.ID, now)
	if deleted, err := store.DeleteDue(ctx, &recordedLog{}, now, now); err != nil || deleted != 1 {
		t.Fatalf("delete of the clean heir: %d %v", deleted, err)
	}
	if n := count(t, pool, `select count(*) from ban_evasion_hashes`); n != 0 {
		t.Fatalf("hashes after a clean deletion: %d", n)
	}
	if fresh, created, err := store.SignIn(ctx, google("g-perm")); err != nil || !created || fresh.State != account.StateActive {
		t.Fatalf("sign-in after the lifted ban and a clean deletion: %+v %v %v", fresh, created, err)
	}
}

func TestDeletingAnIdentityThatAlreadyHasAHashKeepsTheLaterExpiry(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := trackingStore(t, pool)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	ends := now.Add(48 * time.Hour)
	stored := map[string]time.Time{"g-longer": now.Add(72 * time.Hour), "g-shorter": now.Add(24 * time.Hour)}
	hasher := banEvasion(t).Hasher
	for subject, expires := range stored {
		a := signedUp(t, store, pool, subject, subject)
		if _, err := pool.Exec(ctx, `insert into ban_evasion_hashes (identity_hash, category, ban_ends_at, expires_at) values ($1, 'other', $2, $2)`, hasher.Hash("google", subject), expires); err != nil {
			t.Fatal(err)
		}
		ban(t, pool, a.ID, sanction.TemporaryBan, sanction.Cheating, now.Add(-time.Hour), &ends)
		dueForDeletion(t, store, a.ID, now.Add(-time.Minute))
	}
	if deleted, err := store.DeleteDue(ctx, &recordedLog{}, now, now); err != nil || deleted != 2 {
		t.Fatalf("delete due: %d %v", deleted, err)
	}
	want := map[string]struct {
		category string
		expires  time.Time
	}{
		"g-longer":  {"other", stored["g-longer"]},
		"g-shorter": {"cheating", ends},
	}
	for subject, w := range want {
		var category string
		var banEndsAt, expiresAt time.Time
		if err := pool.QueryRow(ctx, `select category, ban_ends_at, expires_at from ban_evasion_hashes where identity_hash = $1`, hasher.Hash("google", subject)).Scan(&category, &banEndsAt, &expiresAt); err != nil {
			t.Fatalf("%s: %v", subject, err)
		}
		if category != w.category || !banEndsAt.Equal(w.expires) || !expiresAt.Equal(w.expires) {
			t.Fatalf("%s: %s %v %v, want %s %v", subject, category, banEndsAt, expiresAt, w.category, w.expires)
		}
	}
}

func TestReapplyOfABannedAccountWritesTheHashFromTheLoggedDeletionTime(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := trackingStore(t, pool)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	restored := signedUp(t, store, pool, "g-perm", "Perm")
	ban(t, pool, restored.ID, sanction.PermanentBan, sanction.Collusion, now.Add(-time.Hour), nil)
	var tombstone string
	if err := pool.QueryRow(ctx, `select gen_random_uuid()::text`).Scan(&tombstone); err != nil {
		t.Fatal(err)
	}
	deletedAt := now.Add(-time.Minute)
	logged := account.Deletion{Account: restored.ID, Tombstone: tombstone, At: deletedAt}
	if reapplied, err := store.Reapply(ctx, logged); err != nil || !reapplied {
		t.Fatalf("replay: %v %v", reapplied, err)
	}
	var category string
	var banEndsAt *time.Time
	var expiresAt time.Time
	if err := pool.QueryRow(ctx, `select category, ban_ends_at, expires_at from ban_evasion_hashes where identity_hash = $1`, banEvasion(t).Hasher.Hash("google", "g-perm")).Scan(&category, &banEndsAt, &expiresAt); err != nil {
		t.Fatal(err)
	}
	if category != "collusion" || banEndsAt != nil || !expiresAt.Equal(deletedAt.Add(permanentBanEvasionRetention)) {
		t.Fatalf("hash after the replay: %s %v %v", category, banEndsAt, expiresAt)
	}
	if reapplied, err := store.Reapply(ctx, logged); err != nil || reapplied {
		t.Fatalf("second replay: %v %v", reapplied, err)
	}
	if n := count(t, pool, `select count(*) from ban_evasion_hashes`); n != 1 {
		t.Fatalf("hashes after the second replay: %d", n)
	}
}

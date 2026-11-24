package adminsession_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/adminaccount"
	"github.com/Alievs-corp/hypedomino/server/internal/adminsession"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

var start = time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

func fixture(t *testing.T) (*adminsession.Store, string) {
	t.Helper()
	_, pool := dbtest.Fresh(t)
	return adminsession.NewStore(pool, adminsession.DefaultConfig()), account(t, pool)
}

func account(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	created, err := adminaccount.NewStore(pool).Create(context.Background(), "ayla@example.com", "Ayla", []rbac.Role{rbac.Content}, start)
	if err != nil {
		t.Fatal(err)
	}
	return created.ID
}

func TestAnOpenedSessionAuthenticatesByItsTokenOnly(t *testing.T) {
	store, accountID := fixture(t)
	ctx := context.Background()
	token, opened, err := store.Open(ctx, accountID, start)
	if err != nil {
		t.Fatal(err)
	}
	want := adminsession.Session{
		ID: opened.ID, AccountID: accountID, CreatedAt: start, AuthenticatedAt: start, LastSeenAt: start, ExpiresAt: start.Add(12 * time.Hour),
	}
	if token == "" || opened != want {
		t.Fatalf("opened %+v with token %q", opened, token)
	}
	later := start.Add(5 * time.Minute)
	seen, err := store.Authenticate(ctx, token, later)
	want.LastSeenAt = later
	if err != nil || seen != want {
		t.Fatalf("authenticated %+v %v", seen, err)
	}
	for _, other := range []string{"", token + "x", opened.ID} {
		if _, err := store.Authenticate(ctx, other, later); !errors.Is(err, adminsession.ErrRejected) {
			t.Fatalf("token %q: %v", other, err)
		}
	}
	second, _, err := store.Open(ctx, accountID, start)
	if err != nil || second == token {
		t.Fatalf("second token %q %v", second, err)
	}
}

func TestASessionEndsAfterTheIdleTimeoutEvenWhileUsed(t *testing.T) {
	store, accountID := fixture(t)
	ctx := context.Background()
	idle, _, err := store.Open(ctx, accountID, start)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Authenticate(ctx, idle, start.Add(30*time.Minute)); !errors.Is(err, adminsession.ErrRejected) {
		t.Fatalf("after 30 idle minutes: %v", err)
	}
	busy, _, err := store.Open(ctx, accountID, start)
	if err != nil {
		t.Fatal(err)
	}
	for now := start.Add(25 * time.Minute); now.Before(start.Add(12 * time.Hour)); now = now.Add(25 * time.Minute) {
		if _, err := store.Authenticate(ctx, busy, now); err != nil {
			t.Fatalf("at %v: %v", now, err)
		}
	}
	if _, err := store.Authenticate(ctx, busy, start.Add(12*time.Hour)); !errors.Is(err, adminsession.ErrRejected) {
		t.Fatalf("at the absolute limit: %v", err)
	}
}

func TestStepUpRenewsTheSignInTimeUntilRevoked(t *testing.T) {
	store, accountID := fixture(t)
	ctx := context.Background()
	token, opened, err := store.Open(ctx, accountID, start)
	if err != nil {
		t.Fatal(err)
	}
	later := start.Add(20 * time.Minute)
	renewed, err := store.StepUp(ctx, opened.ID, later)
	if err != nil || renewed.AuthenticatedAt != later || renewed.LastSeenAt != later || renewed.CreatedAt != start {
		t.Fatalf("stepped up %+v %v", renewed, err)
	}
	if err := store.Revoke(ctx, opened.ID, later); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Authenticate(ctx, token, later); !errors.Is(err, adminsession.ErrRejected) {
		t.Fatalf("revoked session authenticated: %v", err)
	}
	if _, err := store.StepUp(ctx, opened.ID, later); !errors.Is(err, adminsession.ErrRejected) {
		t.Fatalf("revoked session stepped up: %v", err)
	}
}

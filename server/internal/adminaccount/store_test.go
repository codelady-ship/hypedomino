package adminaccount_test

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/adminaccount"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

var start = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

func TestAnAccountIsCreatedWithANormalizedEmailAndSortedRoles(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := adminaccount.NewStore(pool)
	ctx := context.Background()
	created, err := store.Create(ctx, "  Ayla@Example.COM ", " Ayla ", []rbac.Role{rbac.Support, rbac.Content, rbac.Support}, start)
	if err != nil {
		t.Fatal(err)
	}
	want := adminaccount.Account{
		ID: created.ID, Email: "ayla@example.com", DisplayName: "Ayla",
		Roles: []rbac.Role{rbac.Content, rbac.Support}, State: adminaccount.Active, CreatedAt: start,
	}
	if created.ID == "" || !reflect.DeepEqual(created, want) {
		t.Fatalf("created %+v", created)
	}
	found, err := store.ByEmail(ctx, "AYLA@example.com")
	if err != nil || !reflect.DeepEqual(found, want) {
		t.Fatalf("by email %+v %v", found, err)
	}
	byID, err := store.ByID(ctx, created.ID)
	if err != nil || !reflect.DeepEqual(byID, want) {
		t.Fatalf("by id %+v %v", byID, err)
	}
	for _, id := range []string{"00000000-0000-0000-0000-000000000000", "not-an-id"} {
		if _, err := store.ByID(ctx, id); !errors.Is(err, adminaccount.ErrNotFound) {
			t.Fatalf("by id %q: %v", id, err)
		}
	}
	if _, err := store.Create(ctx, "ayla@EXAMPLE.com", "Other", []rbac.Role{rbac.Analyst}, start); !errors.Is(err, adminaccount.ErrTaken) {
		t.Fatalf("second account on the same email: %v", err)
	}
}

func TestInvalidAccountsAreRejected(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := adminaccount.NewStore(pool)
	tests := map[string]struct {
		email, name string
		roles       []rbac.Role
	}{
		"no email":      {"", "Ayla", []rbac.Role{rbac.Content}},
		"no domain":     {"ayla@", "Ayla", []rbac.Role{rbac.Content}},
		"no local part": {"@example.com", "Ayla", []rbac.Role{rbac.Content}},
		"two at signs":  {"ayla@x@example.com", "Ayla", []rbac.Role{rbac.Content}},
		"blank name":    {"ayla@example.com", "  ", []rbac.Role{rbac.Content}},
		"no roles":      {"ayla@example.com", "Ayla", nil},
		"unknown role":  {"ayla@example.com", "Ayla", []rbac.Role{rbac.Content, "owner"}},
	}
	for name, test := range tests {
		if _, err := store.Create(context.Background(), test.email, test.name, test.roles, start); !errors.Is(err, adminaccount.ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestSignInRecordsTheTimeAndRefusesDisabledAndUnknownAccounts(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := adminaccount.NewStore(pool)
	ctx := context.Background()
	ayla, err := store.Create(ctx, "ayla@example.com", "Ayla", []rbac.Role{rbac.LiveOps}, start)
	if err != nil {
		t.Fatal(err)
	}
	signedIn, err := store.SignIn(ctx, "Ayla@Example.com", start.Add(time.Hour))
	if err != nil || signedIn.ID != ayla.ID || !signedIn.LastLoginAt.Equal(start.Add(time.Hour)) {
		t.Fatalf("sign in %+v %v", signedIn, err)
	}
	if _, err := store.SignIn(ctx, "cem@example.com", start); !errors.Is(err, adminaccount.ErrNotFound) {
		t.Fatalf("unknown email: %v", err)
	}
	if _, err := store.Disable(ctx, ayla.ID, start.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SignIn(ctx, "ayla@example.com", start.Add(3*time.Hour)); !errors.Is(err, adminaccount.ErrDisabled) {
		t.Fatalf("disabled account: %v", err)
	}
	after, err := store.ByEmail(ctx, "ayla@example.com")
	if err != nil || !after.LastLoginAt.Equal(start.Add(time.Hour)) {
		t.Fatalf("a refused sign-in changed the last login: %+v %v", after, err)
	}
}

func TestRolesAreReplacedAndValidated(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := adminaccount.NewStore(pool)
	ctx := context.Background()
	ayla, err := store.Create(ctx, "ayla@example.com", "Ayla", []rbac.Role{rbac.Content}, start)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := store.SetRoles(ctx, ayla.ID, []rbac.Role{rbac.Support, rbac.Analyst})
	if err != nil || !reflect.DeepEqual(changed.Roles, []rbac.Role{rbac.Analyst, rbac.Support}) {
		t.Fatalf("set roles %+v %v", changed, err)
	}
	for _, roles := range [][]rbac.Role{nil, {"owner"}} {
		if _, err := store.SetRoles(ctx, ayla.ID, roles); !errors.Is(err, adminaccount.ErrInvalid) {
			t.Errorf("%v: %v", roles, err)
		}
	}
	if _, err := store.SetRoles(ctx, "00000000-0000-0000-0000-000000000000", []rbac.Role{rbac.Content}); !errors.Is(err, adminaccount.ErrNotFound) {
		t.Fatalf("missing account: %v", err)
	}
}

func TestDisablingKeepsTheFirstDisableTime(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := adminaccount.NewStore(pool)
	ctx := context.Background()
	ayla, err := store.Create(ctx, "ayla@example.com", "Ayla", []rbac.Role{rbac.Content}, start)
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.Disable(ctx, ayla.ID, start.Add(time.Hour))
	if err != nil || first.State != adminaccount.Disabled || !first.DisabledAt.Equal(start.Add(time.Hour)) {
		t.Fatalf("disable %+v %v", first, err)
	}
	again, err := store.Disable(ctx, ayla.ID, start.Add(2*time.Hour))
	if err != nil || !again.DisabledAt.Equal(start.Add(time.Hour)) {
		t.Fatalf("second disable %+v %v", again, err)
	}
}

func TestAccountsUnusedForTheWindowAreDisabled(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := adminaccount.NewStore(pool)
	ctx := context.Background()
	window := 30 * 24 * time.Hour
	now := start.Add(40 * 24 * time.Hour)
	neverSignedIn, _ := store.Create(ctx, "never@example.com", "Never", []rbac.Role{rbac.Content}, start)
	recentlyCreated, _ := store.Create(ctx, "new@example.com", "New", []rbac.Role{rbac.Content}, now.Add(-time.Hour))
	stale, _ := store.Create(ctx, "stale@example.com", "Stale", []rbac.Role{rbac.Content}, start)
	fresh, _ := store.Create(ctx, "fresh@example.com", "Fresh", []rbac.Role{rbac.Content}, start)
	for email, at := range map[string]time.Time{"stale@example.com": now.Add(-window - time.Minute), "fresh@example.com": now.Add(-window + time.Minute)} {
		if _, err := store.SignIn(ctx, email, at); err != nil {
			t.Fatal(err)
		}
	}
	disabled, err := store.DisableInactive(ctx, now, window)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, account := range disabled {
		got[account.ID] = true
		if account.State != adminaccount.Disabled || !account.DisabledAt.Equal(now) {
			t.Errorf("disabled account %+v", account)
		}
	}
	if !reflect.DeepEqual(got, map[string]bool{neverSignedIn.ID: true, stale.ID: true}) {
		t.Fatalf("disabled %v; never %s stale %s fresh %s new %s", got, neverSignedIn.ID, stale.ID, fresh.ID, recentlyCreated.ID)
	}
	if again, err := store.DisableInactive(ctx, now, window); err != nil || len(again) != 0 {
		t.Fatalf("second run %+v %v", again, err)
	}
}

package account_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/account"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/idtoken"
)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

func TestFirstSignInCreatesTheAccountAndLaterOnesFindIt(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := account.New(pool)
	ctx := context.Background()
	google := idtoken.Identity{Provider: idtoken.ProviderGoogle, Subject: "g-1", Email: "ayla@example.com", EmailVerified: true}
	first, created, err := store.SignIn(ctx, google)
	if err != nil || !created || first.ID == "" || first.State != account.StateActive || first.Email != "ayla@example.com" || first.CreatedAt.IsZero() {
		t.Fatalf("first sign-in: %+v %v %v", first, created, err)
	}
	again, created, err := store.SignIn(ctx, google)
	if err != nil || created || again != first {
		t.Fatalf("second sign-in: %+v %v %v", again, created, err)
	}
	apple := idtoken.Identity{Provider: idtoken.ProviderApple, Subject: "g-1", Email: "relay@privaterelay.appleid.com", PrivateEmail: true}
	other, created, err := store.SignIn(ctx, apple)
	if err != nil || !created || other.ID == first.ID || other.Email != "relay@privaterelay.appleid.com" {
		t.Fatalf("same subject at another provider: %+v %v %v", other, created, err)
	}
	var identities int
	if err := pool.QueryRow(ctx, `select count(*) from auth_identities`).Scan(&identities); err != nil || identities != 2 {
		t.Fatalf("identities %d %v", identities, err)
	}
}

func TestEmailFollowsTheProviderButIsNeverBlanked(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := account.New(pool)
	ctx := context.Background()
	identity := idtoken.Identity{Provider: idtoken.ProviderApple, Subject: "a-1", Email: "first@example.com"}
	if _, _, err := store.SignIn(ctx, identity); err != nil {
		t.Fatal(err)
	}
	identity.Email = ""
	if a, _, err := store.SignIn(ctx, identity); err != nil || a.Email != "first@example.com" {
		t.Fatalf("sign-in without an email: %+v %v", a, err)
	}
	identity.Email = "second@example.com"
	if a, _, err := store.SignIn(ctx, identity); err != nil || a.Email != "second@example.com" {
		t.Fatalf("sign-in with a new email: %+v %v", a, err)
	}
}

func TestBannedAndDeletedAccountsAreRefused(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := account.New(pool)
	ctx := context.Background()
	identity := idtoken.Identity{Provider: idtoken.ProviderGoogle, Subject: "g-2"}
	a, _, err := store.SignIn(ctx, identity)
	if err != nil {
		t.Fatal(err)
	}
	for state, want := range map[string]error{account.StateBanned: account.ErrBanned, account.StateDeleted: account.ErrDeleted, account.StateDeleting: nil, account.StateActive: nil} {
		if _, err := pool.Exec(ctx, `update accounts set state = $1 where id = $2`, state, a.ID); err != nil {
			t.Fatal(err)
		}
		got, created, err := store.SignIn(ctx, identity)
		if !errors.Is(err, want) || created || got.State != state {
			t.Fatalf("%s: %+v %v %v", state, got, created, err)
		}
	}
}

func TestConcurrentFirstSignInsShareOneAccount(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := account.New(pool)
	identity := idtoken.Identity{Provider: idtoken.ProviderGoogle, Subject: "g-race"}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	var wg sync.WaitGroup
	ids := make([]string, 8)
	errs := make([]error, 8)
	for index := range ids {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a, _, err := store.SignIn(ctx, identity)
			ids[index], errs[index] = a.ID, err
		}()
	}
	wg.Wait()
	for index := range ids {
		if errs[index] != nil || ids[index] != ids[0] {
			t.Fatalf("sign-in %d: %s %v (first %s)", index, ids[index], errs[index], ids[0])
		}
	}
	var accounts int
	if err := pool.QueryRow(ctx, `select count(*) from accounts`).Scan(&accounts); err != nil || accounts != 1 {
		t.Fatalf("accounts %d %v", accounts, err)
	}
}

func TestAnIdentityWithoutASubjectIsInvalid(t *testing.T) {
	store := account.New(nil)
	if _, _, err := store.SignIn(context.Background(), idtoken.Identity{Provider: idtoken.ProviderGoogle}); !errors.Is(err, account.ErrInvalidIdentity) {
		t.Fatalf("empty subject: %v", err)
	}
	if _, _, err := store.SignIn(context.Background(), idtoken.Identity{Subject: "x"}); !errors.Is(err, account.ErrInvalidIdentity) {
		t.Fatalf("no provider: %v", err)
	}
}

func TestCountryIsAdoptedOnceAndChangedWithinTheCooldown(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := account.New(pool)
	ctx := context.Background()
	a, _, err := store.SignIn(ctx, idtoken.Identity{Provider: idtoken.ProviderGoogle, Subject: "g-c", Email: "c@example.com", EmailVerified: true})
	if err != nil {
		t.Fatal(err)
	}
	if c, err := store.Country(ctx, a.ID); err != nil || c.Code != "" {
		t.Fatalf("fresh %+v %v", c, err)
	}
	if err := store.AdoptCountry(ctx, a.ID, "az"); err != nil {
		t.Fatal(err)
	}
	if c, _ := store.Country(ctx, a.ID); c.Code != "" {
		t.Fatalf("lowercase adopted: %+v", c)
	}
	if err := store.AdoptCountry(ctx, a.ID, "AZ"); err != nil {
		t.Fatal(err)
	}
	if err := store.AdoptCountry(ctx, a.ID, "TR"); err != nil {
		t.Fatal(err)
	}
	if c, _ := store.Country(ctx, a.ID); c.Code != "AZ" || !c.ChangedAt.IsZero() {
		t.Fatalf("adoption overwritten: %+v", c)
	}
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	if _, err := store.SetCountry(ctx, a.ID, "A1", now, time.Hour); !errors.Is(err, account.ErrCountryInvalid) {
		t.Fatalf("invalid code: %v", err)
	}
	c, err := store.SetCountry(ctx, a.ID, "TR", now, time.Hour)
	if err != nil || c.Code != "TR" || !c.ChangedAt.Equal(now) {
		t.Fatalf("set %+v %v", c, err)
	}
	if _, err := store.SetCountry(ctx, a.ID, "DE", now.Add(30*time.Minute), time.Hour); !errors.Is(err, account.ErrCountryCooldown) {
		t.Fatalf("cooldown: %v", err)
	}
	if c, err := store.SetCountry(ctx, a.ID, "TR", now.Add(30*time.Minute), time.Hour); err != nil || c.Code != "TR" {
		t.Fatalf("same country inside the cooldown: %+v %v", c, err)
	}
	if c, err := store.SetCountry(ctx, a.ID, "DE", now.Add(2*time.Hour), time.Hour); err != nil || c.Code != "DE" {
		t.Fatalf("after the cooldown: %+v %v", c, err)
	}
	if _, err := store.SetCountry(ctx, "00000000-0000-0000-0000-000000000000", "DE", now, time.Hour); !errors.Is(err, account.ErrUnknown) {
		t.Fatalf("unknown account: %v", err)
	}
	if _, err := store.Country(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, account.ErrUnknown) {
		t.Fatalf("unknown account country: %v", err)
	}
}

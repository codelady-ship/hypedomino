package token_test

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

const (
	ttl   = 30 * 24 * time.Hour
	grace = 30 * time.Second
)

var start = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

func account(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `insert into accounts default values returning id`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func reason(t *testing.T, pool *pgxpool.Pool, family string) string {
	t.Helper()
	var reason *string
	if err := pool.QueryRow(context.Background(), `select revoke_reason from auth_sessions where id = $1`, family).Scan(&reason); err != nil {
		t.Fatal(err)
	}
	if reason == nil {
		return ""
	}
	return *reason
}

func TestRefreshTokensRotateAndKeepTheFamily(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := token.NewStore(pool)
	ctx := context.Background()
	acc := account(t, pool)
	f, refresh, err := store.Open(ctx, acc, "install-1", start, ttl)
	if err != nil || f.ID == "" || f.Generation != 1 || f.Account != acc || f.Device != "install-1" || !f.ExpiresAt.Equal(start.Add(ttl)) {
		t.Fatalf("open: %+v %v", f, err)
	}
	now := start
	current := refresh
	for generation := 2; generation <= 4; generation++ {
		now = now.Add(time.Hour)
		next, rotated, err := store.Rotate(ctx, current, "install-1", now, ttl, grace)
		if err != nil || next.ID != f.ID || next.Generation != generation || rotated == current || !next.ExpiresAt.Equal(now.Add(ttl)) {
			t.Fatalf("rotation %d: %+v %q %v", generation, next, rotated, err)
		}
		current = rotated
	}
	if _, _, err := store.Rotate(ctx, "bm90LWEtdG9rZW4", "install-1", now, ttl, grace); !errors.Is(err, token.ErrUnknownRefresh) {
		t.Fatalf("garbage refresh: %v", err)
	}
	forged := base64.RawURLEncoding.EncodeToString(append(make([]byte, 16), []byte(strings.Repeat("x", 32))...))
	if _, _, err := store.Rotate(ctx, forged, "install-1", now, ttl, grace); !errors.Is(err, token.ErrUnknownRefresh) {
		t.Fatalf("unknown family: %v", err)
	}
}

func TestAReusedRefreshTokenRevokesTheWholeFamily(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := token.NewStore(pool)
	ctx := context.Background()
	f, stolen, err := store.Open(ctx, account(t, pool), "install-1", start, ttl)
	if err != nil {
		t.Fatal(err)
	}
	_, legit, err := store.Rotate(ctx, stolen, "install-1", start.Add(time.Minute), ttl, grace)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Rotate(ctx, stolen, "install-1", start.Add(time.Hour), ttl, grace); !errors.Is(err, token.ErrReused) {
		t.Fatalf("stolen token after the grace: %v", err)
	}
	if _, _, err := store.Rotate(ctx, legit, "install-1", start.Add(time.Hour), ttl, grace); !errors.Is(err, token.ErrRevoked) {
		t.Fatalf("the legitimate token after the reuse: %v", err)
	}
	if got := reason(t, pool, f.ID); got != token.ReasonReuse {
		t.Fatalf("revoke reason %q", got)
	}
}

func TestARetryInsideTheGraceRotatesAgain(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := token.NewStore(pool)
	ctx := context.Background()
	_, first, err := store.Open(ctx, account(t, pool), "install-1", start, ttl)
	if err != nil {
		t.Fatal(err)
	}
	_, lost, err := store.Rotate(ctx, first, "install-1", start.Add(time.Second), ttl, grace)
	if err != nil {
		t.Fatal(err)
	}
	f, retried, err := store.Rotate(ctx, first, "install-1", start.Add(10*time.Second), ttl, grace)
	if err != nil || f.Generation != 3 {
		t.Fatalf("retry inside the grace: %+v %v", f, err)
	}
	if _, _, err := store.Rotate(ctx, lost, "install-1", start.Add(11*time.Second), ttl, grace); !errors.Is(err, token.ErrReused) {
		t.Fatalf("the token the client never saw: %v", err)
	}
	if _, _, err := store.Rotate(ctx, retried, "install-1", start.Add(12*time.Second), ttl, grace); !errors.Is(err, token.ErrRevoked) {
		t.Fatalf("after the reuse: %v", err)
	}
}

func TestAnotherDeviceAnExpiredTokenAndALogoutEndTheFamily(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := token.NewStore(pool)
	ctx := context.Background()
	acc := account(t, pool)
	moved, refresh, err := store.Open(ctx, acc, "install-1", start, ttl)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Rotate(ctx, refresh, "install-2", start.Add(time.Minute), ttl, grace); !errors.Is(err, token.ErrDeviceMismatch) {
		t.Fatalf("another device: %v", err)
	}
	if got := reason(t, pool, moved.ID); got != token.ReasonDevice {
		t.Fatalf("revoke reason %q", got)
	}
	stale, refresh, err := store.Open(ctx, acc, "install-1", start, ttl)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Rotate(ctx, refresh, "install-1", start.Add(ttl), ttl, grace); !errors.Is(err, token.ErrRefreshExpired) {
		t.Fatalf("expired: %v", err)
	}
	if got := reason(t, pool, stale.ID); got != token.ReasonExpired {
		t.Fatalf("revoke reason %q", got)
	}
	out, refresh, err := store.Open(ctx, acc, "install-1", start, ttl)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Revoke(ctx, "garbage", start); err != nil {
		t.Fatalf("logout with garbage: %v", err)
	}
	if err := store.Revoke(ctx, refresh, start.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Rotate(ctx, refresh, "install-1", start.Add(2*time.Minute), ttl, grace); !errors.Is(err, token.ErrRevoked) {
		t.Fatalf("after logout: %v", err)
	}
	if got := reason(t, pool, out.ID); got != token.ReasonLogout {
		t.Fatalf("revoke reason %q", got)
	}
	if _, _, err := store.Open(ctx, acc, "", start, ttl); !errors.Is(err, token.ErrInvalidFamily) {
		t.Fatalf("open without a device: %v", err)
	}
}

func TestLogoutAllRevokesEveryLiveFamilyOfTheAccount(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := token.NewStore(pool)
	ctx := context.Background()
	acc, other := account(t, pool), account(t, pool)
	var refreshes []string
	for _, device := range []string{"phone", "tablet"} {
		_, refresh, err := store.Open(ctx, acc, device, start, ttl)
		if err != nil {
			t.Fatal(err)
		}
		refreshes = append(refreshes, refresh)
	}
	_, keep, err := store.Open(ctx, other, "phone", start, ttl)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `insert into push_devices (token, account_id, platform, registered_at) values ('push-a', $1, 'android', $3), ('push-b', $1, 'ios', $3), ('push-o', $2, 'android', $3)`, acc, other, start); err != nil {
		t.Fatal(err)
	}
	if n, err := store.RevokeAccount(ctx, acc, start.Add(time.Minute)); err != nil || n != 2 {
		t.Fatalf("revoked %d %v", n, err)
	}
	var mine, others int
	if err := pool.QueryRow(ctx, `select count(*) filter (where account_id = $1), count(*) filter (where account_id = $2) from push_devices`, acc, other).Scan(&mine, &others); err != nil || mine != 0 || others != 1 {
		t.Fatalf("push devices after logout-all: own %d other %d %v", mine, others, err)
	}
	for _, refresh := range refreshes {
		if _, _, err := store.Rotate(ctx, refresh, "phone", start.Add(2*time.Minute), ttl, grace); !errors.Is(err, token.ErrRevoked) {
			t.Fatalf("after logout-all: %v", err)
		}
	}
	if _, _, err := store.Rotate(ctx, keep, "phone", start.Add(2*time.Minute), ttl, grace); err != nil {
		t.Fatalf("another account was logged out: %v", err)
	}
	if n, err := store.RevokeAccount(ctx, acc, start.Add(3*time.Minute)); err != nil || n != 0 {
		t.Fatalf("second logout-all revoked %d %v", n, err)
	}
}

func TestTheServiceIssuesVerifiesAndRefreshesPairs(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	config := token.DefaultConfig()
	config.Keys = "k1:" + base64.StdEncoding.EncodeToString([]byte(strings.Repeat("a", token.MinKeyBytes)))
	now := start
	service, err := token.NewService(config, token.NewStore(pool), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	acc := account(t, pool)
	pair, err := service.Issue(ctx, acc, "install-1")
	if err != nil || pair.ExpiresIn != 900 || pair.Session == "" {
		t.Fatalf("issue: %+v %v", pair, err)
	}
	if gotAccount, gotSession, err := service.Authenticate(pair.AccessToken); err != nil || gotAccount != acc || gotSession != pair.Session {
		t.Fatalf("authenticate: %s %s %v", gotAccount, gotSession, err)
	}
	now = now.Add(20 * time.Minute)
	if _, _, err := service.Authenticate(pair.AccessToken); !errors.Is(err, token.ErrExpired) {
		t.Fatalf("expired access token: %v", err)
	}
	next, err := service.Refresh(ctx, pair.RefreshToken, "install-1")
	if err != nil || next.Session != pair.Session || next.AccessToken == pair.AccessToken || next.RefreshToken == pair.RefreshToken {
		t.Fatalf("refresh: %+v %v", next, err)
	}
	if _, _, err := service.Authenticate(next.AccessToken); err != nil {
		t.Fatalf("refreshed access token: %v", err)
	}
	if n, err := service.LogoutAll(ctx, acc); err != nil || n != 1 {
		t.Fatalf("logout-all: %d %v", n, err)
	}
	if _, err := service.Refresh(ctx, next.RefreshToken, "install-1"); !errors.Is(err, token.ErrRevoked) {
		t.Fatalf("refresh after logout-all: %v", err)
	}
	if err := service.Logout(ctx, next.RefreshToken); err != nil {
		t.Fatalf("logout of a revoked family: %v", err)
	}
	if _, err := token.NewService(token.DefaultConfig(), nil, time.Now); !errors.Is(err, token.ErrNoKeys) {
		t.Fatalf("service without keys: %v", err)
	}
}

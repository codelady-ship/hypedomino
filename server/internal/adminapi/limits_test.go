package adminapi_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/adminapi"
	"github.com/Alievs-corp/hypedomino/server/internal/ratelimit"
)

func (a *api) from(ip, method, path, token string) *httptest.ResponseRecorder {
	a.t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader("{}"))
	request.RemoteAddr = "10.0.0.5:4431"
	request.Header.Set("X-Forwarded-For", ip)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	a.handler.ServeHTTP(recorder, request)
	return recorder
}

func expectThrottled(t *testing.T, recorder *httptest.ResponseRecorder, retryAfter string) {
	t.Helper()
	if recorder.Code != http.StatusTooManyRequests || !strings.Contains(recorder.Body.String(), `"error":"rate_limited"`) {
		t.Fatalf("got %d %s, want 429 rate_limited", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Retry-After"); got != retryAfter {
		t.Fatalf("Retry-After %q, want %q", got, retryAfter)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control %q, want no-store", got)
	}
}

func TestSignInIsLimitedPerClientAddress(t *testing.T) {
	a := newAPIWith(t, func(c *adminapi.Config) { c.Limits.SignIn = ratelimit.Rule{PerMinute: 1, Burst: 2} })
	for range 2 {
		if got := a.from("203.0.113.9", "POST", "/admin/nonce", ""); got.Code != http.StatusOK {
			t.Fatalf("nonce within burst: %d %s", got.Code, got.Body.String())
		}
	}
	expectThrottled(t, a.from("203.0.113.9", "POST", "/admin/session", ""), "60")
	expectThrottled(t, a.from("203.0.113.9", "POST", "/admin/nonce", ""), "60")
	if got := a.from("203.0.113.10", "POST", "/admin/nonce", ""); got.Code != http.StatusOK {
		t.Fatalf("another address shares the bucket: %d", got.Code)
	}
	a.now = a.now.Add(time.Minute)
	if got := a.from("203.0.113.9", "POST", "/admin/nonce", ""); got.Code != http.StatusOK {
		t.Fatalf("bucket did not refill after a minute: %d", got.Code)
	}
}

func TestSignedInRequestsAreLimitedPerAccount(t *testing.T) {
	a := newAPIWith(t, func(c *adminapi.Config) { c.Limits.Account = ratelimit.Rule{PerMinute: 2, Burst: 2} })
	root, ayla := a.signIn("root"), a.signIn("ayla")
	for range 2 {
		if got := a.from("203.0.113.9", "GET", "/admin/me", root); got.Code != http.StatusOK {
			t.Fatalf("me within burst: %d %s", got.Code, got.Body.String())
		}
	}
	expectThrottled(t, a.from("203.0.113.11", "GET", "/admin/audit", root), "30")
	if got := a.from("203.0.113.9", "GET", "/admin/me", ayla); got.Code != http.StatusOK {
		t.Fatalf("another account shares the bucket: %d", got.Code)
	}
	if got := a.from("203.0.113.9", "GET", "/admin/me", "unknown-token"); got.Code != http.StatusUnauthorized {
		t.Fatalf("an unknown session is refused before any bucket: %d", got.Code)
	}
	a.now = a.now.Add(30 * time.Second)
	if got := a.from("203.0.113.9", "GET", "/admin/me", root); got.Code != http.StatusOK {
		t.Fatalf("bucket did not refill: %d", got.Code)
	}
}

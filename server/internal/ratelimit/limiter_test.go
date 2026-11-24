package ratelimit

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

var epoch = time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)

type accessTokens struct{}

func (accessTokens) Verify(access string) (token.Claims, error) {
	account, ok := strings.CutPrefix(access, "access:")
	if !ok {
		return token.Claims{}, errors.New("invalid")
	}
	return token.Claims{Account: account}, nil
}

type brokenBuckets struct{}

func (brokenBuckets) Take(context.Context, string, Rule, time.Time) (time.Duration, error) {
	return 0, errors.New("redis down")
}

type clock struct{ now time.Time }

func (c *clock) Now() time.Time { return c.now }

func TestABucketAllowsItsBurstThenRefillsAtItsRate(t *testing.T) {
	memory := NewMemory()
	rule := Rule{PerMinute: 60, Burst: 3}
	steps := []struct {
		after time.Duration
		wait  time.Duration
	}{
		{0, 0}, {0, 0}, {0, 0},
		{0, time.Second},
		{500 * time.Millisecond, 500 * time.Millisecond},
		{500 * time.Millisecond, 0},
		{0, time.Second},
		{10 * time.Second, 0}, {0, 0}, {0, 0},
		{0, time.Second},
	}
	now := epoch
	for i, step := range steps {
		now = now.Add(step.after)
		wait, _ := memory.Take(context.Background(), "ip:1", rule, now)
		if wait != step.wait {
			t.Fatalf("step %d: wait %v, want %v", i, wait, step.wait)
		}
	}
	if wait, _ := memory.Take(context.Background(), "ip:2", rule, now); wait != 0 {
		t.Fatalf("another key waited %v", wait)
	}
}

func TestFullBucketsAreSweptFromMemory(t *testing.T) {
	memory := NewMemory()
	rule := Rule{PerMinute: 60, Burst: 5}
	memory.Take(context.Background(), "ip:1", rule, epoch)
	memory.Take(context.Background(), "ip:2", rule, epoch)
	memory.Take(context.Background(), "ip:3", rule, epoch.Add(2*time.Minute))
	if memory.Len() != 1 {
		t.Fatalf("%d buckets kept", memory.Len())
	}
}

func limiter(buckets Buckets, now *clock) *Limiter {
	config := DefaultConfig()
	config.IP = Rule{PerMinute: 60, Burst: 4}
	config.Account = Rule{PerMinute: 60, Burst: 3}
	config.SignIn = Rule{PerMinute: 60, Burst: 2}
	config.Spend = Rule{PerMinute: 60, Burst: 1}
	config.IPHeader = "CF-Connecting-IP"
	l := New(config, buckets, slog.New(slog.NewTextHandler(io.Discard, nil)), now.Now)
	l.Authenticate(accessTokens{})
	return l
}

type call struct {
	method, path, ip, access string
	status                   int
}

func serve(handler http.Handler, c call) *httptest.ResponseRecorder {
	request := httptest.NewRequest(c.method, c.path, nil)
	request.RemoteAddr = "10.0.0.1:5000"
	if c.ip != "" {
		request.Header.Set("CF-Connecting-IP", c.ip)
	}
	if c.access != "" {
		request.Header.Set("Authorization", "Bearer "+c.access)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestRequestsTakeFromTheirIPAccountSignInAndSpendBuckets(t *testing.T) {
	cases := map[string][]call{
		"ip burst": {
			{"GET", "/wallet", "1.1.1.1", "", 200}, {"GET", "/wallet", "1.1.1.1", "", 200}, {"GET", "/wallet", "1.1.1.1", "", 200}, {"GET", "/wallet", "1.1.1.1", "", 200},
			{"GET", "/wallet", "1.1.1.1", "", 429}, {"GET", "/wallet", "2.2.2.2", "", 200},
		},
		"connection address without the header": {
			{"GET", "/a", "", "", 200}, {"GET", "/a", "", "", 200}, {"GET", "/a", "", "", 200}, {"GET", "/a", "", "", 200}, {"GET", "/a", "", "", 429},
		},
		"account burst across ips": {
			{"GET", "/wallet", "1.1.1.1", "access:a", 200}, {"GET", "/wallet", "2.2.2.2", "access:a", 200}, {"GET", "/wallet", "3.3.3.3", "access:a", 200},
			{"GET", "/wallet", "4.4.4.4", "access:a", 429}, {"GET", "/wallet", "4.4.4.4", "access:b", 200},
		},
		"invalid token counts only the ip": {
			{"GET", "/wallet", "1.1.1.1", "forged", 200}, {"GET", "/wallet", "1.1.1.1", "forged", 200}, {"GET", "/wallet", "1.1.1.1", "forged", 200}, {"GET", "/wallet", "1.1.1.1", "forged", 200},
			{"GET", "/wallet", "1.1.1.1", "forged", 429},
		},
		"sign in and refresh share a bucket": {
			{"POST", "/auth/signin", "1.1.1.1", "", 200}, {"POST", "/auth/refresh", "1.1.1.1", "", 200}, {"POST", "/auth/signin", "1.1.1.1", "", 429},
			{"POST", "/auth/nonce", "1.1.1.1", "", 200}, {"POST", "/auth/signin", "2.2.2.2", "", 200},
		},
		"spend paths share a bucket": {
			{"POST", "/shop/buy", "1.1.1.1", "access:a", 200}, {"POST", "/promo/redeem", "1.1.1.1", "access:a", 429}, {"GET", "/shop", "1.1.1.1", "access:a", 200},
			{"POST", "/shop/complete", "2.2.2.2", "access:b", 200}, {"POST", "/shop/begin", "2.2.2.2", "access:b", 429},
		},
		"probes and store notifications are not limited": {
			{"GET", "/healthz", "1.1.1.1", "", 200}, {"GET", "/readyz", "1.1.1.1", "", 200}, {"GET", "/loadz", "1.1.1.1", "", 200}, {"GET", "/healthz", "1.1.1.1", "", 200},
			{"POST", "/store/google/notifications", "1.1.1.1", "", 200}, {"GET", "/wallet", "1.1.1.1", "", 200},
		},
	}
	for name, calls := range cases {
		now := &clock{now: epoch}
		l := limiter(NewMemory(), now)
		handled := 0
		handler := l.Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { handled++ }))
		refused := 0
		for i, c := range calls {
			recorder := serve(handler, c)
			if recorder.Code != c.status {
				t.Fatalf("%s: call %d %s %s answered %d, want %d", name, i, c.method, c.path, recorder.Code, c.status)
			}
			if c.status == 429 {
				refused++
			}
		}
		if handled != len(calls)-refused || l.Refused() != int64(refused) {
			t.Fatalf("%s: %d handled, %d refused, counter %d", name, handled, refused, l.Refused())
		}
	}
}

func TestARefusalTellsWhenToRetryAndRunsNoHandler(t *testing.T) {
	now := &clock{now: epoch}
	ran := 0
	ok := limiter(NewMemory(), now).Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { ran++ }))
	serve(ok, call{"POST", "/shop/buy", "1.1.1.1", "access:a", 200})
	recorder := serve(ok, call{"POST", "/shop/buy", "1.1.1.1", "access:a", 429})
	if ran != 1 || recorder.Code != http.StatusTooManyRequests || recorder.Header().Get("Retry-After") != "1" || strings.TrimSpace(recorder.Body.String()) != `{"error":"rate_limited"}` {
		t.Fatalf("%d %q %q", recorder.Code, recorder.Header().Get("Retry-After"), recorder.Body.String())
	}
	now.now = now.now.Add(time.Second)
	if recorder := serve(ok, call{"POST", "/shop/buy", "1.1.1.1", "access:a", 200}); recorder.Code != http.StatusOK {
		t.Fatalf("after the retry time: %d", recorder.Code)
	}
}

func TestABrokenStoreLetsRequestsPass(t *testing.T) {
	l := limiter(brokenBuckets{}, &clock{now: epoch})
	handler := l.Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	for range 10 {
		if recorder := serve(handler, call{"POST", "/shop/buy", "1.1.1.1", "access:a", 200}); recorder.Code != http.StatusOK {
			t.Fatalf("answered %d", recorder.Code)
		}
	}
}

func TestInvalidConfigIsRefused(t *testing.T) {
	if err := DefaultConfig().Validate(); err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*Config){
		"zero rate":    func(c *Config) { c.IP.PerMinute = 0 },
		"zero burst":   func(c *Config) { c.Spend.Burst = 0 },
		"no read time": func(c *Config) { c.ReadTimeout = 0 },
		"no idle time": func(c *Config) { c.IdleTimeout = 0 },
	} {
		config := DefaultConfig()
		change(&config)
		if config.Validate() == nil {
			t.Fatalf("%s accepted", name)
		}
	}
}

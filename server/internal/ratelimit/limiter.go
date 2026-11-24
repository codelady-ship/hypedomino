package ratelimit

import (
	"log/slog"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

var (
	unlimitedPaths = map[string]bool{"/healthz": true, "/readyz": true, "/loadz": true}
	signInPaths    = map[string]bool{"/auth/signin": true, "/auth/refresh": true}
	spendPaths     = map[string]bool{"/shop/buy": true, "/shop/begin": true, "/shop/complete": true, "/promo/redeem": true}
)

type Limiter struct {
	config   Config
	buckets  Buckets
	verifier token.Verifier
	logger   *slog.Logger
	now      func() time.Time
	refused  atomic.Int64
}

func New(config Config, buckets Buckets, logger *slog.Logger, now func() time.Time) *Limiter {
	return &Limiter{config: config, buckets: buckets, logger: logger, now: now}
}

func (l *Limiter) Authenticate(verifier token.Verifier) {
	l.verifier = verifier
}

func (l *Limiter) Refused() int64 {
	return l.refused.Load()
}

func (l *Limiter) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if wait := l.wait(r); wait > 0 {
			l.refuse(w, wait)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *Limiter) wait(r *http.Request) time.Duration {
	if unlimitedPaths[r.URL.Path] || strings.HasPrefix(r.URL.Path, "/store/") {
		return 0
	}
	ip := l.clientIP(r)
	if wait := l.take(r, "ip:"+ip, l.config.IP); wait > 0 {
		return wait
	}
	post := r.Method == http.MethodPost
	if post && signInPaths[r.URL.Path] {
		if wait := l.take(r, "signin:"+ip, l.config.SignIn); wait > 0 {
			return wait
		}
	}
	account := l.account(r)
	if account == "" {
		return 0
	}
	if wait := l.take(r, "account:"+account, l.config.Account); wait > 0 {
		return wait
	}
	if post && spendPaths[r.URL.Path] {
		return l.take(r, "spend:"+account, l.config.Spend)
	}
	return 0
}

func (l *Limiter) take(r *http.Request, key string, rule Rule) time.Duration {
	wait, err := l.buckets.Take(r.Context(), key, rule, l.now())
	if err != nil {
		l.logger.Warn("rate limit store failed, request passes", "error", err)
		return 0
	}
	return wait
}

func (l *Limiter) account(r *http.Request) string {
	if r.Header.Get("Authorization") == "" || l.verifier == nil {
		return ""
	}
	claims, err := token.Authorize(l.verifier, r)
	if err != nil {
		return ""
	}
	return claims.Account
}

func (l *Limiter) clientIP(r *http.Request) string {
	if l.config.IPHeader != "" {
		if forwarded := strings.TrimSpace(r.Header.Get(l.config.IPHeader)); forwarded != "" {
			return forwarded
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (l *Limiter) refuse(w http.ResponseWriter, wait time.Duration) {
	l.refused.Add(1)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(wait.Seconds()))))
	w.WriteHeader(http.StatusTooManyRequests)
	w.Write([]byte(`{"error":"rate_limited"}` + "\n"))
}

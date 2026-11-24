package adminapi

import (
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/Alievs-corp/hypedomino/server/internal/ratelimit"
)

type Limits struct {
	Account ratelimit.Rule
	SignIn  ratelimit.Rule
}

func DefaultLimits() Limits {
	return Limits{
		Account: ratelimit.Rule{PerMinute: 120, Burst: 40},
		SignIn:  ratelimit.Rule{PerMinute: 10, Burst: 10},
	}
}

func (l Limits) Validate() error {
	for _, rule := range []ratelimit.Rule{l.Account, l.SignIn} {
		if rule.PerMinute < 1 || rule.Burst < 1 {
			return errors.New("every per-minute rate and burst must be at least 1")
		}
	}
	return nil
}

func (h *Handler) signingIn(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.throttled(w, r, "signin:"+clientIP(r).String(), h.config.Limits.SignIn) {
			return
		}
		next(w, r)
	}
}

func (h *Handler) throttled(w http.ResponseWriter, r *http.Request, key string, rule ratelimit.Rule) bool {
	wait, err := h.buckets.Take(r.Context(), key, rule, h.now())
	if err != nil {
		h.logger.Warn("admin rate limit store failed, request passes", "error", err)
		return false
	}
	if wait <= 0 {
		return false
	}
	h.logger.Info("admin request refused", "reason", "rate_limited")
	w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(wait.Seconds()))))
	reply(w, http.StatusTooManyRequests, errorResponse{Error: "rate_limited"})
	return true
}

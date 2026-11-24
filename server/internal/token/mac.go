package token

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

const (
	MacHeader       = "X-HD-Mac"
	MaxMacBodyBytes = 64 * 1024
)

type RequestKeys interface {
	Verifier
	RequestKey(claims Claims) ([]byte, error)
}

type MacFailures interface {
	Observe(ctx context.Context, account string)
}

func RequireMac(keys RequestKeys, failures MacFailures, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}
		access := bearer(r)
		claims, err := keys.Verify(access)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxMacBodyBytes))
		if err != nil {
			rejectMac(w, http.StatusRequestEntityTooLarge, "body_too_large")
			return
		}
		key, err := keys.RequestKey(claims)
		if err != nil || !validMac(key, r, access, body) {
			failures.Observe(r.Context(), claims.Account)
			rejectMac(w, http.StatusUnauthorized, "bad_mac")
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		next.ServeHTTP(w, r)
	})
}

func SignRequest(key []byte, method, pathAndQuery, access string, body []byte) string {
	return base64.RawURLEncoding.EncodeToString(requestMac(key, method, pathAndQuery, access, body))
}

func requestMac(key []byte, method, pathAndQuery, access string, body []byte) []byte {
	h := hmac.New(sha256.New, key)
	io.WriteString(h, method+"\n"+pathAndQuery+"\n"+access+"\n")
	h.Write(body)
	return h.Sum(nil)
}

func validMac(key []byte, r *http.Request, access string, body []byte) bool {
	presented, err := base64.RawURLEncoding.DecodeString(r.Header.Get(MacHeader))
	if err != nil || len(presented) == 0 {
		return false
	}
	return hmac.Equal(presented, requestMac(key, r.Method, r.URL.RequestURI(), access, body))
}

func bearer(r *http.Request) string {
	scheme, value, _ := strings.Cut(r.Header.Get("Authorization"), " ")
	if !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(value)
}

func rejectMac(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": code})
}

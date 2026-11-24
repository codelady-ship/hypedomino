package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/account"
	"github.com/Alievs-corp/hypedomino/server/internal/appeal"
	"github.com/Alievs-corp/hypedomino/server/internal/idtoken"
	"github.com/Alievs-corp/hypedomino/server/internal/metrics"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

const MaxBodyBytes = 16 * 1024

const unknownProvider = "unknown"

type Verifier interface {
	IssueNonce(ctx context.Context) (string, error)
	Verify(ctx context.Context, provider idtoken.Provider, token, rawNonce string) (idtoken.Identity, error)
}

type Accounts interface {
	SignIn(ctx context.Context, identity idtoken.Identity) (account.Account, bool, error)
	AdoptCountry(ctx context.Context, id, code string) error
}

type Tokens interface {
	Issue(ctx context.Context, account, device string) (token.Pair, error)
	Refresh(ctx context.Context, refresh, device string) (token.Pair, error)
	Logout(ctx context.Context, refresh string) error
	LogoutAll(ctx context.Context, account string) (int, error)
	Verify(access string) (token.Claims, error)
}

type Connections interface {
	RevokeAccount(account string) int
}

type Handler struct {
	countryHeader string
	deletion      Deletion
	appeals       *appeal.Granter
	verifier      Verifier
	accounts      Accounts
	tokens        Tokens
	connections   Connections
	logger        *slog.Logger
	mux           *http.ServeMux
	signIns       *metrics.Counter
}

func New(verifier Verifier, accounts Accounts, tokens Tokens, connections Connections, logger *slog.Logger) *Handler {
	h := &Handler{verifier: verifier, accounts: accounts, tokens: tokens, connections: connections, logger: logger, mux: http.NewServeMux(), signIns: newSignIns()}
	h.mux.HandleFunc("POST /auth/nonce", h.nonce)
	h.mux.HandleFunc("POST /auth/signin", h.signIn)
	h.mux.HandleFunc("POST /auth/refresh", h.refresh)
	h.mux.HandleFunc("POST /auth/logout", h.logout)
	h.mux.HandleFunc("POST /auth/logout-all", h.logoutAll)
	return h
}

func newSignIns() *metrics.Counter {
	return metrics.NewCounter("hd_signins_total", "Sign-in attempts by identity provider and result: ok or the refusal reason sent to the client.", metrics.Labels{"provider", "result"})
}

func (h *Handler) SignIns() io.WriterTo {
	return h.signIns
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

type nonceResponse struct {
	Nonce string `json:"nonce"`
}

type signInRequest struct {
	Provider string `json:"provider"`
	IDToken  string `json:"id_token"`
	Nonce    string `json:"nonce"`
	DeviceID string `json:"device_id"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
	DeviceID     string `json:"device_id"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	RequestKey   string `json:"request_key"`
	AccountID    string `json:"account_id"`
	Created      bool   `json:"created,omitempty"`
}

type errorResponse struct {
	Error       string        `json:"error"`
	Category    string        `json:"category,omitempty"`
	BannedUntil *time.Time    `json:"banned_until,omitempty"`
	Appeal      *appeal.Grant `json:"appeal,omitempty"`
}

func (h *Handler) nonce(w http.ResponseWriter, r *http.Request) {
	nonce, err := h.verifier.IssueNonce(r.Context())
	if err != nil {
		h.logger.Error("nonce issue failed", "error", err)
		reply(w, http.StatusServiceUnavailable, errorResponse{Error: "unavailable"})
		return
	}
	reply(w, http.StatusOK, nonceResponse{Nonce: nonce})
}

func (h *Handler) signIn(w http.ResponseWriter, r *http.Request) {
	provider, result := h.attemptSignIn(w, r)
	h.signIns.Inc(metrics.Labels{provider, result})
}

func (h *Handler) attemptSignIn(w http.ResponseWriter, r *http.Request) (string, string) {
	var request signInRequest
	if !decode(w, r, &request) {
		return unknownProvider, "malformed"
	}
	provider, ok := parseProvider(request.Provider)
	if !ok {
		reply(w, http.StatusBadRequest, errorResponse{Error: "unknown_provider"})
		return unknownProvider, "unknown_provider"
	}
	name := provider.String()
	if request.DeviceID == "" {
		reply(w, http.StatusBadRequest, errorResponse{Error: "device_required"})
		return name, "device_required"
	}
	identity, err := h.verifier.Verify(r.Context(), provider, request.IDToken, request.Nonce)
	if err != nil {
		return name, h.refuse(w, err)
	}
	a, created, err := h.accounts.SignIn(r.Context(), identity)
	if errors.Is(err, account.ErrDeletionPending) {
		reply(w, http.StatusConflict, deletionResponse{Error: "deletion_pending", DeleteAt: h.deleteAt(a)})
		return name, "deletion_pending"
	}
	if err != nil {
		return name, h.refuse(w, err)
	}
	if a.Country == "" && h.countryHeader != "" {
		if err := h.accounts.AdoptCountry(r.Context(), a.ID, strings.ToUpper(strings.TrimSpace(r.Header.Get(h.countryHeader)))); err != nil {
			return name, h.refuse(w, err)
		}
	}
	pair, err := h.tokens.Issue(r.Context(), a.ID, request.DeviceID)
	if err != nil {
		return name, h.refuse(w, err)
	}
	reply(w, http.StatusOK, tokenResponse{AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken, ExpiresIn: pair.ExpiresIn, RequestKey: base64.StdEncoding.EncodeToString(pair.RequestKey), AccountID: a.ID, Created: created})
	return name, "ok"
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	var request refreshRequest
	if !decode(w, r, &request) {
		return
	}
	if request.DeviceID == "" {
		reply(w, http.StatusBadRequest, errorResponse{Error: "device_required"})
		return
	}
	pair, err := h.tokens.Refresh(r.Context(), request.RefreshToken, request.DeviceID)
	if err != nil {
		h.refuse(w, err)
		return
	}
	claims, err := h.tokens.Verify(pair.AccessToken)
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, tokenResponse{AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken, ExpiresIn: pair.ExpiresIn, RequestKey: base64.StdEncoding.EncodeToString(pair.RequestKey), AccountID: claims.Account})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	var request logoutRequest
	if !decode(w, r, &request) {
		return
	}
	if err := h.tokens.Logout(r.Context(), request.RefreshToken); err != nil {
		h.refuse(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) logoutAll(w http.ResponseWriter, r *http.Request) {
	claims, err := h.tokens.Verify(bearer(r))
	if err != nil {
		h.refuse(w, err)
		return
	}
	if _, err := h.tokens.LogoutAll(r.Context(), claims.Account); err != nil {
		h.refuse(w, err)
		return
	}
	h.connections.RevokeAccount(claims.Account)
	w.WriteHeader(http.StatusNoContent)
}

func bearer(r *http.Request) string {
	scheme, value, _ := strings.Cut(r.Header.Get("Authorization"), " ")
	if !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(value)
}

func decode(w http.ResponseWriter, r *http.Request, into any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxBodyBytes)).Decode(into); err != nil {
		reply(w, http.StatusBadRequest, errorResponse{Error: "malformed"})
		return false
	}
	return true
}

func (h *Handler) refuse(w http.ResponseWriter, err error) string {
	status, code := classify(err)
	if status >= http.StatusInternalServerError {
		h.logger.Error("auth request failed", "error", err)
	} else {
		h.logger.Info("auth request refused", "reason", code)
	}
	reply(w, status, h.refusal(code, err))
	return code
}

func (h *Handler) EnableAppeals(appeals *appeal.Granter) {
	h.appeals = appeals
}

func (h *Handler) refusal(code string, err error) errorResponse {
	var banned *account.BannedError
	if !errors.As(err, &banned) {
		return errorResponse{Error: code}
	}
	grant, grantErr := h.appeals.Grant(banned.Account, banned.SanctionID)
	if grantErr != nil {
		h.logger.Error("appeal grant failed", "error", grantErr)
	}
	return errorResponse{Error: code, Category: banned.Category, BannedUntil: banned.Until, Appeal: grant}
}

func classify(err error) (int, string) {
	switch {
	case errors.Is(err, idtoken.ErrKeyFetch):
		return http.StatusServiceUnavailable, "provider_unavailable"
	case errors.Is(err, idtoken.ErrExpired), errors.Is(err, idtoken.ErrNotYetValid):
		return http.StatusUnauthorized, "token_expired"
	case errors.Is(err, idtoken.ErrNonce), errors.Is(err, idtoken.ErrNonceUsed):
		return http.StatusUnauthorized, "nonce_rejected"
	case errors.Is(err, idtoken.ErrMalformed), errors.Is(err, idtoken.ErrAlgorithm), errors.Is(err, idtoken.ErrUnknownKey),
		errors.Is(err, idtoken.ErrSignature), errors.Is(err, idtoken.ErrIssuer), errors.Is(err, idtoken.ErrAudience),
		errors.Is(err, idtoken.ErrSubject), errors.Is(err, idtoken.ErrUnknownProvider), errors.Is(err, account.ErrInvalidIdentity):
		return http.StatusUnauthorized, "token_rejected"
	case errors.Is(err, token.ErrExpired):
		return http.StatusUnauthorized, "access_expired"
	case errors.Is(err, token.ErrMalformed), errors.Is(err, token.ErrSignature), errors.Is(err, token.ErrUnknownKey), errors.Is(err, token.ErrIssuer):
		return http.StatusUnauthorized, "access_rejected"
	case errors.Is(err, token.ErrReused), errors.Is(err, token.ErrDeviceMismatch):
		return http.StatusUnauthorized, "refresh_reused"
	case errors.Is(err, token.ErrRefreshExpired):
		return http.StatusUnauthorized, "refresh_expired"
	case errors.Is(err, token.ErrUnknownRefresh), errors.Is(err, token.ErrRevoked), errors.Is(err, token.ErrInvalidFamily):
		return http.StatusUnauthorized, "refresh_rejected"
	case errors.Is(err, account.ErrBanned):
		return http.StatusForbidden, "banned"
	case errors.Is(err, account.ErrDeleted):
		return http.StatusGone, "deleted"
	case errors.Is(err, account.ErrUnknown):
		return http.StatusNotFound, "account_unknown"
	case errors.Is(err, account.ErrNoDeletionPending):
		return http.StatusConflict, "no_deletion_pending"
	}
	return http.StatusInternalServerError, "internal"
}

func parseProvider(name string) (idtoken.Provider, bool) {
	switch name {
	case idtoken.ProviderGoogle.String():
		return idtoken.ProviderGoogle, true
	case idtoken.ProviderApple.String():
		return idtoken.ProviderApple, true
	}
	return 0, false
}

func reply(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func (h *Handler) SetCountryHeader(name string) {
	h.countryHeader = name
}

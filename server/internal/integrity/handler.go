package integrity

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

const (
	MaxBodyBytes       = 1024
	MaxAttestBodyBytes = 16 * 1024
)

var ErrKeyExists = errors.New("integrity: key already attested")

type KeyStore interface {
	Save(ctx context.Context, account string, key AttestedKey, at time.Time) error
}

type Handler struct {
	verifier token.Verifier
	nonces   *Nonces
	logger   *slog.Logger
	mux      *http.ServeMux
	attest   *AppAttest
	keys     KeyStore
	clock    Clock
}

func NewHandler(verifier token.Verifier, nonces *Nonces, logger *slog.Logger) *Handler {
	h := &Handler{verifier: verifier, nonces: nonces, logger: logger, mux: http.NewServeMux()}
	h.mux.HandleFunc("POST /integrity/nonce", h.nonce)
	return h
}

func (h *Handler) EnableAppAttest(attest *AppAttest, keys KeyStore, clock Clock) {
	h.attest, h.keys, h.clock = attest, keys, clock
	h.mux.HandleFunc("POST /integrity/attest", h.attestKey)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

type nonceRequest struct {
	Purpose string `json:"purpose"`
}

type nonceResponse struct {
	Nonce     string `json:"nonce"`
	ExpiresIn int    `json:"expires_in"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) nonce(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	var request nonceRequest
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxBodyBytes)).Decode(&request) != nil {
		reply(w, http.StatusBadRequest, errorResponse{Error: "bad_request"})
		return
	}
	purpose, ok := parsePurpose(request.Purpose)
	if !ok {
		reply(w, http.StatusBadRequest, errorResponse{Error: "unknown_purpose"})
		return
	}
	nonce, err := h.nonces.Issue(r.Context(), Binding{Account: claims.Account, Session: claims.Session, Purpose: purpose})
	if err != nil {
		h.unavailable(w, claims.Account, "integrity nonce not issued", err)
		return
	}
	reply(w, http.StatusOK, nonceResponse{Nonce: nonce.Value, ExpiresIn: int(h.nonces.config.TTL.Seconds())})
}

func parsePurpose(name string) (Purpose, bool) {
	for purpose := PurposeSessionStart; purpose.valid(); purpose++ {
		if purpose.String() == name {
			return purpose, true
		}
	}
	return 0, false
}

func reply(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

type attestRequest struct {
	Nonce       string `json:"nonce"`
	KeyID       string `json:"key_id"`
	Attestation string `json:"attestation"`
}

func (h *Handler) attestKey(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	var request attestRequest
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxAttestBodyBytes)).Decode(&request) != nil {
		reply(w, http.StatusBadRequest, errorResponse{Error: "bad_request"})
		return
	}
	keyID, keyErr := base64.StdEncoding.DecodeString(request.KeyID)
	attestation, attestErr := base64.StdEncoding.DecodeString(request.Attestation)
	if keyErr != nil || attestErr != nil || len(keyID) == 0 || len(attestation) == 0 {
		reply(w, http.StatusBadRequest, errorResponse{Error: "bad_request"})
		return
	}
	binding := Binding{Account: claims.Account, Session: claims.Session, Purpose: PurposeSessionStart}
	decision, key, err := h.attest.Attest(r.Context(), Policy{}, binding, request.Nonce, keyID, attestation)
	if err != nil {
		h.unavailable(w, claims.Account, "attestation not checked", err)
		return
	}
	if !decision.Passed() {
		h.logger.Warn("attestation refused", "account", claims.Account, "reasons", decision.ReasonNames())
		reply(w, http.StatusForbidden, errorResponse{Error: "attestation_failed"})
		return
	}
	err = h.keys.Save(r.Context(), claims.Account, key, h.clock.Now())
	if errors.Is(err, ErrKeyExists) {
		reply(w, http.StatusConflict, errorResponse{Error: "key_exists"})
		return
	}
	if err != nil {
		h.unavailable(w, claims.Account, "attested key not stored", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) authenticate(w http.ResponseWriter, r *http.Request) (token.Claims, bool) {
	claims, err := token.Authorize(h.verifier, r)
	switch {
	case errors.Is(err, token.ErrExpired):
		reply(w, http.StatusUnauthorized, errorResponse{Error: "access_expired"})
		return token.Claims{}, false
	case err != nil:
		reply(w, http.StatusUnauthorized, errorResponse{Error: "access_rejected"})
		return token.Claims{}, false
	}
	return claims, true
}

func (h *Handler) unavailable(w http.ResponseWriter, account, message string, err error) {
	h.logger.Error(message, "account", account, "error", err)
	reply(w, http.StatusServiceUnavailable, errorResponse{Error: "unavailable"})
}

package adminapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/adminaccount"
	"github.com/Alievs-corp/hypedomino/server/internal/adminsession"
	"github.com/Alievs-corp/hypedomino/server/internal/idtoken"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

type nonceResponse struct {
	Nonce string `json:"nonce"`
}

type signInRequest struct {
	IDToken string `json:"id_token"`
	Nonce   string `json:"nonce"`
}

type accountView struct {
	ID          string      `json:"id"`
	Email       string      `json:"email"`
	DisplayName string      `json:"display_name"`
	Roles       []rbac.Role `json:"roles"`
	State       string      `json:"state"`
	CreatedAt   time.Time   `json:"created_at"`
	LastLoginAt *time.Time  `json:"last_login_at,omitempty"`
	DisabledAt  *time.Time  `json:"disabled_at,omitempty"`
	Locale      string      `json:"locale,omitempty"`
}

type sessionView struct {
	ID              string    `json:"id"`
	AuthenticatedAt time.Time `json:"authenticated_at"`
	StepUpUntil     time.Time `json:"step_up_until"`
	IdleUntil       time.Time `json:"idle_until"`
	ExpiresAt       time.Time `json:"expires_at"`
}

type signInResponse struct {
	Token   string      `json:"token"`
	Session sessionView `json:"session"`
	Account accountView `json:"account"`
}

type meResponse struct {
	Session     sessionView                   `json:"session"`
	Account     accountView                   `json:"account"`
	Permissions map[rbac.Object][]rbac.Action `json:"permissions"`
}

func viewAccount(a adminaccount.Account) accountView {
	return accountView{
		ID: a.ID, Email: a.Email, DisplayName: a.DisplayName, Roles: a.Roles, State: string(a.State),
		CreatedAt: a.CreatedAt, LastLoginAt: optionalTime(a.LastLoginAt), DisabledAt: optionalTime(a.DisabledAt), Locale: a.Locale,
	}
}

func optionalTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func (h *Handler) viewSession(s adminsession.Session) sessionView {
	idleUntil := s.LastSeenAt.Add(h.config.Session.Idle)
	if idleUntil.After(s.ExpiresAt) {
		idleUntil = s.ExpiresAt
	}
	return sessionView{
		ID:              s.ID,
		AuthenticatedAt: s.AuthenticatedAt,
		StepUpUntil:     s.AuthenticatedAt.Add(h.config.Session.StepUp),
		IdleUntil:       idleUntil,
		ExpiresAt:       s.ExpiresAt,
	}
}

func permissions(roles []rbac.Role) map[rbac.Object][]rbac.Action {
	granted := map[rbac.Object][]rbac.Action{}
	for _, object := range rbac.Objects() {
		if actions := rbac.Granted(roles, object); len(actions) > 0 {
			granted[object] = actions
		}
	}
	return granted
}

func (h *Handler) nonce(w http.ResponseWriter, r *http.Request) {
	nonce, err := h.verifier.IssueNonce(r.Context())
	if err != nil {
		h.logger.Error("admin nonce issue failed", "error", err)
		reply(w, http.StatusServiceUnavailable, errorResponse{Error: "unavailable"})
		return
	}
	reply(w, http.StatusOK, nonceResponse{Nonce: nonce})
}

func (h *Handler) companyEmail(ctx context.Context, request signInRequest) (string, error) {
	identity, err := h.verifier.Verify(ctx, idtoken.ProviderGoogle, request.IDToken, request.Nonce)
	if err != nil {
		return "", err
	}
	email := adminaccount.NormalizeEmail(identity.Email)
	domain := strings.ToLower(h.config.GoogleDomain)
	if !identity.EmailVerified || strings.ToLower(identity.HostedDomain) != domain || !strings.HasSuffix(email, "@"+domain) {
		return "", errNotWorkspace
	}
	return email, nil
}

func (h *Handler) signIn(w http.ResponseWriter, r *http.Request) {
	var request signInRequest
	if !decode(w, r, &request) {
		return
	}
	email, err := h.companyEmail(r.Context(), request)
	if err != nil {
		h.refuse(w, err)
		return
	}
	now := h.now()
	account, err := h.accounts.SignIn(r.Context(), email, now)
	if errors.Is(err, adminaccount.ErrNotFound) || errors.Is(err, adminaccount.ErrDisabled) {
		err = errAccountRefused
	}
	if err != nil {
		h.refuse(w, err)
		return
	}
	token, session, err := h.sessions.Open(r.Context(), account.ID, now)
	if err != nil {
		h.refuse(w, err)
		return
	}
	h.logger.Info("admin signed in", "account", account.ID, "session", session.ID)
	reply(w, http.StatusOK, signInResponse{Token: token, Session: h.viewSession(session), Account: viewAccount(account)})
}

func (h *Handler) stepUp(w http.ResponseWriter, r *http.Request, p principal) {
	var request signInRequest
	if !decode(w, r, &request) {
		return
	}
	email, err := h.companyEmail(r.Context(), request)
	if err == nil && email != p.account.Email {
		err = errDifferentAccount
	}
	if err != nil {
		h.refuse(w, err)
		return
	}
	session, err := h.sessions.StepUp(r.Context(), p.session.ID, h.now())
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, h.viewSession(session))
}

func (h *Handler) signOut(w http.ResponseWriter, r *http.Request, p principal) {
	if err := h.sessions.Revoke(r.Context(), p.session.ID, h.now()); err != nil {
		h.refuse(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request, p principal) {
	reply(w, http.StatusOK, meResponse{Session: h.viewSession(p.session), Account: viewAccount(p.account), Permissions: permissions(p.account.Roles)})
}

type localeRequest struct {
	Locale string `json:"locale"`
}

func (h *Handler) chooseLocale(w http.ResponseWriter, r *http.Request, p principal) {
	var request localeRequest
	if !decode(w, r, &request) {
		return
	}
	account, err := h.accounts.SetLocale(r.Context(), p.account.ID, request.Locale)
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, viewAccount(account))
}

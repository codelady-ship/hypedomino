package profile

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/account"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

const MaxBodyBytes = 4 * 1024

type Profiles interface {
	Get(ctx context.Context, account string) (Profile, bool, error)
	Exists(ctx context.Context, account string) (bool, error)
	SetNickname(ctx context.Context, account, raw string, now time.Time) (Profile, error)
	Suggest(ctx context.Context, now time.Time) (string, error)
	RenameAvailableAt(p Profile) (time.Time, bool)
}

type Countries interface {
	Country(ctx context.Context, id string) (account.Country, error)
	SetCountry(ctx context.Context, id, code string, now time.Time, cooldown time.Duration) (account.Country, error)
}

type Languages interface {
	Language(ctx context.Context, id string) (string, error)
	SetLanguage(ctx context.Context, id, code string) (string, error)
}

type Handler struct {
	verifier  token.Verifier
	profiles  Profiles
	countries Countries
	languages Languages
	cooldown  time.Duration
	ranked    func(ctx context.Context, account string) (any, error)
	medals    func(ctx context.Context, account string) (any, error)
	appeal    func(ctx context.Context, account string) (any, error)
	standing  func(ctx context.Context, account string, now time.Time) (int, int64, error)
	logger    *slog.Logger
	now       func() time.Time
	mux       *http.ServeMux
}

func NewHandler(verifier token.Verifier, profiles Profiles, logger *slog.Logger, now func() time.Time) *Handler {
	h := &Handler{verifier: verifier, profiles: profiles, logger: logger, now: now, mux: http.NewServeMux()}
	h.mux.HandleFunc("GET /profile/me", h.me)
	h.mux.HandleFunc("PUT /profile/nickname", h.nickname)
	h.mux.HandleFunc("GET /profile/country", h.country)
	h.mux.HandleFunc("PUT /profile/country", h.setCountry)
	h.mux.HandleFunc("GET /profile/language", h.language)
	h.mux.HandleFunc("PUT /profile/language", h.setLanguage)
	h.mux.HandleFunc("GET /profile/{account}", h.public)
	return h
}

func (h *Handler) SetCountries(countries Countries, cooldown time.Duration) {
	h.countries, h.cooldown = countries, cooldown
}

func (h *Handler) SetLanguages(languages Languages) {
	h.languages = languages
}

func (h *Handler) SetRanked(ranked func(ctx context.Context, account string) (any, error)) {
	h.ranked = ranked
}

func (h *Handler) SetMedals(medals func(ctx context.Context, account string) (any, error)) {
	h.medals = medals
}

func (h *Handler) SetRenameAppeals(appeal func(ctx context.Context, account string) (any, error)) {
	h.appeal = appeal
}

func (h *Handler) SetStanding(standing func(ctx context.Context, account string, now time.Time) (int, int64, error)) {
	h.standing = standing
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

type ownView struct {
	AccountID         string   `json:"account_id"`
	Nickname          string   `json:"nickname"`
	SetupRequired     bool     `json:"setup_required"`
	RenameRequired    bool     `json:"rename_required"`
	RenameAppeal      any      `json:"rename_appeal,omitempty"`
	SuggestedNickname string   `json:"suggested_nickname,omitempty"`
	RenameAvailableAt string   `json:"rename_available_at,omitempty"`
	Level             int      `json:"level"`
	XP                int64    `json:"xp"`
	Loadout           Loadout  `json:"loadout"`
	Stats             ownStats `json:"stats"`
}

type ownStats struct {
	Games          int                  `json:"games"`
	Wins           int                  `json:"wins"`
	Losses         int                  `json:"losses"`
	WinRatePercent int                  `json:"win_rate_percent"`
	Streak         int                  `json:"streak"`
	BestStreak     int                  `json:"best_streak"`
	Modes          map[string]ModeStats `json:"modes"`
}

type publicView struct {
	AccountID string      `json:"account_id"`
	Nickname  string      `json:"nickname"`
	Level     int         `json:"level"`
	Loadout   Loadout     `json:"loadout"`
	Stats     publicStats `json:"stats"`
	Ranked    any         `json:"ranked,omitempty"`
	Medals    any         `json:"medals,omitempty"`
}

type publicStats struct {
	Games          int `json:"games"`
	WinRatePercent int `json:"win_rate_percent"`
	BestStreak     int `json:"best_streak"`
}

type nicknameRequest struct {
	Nickname string `json:"nickname"`
}

type errorResponse struct {
	Error   string `json:"error"`
	RetryAt string `json:"retry_at,omitempty"`
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	p, found, err := h.profiles.Get(r.Context(), claims.Account)
	if err != nil {
		h.fail(w, err)
		return
	}
	if found {
		h.replyOwn(w, r, h.own(p))
		return
	}
	suggestion, err := h.profiles.Suggest(r.Context(), h.now())
	if err != nil {
		h.fail(w, err)
		return
	}
	h.replyOwn(w, r, ownView{AccountID: claims.Account, SetupRequired: true, SuggestedNickname: suggestion, Stats: ownStats{Modes: map[string]ModeStats{}}})
}

func (h *Handler) nickname(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	var request nicknameRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxBodyBytes)).Decode(&request); err != nil {
		reply(w, http.StatusBadRequest, errorResponse{Error: "malformed"})
		return
	}
	p, err := h.profiles.SetNickname(r.Context(), claims.Account, request.Nickname, h.now())
	if err != nil {
		h.refuseNickname(w, err)
		return
	}
	h.replyOwn(w, r, h.own(p))
}

func (h *Handler) public(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authenticate(w, r); !ok {
		return
	}
	account := r.PathValue("account")
	p, found, err := h.profiles.Get(r.Context(), account)
	if err != nil {
		h.fail(w, err)
		return
	}
	if !found {
		reply(w, http.StatusNotFound, errorResponse{Error: "profile.unknown"})
		return
	}
	view := publicView{
		AccountID: p.Account,
		Nickname:  p.Nickname,
		Loadout:   p.Loadout,
		Stats:     publicStats{Games: p.Stats.Games, WinRatePercent: p.Stats.WinRatePercent(), BestStreak: p.Stats.BestStreak},
	}
	if view.Level, _, err = h.progress(r.Context(), p.Account); err != nil {
		h.fail(w, err)
		return
	}
	if view.Ranked, err = section(r.Context(), h.ranked, p.Account); err != nil {
		h.fail(w, err)
		return
	}
	if view.Medals, err = section(r.Context(), h.medals, p.Account); err != nil {
		h.fail(w, err)
		return
	}
	reply(w, http.StatusOK, view)
}

func (h *Handler) progress(ctx context.Context, account string) (int, int64, error) {
	if h.standing == nil {
		return 1, 0, nil
	}
	return h.standing(ctx, account, h.now())
}

func (h *Handler) replyOwn(w http.ResponseWriter, r *http.Request, view ownView) {
	var err error
	if view.Level, view.XP, err = h.progress(r.Context(), view.AccountID); err != nil {
		h.fail(w, err)
		return
	}
	if view.RenameRequired {
		if view.RenameAppeal, err = section(r.Context(), h.appeal, view.AccountID); err != nil {
			h.fail(w, err)
			return
		}
	}
	reply(w, http.StatusOK, view)
}

func section(ctx context.Context, read func(ctx context.Context, account string) (any, error), account string) (any, error) {
	if read == nil {
		return nil, nil
	}
	return read(ctx, account)
}

func (h *Handler) own(p Profile) ownView {
	view := ownView{
		RenameRequired: p.RenameRequired,
		AccountID:      p.Account,
		Nickname:       p.Nickname,
		Loadout:        p.Loadout,
		Stats: ownStats{
			Games: p.Stats.Games, Wins: p.Stats.Wins, Losses: p.Stats.Losses, WinRatePercent: p.Stats.WinRatePercent(),
			Streak: p.Stats.Streak, BestStreak: p.Stats.BestStreak, Modes: p.Stats.Modes,
		},
	}
	if view.Stats.Modes == nil {
		view.Stats.Modes = map[string]ModeStats{}
	}
	if available, cooling := h.profiles.RenameAvailableAt(p); cooling && h.now().Before(available) {
		view.RenameAvailableAt = available.UTC().Format(time.RFC3339)
	}
	return view
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

func (h *Handler) refuseNickname(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrTaken):
		reply(w, http.StatusConflict, errorResponse{Error: ErrTaken.Error()})
	case errors.Is(err, ErrCooldown):
		_, retry, _ := strings.Cut(err.Error(), " until ")
		reply(w, http.StatusTooManyRequests, errorResponse{Error: ErrCooldown.Error(), RetryAt: retry})
	case errors.Is(err, ErrInvalidLength), errors.Is(err, ErrInvalidCharacters), errors.Is(err, ErrInvalidSeparators), errors.Is(err, ErrMixedScripts), errors.Is(err, ErrDigitsOnly),
		errors.Is(err, ErrUnchanged), errors.Is(err, ErrNotAllowed):
		reply(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
	default:
		h.fail(w, err)
	}
}

func (h *Handler) fail(w http.ResponseWriter, err error) {
	h.logger.Error("profile request failed", "error", err)
	reply(w, http.StatusInternalServerError, errorResponse{Error: "internal"})
}

func reply(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

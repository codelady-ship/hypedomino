package pass

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

type Passes interface {
	View(ctx context.Context, account string, now time.Time) (View, error)
	BuyLevels(ctx context.Context, account string, count int, currency wallet.Currency, requestID string, now time.Time) (LevelPurchase, error)
}

type Handler struct {
	verifier token.Verifier
	store    Passes
	logger   *slog.Logger
	now      func() time.Time
	mux      *http.ServeMux
}

func NewHandler(verifier token.Verifier, store Passes, logger *slog.Logger, now func() time.Time) *Handler {
	h := &Handler{verifier: verifier, store: store, logger: logger, now: now, mux: http.NewServeMux()}
	h.mux.HandleFunc("GET /pass", h.view)
	h.mux.HandleFunc("POST /pass/levels", h.levels)
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

type errorResponse struct {
	Error string `json:"error"`
}

type View struct {
	Season            string                           `json:"season"`
	StartsAt          time.Time                        `json:"starts_at"`
	EndsAt            time.Time                        `json:"ends_at"`
	ConfigVersion     int64                            `json:"config_version"`
	Levels            int                              `json:"levels"`
	XP                int64                            `json:"xp"`
	Level             int                              `json:"level"`
	LevelXP           int64                            `json:"level_xp"`
	NextLevelXP       *int64                           `json:"next_level_xp"`
	Premium           bool                             `json:"premium"`
	LevelCoinPrice    int64                            `json:"level_coin_price"`
	LevelCrystalPrice int64                            `json:"level_crystal_price"`
	Tracks            map[string]map[int]reward.Bundle `json:"tracks"`
}

func NewView(config *Snapshot, season Season, progress Progress, version int64) View {
	level := season.Curve.LevelFor(progress.XP)
	view := View{
		Season:            season.ID,
		StartsAt:          season.Start,
		EndsAt:            season.End,
		ConfigVersion:     version,
		Levels:            season.Curve.Levels,
		XP:                progress.XP,
		Level:             level,
		LevelXP:           season.Curve.XpToReach(level),
		Premium:           progress.Premium,
		LevelCoinPrice:    config.LevelPrice(wallet.Coin),
		LevelCrystalPrice: config.LevelPrice(wallet.Crystal),
		Tracks:            map[string]map[int]reward.Bundle{Free: season.Free, Premium: season.Premium},
	}
	if level < season.Curve.Levels {
		next := season.Curve.XpToReach(level + 1)
		view.NextLevelXP = &next
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

func (h *Handler) view(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	view, err := h.store.View(r.Context(), claims.Account, h.now())
	switch {
	case errors.Is(err, ErrNoSeason):
		reply(w, http.StatusNotFound, errorResponse{Error: "pass.no_season"})
	case err != nil:
		h.logger.Error("pass view failed", "error", err)
		reply(w, http.StatusInternalServerError, errorResponse{Error: "internal"})
	default:
		reply(w, http.StatusOK, view)
	}
}

func reply(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

type levelsRequest struct {
	Count     int             `json:"count"`
	Currency  wallet.Currency `json:"currency"`
	RequestID string          `json:"request_id"`
}

var levelRefusals = []struct {
	err    error
	status int
	code   string
}{
	{ErrRequest, http.StatusBadRequest, ErrRequest.Error()},
	{ErrLevelCount, http.StatusConflict, ErrLevelCount.Error()},
	{ErrNoSeason, http.StatusConflict, "pass.no_season"},
	{wallet.ErrInsufficient, http.StatusPaymentRequired, "wallet.insufficient"},
}

func (h *Handler) levels(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	var request levelsRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&request); err != nil {
		reply(w, http.StatusBadRequest, errorResponse{Error: "malformed"})
		return
	}
	purchase, err := h.store.BuyLevels(r.Context(), claims.Account, request.Count, cmp.Or(request.Currency, wallet.Coin), request.RequestID, h.now())
	if err == nil {
		reply(w, http.StatusOK, purchase)
		return
	}
	for _, refusal := range levelRefusals {
		if errors.Is(err, refusal.err) {
			reply(w, refusal.status, errorResponse{Error: refusal.code})
			return
		}
	}
	h.logger.Error("pass level purchase failed", "error", err)
	reply(w, http.StatusInternalServerError, errorResponse{Error: "internal"})
}

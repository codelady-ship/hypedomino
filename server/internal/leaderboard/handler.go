package leaderboard

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/rating"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

const Radius = 2

type Reader interface {
	Top(ctx context.Context, season string, mode rules.Mode, board string, size int) ([]Row, error)
	Around(ctx context.Context, season string, mode rules.Mode, board, account string, radius int) ([]Row, bool, error)
}

type Seasons interface {
	SeasonAt(ctx context.Context, at time.Time) (rating.Season, error)
	TopSize() int
}

type Handler struct {
	verifier token.Verifier
	boards   Reader
	seasons  Seasons
	logger   *slog.Logger
	now      func() time.Time
	mux      *http.ServeMux
}

func NewHandler(verifier token.Verifier, boards Reader, seasons Seasons, logger *slog.Logger, now func() time.Time) *Handler {
	h := &Handler{verifier: verifier, boards: boards, seasons: seasons, logger: logger, now: now, mux: http.NewServeMux()}
	h.mux.HandleFunc("GET /leaderboard/{mode}", h.view)
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

type View struct {
	Season string `json:"season"`
	Mode   string `json:"mode"`
	Board  string `json:"board"`
	Top    []Row  `json:"top"`
	Me     *Row   `json:"me"`
	Around []Row  `json:"around"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) view(w http.ResponseWriter, r *http.Request) {
	claims, err := token.Authorize(h.verifier, r)
	switch {
	case errors.Is(err, token.ErrExpired):
		reply(w, http.StatusUnauthorized, errorResponse{Error: "access_expired"})
		return
	case err != nil:
		reply(w, http.StatusUnauthorized, errorResponse{Error: "access_rejected"})
		return
	}
	if h.boards == nil {
		reply(w, http.StatusServiceUnavailable, errorResponse{Error: ErrUnavailable.Error()})
		return
	}
	mode, err := rules.ParseMode(r.PathValue("mode"))
	if err != nil {
		reply(w, http.StatusBadRequest, errorResponse{Error: "mode_invalid"})
		return
	}
	board := r.URL.Query().Get("board")
	if board == "" {
		board = BoardGlobal
	}
	season := r.URL.Query().Get("season")
	if season == "" {
		current, err := h.seasons.SeasonAt(r.Context(), h.now())
		if err != nil {
			h.refuse(w, err)
			return
		}
		season = current.ID
	}
	top, err := h.boards.Top(r.Context(), season, mode, board, h.seasons.TopSize())
	if err != nil {
		h.refuse(w, err)
		return
	}
	around, listed, err := h.boards.Around(r.Context(), season, mode, board, claims.Account, Radius)
	if err != nil {
		h.refuse(w, err)
		return
	}
	view := View{Season: season, Mode: mode.String(), Board: board, Top: top, Around: []Row{}}
	if listed {
		view.Around = around
		for i := range around {
			if around[i].Account == claims.Account {
				view.Me = &around[i]
			}
		}
	}
	reply(w, http.StatusOK, view)
}

func (h *Handler) refuse(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, rating.ErrNoSeason), errors.Is(err, rating.ErrSeasonUnknown):
		reply(w, http.StatusNotFound, errorResponse{Error: rating.ErrSeasonUnknown.Error()})
	case errors.Is(err, ErrUnknownBoard):
		reply(w, http.StatusNotFound, errorResponse{Error: ErrUnknownBoard.Error()})
	case errors.Is(err, ErrUnavailable):
		h.logger.Error("leaderboard unavailable", "error", err)
		reply(w, http.StatusServiceUnavailable, errorResponse{Error: ErrUnavailable.Error()})
	default:
		h.logger.Error("leaderboard request failed", "error", err)
		reply(w, http.StatusInternalServerError, errorResponse{Error: "internal"})
	}
}

func reply(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

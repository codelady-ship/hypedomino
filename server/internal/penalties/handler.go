package penalties

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

type Records interface {
	Record(ctx context.Context, account string) (Record, error)
	Config() (Config, int)
}

type Handler struct {
	verifier token.Verifier
	records  Records
	logger   *slog.Logger
	now      func() time.Time
}

func NewHandler(verifier token.Verifier, records Records, logger *slog.Logger, now func() time.Time) *Handler {
	return &Handler{verifier: verifier, records: records, logger: logger, now: now}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet || r.URL.Path != "/penalties" {
		http.NotFound(w, r)
		return
	}
	claims, err := token.Authorize(h.verifier, r)
	switch {
	case errors.Is(err, token.ErrExpired):
		reply(w, http.StatusUnauthorized, map[string]string{"error": "access_expired"})
		return
	case err != nil:
		reply(w, http.StatusUnauthorized, map[string]string{"error": "access_rejected"})
		return
	}
	record, err := h.records.Record(r.Context(), claims.Account)
	if err != nil {
		h.logger.Error("penalty record read failed", "account", claims.Account, "error", err)
		reply(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	config, _ := h.records.Config()
	reply(w, http.StatusOK, record.View(config, h.now()))
}

func reply(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

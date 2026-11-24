package adminapi

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/economy"
)

type economyComparison struct {
	ConfigVersion int64          `json:"config_version"`
	Domain        string         `json:"domain"`
	PublishedAt   time.Time      `json:"published_at"`
	Before        economy.Report `json:"before"`
	After         economy.Report `json:"after"`
}

func (h *Handler) economyMonitors(w http.ResponseWriter, r *http.Request, p principal) {
	report, err := h.economy.Report(r.Context(), h.now())
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, report)
}

func (h *Handler) compareEconomyMonitors(w http.ResponseWriter, r *http.Request, p principal) {
	version, err := strconv.ParseInt(r.URL.Query().Get("version"), 10, 64)
	if err != nil {
		h.refuse(w, errMalformed)
		return
	}
	comparison := economyComparison{ConfigVersion: version}
	err = h.pool.QueryRow(r.Context(), `select domain, published_at from config_snapshots where version = $1`, version).Scan(&comparison.Domain, &comparison.PublishedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		err = errNoSuchVersion
	}
	if err != nil {
		h.refuse(w, err)
		return
	}
	if comparison.Before, err = h.economy.Report(r.Context(), comparison.PublishedAt); err != nil {
		h.refuse(w, err)
		return
	}
	after := comparison.PublishedAt.Add(economy.Window)
	if now := h.now(); now.Before(after) {
		after = now
	}
	if comparison.After, err = h.economy.Report(r.Context(), after); err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, comparison)
}

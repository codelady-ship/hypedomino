package adminapi

import (
	"net/http"
	"strconv"

	"github.com/Alievs-corp/hypedomino/server/internal/kpi"
)

func (h *Handler) kpis(w http.ResponseWriter, r *http.Request, p principal) {
	weeks := kpi.DefaultWeeks
	if text := r.URL.Query().Get("weeks"); text != "" {
		parsed, err := strconv.Atoi(text)
		if err != nil || parsed < 1 || parsed > kpi.MaxWeeks {
			h.refuse(w, errMalformed)
			return
		}
		weeks = parsed
	}
	report, err := kpi.Build(r.Context(), h.pool, h.now(), weeks)
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, report)
}

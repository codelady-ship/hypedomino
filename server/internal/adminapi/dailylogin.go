package adminapi

import (
	"net/http"

	"github.com/Alievs-corp/hypedomino/server/internal/dailylogin"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

type dailyLoginRulesView struct {
	Default *dailylogin.Calendar `json:"default"`
}

type dailyLoginView struct {
	catalogView
	Rules dailyLoginRulesView `json:"rules"`
}

func dailyLoginDomain() configDomain {
	return configDomain{name: dailylogin.Domain, object: rbac.Missions, accept: func(_, after []byte) error {
		_, err := dailylogin.Parse(after)
		return err
	}}
}

func (h *Handler) dailyLoginCalendar(w http.ResponseWriter, r *http.Request, _ principal) {
	version, raw, err := latestSnapshot(r.Context(), h.pool, dailylogin.Domain)
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, dailyLoginView{catalogView: viewCatalog(version, raw), Rules: dailyLoginRulesView{Default: dailylogin.Default()}})
}

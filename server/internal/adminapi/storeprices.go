package adminapi

import (
	"net/http"

	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/storeprice"
)

type storePricesRulesView struct {
	Default   *storeprice.Config  `json:"default"`
	Products  []string            `json:"products"`
	Countries []string            `json:"countries"`
	Targets   []storeprice.Target `json:"targets"`
}

type storePricesView struct {
	catalogView
	Rules storePricesRulesView `json:"rules"`
}

func storePricesDomain() configDomain {
	return configDomain{name: storeprice.Domain, object: rbac.StoreProducts, accept: func(_, after []byte) error {
		_, err := storeprice.Parse(after)
		return err
	}}
}

func (h *Handler) storePrices(w http.ResponseWriter, r *http.Request, _ principal) {
	version, raw, err := latestSnapshot(r.Context(), h.pool, storeprice.Domain)
	if err != nil {
		h.refuse(w, err)
		return
	}
	current := storeprice.Default()
	if raw != nil {
		if current, err = storeprice.Parse(raw); err != nil {
			h.refuse(w, err)
			return
		}
	}
	rules := storePricesRulesView{Default: storeprice.Default(), Products: storeprice.Products, Countries: storeprice.StoreCountries(), Targets: current.Targets()}
	reply(w, http.StatusOK, storePricesView{catalogView: viewCatalog(version, raw), Rules: rules})
}

package profile

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/account"
)

type countryView struct {
	Country           string `json:"country"`
	ChangeAvailableAt string `json:"change_available_at,omitempty"`
}

type countryRequest struct {
	Country string `json:"country"`
}

func (h *Handler) countryView(c account.Country) countryView {
	view := countryView{Country: c.Code}
	if available := c.ChangedAt.Add(h.cooldown); !c.ChangedAt.IsZero() && h.now().Before(available) {
		view.ChangeAvailableAt = available.UTC().Format(time.RFC3339)
	}
	return view
}

func (h *Handler) country(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	if h.countries == nil {
		reply(w, http.StatusNotFound, errorResponse{Error: "not_found"})
		return
	}
	c, err := h.countries.Country(r.Context(), claims.Account)
	if err != nil {
		h.refuseCountry(w, err)
		return
	}
	reply(w, http.StatusOK, h.countryView(c))
}

func (h *Handler) setCountry(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	if h.countries == nil {
		reply(w, http.StatusNotFound, errorResponse{Error: "not_found"})
		return
	}
	var request countryRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxBodyBytes)).Decode(&request); err != nil {
		reply(w, http.StatusBadRequest, errorResponse{Error: "malformed"})
		return
	}
	c, err := h.countries.SetCountry(r.Context(), claims.Account, strings.ToUpper(strings.TrimSpace(request.Country)), h.now(), h.cooldown)
	if err != nil {
		h.refuseCountry(w, err)
		return
	}
	reply(w, http.StatusOK, h.countryView(c))
}

func (h *Handler) refuseCountry(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, account.ErrCountryInvalid):
		reply(w, http.StatusBadRequest, errorResponse{Error: account.ErrCountryInvalid.Error()})
	case errors.Is(err, account.ErrCountryCooldown):
		reply(w, http.StatusConflict, errorResponse{Error: account.ErrCountryCooldown.Error()})
	case errors.Is(err, account.ErrUnknown):
		reply(w, http.StatusNotFound, errorResponse{Error: "not_found"})
	default:
		h.fail(w, err)
	}
}

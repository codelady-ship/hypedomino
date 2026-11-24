package profile

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Alievs-corp/hypedomino/server/internal/account"
)

type languageView struct {
	Language string `json:"language"`
}

func (h *Handler) language(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	if h.languages == nil {
		reply(w, http.StatusNotFound, errorResponse{Error: "not_found"})
		return
	}
	language, err := h.languages.Language(r.Context(), claims.Account)
	if err != nil {
		h.refuseLanguage(w, err)
		return
	}
	reply(w, http.StatusOK, languageView{Language: language})
}

func (h *Handler) setLanguage(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	if h.languages == nil {
		reply(w, http.StatusNotFound, errorResponse{Error: "not_found"})
		return
	}
	var request languageView
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxBodyBytes)).Decode(&request); err != nil {
		reply(w, http.StatusBadRequest, errorResponse{Error: "malformed"})
		return
	}
	language, err := h.languages.SetLanguage(r.Context(), claims.Account, strings.ToLower(strings.TrimSpace(request.Language)))
	if err != nil {
		h.refuseLanguage(w, err)
		return
	}
	reply(w, http.StatusOK, languageView{Language: language})
}

func (h *Handler) refuseLanguage(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, account.ErrLanguageInvalid):
		reply(w, http.StatusBadRequest, errorResponse{Error: account.ErrLanguageInvalid.Error()})
	case errors.Is(err, account.ErrUnknown):
		reply(w, http.StatusNotFound, errorResponse{Error: "not_found"})
	default:
		h.fail(w, err)
	}
}

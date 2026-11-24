package wallet

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

const HistoryPage = 50

type Wallet interface {
	Balance(ctx context.Context, account string, currency Currency) (int64, error)
	History(ctx context.Context, account string, limit int) ([]Applied, error)
}

type Handler struct {
	verifier token.Verifier
	wallet   Wallet
	logger   *slog.Logger
}

func NewHandler(verifier token.Verifier, wallet Wallet, logger *slog.Logger) *Handler {
	return &Handler{verifier: verifier, wallet: wallet, logger: logger}
}

type entryView struct {
	ID           int64    `json:"id"`
	Currency     Currency `json:"currency"`
	Amount       int64    `json:"amount"`
	Reason       Reason   `json:"reason"`
	SourceType   string   `json:"source_type"`
	SourceRef    string   `json:"source_ref"`
	BalanceAfter int64    `json:"balance_after"`
	At           string   `json:"at"`
}

type walletView struct {
	Coins    int64       `json:"coins"`
	Crystals int64       `json:"crystals"`
	Entries  []entryView `json:"entries"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		reply(w, http.StatusMethodNotAllowed, errorResponse{Error: "method_not_allowed"})
		return
	}
	claims, err := token.Authorize(h.verifier, r)
	switch {
	case errors.Is(err, token.ErrExpired):
		reply(w, http.StatusUnauthorized, errorResponse{Error: "access_expired"})
		return
	case err != nil:
		reply(w, http.StatusUnauthorized, errorResponse{Error: "access_rejected"})
		return
	}
	coins, err := h.wallet.Balance(r.Context(), claims.Account, Coin)
	if err != nil {
		h.fail(w, err)
		return
	}
	crystals, err := h.wallet.Balance(r.Context(), claims.Account, Crystal)
	if err != nil {
		h.fail(w, err)
		return
	}
	history, err := h.wallet.History(r.Context(), claims.Account, HistoryPage)
	if err != nil {
		h.fail(w, err)
		return
	}
	view := walletView{Coins: coins, Crystals: crystals, Entries: make([]entryView, 0, len(history))}
	for _, e := range history {
		view.Entries = append(view.Entries, entryView{ID: e.ID, Currency: e.Currency, Amount: e.Amount, Reason: e.Reason, SourceType: e.Source.Type, SourceRef: e.Source.Ref, BalanceAfter: e.BalanceAfter, At: e.At.UTC().Format(time.RFC3339)})
	}
	reply(w, http.StatusOK, view)
}

func (h *Handler) fail(w http.ResponseWriter, err error) {
	h.logger.Error("wallet request failed", "error", err)
	reply(w, http.StatusInternalServerError, errorResponse{Error: "internal"})
}

func reply(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

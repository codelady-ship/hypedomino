package adminapi

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/playerrecord"
	"github.com/Alievs-corp/hypedomino/server/internal/rating"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

const RemoveFromLeaderboardAction = "remove_from_leaderboard"

type leaderboardRemovalRequest struct {
	Season string `json:"season"`
	Mode   string `json:"mode"`
	Reason string `json:"reason"`
	Ref    string `json:"ref"`
}

func (h *Handler) OnLeaderboardRemoval(drop func(ctx context.Context, removal rating.BoardRemoval) error) {
	h.dropFromBoards = drop
}

func (h *Handler) removeFromLeaderboard(w http.ResponseWriter, r *http.Request, p principal) {
	var request leaderboardRemovalRequest
	if !decode(w, r, &request) {
		return
	}
	mode, err := rules.ParseMode(request.Mode)
	if err != nil {
		reply(w, http.StatusBadRequest, errorResponse{Error: "invalid", Detail: "mode must be 1v1 or 2v2"})
		return
	}
	accountID := r.PathValue("id")
	note := changeNote{Reason: request.Reason, Ref: request.Ref}
	var removal rating.BoardRemoval
	err = h.audited(r.Context(), p, func(tx pgx.Tx, now time.Time) ([]audit.Entry, error) {
		if err := h.sensitive(p, note, now); err != nil {
			return nil, err
		}
		if err := playerrecord.Lock(r.Context(), tx, accountID); err != nil {
			return nil, err
		}
		var err error
		if removal, err = rating.RemoveFromBoardIn(r.Context(), tx, request.Season, mode, accountID, now); err != nil {
			return nil, err
		}
		return []audit.Entry{{Action: RemoveFromLeaderboardAction, Object: audit.Object{Type: rbac.Players, ID: accountID}, After: marshal(removal), Reason: note.Reason, Ref: note.Ref}}, nil
	})
	if err != nil {
		h.refuse(w, err)
		return
	}
	if h.dropFromBoards != nil {
		if err := h.dropFromBoards(r.Context(), removal); err != nil {
			h.logger.Error("removed account still on a board until it rebuilds", "account", accountID, "season", removal.Season, "mode", removal.Mode, "error", err)
		}
	}
	reply(w, http.StatusOK, removal)
}

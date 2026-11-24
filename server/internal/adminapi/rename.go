package adminapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/playerrecord"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/sanction"
)

const ForceRenameAction = "force_rename"

var errNoProfile = errors.New("adminapi: the account has no nickname yet")

type renameState struct {
	Nickname       string `json:"nickname"`
	RenameRequired bool   `json:"rename_required"`
	SanctionID     int64  `json:"sanction_id,omitempty"`
}

type renameView struct {
	AccountID string `json:"account_id"`
	renameState
}

func (h *Handler) forceRename(w http.ResponseWriter, r *http.Request, p principal) {
	var note changeNote
	if !decode(w, r, &note) {
		return
	}
	accountID := r.PathValue("id")
	var state renameState
	err := h.audited(r.Context(), p, func(tx pgx.Tx, now time.Time) ([]audit.Entry, error) {
		if err := h.sensitive(p, note, now); err != nil {
			return nil, err
		}
		if err := playerrecord.Lock(r.Context(), tx, accountID); err != nil {
			return nil, err
		}
		err := tx.QueryRow(r.Context(), `select nickname, rename_required from profiles where account_id = $1 for update`, accountID).Scan(&state.Nickname, &state.RenameRequired)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errNoProfile
		}
		if err != nil || state.RenameRequired {
			return nil, err
		}
		if _, err := tx.Exec(r.Context(), `update profiles set rename_required = true, updated_at = $2 where account_id = $1`, accountID, now); err != nil {
			return nil, err
		}
		imposed, err := sanction.Impose(r.Context(), tx, accountID, sanction.ForcedRename, sanction.OffensiveName, now, nil)
		if err != nil {
			return nil, err
		}
		before := state
		state.RenameRequired, state.SanctionID = true, imposed.ID
		return []audit.Entry{{
			Action: ForceRenameAction, Object: audit.Object{Type: rbac.Players, ID: accountID},
			Before: marshal(before), After: marshal(state), Reason: note.Reason, Ref: note.Ref,
		}}, nil
	})
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, renameView{AccountID: accountID, renameState: state})
}

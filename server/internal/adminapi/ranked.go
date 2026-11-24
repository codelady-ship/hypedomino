package adminapi

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/rating"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

const (
	CreateSeasonAction     = "create_season"
	RescheduleSeasonAction = "reschedule_season"
	ExtendSeasonAction     = "extend_season"
	DeleteSeasonAction     = "delete_season"
)

var errSeasonNotLive = errors.New("adminapi: only a live season can be extended")

type seasonView struct {
	rating.Season
	State string `json:"state"`
}

type seasonsView struct {
	ServerTime time.Time    `json:"server_time"`
	NextStart  *time.Time   `json:"next_start"`
	Seasons    []seasonView `json:"seasons"`
}

type scheduleRequest struct {
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	NoRewards bool      `json:"no_rewards"`
	changeNote
}

type createSeasonRequest struct {
	ID string `json:"id"`
	scheduleRequest
}

type extendSeasonRequest struct {
	End time.Time `json:"end"`
	changeNote
}

func (h *Handler) rankedSeasons(w http.ResponseWriter, r *http.Request, p principal) {
	seasons, err := rating.ListSeasons(r.Context(), h.pool)
	if err != nil {
		h.refuse(w, err)
		return
	}
	now := h.now().UTC()
	view := seasonsView{ServerTime: now, Seasons: make([]seasonView, 0, len(seasons))}
	for _, season := range seasons {
		view.Seasons = append(view.Seasons, seasonView{Season: season, State: season.State(now)})
		end := season.End
		view.NextStart = &end
	}
	reply(w, http.StatusOK, view)
}

func seasonEntry(action string, note changeNote, before, after *rating.Season) audit.Entry {
	entry := audit.Entry{Action: action, Reason: note.Reason, Ref: note.Ref}
	if before != nil {
		entry.Before, entry.Object = marshal(before), audit.Object{Type: rbac.RankedSeasons, ID: before.ID}
	}
	if after != nil {
		entry.After, entry.Object = marshal(after), audit.Object{Type: rbac.RankedSeasons, ID: after.ID}
	}
	return entry
}

func (h *Handler) changeSeason(w http.ResponseWriter, r *http.Request, p principal, note changeNote, status int, change func(tx pgx.Tx, now time.Time) (audit.Entry, *rating.Season, error)) {
	if err := h.noted(note); err != nil {
		h.refuse(w, err)
		return
	}
	var changed *rating.Season
	var now time.Time
	err := h.audited(r.Context(), p, func(tx pgx.Tx, at time.Time) ([]audit.Entry, error) {
		entry, season, err := change(tx, at)
		changed, now = season, at
		return []audit.Entry{entry}, err
	})
	if err != nil {
		h.refuse(w, err)
		return
	}
	if changed == nil {
		w.WriteHeader(status)
		return
	}
	reply(w, status, seasonView{Season: *changed, State: changed.State(now)})
}

func (h *Handler) createSeason(w http.ResponseWriter, r *http.Request, p principal) {
	var request createSeasonRequest
	if !decode(w, r, &request) {
		return
	}
	h.changeSeason(w, r, p, request.changeNote, http.StatusCreated, func(tx pgx.Tx, now time.Time) (audit.Entry, *rating.Season, error) {
		if !now.Before(request.Start) {
			return audit.Entry{}, nil, fmt.Errorf("%w: the start must be in the future", rating.ErrSeason)
		}
		season := rating.Season{ID: request.ID, NameKey: request.ID + ".name", Start: request.Start, End: request.End, NoRewards: request.NoRewards}
		if err := h.keysApprovedInEnglish(r.Context(), tx, rating.ErrSeason, []string{season.NameKey}); err != nil {
			return audit.Entry{}, nil, err
		}
		created, err := rating.CreateSeasonIn(r.Context(), tx, season, now)
		return seasonEntry(CreateSeasonAction, request.changeNote, nil, &created), &created, err
	})
}

func (h *Handler) rescheduleSeason(w http.ResponseWriter, r *http.Request, p principal) {
	var request scheduleRequest
	if !decode(w, r, &request) {
		return
	}
	h.changeSeason(w, r, p, request.changeNote, http.StatusOK, func(tx pgx.Tx, now time.Time) (audit.Entry, *rating.Season, error) {
		schedule := rating.Schedule{Start: request.Start, End: request.End, NoRewards: request.NoRewards}
		before, after, err := rating.RescheduleSeasonIn(r.Context(), tx, r.PathValue("id"), schedule, now)
		return seasonEntry(RescheduleSeasonAction, request.changeNote, &before, &after), &after, err
	})
}

func (h *Handler) extendSeason(w http.ResponseWriter, r *http.Request, p principal) {
	var request extendSeasonRequest
	if !decode(w, r, &request) {
		return
	}
	h.changeSeason(w, r, p, request.changeNote, http.StatusOK, func(tx pgx.Tx, now time.Time) (audit.Entry, *rating.Season, error) {
		season, err := rating.LockSeason(r.Context(), tx, r.PathValue("id"))
		if err != nil {
			return audit.Entry{}, nil, err
		}
		if season.State(now) != rating.StateLive {
			return audit.Entry{}, nil, fmt.Errorf("%w: %s is %s", errSeasonNotLive, season.ID, season.State(now))
		}
		before, after, err := rating.ExtendSeasonIn(r.Context(), tx, season.ID, request.End)
		return seasonEntry(ExtendSeasonAction, request.changeNote, &before, &after), &after, err
	})
}

func (h *Handler) deleteSeason(w http.ResponseWriter, r *http.Request, p principal) {
	var note changeNote
	if !decode(w, r, &note) {
		return
	}
	h.changeSeason(w, r, p, note, http.StatusNoContent, func(tx pgx.Tx, now time.Time) (audit.Entry, *rating.Season, error) {
		deleted, err := rating.DeleteSeasonIn(r.Context(), tx, r.PathValue("id"), now)
		return seasonEntry(DeleteSeasonAction, note, &deleted, nil), nil, err
	})
}

package adminapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/contentpublish"
	"github.com/Alievs-corp/hypedomino/server/internal/gamestrings"
	"github.com/Alievs-corp/hypedomino/server/internal/objectstore"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

const MaxImportBytes = 4 << 20

var (
	errContentStoreOff = errors.New("adminapi: no content store is configured")
	errContentUpload   = errors.New("adminapi: the content store did not take the bundle")
)

type ContentStore interface {
	PutObject(ctx context.Context, key string, object objectstore.Object, now time.Time) error
}

type valueView struct {
	Text   string             `json:"text"`
	Status gamestrings.Status `json:"status"`
}

type stringView struct {
	Key          string                             `json:"key"`
	Note         string                             `json:"note"`
	MaxLength    int                                `json:"max_length"`
	Placeholders []gamestrings.Placeholder          `json:"placeholders"`
	Deprecated   bool                               `json:"deprecated"`
	Values       map[gamestrings.Language]valueView `json:"values"`
}

type coverageView struct {
	Language gamestrings.Language `json:"language"`
	Keys     int                  `json:"keys"`
	Approved int                  `json:"approved"`
	Percent  float64              `json:"percent"`
}

type stringsResponse struct {
	Strings []stringView `json:"strings"`
}

type coverageResponse struct {
	Coverage []coverageView `json:"coverage"`
}

type saveStringRequest struct {
	Note         string                    `json:"note"`
	MaxLength    int                       `json:"max_length"`
	Placeholders []gamestrings.Placeholder `json:"placeholders"`
	Deprecated   bool                      `json:"deprecated"`
	changeNote
}

type setValueRequest struct {
	Text   string             `json:"text"`
	Status gamestrings.Status `json:"status"`
	changeNote
}

type bundleView struct {
	Language gamestrings.Language `json:"language"`
	Version  int                  `json:"version"`
	Hash     string               `json:"hash"`
	By       string               `json:"by"`
	At       time.Time            `json:"at"`
}

type bundlesResponse struct {
	Bundles []bundleView `json:"bundles"`
}

type problemView struct {
	Key   string `json:"key"`
	Error string `json:"error"`
}

type blockedResponse struct {
	Error    string        `json:"error"`
	Problems []problemView `json:"problems"`
}

type keysRequest struct {
	Keys []string `json:"keys"`
}

type keysResponse struct {
	Keys []string `json:"keys"`
}

type importResponse struct {
	Changed int `json:"changed"`
}

func viewString(e gamestrings.Entry) stringView {
	values := make(map[gamestrings.Language]valueView, len(e.Values))
	for language, value := range e.Values {
		values[language] = valueView{Text: value.Text, Status: value.Status}
	}
	placeholders := e.Placeholders
	if placeholders == nil {
		placeholders = []gamestrings.Placeholder{}
	}
	return stringView{Key: e.Key, Note: e.Note, MaxLength: e.MaxLength, Placeholders: placeholders, Deprecated: e.Deprecated, Values: values}
}

func viewBundle(b gamestrings.Published) bundleView {
	return bundleView{Language: b.Language, Version: b.Version, Hash: b.Hash, By: b.By, At: b.At}
}

func marshal(value any) json.RawMessage {
	raw, _ := json.Marshal(value)
	return raw
}

type auditedString struct {
	Note         string                    `json:"note"`
	MaxLength    int                       `json:"max_length"`
	Placeholders []gamestrings.Placeholder `json:"placeholders"`
	Deprecated   bool                      `json:"deprecated"`
}

func auditedEntry(e gamestrings.Entry) json.RawMessage {
	return marshal(auditedString{Note: e.Note, MaxLength: e.MaxLength, Placeholders: e.Placeholders, Deprecated: e.Deprecated})
}

type auditedValue struct {
	Language gamestrings.Language `json:"language"`
	Text     string               `json:"text"`
	Status   gamestrings.Status   `json:"status"`
}

func valueEntry(key string, language gamestrings.Language, before, after gamestrings.Value, note changeNote) audit.Entry {
	action := "edit"
	if after.Status == gamestrings.Approved {
		action = "approve"
	}
	return audit.Entry{
		Action: action, Object: audit.Object{Type: rbac.GameStrings, ID: key}, Reason: note.Reason, Ref: note.Ref,
		Before: marshal(auditedValue{Language: language, Text: before.Text, Status: before.Status}),
		After:  marshal(auditedValue{Language: language, Text: after.Text, Status: after.Status}),
	}
}

func (h *Handler) allows(p principal, object rbac.Object, actions ...rbac.Action) error {
	for _, action := range actions {
		if !rbac.Allows(p.account.Roles, object, action) {
			h.logger.Info("admin request forbidden", "account", p.account.ID, "object", object, "action", action)
			return errForbidden
		}
	}
	return nil
}

func (h *Handler) refuseStrings(w http.ResponseWriter, err error, problems []gamestrings.Problem) {
	if problems == nil {
		h.refuse(w, err)
		return
	}
	views := make([]problemView, 0, len(problems))
	for _, problem := range problems {
		views = append(views, problemView{Key: problem.Key, Error: problem.Err.Error()})
	}
	reply(w, http.StatusUnprocessableEntity, blockedResponse{Error: "blocked", Problems: views})
}

func (h *Handler) listStrings(w http.ResponseWriter, r *http.Request, p principal) {
	query, err := stringsQuery(r)
	if err != nil {
		h.refuse(w, err)
		return
	}
	entries, err := h.strings.Page(r.Context(), query)
	if err != nil {
		h.refuse(w, err)
		return
	}
	views := make([]stringView, 0, len(entries))
	for _, entry := range entries {
		views = append(views, viewString(entry))
	}
	reply(w, http.StatusOK, stringsResponse{Strings: views})
}

func stringsQuery(r *http.Request) (gamestrings.EntryQuery, error) {
	query := r.URL.Query()
	limit, err := queuePageLimit(query.Get("limit"))
	missing := gamestrings.Language(query.Get("missing"))
	if err != nil || (missing != "" && !slices.Contains(gamestrings.Languages, missing)) {
		return gamestrings.EntryQuery{}, errMalformed
	}
	return gamestrings.EntryQuery{Text: strings.TrimSpace(query.Get("q")), Missing: missing, After: query.Get("after"), Limit: limit}, nil
}

func (h *Handler) stringsCoverage(w http.ResponseWriter, r *http.Request, p principal) {
	report, err := h.strings.Coverage(r.Context())
	if err != nil {
		h.refuse(w, err)
		return
	}
	coverage := make([]coverageView, 0, len(report))
	for _, c := range report {
		coverage = append(coverage, coverageView{Language: c.Language, Keys: c.Keys, Approved: c.Approved, Percent: c.Percent})
	}
	reply(w, http.StatusOK, coverageResponse{Coverage: coverage})
}

func (h *Handler) unapprovedInEnglish(w http.ResponseWriter, r *http.Request, p principal) {
	var request keysRequest
	if !decode(w, r, &request) {
		return
	}
	unapproved, err := h.strings.Unapproved(r.Context(), request.Keys, gamestrings.English)
	if err != nil {
		h.refuse(w, err)
		return
	}
	if unapproved == nil {
		unapproved = []string{}
	}
	reply(w, http.StatusOK, keysResponse{Keys: unapproved})
}

func (h *Handler) saveString(w http.ResponseWriter, r *http.Request, p principal) {
	var request saveStringRequest
	if !decode(w, r, &request) {
		return
	}
	key := r.PathValue("key")
	saved := gamestrings.Entry{Key: key, Note: request.Note, MaxLength: request.MaxLength, Placeholders: request.Placeholders, Deprecated: request.Deprecated}
	err := h.audited(r.Context(), p, func(tx pgx.Tx, now time.Time) ([]audit.Entry, error) {
		store := h.strings.In(tx)
		before, err := store.Entry(r.Context(), key)
		exists := err == nil
		if err != nil && !errors.Is(err, gamestrings.ErrUnknownKey) {
			return nil, err
		}
		action, needed := stringAction(exists, before, saved)
		if err := h.allows(p, rbac.GameStrings, needed); err != nil {
			return nil, err
		}
		if err := store.SaveEntry(r.Context(), saved, p.account.ID, now); err != nil {
			return nil, err
		}
		entry := audit.Entry{Action: action, Object: audit.Object{Type: rbac.GameStrings, ID: key}, After: auditedEntry(saved), Reason: request.Reason, Ref: request.Ref}
		if exists {
			entry.Before = auditedEntry(before)
		}
		return []audit.Entry{entry}, nil
	})
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, viewString(saved))
}

func stringAction(exists bool, before, after gamestrings.Entry) (string, rbac.Action) {
	switch {
	case !exists:
		return "create", rbac.Create
	case after.Deprecated && !before.Deprecated:
		return "retire", rbac.Retire
	}
	return "edit", rbac.Edit
}

func (h *Handler) setValue(w http.ResponseWriter, r *http.Request, p principal) {
	var request setValueRequest
	if !decode(w, r, &request) {
		return
	}
	needed := []rbac.Action{rbac.Edit}
	if request.Status == gamestrings.Approved {
		needed = append(needed, rbac.Publish)
	}
	if err := h.allows(p, rbac.GameStrings, needed...); err != nil {
		h.refuse(w, err)
		return
	}
	key, language := r.PathValue("key"), gamestrings.Language(r.PathValue("language"))
	after := gamestrings.Value{Text: request.Text, Status: request.Status}
	err := h.audited(r.Context(), p, func(tx pgx.Tx, now time.Time) ([]audit.Entry, error) {
		store := h.strings.In(tx)
		before, err := store.Value(r.Context(), key, language)
		if err != nil {
			return nil, err
		}
		if err := store.SetValue(r.Context(), key, language, after, p.account.ID, now); err != nil {
			return nil, err
		}
		return []audit.Entry{valueEntry(key, language, before, after, request.changeNote)}, nil
	})
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, valueView{Text: after.Text, Status: after.Status})
}

func transferQuery(r *http.Request) (gamestrings.Language, gamestrings.Format) {
	query := r.URL.Query()
	return gamestrings.Language(query.Get("language")), gamestrings.Format(query.Get("format"))
}

func (h *Handler) exportStrings(w http.ResponseWriter, r *http.Request, p principal) {
	language, format := transferQuery(r)
	data, err := h.strings.Export(r.Context(), language, format)
	if err != nil {
		h.refuse(w, err)
		return
	}
	contentType := "application/json"
	if format == gamestrings.CSV {
		contentType = "text/csv; charset=utf-8"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", `attachment; filename="strings-`+string(language)+"."+string(format)+`"`)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (h *Handler) importStrings(w http.ResponseWriter, r *http.Request, p principal) {
	language, format := transferQuery(r)
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxImportBytes))
	if err != nil {
		reply(w, http.StatusRequestEntityTooLarge, errorResponse{Error: "too_large"})
		return
	}
	var changed int
	var problems []gamestrings.Problem
	err = h.audited(r.Context(), p, func(tx pgx.Tx, now time.Time) ([]audit.Entry, error) {
		store := h.strings.In(tx)
		entries, err := store.Entries(r.Context())
		if err != nil {
			return nil, err
		}
		changes, blocked, err := store.Import(r.Context(), language, format, data, p.account.ID, now)
		if err != nil {
			problems = blocked
			return nil, err
		}
		changed = len(changes)
		return importEntries(entries, language, changes), nil
	})
	if err != nil {
		h.refuseStrings(w, err, problems)
		return
	}
	reply(w, http.StatusOK, importResponse{Changed: changed})
}

func importEntries(entries []gamestrings.Entry, language gamestrings.Language, changes []gamestrings.Change) []audit.Entry {
	before := make(map[string]gamestrings.Value, len(entries))
	for _, entry := range entries {
		value, has := entry.Values[language]
		if !has {
			value.Status = gamestrings.Missing
		}
		before[entry.Key] = value
	}
	audited := make([]audit.Entry, 0, len(changes))
	for _, change := range changes {
		after := gamestrings.Value{Text: change.Text, Status: gamestrings.Draft}
		audited = append(audited, valueEntry(change.Key, language, before[change.Key], after, changeNote{Reason: "import"}))
	}
	return audited
}

func (h *Handler) listBundles(w http.ResponseWriter, r *http.Request, p principal) {
	query := r.URL.Query()
	language := gamestrings.Language(query.Get("language"))
	limit, limitErr := queuePageLimit(query.Get("limit"))
	before, beforeErr := optionalNumber(query.Get("before"))
	if limitErr != nil || beforeErr != nil || !slices.Contains(gamestrings.Languages, language) {
		h.refuse(w, errMalformed)
		return
	}
	bundles, err := h.strings.Bundles(r.Context(), language, int(before), limit)
	if err != nil {
		h.refuse(w, err)
		return
	}
	views := make([]bundleView, 0, len(bundles))
	for _, bundle := range bundles {
		views = append(views, viewBundle(bundle))
	}
	reply(w, http.StatusOK, bundlesResponse{Bundles: views})
}

func (h *Handler) publishBundle(w http.ResponseWriter, r *http.Request, p principal) {
	var request changeNote
	if !decode(w, r, &request) {
		return
	}
	if h.config.Content == nil {
		h.refuse(w, errContentStoreOff)
		return
	}
	language := gamestrings.Language(r.PathValue("language"))
	var published gamestrings.Published
	var problems []gamestrings.Problem
	err := h.audited(r.Context(), p, func(tx pgx.Tx, now time.Time) ([]audit.Entry, error) {
		var err error
		published, problems, err = h.strings.In(tx).Publish(r.Context(), language, p.account.ID, now)
		if err != nil {
			return nil, err
		}
		if err := h.uploadBundle(r.Context(), published, now); err != nil {
			return nil, err
		}
		return []audit.Entry{{
			Action: "publish", Object: audit.Object{Type: rbac.GameStrings, ID: "bundle:" + string(language)},
			After: marshal(bundleView{Language: published.Language, Version: published.Version, Hash: published.Hash}), Reason: request.Reason, Ref: request.Ref,
		}}, nil
	})
	if err != nil {
		h.refuseStrings(w, err, problems)
		return
	}
	reply(w, http.StatusOK, viewBundle(published))
}

func (h *Handler) uploadBundle(ctx context.Context, bundle gamestrings.Published, now time.Time) error {
	object := objectstore.Object{ContentType: "application/json", CacheControl: contentpublish.ImmutableCacheControl, Body: bundle.Content}
	if err := h.config.Content.PutObject(ctx, bundle.File(), object, now); err != nil {
		return fmt.Errorf("%w: %s: %v", errContentUpload, bundle.File(), err)
	}
	return nil
}

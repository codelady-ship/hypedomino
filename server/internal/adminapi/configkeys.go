package adminapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
)

type configKey struct {
	Name    string
	Unit    string
	Min     int
	Max     *int
	Default int
}

type keyedDomain struct {
	configDomain
	keys  []configKey
	parse func(raw []byte) (map[string]int, error)
}

func newKeyedDomain(name string, keys []configKey, parse func(raw []byte) (map[string]int, error)) keyedDomain {
	return keyedDomain{
		configDomain: configDomain{name: name, object: rbac.GameConfig, accept: func(_, after []byte) error {
			_, err := parse(after)
			return err
		}},
		keys:  keys,
		parse: parse,
	}
}

type configKeyView struct {
	Key     string `json:"key"`
	Type    string `json:"type"`
	Unit    string `json:"unit"`
	Min     int    `json:"min"`
	Max     *int   `json:"max,omitempty"`
	Default int    `json:"default"`
	Value   int    `json:"value"`
	Set     bool   `json:"set"`
}

type configKeysView struct {
	ConfigVersion int64           `json:"config_version"`
	Keys          []configKeyView `json:"keys"`
}

type configValuesVersionView struct {
	ConfigVersion int64          `json:"config_version"`
	PublishedAt   time.Time      `json:"published_at"`
	Values        map[string]int `json:"values"`
}

type configValuesHistoryView struct {
	Versions []configValuesVersionView `json:"versions"`
}

type publishValuesRequest struct {
	Values map[string]json.RawMessage `json:"values"`
	changeNote
}

type rollbackRequest struct {
	Version int64 `json:"version"`
	changeNote
}

func (d keyedDomain) view(version int64, raw []byte) (configKeysView, error) {
	values := map[string]int{}
	if raw != nil {
		var err error
		if values, err = d.parse(raw); err != nil {
			return configKeysView{}, err
		}
	}
	keys := make([]configKeyView, 0, len(d.keys))
	for _, key := range d.keys {
		value, set := values[key.Name]
		if !set {
			value = key.Default
		}
		keys = append(keys, configKeyView{Key: key.Name, Type: "integer", Unit: key.Unit, Min: key.Min, Max: key.Max, Default: key.Default, Value: value, Set: set})
	}
	return configKeysView{ConfigVersion: version, Keys: keys}, nil
}

func (h *Handler) replyConfigKeys(w http.ResponseWriter, d keyedDomain, version int64, raw []byte) {
	view, err := d.view(version, raw)
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, view)
}

func (h *Handler) configKeys(d keyedDomain) func(http.ResponseWriter, *http.Request, principal) {
	return func(w http.ResponseWriter, r *http.Request, p principal) {
		version, raw, err := latestSnapshot(r.Context(), h.pool, d.name)
		if err != nil {
			h.refuse(w, err)
			return
		}
		h.replyConfigKeys(w, d, version, raw)
	}
}

func (h *Handler) configValuesHistory(d keyedDomain) func(http.ResponseWriter, *http.Request, principal) {
	return func(w http.ResponseWriter, r *http.Request, p principal) {
		history, err := snapshot.History(r.Context(), h.pool, d.name, ConfigHistoryLimit)
		if err != nil {
			h.refuse(w, err)
			return
		}
		versions := make([]configValuesVersionView, 0, len(history))
		for _, published := range history {
			var values map[string]int
			if err := json.Unmarshal(published.Content, &values); err != nil {
				h.refuse(w, err)
				return
			}
			versions = append(versions, configValuesVersionView{ConfigVersion: published.Version, PublishedAt: published.At, Values: values})
		}
		reply(w, http.StatusOK, configValuesHistoryView{Versions: versions})
	}
}

func (h *Handler) publishConfigValues(d keyedDomain) func(http.ResponseWriter, *http.Request, principal) {
	return func(w http.ResponseWriter, r *http.Request, p principal) {
		var request publishValuesRequest
		if !decode(w, r, &request) {
			return
		}
		if request.Values == nil {
			request.Values = map[string]json.RawMessage{}
		}
		raw := marshal(request.Values)
		version, _, err := h.replaceSnapshot(r.Context(), p, d.configDomain, "publish", raw, request.changeNote, nil)
		if err != nil {
			h.refuse(w, err)
			return
		}
		h.replyConfigKeys(w, d, version, raw)
	}
}

func (h *Handler) rollbackConfigValues(d keyedDomain) func(http.ResponseWriter, *http.Request, principal) {
	return func(w http.ResponseWriter, r *http.Request, p principal) {
		var request rollbackRequest
		if !decode(w, r, &request) {
			return
		}
		raw, err := h.earlierSnapshot(r.Context(), d.name, request.Version)
		if err != nil {
			h.refuse(w, err)
			return
		}
		version, _, err := h.replaceSnapshot(r.Context(), p, d.configDomain, "rollback", raw, request.changeNote, h.mayRollBack)
		if err != nil {
			h.refuse(w, err)
			return
		}
		h.replyConfigKeys(w, d, version, raw)
	}
}

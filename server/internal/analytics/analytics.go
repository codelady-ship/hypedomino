package analytics

import (
	"errors"
	"regexp"

	"github.com/google/uuid"
)

const MaxBodyBytes = 64 * 1024

var (
	ErrMalformed = errors.New("analytics.malformed")
	ErrIDTaken   = errors.New("analytics.id_taken")
	ErrNoConsent = errors.New("analytics.consent_required")
)

type labelRule int

const (
	noLabel labelRule = iota + 1
	providerLabel
	idLabel
)

var clientEvents = map[string]labelRule{
	"app_open":        noLabel,
	"sign_in":         providerLabel,
	"nickname_set":    noLabel,
	"screen_view":     idLabel,
	"queue_join":      idLabel,
	"purchase_start":  idLabel,
	"purchase_cancel": idLabel,
}

var idPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$`)

type Event struct {
	Name         string `json:"name"`
	Label        string `json:"label"`
	Value        int64  `json:"value"`
	ClientTimeMs int64  `json:"clientTimeMs"`
}

type Batch struct {
	AnalyticsID string  `json:"analyticsId"`
	Events      []Event `json:"events"`
}

func (e Event) Valid() bool {
	if e.Value != 0 || e.ClientTimeMs <= 0 {
		return false
	}
	switch clientEvents[e.Name] {
	case noLabel:
		return e.Label == ""
	case providerLabel:
		return e.Label == "google" || e.Label == "apple"
	case idLabel:
		return idPattern.MatchString(e.Label)
	}
	return false
}

func (b Batch) Accepted() (Batch, error) {
	id, err := uuid.Parse(b.AnalyticsID)
	if err != nil || id == uuid.Nil || len(b.AnalyticsID) != 36 {
		return Batch{}, ErrMalformed
	}
	accepted := Batch{AnalyticsID: id.String()}
	for _, e := range b.Events {
		if e.Valid() {
			accepted.Events = append(accepted.Events, e)
		}
	}
	return accepted, nil
}

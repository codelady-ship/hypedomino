package consent

import "errors"

type Purpose string

const (
	Analytics   Purpose = "analytics"
	Diagnostics Purpose = "diagnostics"
)

var (
	ErrMalformed = errors.New("consent.malformed")
	ErrContext   = errors.New("consent.context_required")
)

type Choices struct {
	Analytics   bool `json:"analytics"`
	Diagnostics bool `json:"diagnostics"`
}

type Change struct {
	Analytics     *bool  `json:"analytics"`
	Diagnostics   *bool  `json:"diagnostics"`
	PolicyVersion string `json:"policyVersion"`
	AppVersion    string `json:"appVersion"`
}

type Choice struct {
	Purpose Purpose `json:"purpose"`
	Granted bool    `json:"granted"`
}

type FirstLaunch struct {
	Choices       []Choice `json:"choices"`
	PolicyVersion string   `json:"policyVersion"`
	AppVersion    string   `json:"appVersion"`
}

func (f FirstLaunch) Known() bool {
	for _, choice := range f.Choices {
		if choice.Purpose != Analytics && choice.Purpose != Diagnostics {
			return false
		}
	}
	return true
}

func (c Choices) With(change Change) Choices {
	if change.Analytics != nil {
		c.Analytics = *change.Analytics
	}
	if change.Diagnostics != nil {
		c.Diagnostics = *change.Diagnostics
	}
	return c
}

func (c Choices) Of(purpose Purpose) bool {
	switch purpose {
	case Analytics:
		return c.Analytics
	case Diagnostics:
		return c.Diagnostics
	}
	return false
}

func (c *Choices) set(purpose Purpose, granted bool) {
	switch purpose {
	case Analytics:
		c.Analytics = granted
	case Diagnostics:
		c.Diagnostics = granted
	}
}

func (c Choices) Changed(next Choices) []Purpose {
	var changed []Purpose
	for _, purpose := range []Purpose{Analytics, Diagnostics} {
		if c.Of(purpose) != next.Of(purpose) {
			changed = append(changed, purpose)
		}
	}
	return changed
}

package rules

import (
	"fmt"
	"time"
)

type Controller uint8

const (
	ControllerPlayer Controller = iota
	ControllerBot
)

func (c Controller) String() string {
	if c == ControllerBot {
		return "bot"
	}
	return "player"
}

func (c Controller) MarshalText() ([]byte, error) {
	return []byte(c.String()), nil
}

func (c *Controller) UnmarshalText(text []byte) error {
	switch string(text) {
	case "player":
		*c = ControllerPlayer
	case "bot":
		*c = ControllerBot
	default:
		return fmt.Errorf("rules: invalid controller %q", text)
	}
	return nil
}

func TurnLimit(config Config) time.Duration {
	return time.Duration(config.TurnSeconds) * time.Second
}

func LoadingLimit(config Config) time.Duration {
	return time.Duration(config.MatchLoadingSeconds) * time.Second
}

func MovedBy(config Config, acted bool, actedAfter time.Duration) Controller {
	if acted && actedAfter < TurnLimit(config) {
		return ControllerPlayer
	}
	return ControllerBot
}

type seatPresence struct {
	consecutiveTimeouts int
	byBot               bool
	timeouts            int
	markedAfk           bool
	left                bool
}

type PresenceReport struct {
	Timeouts  int
	MarkedAfk bool
	Left      bool
	ByBot     bool
}

type Presence struct {
	config Config
	seats  [seatCount]seatPresence
}

func NewPresence(config Config) Presence {
	return Presence{config: config}
}

func (p *Presence) Controller(seat Seat) Controller {
	if p.seats[seat].byBot {
		return ControllerBot
	}
	return ControllerPlayer
}

func (p *Presence) ConsecutiveTimeouts(seat Seat) int {
	return p.seats[seat].consecutiveTimeouts
}

func (p *Presence) Timeout(seat Seat) Controller {
	s := &p.seats[seat]
	s.consecutiveTimeouts++
	s.timeouts++
	if s.consecutiveTimeouts >= p.config.AfkConsecutiveTimeouts {
		s.byBot = true
		s.markedAfk = true
	}
	return p.Controller(seat)
}

func (p *Presence) Move(seat Seat) {
	p.seats[seat].consecutiveTimeouts = 0
}

func (p *Presence) Reconnect(seat Seat) Controller {
	s := &p.seats[seat]
	s.consecutiveTimeouts = 0
	s.byBot = false
	return p.Controller(seat)
}

func (p *Presence) Leave(seat Seat) {
	s := &p.seats[seat]
	s.byBot = true
	s.left = true
}

func (p *Presence) Absent(seat Seat) {
	p.seats[seat].byBot = true
}

func (p *Presence) Report(seat Seat) PresenceReport {
	s := p.seats[seat]
	return PresenceReport{Timeouts: s.timeouts, MarkedAfk: s.markedAfk, Left: s.left, ByBot: s.byBot}
}

package rules

import (
	"testing"
	"time"
)

func TestMovedByTreatsTheLimitItselfAsLate(t *testing.T) {
	config := DefaultConfig()
	cases := []struct {
		acted bool
		after time.Duration
		want  Controller
	}{
		{true, 0, ControllerPlayer},
		{true, 19999 * time.Millisecond, ControllerPlayer},
		{true, 20 * time.Second, ControllerBot},
		{true, time.Minute, ControllerBot},
		{false, 0, ControllerBot},
	}
	for _, c := range cases {
		if got := MovedBy(config, c.acted, c.after); got != c.want {
			t.Errorf("acted %v after %s: %s, want %s", c.acted, c.after, got, c.want)
		}
	}
	config.TurnSeconds = 15
	if TurnLimit(config) != 15*time.Second || MovedBy(config, true, 16*time.Second) != ControllerBot {
		t.Error("admin turn limit is not used")
	}
}

func TestPresenceAfkLeaveAndReconnect(t *testing.T) {
	config := DefaultConfig()
	p := NewPresence(config)
	p.Timeout(SeatSouth)
	p.Timeout(SeatSouth)
	if p.Controller(SeatSouth) != ControllerPlayer {
		t.Fatal("two timeouts marked the seat AFK")
	}
	p.Move(SeatSouth)
	p.Timeout(SeatSouth)
	p.Timeout(SeatSouth)
	if p.Controller(SeatSouth) != ControllerPlayer {
		t.Fatal("an own move did not reset the count")
	}
	if p.Timeout(SeatSouth) != ControllerBot || p.Controller(SeatNorth) != ControllerPlayer {
		t.Fatal("the third timeout in a row did not hand the seat to the bot")
	}
	if p.Reconnect(SeatSouth) != ControllerPlayer {
		t.Fatal("reconnecting did not take the seat back")
	}
	p.Timeout(SeatSouth)
	p.Timeout(SeatSouth)
	if p.Controller(SeatSouth) != ControllerPlayer {
		t.Fatal("the count did not restart from zero after reconnecting")
	}
	p.Leave(SeatSouth)
	if p.Controller(SeatSouth) != ControllerBot || p.Controller(SeatNorth) != ControllerPlayer {
		t.Fatal("leaving did not hand only the leaver's seat to the bot")
	}
	if p.Reconnect(SeatSouth) != ControllerPlayer {
		t.Fatal("a returning leaver did not take the seat back")
	}
	p.Timeout(SeatSouth)
	p.Timeout(SeatSouth)
	if p.Controller(SeatSouth) != ControllerPlayer {
		t.Fatal("the count did not restart from zero after a leaver came back")
	}
	var parsed Controller
	if err := parsed.UnmarshalText([]byte("bot")); err != nil || parsed != ControllerBot || ControllerPlayer.String() != "player" {
		t.Fatal("controller text")
	}
	if err := parsed.UnmarshalText([]byte("ghost")); err == nil {
		t.Fatal("ghost parsed as a controller")
	}
	allocations := testing.AllocsPerRun(100, func() {
		p.Timeout(SeatEast)
		p.Move(SeatEast)
		p.Controller(SeatEast)
	})
	if allocations != 0 {
		t.Fatalf("presence allocates %.0f times", allocations)
	}
}

func TestPresenceReportKeepsTheHistoryAcrossReconnects(t *testing.T) {
	p := NewPresence(DefaultConfig())
	if r := p.Report(SeatSouth); r != (PresenceReport{}) {
		t.Fatalf("fresh seat reports %+v", r)
	}
	p.Timeout(SeatSouth)
	p.Timeout(SeatSouth)
	if n := p.ConsecutiveTimeouts(SeatSouth); n != 2 {
		t.Fatalf("consecutive timeouts before a move: %d", n)
	}
	p.Move(SeatSouth)
	if n := p.ConsecutiveTimeouts(SeatSouth); n != 0 {
		t.Fatalf("consecutive timeouts after a move: %d", n)
	}
	if r := p.Report(SeatSouth); r != (PresenceReport{Timeouts: 2}) {
		t.Fatalf("after two timeouts and a move: %+v", r)
	}
	for i := 0; i < 3; i++ {
		p.Timeout(SeatSouth)
	}
	if r := p.Report(SeatSouth); r != (PresenceReport{Timeouts: 5, MarkedAfk: true, ByBot: true}) {
		t.Fatalf("marked AFK: %+v", r)
	}
	p.Reconnect(SeatSouth)
	if r := p.Report(SeatSouth); r != (PresenceReport{Timeouts: 5, MarkedAfk: true}) {
		t.Fatalf("after taking the seat back: %+v", r)
	}
	if p.Timeout(SeatSouth) != ControllerPlayer {
		t.Fatal("the consecutive count did not restart on reconnect")
	}
	p.Leave(SeatSouth)
	if r := p.Report(SeatSouth); r != (PresenceReport{Timeouts: 6, MarkedAfk: true, Left: true, ByBot: true}) {
		t.Fatalf("after leaving: %+v", r)
	}
	p.Reconnect(SeatSouth)
	if r := p.Report(SeatSouth); r != (PresenceReport{Timeouts: 6, MarkedAfk: true, Left: true}) {
		t.Fatalf("returned leaver: %+v", r)
	}
	if r := p.Report(SeatNorth); r != (PresenceReport{}) {
		t.Fatalf("another seat was touched: %+v", r)
	}
}

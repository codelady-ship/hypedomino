package room

import (
	"context"
	"slices"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/bot"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func quickTable(t *testing.T, roster []Player) *table {
	t.Helper()
	clock := newFakeClock()
	allowed := func(key string) bool { return slices.Contains([]string{"quick.hello", "quick.well_played"}, key) }
	room := New(Options{Mode: rules.Mode2v2, Config: rules.DefaultConfig(), Bot: bot.DefaultConfig(), BotLevel: bot.Hard, Seed: 5, Clock: clock, Roster: roster, Emoji: DefaultEmojiLimit(), QuickMessages: allowed})
	ctx, cancel := context.WithCancel(context.Background())
	go room.Run(ctx)
	t.Cleanup(func() {
		cancel()
		<-room.Done()
	})
	tab := &table{t: t, room: room, clock: clock, sinks: map[rules.Seat]chanSink{}, mode: rules.Mode2v2, stop: cancel}
	for _, seat := range rules.Mode2v2.Seats() {
		tab.sinks[seat] = make(chanSink, 256)
		room.Join(seat, tab.sinks[seat])
	}
	tab.settle()
	return tab
}

func (tab *table) quickMessages() map[rules.Seat][]QuickMessage {
	seen := map[rules.Seat][]QuickMessage{}
	for _, seat := range tab.mode.Seats() {
		for _, m := range tab.inbox(seat) {
			if m.Quick != nil {
				seen[seat] = append(seen[seat], *m.Quick)
			}
		}
	}
	return seen
}

func TestQuickMessagesReachTheTableOrOnlyTheTeam(t *testing.T) {
	tab := quickTable(t, []Player{{Seat: rules.SeatNorth}, {Seat: rules.SeatEast}, {Seat: rules.SeatSouth}, {Seat: rules.SeatWest}})
	tab.room.QuickMessage(rules.SeatNorth, "quick.hello", false)
	everyone := tab.quickMessages()
	for _, seat := range rules.Mode2v2.Seats() {
		if len(everyone[seat]) != 1 || everyone[seat][0] != (QuickMessage{Seat: rules.SeatNorth, Key: "quick.hello"}) {
			t.Fatalf("%s saw %v", seat, everyone[seat])
		}
	}

	tab.room.QuickMessage(rules.SeatEast, "quick.well_played", true)
	team := tab.quickMessages()
	want := QuickMessage{Seat: rules.SeatEast, Key: "quick.well_played", TeamOnly: true}
	for _, seat := range rules.Mode2v2.Seats() {
		mate := seat.Team() == rules.SeatEast.Team()
		if mate && (len(team[seat]) != 1 || team[seat][0] != want) || !mate && len(team[seat]) != 0 {
			t.Fatalf("%s saw %v", seat, team[seat])
		}
	}
}

func TestUnlistedKeysMutedBotAndThrottledSeatsSendNoQuickMessage(t *testing.T) {
	tab := quickTable(t, []Player{{Seat: rules.SeatNorth, Emoji: []string{"emoji.laugh"}}, {Seat: rules.SeatEast, Muted: true}, {Seat: rules.SeatSouth, Bot: true}})
	tab.room.QuickMessage(rules.SeatNorth, "quick.free_text", false)
	tab.room.QuickMessage(rules.SeatEast, "quick.hello", false)
	tab.room.QuickMessage(rules.SeatSouth, "quick.hello", false)
	tab.room.QuickMessage(rules.SeatWest, "quick.hello", false)
	if seen := tab.quickMessages(); len(seen) != 0 {
		t.Fatalf("sent %v", seen)
	}

	tab.room.Emoji(rules.SeatNorth, "emoji.laugh")
	for range DefaultEmojiLimit().Burst {
		tab.room.QuickMessage(rules.SeatNorth, "quick.hello", false)
	}
	if got := len(tab.quickMessages()[rules.SeatWest]); got != DefaultEmojiLimit().Burst-1 {
		t.Fatalf("west saw %d, want the burst left after the emoji", got)
	}
}

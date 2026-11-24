package room

import (
	"context"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/bot"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func emojiTable(t *testing.T, roster []Player) *table {
	t.Helper()
	clock := newFakeClock()
	room := New(Options{Mode: rules.Mode1v1, Config: rules.DefaultConfig(), Bot: bot.DefaultConfig(), BotLevel: bot.Hard, Seed: 5, Clock: clock, Roster: roster, Emoji: DefaultEmojiLimit()})
	ctx, cancel := context.WithCancel(context.Background())
	go room.Run(ctx)
	t.Cleanup(func() {
		cancel()
		<-room.Done()
	})
	tab := &table{t: t, room: room, clock: clock, sinks: map[rules.Seat]chanSink{}, mode: rules.Mode1v1, stop: cancel}
	for _, seat := range rules.Mode1v1.Seats() {
		tab.sinks[seat] = make(chanSink, 256)
		room.Join(seat, tab.sinks[seat])
	}
	return tab
}

func (tab *table) emojis(seat rules.Seat) []Emoji {
	var shown []Emoji
	for _, m := range tab.inbox(seat) {
		if m.Emoji != nil {
			shown = append(shown, *m.Emoji)
		}
	}
	return shown
}

func (tab *table) expectEmojis(sent int, want ...Emoji) {
	tab.t.Helper()
	for _, seat := range tab.mode.Seats() {
		got := tab.emojis(seat)
		if len(got) != len(want) {
			tab.t.Fatalf("after %d sent, %s saw %v, want %v", sent, seat, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				tab.t.Fatalf("after %d sent, %s saw %v, want %v", sent, seat, got, want)
			}
		}
	}
}

func TestAnEquippedEmojiReachesEverySeatWithinTheSpamLimit(t *testing.T) {
	tab := emojiTable(t, []Player{
		{Seat: rules.SeatX, Emoji: []string{"emoji.laugh", "emoji.cry"}},
		{Seat: rules.SeatY, Emoji: []string{"emoji.wave"}},
	})
	laugh := Emoji{Seat: rules.SeatX, Item: "emoji.laugh"}

	tab.room.Emoji(rules.SeatX, "emoji.laugh")
	tab.expectEmojis(1, laugh)

	tab.room.Emoji(rules.SeatX, "emoji.wave")
	tab.room.Emoji(rules.SeatX, "")
	tab.expectEmojis(3)

	tab.room.Emoji(rules.SeatX, "emoji.cry")
	tab.room.Emoji(rules.SeatX, "emoji.laugh")
	tab.room.Emoji(rules.SeatX, "emoji.laugh")
	tab.expectEmojis(6, Emoji{Seat: rules.SeatX, Item: "emoji.cry"}, laugh)

	tab.room.Emoji(rules.SeatY, "emoji.wave")
	tab.expectEmojis(7, Emoji{Seat: rules.SeatY, Item: "emoji.wave"})

	tab.clock.advance(1900 * time.Millisecond)
	tab.room.Emoji(rules.SeatX, "emoji.laugh")
	tab.expectEmojis(8)

	tab.clock.advance(100 * time.Millisecond)
	tab.room.Emoji(rules.SeatX, "emoji.laugh")
	tab.room.Emoji(rules.SeatX, "emoji.laugh")
	tab.expectEmojis(10, laugh)

	tab.clock.advance(time.Minute)
	for range 5 {
		tab.room.Emoji(rules.SeatX, "emoji.cry")
	}
	cry := Emoji{Seat: rules.SeatX, Item: "emoji.cry"}
	tab.expectEmojis(15, cry, cry, cry)
}

func TestBotsAbsentSeatsAndEmptySeatsSendNoEmoji(t *testing.T) {
	tab := emojiTable(t, []Player{
		{Seat: rules.SeatX, Emoji: []string{"emoji.laugh"}},
		{Seat: rules.SeatY, Bot: true, Emoji: []string{"emoji.laugh"}},
	})
	tab.room.Emoji(rules.SeatY, "emoji.laugh")
	tab.room.Emoji(rules.SeatNorth, "emoji.laugh")
	tab.expectEmojis(2)

	tab.room.Disconnect(rules.SeatX, tab.sinks[rules.SeatX])
	tab.room.Emoji(rules.SeatX, "emoji.laugh")
	if got := tab.emojis(rules.SeatY); len(got) != 0 {
		t.Fatalf("a disconnected seat sent %v", got)
	}
}

func TestAMutedSeatSendsNoEmojiWhileOthersStillDo(t *testing.T) {
	tab := emojiTable(t, []Player{
		{Seat: rules.SeatX, Muted: true, Emoji: []string{"emoji.laugh"}},
		{Seat: rules.SeatY, Emoji: []string{"emoji.wave"}},
	})
	tab.room.Emoji(rules.SeatX, "emoji.laugh")
	tab.room.Emoji(rules.SeatY, "emoji.wave")
	tab.expectEmojis(2, Emoji{Seat: rules.SeatY, Item: "emoji.wave"})
}

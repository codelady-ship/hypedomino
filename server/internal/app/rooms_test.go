package app

import (
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/bot"
	"github.com/Alievs-corp/hypedomino/server/internal/room"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
	"github.com/Alievs-corp/hypedomino/server/internal/rulesconfig"
)

func openTimed(t *testing.T, rooms *Rooms) (*room.Room, time.Time) {
	t.Helper()
	opened := time.Now()
	r, err := rooms.Open(room.Options{Mode: rules.Mode1v1, Config: rules.DefaultConfig(), Bot: bot.DefaultConfig(), BotLevel: bot.Hard, Seed: 9, Clock: realClock{}})
	if err != nil {
		t.Fatal(err)
	}
	return r, opened
}

func loadingTime(r *room.Room, opened time.Time) time.Duration {
	reply := make(chan room.View, 1)
	r.Snapshot(rules.SeatX, reply)
	return (<-reply).LoadingDeadline.Sub(opened).Round(time.Second)
}

func TestNewRoomsTakeTheTunedRulesAndOpenRoomsKeepTheirs(t *testing.T) {
	rooms := NewRooms()
	defer rooms.Stop()
	before, beforeOpened := openTimed(t, rooms)
	tuning, err := rulesconfig.Parse([]byte(`{"MatchLoadingSeconds": 40}`))
	if err != nil {
		t.Fatal(err)
	}
	rooms.Tune(tuning.Apply)
	after, afterOpened := openTimed(t, rooms)
	if got := loadingTime(after, afterOpened); got != 40*time.Second {
		t.Fatalf("a room opened after the change loads for %v", got)
	}
	if got := loadingTime(before, beforeOpened); got != 10*time.Second {
		t.Fatalf("a room opened before the change loads for %v", got)
	}
}

type emojiSink chan room.Message

func (s emojiSink) Deliver(m room.Message) {
	if m.Emoji != nil {
		s <- m
	}
}

func TestRoomsApplyTheConfiguredEmojiLimit(t *testing.T) {
	rooms := NewRooms()
	defer rooms.Stop()
	rooms.LimitEmoji(room.EmojiLimit{Burst: 1, Refill: time.Hour})
	r, err := rooms.Open(room.Options{Mode: rules.Mode1v1, Config: rules.DefaultConfig(), Bot: bot.DefaultConfig(), Seed: 9, Clock: realClock{}, Roster: []room.Player{{Seat: rules.SeatX, Emoji: []string{"emoji.laugh"}}}})
	if err != nil {
		t.Fatal(err)
	}
	sink := make(emojiSink, 4)
	r.Join(rules.SeatX, sink)
	r.Emoji(rules.SeatX, "emoji.laugh")
	r.Emoji(rules.SeatX, "emoji.laugh")
	reply := make(chan room.View, 1)
	r.Snapshot(rules.SeatX, reply)
	<-reply
	if len(sink) != 1 {
		t.Fatalf("%d emoji shown under a burst of 1", len(sink))
	}
}

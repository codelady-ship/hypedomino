package room

import (
	"bytes"
	"log/slog"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/panics"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func roomPanics(t *testing.T) string {
	t.Helper()
	var out strings.Builder
	if err := panics.WriteTo(&out); err != nil {
		t.Fatal(err)
	}
	match := regexp.MustCompile(`hd_panics_total\{where="room"\} (\d+)`).FindStringSubmatch(out.String())
	if match == nil {
		return "0"
	}
	return match[1]
}

func TestAPanickingRoomEndsAsCancelledWithoutTakingDownTheProcess(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	before := roomPanics(t)
	tab := newTable(t, rules.Mode2v2, 3)
	tab.snapshot(rules.SeatX)
	tab.room.enqueue(func(*Room) { panic("room broke") })
	for _, signal := range []<-chan struct{}{tab.room.Finished(), tab.room.Done()} {
		select {
		case <-signal:
		case <-time.After(5 * time.Second):
			t.Fatal("the room did not end after a panic")
		}
	}
	if record, recorded := tab.room.FinalRecord(); recorded && record.HasResult {
		t.Fatal("a room that panicked mid-match reported a result")
	}
	if before == roomPanics(t) {
		t.Fatal("the room panic was not counted")
	}
	for _, field := range []string{`"level":"ERROR"`, `"where":"room"`, `"panic":"room broke"`, `"stack":"goroutine `} {
		if !strings.Contains(logs.String(), field) {
			t.Fatalf("missing %s in %s", field, logs.String())
		}
	}
}

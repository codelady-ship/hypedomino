package matchqueue_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/matchmaking"
	"github.com/Alievs-corp/hypedomino/server/internal/matchqueue"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func TestPlacedObservesTheWaitOfEveryTicketByQueueAndMode(t *testing.T) {
	placedAt := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	waited := func(d time.Duration) matchqueue.Ticket {
		return matchqueue.Ticket{Entry: matchmaking.Entry{EnqueuedAt: placedAt.Add(-d)}}
	}
	m := matchqueue.NewMetrics()
	m.Placed(matchqueue.Placement{Match: matchmaking.Match{Mode: rules.Mode2v2, Queue: matchmaking.QueueRanked}, Tickets: []matchqueue.Ticket{waited(4 * time.Second), waited(40 * time.Second)}}, placedAt)
	m.Placed(matchqueue.Placement{Match: matchmaking.Match{Mode: rules.Mode1v1, Queue: matchmaking.QueueCasual}, Tickets: []matchqueue.Ticket{waited(30 * time.Second)}}, placedAt)
	var out strings.Builder
	if _, err := m.WriteTo(&out); err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{
		`hd_matchmaking_wait_seconds_bucket{queue="ranked",mode="2v2",le="5"} 1`,
		`hd_matchmaking_wait_seconds_bucket{queue="ranked",mode="2v2",le="45"} 2`,
		`hd_matchmaking_wait_seconds_sum{queue="ranked",mode="2v2"} 44`,
		`hd_matchmaking_wait_seconds_count{queue="casual",mode="1v1"} 1`,
		`# TYPE hd_matchmaking_left_total counter`,
	} {
		if !strings.Contains(out.String(), line+"\n") {
			t.Fatalf("missing %q in\n%s", line, out.String())
		}
	}
}

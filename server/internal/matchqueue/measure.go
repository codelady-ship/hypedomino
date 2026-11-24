package matchqueue

import (
	"io"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/metrics"
)

var waitBounds = []time.Duration{
	time.Second, 2 * time.Second, 5 * time.Second, 10 * time.Second, 15 * time.Second, 20 * time.Second, 30 * time.Second,
	45 * time.Second, time.Minute, 90 * time.Second, 2 * time.Minute, 3 * time.Minute, 5 * time.Minute,
}

type Metrics struct {
	waits *metrics.Histograms
	left  *metrics.Counter
}

func NewMetrics() *Metrics {
	return &Metrics{
		waits: metrics.NewHistograms("hd_matchmaking_wait_seconds", "Time from joining the queue to the match being placed on a node, per human player.", metrics.Labels{"queue", "mode"}, waitBounds),
		left:  metrics.NewCounter("hd_matchmaking_left_total", "Players who left the queue before a match formed, by a leave intent or a closed connection.", metrics.Labels{}),
	}
}

func (m *Metrics) Placed(placement Placement, now time.Time) {
	labels := metrics.Labels{placement.Match.Queue.String(), placement.Match.Mode.String()}
	for _, ticket := range placement.Tickets {
		m.waits.Observe(labels, now.Sub(ticket.Entry.EnqueuedAt))
	}
}

func (m *Metrics) WriteTo(w io.Writer) (int64, error) {
	waits, err := m.waits.WriteTo(w)
	if err != nil {
		return waits, err
	}
	left, err := m.left.WriteTo(w)
	return waits + left, err
}

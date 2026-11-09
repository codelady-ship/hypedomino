package loadbot

import (
	"context"
	"io"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/Alievs-corp/hypedomino/server/internal/matchstore"
	"github.com/Alievs-corp/hypedomino/server/internal/metrics"
	"github.com/Alievs-corp/hypedomino/server/internal/room"
)

type Archive interface {
	Begin(ctx context.Context, m matchstore.Started) error
	Finish(ctx context.Context, f matchstore.Finished) error
}

type Router interface {
	Claim(ctx context.Context, token string) error
	Release(ctx context.Context, token string) error
}

var storeBounds = []time.Duration{
	time.Millisecond, 2 * time.Millisecond, 5 * time.Millisecond, 10 * time.Millisecond, 25 * time.Millisecond,
	50 * time.Millisecond, 100 * time.Millisecond, 250 * time.Millisecond, 500 * time.Millisecond, time.Second,
}

type Stores struct {
	archive  Archive
	router   Router
	begins   *metrics.Histogram
	finishes *metrics.Histogram
	claims   *metrics.Histogram
	releases *metrics.Histogram
	failures atomic.Uint64
}

func NewStores(archive Archive, router Router) *Stores {
	return &Stores{
		archive:  archive,
		router:   router,
		begins:   metrics.NewHistogram("match_begin_seconds", "Match row and its participants written when a room opens.", storeBounds),
		finishes: metrics.NewHistogram("match_finish_seconds", "Match result, replay and seat results written when a room finishes.", storeBounds),
		claims:   metrics.NewHistogram("route_claim_seconds", "Route of one seat token claimed for this node.", storeBounds),
		releases: metrics.NewHistogram("route_release_seconds", "Route of one seat token released after the match.", storeBounds),
	}
}

func (s *Stores) Failures() uint64 {
	return s.failures.Load()
}

func (s *Stores) WriteTo(w io.Writer) (int64, error) {
	var written int64
	for _, h := range []*metrics.Histogram{s.begins, s.finishes, s.claims, s.releases} {
		n, err := h.WriteTo(w)
		written += n
		if err != nil {
			return written, err
		}
	}
	metrics.WriteGauge(w, "hd_store_failures", "Match writes and route claims or releases that returned an error.", float64(s.failures.Load()))
	return written, nil
}

func (s *Stores) timed(h *metrics.Histogram, write func(context.Context) error) {
	started := time.Now()
	err := write(context.Background())
	h.Observe(time.Since(started))
	if err != nil {
		s.failures.Add(1)
	}
}

func (s *Stores) track(c claim, r *room.Room) {
	started := matchstore.Started{ID: uuid.NewString(), StartedAt: time.Now(), Mode: c.mode, Kind: matchstore.KindCasual, Seed: c.room}
	for _, seat := range c.mode.Seats() {
		started.Participants = append(started.Participants, matchstore.Participant{Seat: seat, Bot: true})
	}
	if s.archive != nil {
		s.timed(s.begins, func(ctx context.Context) error { return s.archive.Begin(ctx, started) })
	}
	if s.router != nil {
		s.route(c, s.claims, s.router.Claim)
	}
	<-r.Finished()
	record, recorded := r.FinalRecord()
	if s.archive != nil && recorded {
		finished := matchstore.FinishedFrom(started, record, time.Now())
		s.timed(s.finishes, func(ctx context.Context) error { return s.archive.Finish(ctx, finished) })
	}
	if s.router != nil {
		s.route(c, s.releases, s.router.Release)
	}
}

func (s *Stores) route(c claim, h *metrics.Histogram, call func(context.Context, string) error) {
	for _, seat := range c.mode.Seats() {
		token := Token(c.room, c.mode, seat)
		s.timed(h, func(ctx context.Context) error { return call(ctx, token) })
	}
}

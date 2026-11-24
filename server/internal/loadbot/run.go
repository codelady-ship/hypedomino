package loadbot

import (
	"context"
	"errors"
	"log/slog"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/botclient"
	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

type Config struct {
	Target          string
	Rooms           int
	TwoVersusTwoPct int
	Ramp            time.Duration
	Hold            time.Duration
	ThinkMin        time.Duration
	ThinkMax        time.Duration
	ThinkFollow     time.Duration
	DropsPerMinute  float64
	StormAt         time.Duration
	Window          time.Duration
	Warmup          time.Duration
	ReportEvery     time.Duration
	AppVersion      string
	Seed            uint64
	Budget          time.Duration
	Node            func() (NodeSample, error)
}

func DefaultConfig() Config {
	return Config{
		Rooms:           100,
		TwoVersusTwoPct: 50,
		Ramp:            10 * time.Second,
		Hold:            time.Minute,
		ThinkMin:        time.Second,
		ThinkMax:        3 * time.Second,
		ThinkFollow:     200 * time.Millisecond,
		DropsPerMinute:  0.02,
		Window:          time.Minute,
		Warmup:          2 * time.Minute,
		ReportEvery:     10 * time.Second,
		AppVersion:      "1.0.0",
		Seed:            1,
		Budget:          100 * time.Millisecond,
	}
}

type Counters struct {
	Moves        atomic.Uint64
	Rejected     atomic.Uint64
	DialFailures atomic.Uint64
	ReadFailures atomic.Uint64
	Drops        atomic.Uint64
	Reconnects   atomic.Uint64
	Matches      atomic.Uint64
	Connections  atomic.Int64
	Peak         atomic.Int64
	SentBytes    atomic.Uint64
	Sent         atomic.Uint64
	StormDrops   atomic.Uint64
	Recovered    atomic.Uint64
}

type NodeSample struct {
	At         time.Duration
	HeapBytes  uint64
	Goroutines uint64
}

type Runner struct {
	config   Config
	logger   *slog.Logger
	start    time.Time
	overall  *Histogram
	steady   *Histogram
	windows  *Windows
	counters Counters
	nodeMu   sync.Mutex
	node     []NodeSample
	nextRoom atomic.Uint64
	storm    chan struct{}
	stormed  time.Time
	recovery atomic.Int64
	wg       sync.WaitGroup
}

func New(config Config, logger *slog.Logger) *Runner {
	return &Runner{config: config, logger: logger, overall: &Histogram{}, steady: &Histogram{}}
}

func (r *Runner) Run(ctx context.Context) (*Report, error) {
	if r.config.Rooms <= 0 || r.config.Target == "" {
		return nil, errors.New("loadbot: rooms and target are required")
	}
	r.start = time.Now()
	r.windows = NewWindows(r.start, r.config.Window)
	r.storm = make(chan struct{})
	if r.config.StormAt > 0 {
		storm := time.AfterFunc(r.config.StormAt, func() {
			r.stormed = time.Now()
			close(r.storm)
		})
		defer storm.Stop()
	}
	ctx, cancel := context.WithTimeout(ctx, r.config.Ramp+r.config.Hold)
	defer cancel()
	observed := make(chan struct{})
	go r.observe(ctx, observed)
	r.ramp(ctx)
	<-ctx.Done()
	r.wg.Wait()
	<-observed
	return r.report(), nil
}

func (r *Runner) ramp(ctx context.Context) {
	gap := time.Duration(0)
	if r.config.Rooms > 1 {
		gap = r.config.Ramp / time.Duration(r.config.Rooms)
	}
	for index := 0; index < r.config.Rooms; index++ {
		if ctx.Err() != nil {
			return
		}
		r.wg.Add(1)
		go r.keepRoom(ctx)
		if gap > 0 {
			select {
			case <-time.After(gap):
			case <-ctx.Done():
				return
			}
		}
	}
}

func (r *Runner) keepRoom(ctx context.Context) {
	defer r.wg.Done()
	random := rand.New(rand.NewPCG(r.config.Seed, r.nextRoom.Add(1)))
	for ctx.Err() == nil {
		id := r.config.Seed*1_000_000 + r.nextRoom.Add(1)
		mode := rules.Mode1v1
		if random.IntN(100) < r.config.TwoVersusTwoPct {
			mode = rules.Mode2v2
		}
		if r.playRoom(ctx, id, mode, random.Uint64()) {
			r.counters.Matches.Add(1)
		}
	}
}

func (r *Runner) playRoom(ctx context.Context, id uint64, mode rules.Mode, seed uint64) bool {
	roomCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	finished := make(chan bool, len(mode.Seats()))
	var seats sync.WaitGroup
	for index, seat := range mode.Seats() {
		seats.Add(1)
		go func() {
			defer seats.Done()
			finished <- r.playSeat(roomCtx, Token(id, mode, seat), rand.New(rand.NewPCG(seed, uint64(index))))
		}()
	}
	seats.Wait()
	close(finished)
	done := false
	for f := range finished {
		done = done || f
	}
	return done
}

func (r *Runner) dial(ctx context.Context, token string, acked uint64) (*botclient.Client, error) {
	c, err := botclient.Dial(ctx, r.config.Target, botclient.Options{Token: token, AppVersion: r.config.AppVersion, LastAcked: acked})
	if err != nil {
		if ctx.Err() == nil {
			r.counters.DialFailures.Add(1)
			r.debug("dial failed", "error", err)
		}
		return nil, err
	}
	n := r.counters.Connections.Add(1)
	for {
		peak := r.counters.Peak.Load()
		if n <= peak || r.counters.Peak.CompareAndSwap(peak, n) {
			break
		}
	}
	return c, nil
}

func (r *Runner) release(c *botclient.Client) {
	r.counters.Connections.Add(-1)
	r.counters.Sent.Add(uint64(c.Sent))
	r.counters.SentBytes.Add(uint64(c.SentBytes))
	c.Drop()
}

type seatRun struct {
	runner     *Runner
	ctx        context.Context
	random     *rand.Rand
	client     *botclient.Client
	pending    uint64
	sentAt     time.Time
	actedOn    *pb.MatchView
	turn       turnKey
	stormed    bool
	recovering bool
}

type turnKey struct {
	hands   uint32
	current pb.Seat
	state   pb.MatchState
}

func keyOf(view *pb.MatchView) turnKey {
	return turnKey{hands: view.GetHandsPlayed(), current: view.GetCurrent(), state: view.GetState()}
}

func (r *Runner) playSeat(ctx context.Context, token string, random *rand.Rand) bool {
	s := &seatRun{runner: r, ctx: ctx, random: random, stormed: r.stormOver()}
	acked := uint64(0)
	for ctx.Err() == nil {
		c, err := r.dial(ctx, token, acked)
		if err != nil {
			select {
			case <-time.After(time.Second):
			case <-ctx.Done():
			}
			continue
		}
		s.client, s.pending = c, 0
		finished, dropped := s.play()
		acked = c.Acked
		r.release(c)
		if finished {
			return true
		}
		if dropped {
			r.counters.Reconnects.Add(1)
		}
	}
	return false
}

var errDrop = errors.New("loadbot: dropped on purpose")

func (s *seatRun) play() (finished bool, dropped bool) {
	r := s.runner
	drop := s.nextDrop()
	frames := make(chan readResult, 16)
	go s.reader(s.client, frames)
	for {
		frame, err := s.next(frames, drop)
		if errors.Is(err, errDrop) {
			r.counters.Drops.Add(1)
			return false, true
		}
		if err != nil {
			if s.ctx.Err() == nil {
				r.counters.ReadFailures.Add(1)
				r.debug("read failed", "error", err)
			}
			return false, false
		}
		if rejected := frame.GetError(); rejected != nil {
			r.counters.Rejected.Add(1)
			r.debug("intent rejected", "code", rejected.GetCode().String(), "sequence", rejected.GetRejectedSequence(), "view", s.actedOn.String())
			s.pending = 0
		}
		s.client.Observe(frame)
		view := frame.GetMatchView()
		if view == nil {
			continue
		}
		if s.recovering {
			s.recovering = false
			r.recovered()
		}
		s.settle(frame.GetAckedSequence())
		if botclient.Finished(view) {
			return true, false
		}
		if botclient.NeedsReady(view) {
			if err := s.client.Ready(); err != nil && s.ctx.Err() == nil {
				r.counters.ReadFailures.Add(1)
				r.debug("ready failed", "error", err)
				return false, false
			}
			continue
		}
		if botclient.Idle(view) || s.pending != 0 {
			continue
		}
		key := keyOf(view)
		if !s.think(key != s.turn) {
			return false, false
		}
		s.turn = key
		s.sentAt, s.actedOn = time.Now(), view
		if err := s.client.Act(view); err != nil {
			if s.ctx.Err() == nil {
				r.counters.ReadFailures.Add(1)
				r.debug("send failed", "error", err)
			}
			return false, false
		}
		s.pending = s.client.Sequence
	}
}

func (s *seatRun) nextDrop() <-chan time.Time {
	if s.runner.config.DropsPerMinute <= 0 {
		return nil
	}
	wait := time.Duration(s.random.ExpFloat64() / s.runner.config.DropsPerMinute * float64(time.Minute))
	return time.After(wait)
}

type readResult struct {
	frame *pb.Envelope
	err   error
}

func (s *seatRun) reader(client *botclient.Client, frames chan<- readResult) {
	defer close(frames)
	for {
		frame, err := client.Receive()
		frames <- readResult{frame, err}
		if err != nil {
			return
		}
	}
}

func (s *seatRun) next(frames <-chan readResult, drop <-chan time.Time) (*pb.Envelope, error) {
	select {
	case res := <-frames:
		return res.frame, res.err
	case <-drop:
		s.client.Drop()
		return nil, errDrop
	case <-s.nextStorm():
		s.stormed, s.recovering = true, true
		s.runner.counters.StormDrops.Add(1)
		s.client.Drop()
		return nil, errDrop
	case <-s.ctx.Done():
		s.client.Drop()
		return nil, s.ctx.Err()
	}
}

func (s *seatRun) settle(acked uint64) {
	if s.pending == 0 || acked < s.pending {
		return
	}
	now := time.Now()
	latency := now.Sub(s.sentAt)
	s.pending = 0
	s.runner.record(latency, now)
}

func (s *seatRun) think(firstOfTurn bool) bool {
	wait := s.runner.config.ThinkFollow
	if firstOfTurn {
		wait = s.runner.config.ThinkMin
		if span := s.runner.config.ThinkMax - s.runner.config.ThinkMin; span > 0 {
			wait += time.Duration(s.random.Int64N(int64(span)))
		}
	}
	if wait <= 0 {
		return true
	}
	select {
	case <-time.After(wait):
		return true
	case <-s.ctx.Done():
		return false
	}
}

func (r *Runner) record(latency time.Duration, now time.Time) {
	r.counters.Moves.Add(1)
	r.overall.Add(latency)
	r.windows.At(now).Add(latency)
	if now.Sub(r.start) >= r.config.Warmup {
		r.steady.Add(latency)
	}
}

func (r *Runner) observe(ctx context.Context, done chan<- struct{}) {
	defer close(done)
	every := r.config.ReportEvery
	if every <= 0 {
		every = 10 * time.Second
	}
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			r.sample()
		case <-ctx.Done():
			return
		}
	}
}

func (r *Runner) sample() {
	elapsed := time.Since(r.start)
	attrs := []any{
		"elapsed", elapsed.Round(time.Second).String(),
		"connections", r.counters.Connections.Load(),
		"moves", r.counters.Moves.Load(),
		"p99", r.overall.Percentile(99).String(),
		"matches", r.counters.Matches.Load(),
		"drops", r.counters.Drops.Load(),
		"errors", r.counters.Rejected.Load() + r.counters.DialFailures.Load() + r.counters.ReadFailures.Load(),
	}
	if r.config.Node != nil {
		attrs = append(attrs, r.sampleNode(elapsed)...)
	}
	if r.logger != nil {
		r.logger.Info("load", attrs...)
	}
}

func (r *Runner) sampleNode(elapsed time.Duration) []any {
	sample, err := r.config.Node()
	if err != nil {
		return []any{"node_error", err.Error()}
	}
	sample.At = elapsed
	r.nodeMu.Lock()
	r.node = append(r.node, sample)
	r.nodeMu.Unlock()
	return []any{"heap_bytes", sample.HeapBytes, "goroutines", sample.Goroutines}
}

func (r *Runner) debug(msg string, args ...any) {
	if r.logger != nil {
		r.logger.Debug(msg, args...)
	}
}

func (r *Runner) stormOver() bool {
	select {
	case <-r.storm:
		return true
	default:
		return false
	}
}

func (s *seatRun) nextStorm() <-chan struct{} {
	if s.stormed {
		return nil
	}
	return s.runner.storm
}

func (r *Runner) recovered() {
	r.counters.Recovered.Add(1)
	took := int64(time.Since(r.stormed))
	for {
		longest := r.recovery.Load()
		if took <= longest || r.recovery.CompareAndSwap(longest, took) {
			return
		}
	}
}

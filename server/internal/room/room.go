package room

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/bot"
	"github.com/Alievs-corp/hypedomino/server/internal/replay"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

var (
	ErrSeatTaken     = errors.New("room: seat already taken")
	ErrNotSeated     = errors.New("room: seat is empty")
	ErrNotStarted    = errors.New("room: the match has not started")
	ErrFinished      = errors.New("room: the match is over")
	ErrSeatByBot     = errors.New("room: the seat is played by the bot")
	ErrStaleSequence = errors.New("room: sequence already handled")
	ErrBarred        = errors.New("room: seat barred by the room creator")
)

type Kind uint8

const (
	KindRanked Kind = iota
	KindCasual
	KindCustom
)

func (k Kind) String() string {
	switch k {
	case KindCasual:
		return "casual"
	case KindCustom:
		return "custom"
	}
	return "ranked"
}

type State uint8

const (
	StateWaiting State = iota
	StateOpening
	StatePlaying
	StateFinished
	StateLoading
)

type Clock interface {
	Now() time.Time
	After(d time.Duration) <-chan time.Time
}

type Message struct {
	View     View
	Rejected bool
	Sequence uint64
	Reason   error
	Emoji    *Emoji
	Quick    *QuickMessage
}

type QuickMessage struct {
	Seat     rules.Seat
	Key      string
	TeamOnly bool
}

type Emoji struct {
	Seat rules.Seat
	Item string
}

type EmojiLimit struct {
	Burst  int
	Refill time.Duration
}

func DefaultEmojiLimit() EmojiLimit {
	return EmojiLimit{Burst: 3, Refill: 2 * time.Second}
}

type Sink interface {
	Deliver(Message)
}

type MoveObserver interface {
	Observe(elapsed time.Duration)
}

type Player struct {
	Seat      rules.Seat
	Nickname  string
	Character string
	Frame     string
	Avatar    string
	RankLabel string
	Emblems   []string
	Emoji     []string
	Bot       bool
	Muted     bool
}

type Options struct {
	Kind           Kind
	Mode           rules.Mode
	Config         rules.Config
	Bot            bot.Config
	BotLevel       bot.Level
	Seed           uint64
	Table          string
	TableDraw      [32]byte
	Clock          Clock
	Roster         []Player
	ReconnectGrace time.Duration
	Moves          MoveObserver
	Emoji          EmojiLimit
	QuickMessages  func(key string) bool
}

type seatState struct {
	sink        Sink
	acked       uint64
	loading     int
	ready       bool
	absentSince time.Time
	emojiTokens float64
	emojiAt     time.Time
}

func (s *seatState) spendEmoji(limit EmojiLimit, now time.Time) bool {
	switch {
	case s.emojiAt.IsZero():
		s.emojiTokens = float64(limit.Burst)
	case limit.Refill > 0:
		s.emojiTokens = min(float64(limit.Burst), s.emojiTokens+float64(now.Sub(s.emojiAt))/float64(limit.Refill))
	}
	s.emojiAt = now
	if s.emojiTokens < 1 {
		return false
	}
	s.emojiTokens--
	return true
}

type turnKey struct {
	phase   rules.Phase
	current rules.Seat
	hands   int
}

type command func(*Room)

type Room struct {
	options  Options
	commands chan command
	state    State
	seats    [6]seatState
	recorder *replay.Recorder
	presence rules.Presence
	bots     [6]*bot.Player
	armed    turnKey
	deadline time.Time
	timer    <-chan time.Time
	moves    []rules.Move
	done     chan struct{}
	barred   [6]bool
	finished chan struct{}
	ended    bool
}

func New(options Options) *Room {
	if options.Emoji == (EmojiLimit{}) {
		options.Emoji = DefaultEmojiLimit()
	}
	r := &Room{
		options:  options,
		commands: make(chan command, 64),
		presence: rules.NewPresence(options.Config),
		armed:    turnKey{hands: -1},
		moves:    make([]rules.Move, 0, 4*rules.SetSize),
		done:     make(chan struct{}),
		finished: make(chan struct{}),
	}
	random := rules.NewSeededRandom(options.Seed ^ 0x9e3779b97f4a7c15)
	for _, seat := range options.Mode.Seats() {
		r.bots[seat] = bot.New(options.BotLevel, options.Bot, random)
	}
	return r
}

func (r *Room) Run(ctx context.Context) {
	defer close(r.done)
	r.open()
	for {
		select {
		case <-ctx.Done():
			return
		case cmd := <-r.commands:
			cmd(r)
		case <-r.timer:
			r.onTimer()
		}
	}
}

func (r *Room) Done() <-chan struct{} {
	return r.done
}

func (r *Room) Finished() <-chan struct{} {
	return r.finished
}

func (r *Room) end() {
	if !r.ended {
		r.settleAbsences()
		r.ended = true
		close(r.finished)
	}
}

func (r *Room) enqueue(cmd command) {
	select {
	case r.commands <- cmd:
	case <-r.done:
	}
}

func (r *Room) Join(seat rules.Seat, sink Sink) {
	r.enqueue(func(r *Room) { r.join(seat, sink) })
}

func (r *Room) Reconnect(seat rules.Seat, sink Sink) {
	r.enqueue(func(r *Room) { r.reconnect(seat, sink) })
}

func (r *Room) Leave(seat rules.Seat) {
	r.enqueue(func(r *Room) { r.leave(seat) })
}

func (r *Room) Kick(seat rules.Seat) {
	r.enqueue(func(r *Room) {
		if r.seated(seat) {
			r.barred[seat] = true
			r.leave(seat)
		}
	})
}

func (r *Room) Admit(seat rules.Seat) {
	r.enqueue(func(r *Room) { r.barred[seat] = false })
}

func (r *Room) Kind() Kind {
	return r.options.Kind
}

func (r *Room) Loading(seat rules.Seat, progress int, ready bool) {
	r.enqueue(func(r *Room) { r.loading(seat, progress, ready) })
}

func (r *Room) Disconnect(seat rules.Seat, sink Sink) {
	r.enqueue(func(r *Room) { r.disconnect(seat, sink) })
}

func (r *Room) Intent(seat rules.Seat, sequence uint64, intent replay.Intent) {
	received := r.options.Clock.Now()
	r.enqueue(func(r *Room) { r.intent(seat, sequence, intent, received) })
}

func (r *Room) Emoji(seat rules.Seat, item string) {
	received := r.options.Clock.Now()
	r.enqueue(func(r *Room) { r.emoji(seat, item, received) })
}

func (r *Room) QuickMessage(seat rules.Seat, key string, teamOnly bool) {
	received := r.options.Clock.Now()
	r.enqueue(func(r *Room) { r.quickMessage(seat, key, teamOnly, received) })
}

func (r *Room) Snapshot(seat rules.Seat, reply chan<- View) {
	r.enqueue(func(r *Room) { reply <- r.view(seat) })
}

func (r *Room) Record(reply chan<- replay.Record) {
	r.enqueue(func(r *Room) {
		if r.recorder == nil {
			reply <- replay.Record{}
			return
		}
		reply <- r.recorder.Record()
	})
}

func (r *Room) FinalRecord() (replay.Record, bool) {
	<-r.done
	if r.recorder == nil {
		return replay.Record{}, false
	}
	return r.recorder.Record(), true
}

func (r *Room) FinalPresence(seat rules.Seat) rules.PresenceReport {
	<-r.done
	return r.presence.Report(seat)
}

func (r *Room) join(seat rules.Seat, sink Sink) {
	if !r.seated(seat) || r.seats[seat].sink != nil {
		sink.Deliver(Message{Rejected: true, Reason: ErrSeatTaken})
		return
	}
	if r.barred[seat] {
		sink.Deliver(Message{Rejected: true, Reason: ErrBarred})
		return
	}
	r.seats[seat].sink = sink
	r.returned(seat)
	r.settle()
}

func (r *Room) seated(seat rules.Seat) bool {
	for _, candidate := range r.options.Mode.Seats() {
		if candidate == seat {
			return true
		}
	}
	return false
}

func (r *Room) allReady() bool {
	for _, seat := range r.options.Mode.Seats() {
		if !r.seats[seat].ready {
			return false
		}
	}
	return true
}

func (r *Room) reconnect(seat rules.Seat, sink Sink) {
	if !r.seated(seat) {
		sink.Deliver(Message{Rejected: true, Reason: ErrNotSeated})
		return
	}
	if r.barred[seat] {
		sink.Deliver(Message{Rejected: true, Reason: ErrBarred})
		return
	}
	r.seats[seat].sink = sink
	r.returned(seat)
	r.settle()
}

func (r *Room) disconnect(seat rules.Seat, sink Sink) {
	if r.seated(seat) && r.seats[seat].sink == sink {
		r.seats[seat].sink = nil
	}
}

func (r *Room) leave(seat rules.Seat) {
	if !r.seated(seat) {
		return
	}
	r.seats[seat].sink = nil
	r.seats[seat].absentSince = time.Time{}
	r.presence.Leave(seat)
	r.settle()
}

func (r *Room) intent(seat rules.Seat, sequence uint64, intent replay.Intent, received time.Time) {
	if err := r.admit(seat, sequence); err != nil {
		if !errors.Is(err, ErrStaleSequence) {
			r.reject(seat, sequence, err)
		}
		return
	}
	intent.Seat = seat
	if err := replay.Apply(r.recorder, intent); err != nil {
		r.reject(seat, sequence, err)
		return
	}
	r.seats[seat].acked = sequence
	r.presence.Move(seat)
	r.settle()
	if r.options.Moves != nil {
		r.options.Moves.Observe(r.options.Clock.Now().Sub(received))
	}
}

func (r *Room) emoji(seat rules.Seat, item string, received time.Time) {
	if !r.seated(seat) || r.seats[seat].sink == nil || !r.equipped(seat, item) || !r.seats[seat].spendEmoji(r.options.Emoji, received) {
		return
	}
	shown := &Emoji{Seat: seat, Item: item}
	for _, other := range r.options.Mode.Seats() {
		if sink := r.seats[other].sink; sink != nil {
			sink.Deliver(Message{Emoji: shown})
		}
	}
}

func (r *Room) equipped(seat rules.Seat, item string) bool {
	player, ok := r.speaker(seat)
	return ok && slices.Contains(player.Emoji, item)
}

func (r *Room) speaker(seat rules.Seat) (Player, bool) {
	for _, player := range r.options.Roster {
		if player.Seat == seat {
			return player, !player.Bot && !player.Muted
		}
	}
	return Player{}, false
}

func (r *Room) quickMessage(seat rules.Seat, key string, teamOnly bool, received time.Time) {
	if !r.seated(seat) || r.seats[seat].sink == nil || r.options.QuickMessages == nil || !r.options.QuickMessages(key) {
		return
	}
	if _, ok := r.speaker(seat); !ok || !r.seats[seat].spendEmoji(r.options.Emoji, received) {
		return
	}
	shown := &QuickMessage{Seat: seat, Key: key, TeamOnly: teamOnly}
	for _, other := range r.options.Mode.Seats() {
		if sink := r.seats[other].sink; sink != nil && (!teamOnly || other.Team() == seat.Team()) {
			sink.Deliver(Message{Quick: shown})
		}
	}
}

func (r *Room) admit(seat rules.Seat, sequence uint64) error {
	switch {
	case !r.seated(seat):
		return ErrNotSeated
	case r.recorder == nil:
		return ErrNotStarted
	case r.state == StateFinished:
		return ErrFinished
	case r.presence.Controller(seat) == rules.ControllerBot:
		return ErrSeatByBot
	case sequence <= r.seats[seat].acked:
		return ErrStaleSequence
	}
	return nil
}

func (r *Room) match() *rules.Match {
	return r.recorder.Match
}

func (r *Room) key() turnKey {
	m := r.match()
	return turnKey{phase: m.Phase(), current: m.Current(), hands: m.HandsPlayed()}
}

func (r *Room) actingSeat() (rules.Seat, bool) {
	m := r.match()
	switch m.Phase() {
	case rules.PhasePlaying:
		return m.Current(), true
	case rules.PhaseOpening:
		if opening, forced := m.ForcedOpening(); forced {
			return opening.Seat, true
		}
		var buffer [2]rules.Seat
		return r.options.Mode.TeamSeats(m.OpeningTeam(), buffer[:0])[0], true
	}
	return 0, false
}

func (r *Room) settle() {
	if r.recorder != nil {
		for guard := 0; guard < 8*rules.SetSize; guard++ {
			r.state = stateOf(r.match().Phase())
			if r.state == StateFinished {
				r.timer = nil
				r.end()
				break
			}
			if key := r.key(); key != r.armed {
				r.armed = key
				limit := rules.TurnLimit(r.options.Config)
				r.deadline = r.options.Clock.Now().Add(limit)
				r.timer = r.options.Clock.After(limit)
			}
			seat, acting := r.actingSeat()
			if !acting || r.presence.Controller(seat) != rules.ControllerBot {
				break
			}
			if _, err := r.bots[seat].Act(r.recorder, seat); err != nil {
				break
			}
			r.presence.Move(seat)
		}
	}
	r.broadcast()
}

func stateOf(phase rules.Phase) State {
	switch phase {
	case rules.PhaseOpening:
		return StateOpening
	case rules.PhasePlaying:
		return StatePlaying
	}
	return StateFinished
}

func (r *Room) onTimer() {
	if r.state == StateLoading {
		r.deal()
		return
	}
	if r.recorder == nil || r.state == StateFinished {
		return
	}
	r.settleAbsences()
	seat, acting := r.actingSeat()
	if !acting {
		return
	}
	r.presence.Timeout(seat)
	key := r.key()
	for guard := 0; guard < rules.SetSize && r.key() == key; guard++ {
		if _, err := r.bots[seat].Act(r.recorder, seat); err != nil {
			break
		}
	}
	r.settle()
}

func (r *Room) reject(seat rules.Seat, sequence uint64, reason error) {
	if sink := r.seats[seat].sink; sink != nil {
		sink.Deliver(Message{View: r.view(seat), Rejected: true, Sequence: sequence, Reason: fmt.Errorf("rejected intent %d: %w", sequence, reason)})
	}
}

func (r *Room) broadcast() {
	for _, seat := range r.options.Mode.Seats() {
		if sink := r.seats[seat].sink; sink != nil {
			sink.Deliver(Message{View: r.view(seat)})
		}
	}
}

func (r *Room) open() {
	r.state = StateLoading
	limit := rules.LoadingLimit(r.options.Config)
	r.deadline = r.options.Clock.Now().Add(limit)
	r.timer = r.options.Clock.After(limit)
}

func (r *Room) loading(seat rules.Seat, progress int, ready bool) {
	if !r.seated(seat) || r.state != StateLoading {
		return
	}
	s := &r.seats[seat]
	progress = min(100, max(0, progress)) / 10 * 10
	ready = s.ready || ready
	if ready {
		progress = 100
	}
	if s.loading == progress && s.ready == ready {
		return
	}
	s.loading, s.ready = progress, ready
	if r.allReady() {
		r.deal()
		return
	}
	r.broadcast()
}

func (r *Room) deal() {
	r.recorder = replay.NewRecorder(r.options.Mode, r.options.Config, r.options.Seed, r.options.TableDraw)
	now := r.options.Clock.Now()
	for _, seat := range r.options.Mode.Seats() {
		if r.seats[seat].ready && r.seats[seat].sink != nil {
			continue
		}
		r.presence.Absent(seat)
		r.seats[seat].absentSince = now
	}
	r.settle()
}

func (r *Room) settleAbsence(seat rules.Seat) {
	since := r.seats[seat].absentSince
	if since.IsZero() {
		return
	}
	if r.options.Clock.Now().Sub(since) >= r.options.ReconnectGrace {
		r.presence.Leave(seat)
		r.seats[seat].absentSince = time.Time{}
	}
}

func (r *Room) settleAbsences() {
	for _, seat := range r.options.Mode.Seats() {
		r.settleAbsence(seat)
	}
}

func (r *Room) returned(seat rules.Seat) {
	r.settleAbsence(seat)
	r.seats[seat].absentSince = time.Time{}
	r.presence.Reconnect(seat)
}

package custom

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/room"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

var (
	ErrUnknownCode    = errors.New("custom: unknown room code")
	ErrNotLeader      = errors.New("custom: only the room leader may do that")
	ErrSeatTaken      = errors.New("custom: seat taken")
	ErrNotInRoom      = errors.New("custom: account is not in the room")
	ErrAlreadyInRoom  = errors.New("custom: account is already in the room")
	ErrFull           = errors.New("custom: the room is full")
	ErrSeatsOpen      = errors.New("custom: every seat of the mode must be filled")
	ErrStarted        = errors.New("custom: the match has started")
	ErrInvalidSeat    = errors.New("custom: seat outside the mode")
	ErrInvalidMode    = errors.New("custom: invalid mode")
	ErrInMatch        = errors.New("custom: account is in a running match")
	ErrWrongPhase     = errors.New("custom: not allowed in this phase")
	ErrBlocked        = errors.New("custom: a block stands between the accounts")
	ErrUnknownEmoji   = errors.New("custom: emoji not equipped by the member")
	ErrMuted          = errors.New("custom: the member is muted")
	ErrEmojiThrottled = errors.New("custom: emoji sent too fast")
)

type Gate interface {
	InMatch(account string) bool
}

type Blocks interface {
	Blocked(ctx context.Context, a, b string) (bool, error)
}

type Sink interface {
	Deliver(Event)
}

type Event struct {
	State   *State
	Account string
	Emoji   string
	Request bool
	Seat    rules.Seat
}

type Phase uint8

const (
	PhaseLobby Phase = iota + 1
	PhaseTeams
	PhaseMatch
)

const codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

const CodeLength = 6

func Capacity() int {
	return len(rules.Mode2v2.Seats())
}

type Random interface {
	Uint64() uint64
}

type Opener func(options room.Options, seats []SeatState) (*room.Room, error)

type Directory interface {
	Claim(ctx context.Context, code string) error
	Release(ctx context.Context, code string) error
}

type Lobby struct {
	random    Random
	opener    Opener
	gate      Gate
	blocks    Blocks
	emoji     room.EmojiLimit
	directory Directory
	mu        sync.Mutex
	rooms     map[string]*Room
}

func NewLobby(random Random, opener Opener) *Lobby {
	return &Lobby{random: random, opener: opener, emoji: room.DefaultEmojiLimit(), rooms: map[string]*Room{}}
}

func (l *Lobby) Guard(gate Gate) {
	l.gate = gate
}

func (l *Lobby) Screen(blocks Blocks) {
	l.blocks = blocks
}

func (l *Lobby) LimitEmoji(limit room.EmojiLimit) {
	l.emoji = limit
}

func (l *Lobby) Route(directory Directory) {
	l.directory = directory
}

func (l *Lobby) busy(account string) bool {
	return l.gate != nil && l.gate.InMatch(account)
}

func (l *Lobby) Create(creator Member) (*Room, error) {
	if l.busy(creator.Account) {
		return nil, ErrInMatch
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	code := l.code()
	for l.rooms[code] != nil {
		code = l.code()
	}
	if l.directory != nil {
		if err := l.directory.Claim(context.Background(), code); err != nil {
			return nil, err
		}
	}
	r := &Room{lobby: l, code: code, phase: PhaseLobby, members: []*member{{Member: creator}}}
	l.rooms[code] = r
	return r, nil
}

func (l *Lobby) Renew(ctx context.Context) error {
	if l.directory == nil {
		return nil
	}
	l.mu.Lock()
	codes := make([]string, 0, len(l.rooms))
	for code := range l.rooms {
		codes = append(codes, code)
	}
	l.mu.Unlock()
	var failed error
	for _, code := range codes {
		failed = errors.Join(failed, l.directory.Claim(ctx, code))
	}
	return failed
}

func (l *Lobby) code() string {
	code := make([]byte, CodeLength)
	value := l.random.Uint64()
	for index := range code {
		code[index] = codeAlphabet[value%uint64(len(codeAlphabet))]
		value /= uint64(len(codeAlphabet))
	}
	return string(code)
}

func (l *Lobby) Find(code string) (*Room, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	r, ok := l.rooms[code]
	if !ok {
		return nil, ErrUnknownCode
	}
	return r, nil
}

func (l *Lobby) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.rooms)
}

func (l *Lobby) close(code string) {
	l.mu.Lock()
	delete(l.rooms, code)
	l.mu.Unlock()
	if l.directory != nil {
		l.directory.Release(context.Background(), code)
	}
}

type Member struct {
	Account string
	Emoji   []string
	Muted   bool
	Sink    Sink
}

type member struct {
	Member
	emoji room.EmojiBucket
}

type Room struct {
	lobby   *Lobby
	code    string
	mu      sync.Mutex
	members []*member
	phase   Phase
	mode    rules.Mode
	seats   [6]string
	table   *room.Room
	closed  bool
}

type SeatState struct {
	Seat    rules.Seat
	Account string
}

type State struct {
	Code    string
	Leader  string
	Members []string
	Phase   Phase
	Mode    rules.Mode
	Seats   []SeatState
	Closed  bool
}

func (r *Room) Code() string {
	return r.code
}

func (r *Room) Table() *room.Room {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.table
}

func (r *Room) State() State {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.state()
}

func (r *Room) state() State {
	state := State{Code: r.code, Phase: r.phase, Mode: r.mode, Closed: r.closed}
	for _, m := range r.members {
		state.Members = append(state.Members, m.Account)
	}
	if len(r.members) > 0 {
		state.Leader = r.members[0].Account
	}
	for _, seat := range r.mode.Seats() {
		state.Seats = append(state.Seats, SeatState{Seat: seat, Account: r.seats[seat]})
	}
	return state
}

func (r *Room) changed() {
	state := r.state()
	r.deliver(Event{State: &state})
}

func (r *Room) deliver(event Event) {
	for _, m := range r.members {
		if m.Sink != nil {
			m.Sink.Deliver(event)
		}
	}
}

func (r *Room) find(account string) *member {
	for _, m := range r.members {
		if m.Account == account {
			return m
		}
	}
	return nil
}

func (r *Room) leads(account string) bool {
	return len(r.members) > 0 && r.members[0].Account == account
}

func (r *Room) seatOf(account string) (rules.Seat, bool) {
	for _, seat := range r.mode.Seats() {
		if r.seats[seat] == account {
			return seat, true
		}
	}
	return 0, false
}

func (r *Room) valid(seat rules.Seat) bool {
	return slices.Contains(r.mode.Seats(), seat)
}

func (r *Room) open() error {
	switch {
	case r.closed:
		return ErrUnknownCode
	case r.phase == PhaseMatch:
		return ErrStarted
	}
	return nil
}

func (r *Room) admissible(account string) error {
	switch {
	case r.find(account) != nil:
		return ErrAlreadyInRoom
	case len(r.members) >= Capacity():
		return ErrFull
	case r.lobby.busy(account):
		return ErrInMatch
	}
	return nil
}

func (r *Room) Join(joiner Member) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.open(); err != nil {
		return err
	}
	if err := r.admissible(joiner.Account); err != nil {
		return err
	}
	r.members = append(r.members, &member{Member: joiner})
	r.changed()
	return nil
}

func (r *Room) Invite(ctx context.Context, from, to string) error {
	if err := r.invitable(from, to); err != nil {
		return err
	}
	if r.lobby.blocks == nil {
		return nil
	}
	blocked, err := r.lobby.blocks.Blocked(ctx, from, to)
	if err != nil {
		return err
	}
	if blocked {
		return ErrBlocked
	}
	return nil
}

func (r *Room) invitable(from, to string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.open(); err != nil {
		return err
	}
	if r.find(from) == nil {
		return ErrNotInRoom
	}
	return r.admissible(to)
}

func (r *Room) Leave(account string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.find(account) == nil {
		return ErrNotInRoom
	}
	r.drop(account)
	seat, seated := r.seatOf(account)
	switch {
	case seated && r.phase == PhaseMatch:
		r.table.Leave(seat)
	case seated:
		r.seats[seat] = ""
	}
	r.settle()
	return nil
}

func (r *Room) drop(account string) {
	r.members = slices.DeleteFunc(r.members, func(m *member) bool { return m.Account == account })
}

func (r *Room) settle() {
	if len(r.members) == 0 {
		r.dissolve()
		return
	}
	r.changed()
}

func (r *Room) dissolve() {
	r.closed = true
	r.members, r.seats = nil, [6]string{}
	r.lobby.close(r.code)
}

func (r *Room) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dissolve()
}

func (r *Room) Remove(leader, account string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch {
	case !r.leads(leader), account == leader:
		return ErrNotLeader
	}
	seat, seated := r.seatOf(account)
	if r.find(account) == nil && !seated {
		return ErrNotInRoom
	}
	r.drop(account)
	if seated {
		r.seats[seat] = ""
	}
	if seated && r.phase == PhaseMatch {
		r.table.Kick(seat)
	}
	r.changed()
	return nil
}

func (r *Room) Teams(leader string, mode rules.Mode) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch {
	case !r.leads(leader):
		return ErrNotLeader
	case r.phase == PhaseMatch:
		return ErrStarted
	case mode != rules.Mode1v1 && mode != rules.Mode2v2:
		return ErrInvalidMode
	}
	if mode != r.mode {
		r.mode, r.seats = mode, [6]string{}
	}
	r.phase = PhaseTeams
	r.changed()
	return nil
}

func (r *Room) Back(leader string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch {
	case !r.leads(leader):
		return ErrNotLeader
	case r.phase != PhaseTeams:
		return ErrWrongPhase
	}
	r.phase = PhaseLobby
	r.changed()
	return nil
}

func (r *Room) Place(leader, account string, seat rules.Seat) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch {
	case !r.leads(leader):
		return ErrNotLeader
	case r.phase != PhaseTeams:
		return ErrWrongPhase
	case !r.valid(seat):
		return ErrInvalidSeat
	case r.find(account) == nil:
		return ErrNotInRoom
	}
	occupant := r.seats[seat]
	if from, seated := r.seatOf(account); seated {
		r.seats[from] = occupant
	}
	r.seats[seat] = account
	r.changed()
	return nil
}

func (r *Room) Add(leader string, added Member, seat rules.Seat) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch {
	case !r.leads(leader):
		return ErrNotLeader
	case r.phase != PhaseTeams && r.phase != PhaseMatch:
		return ErrWrongPhase
	case !r.valid(seat):
		return ErrInvalidSeat
	case r.seats[seat] != "":
		return ErrSeatTaken
	}
	if _, seated := r.seatOf(added.Account); seated {
		return ErrAlreadyInRoom
	}
	if r.find(added.Account) == nil {
		if err := r.admissible(added.Account); err != nil {
			return err
		}
		r.members = append(r.members, &member{Member: added})
	}
	r.seats[seat] = added.Account
	if r.phase == PhaseMatch {
		r.table.Admit(seat)
	}
	r.changed()
	return nil
}

func (r *Room) Start(leader string, options room.Options) (*room.Room, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch {
	case !r.leads(leader):
		return nil, ErrNotLeader
	case r.phase == PhaseMatch:
		return nil, ErrStarted
	case r.phase != PhaseTeams:
		return nil, ErrWrongPhase
	}
	for _, seat := range r.mode.Seats() {
		if r.seats[seat] == "" {
			return nil, ErrSeatsOpen
		}
	}
	options.Kind, options.Mode = room.KindCustom, r.mode
	table, err := r.lobby.opener(options, r.state().Seats)
	if err != nil {
		return nil, err
	}
	r.table, r.phase = table, PhaseMatch
	go r.await(table)
	r.changed()
	return table, nil
}

func (r *Room) await(table *room.Room) {
	select {
	case <-table.Finished():
	case <-table.Done():
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed || r.table != table {
		return
	}
	r.table, r.phase = nil, PhaseLobby
	for _, seat := range r.mode.Seats() {
		if r.find(r.seats[seat]) == nil {
			r.seats[seat] = ""
		}
	}
	r.changed()
}

func (r *Room) Request(account string, seat rules.Seat) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch {
	case r.find(account) == nil:
		return ErrNotInRoom
	case r.phase != PhaseTeams:
		return ErrWrongPhase
	case !r.valid(seat):
		return ErrInvalidSeat
	}
	if leader := r.members[0]; leader.Sink != nil {
		leader.Sink.Deliver(Event{Account: account, Request: true, Seat: seat})
	}
	return nil
}

func (r *Room) Emote(account, item string, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, err := r.inLobby(account)
	switch {
	case err != nil:
		return err
	case m.Muted:
		return ErrMuted
	case !slices.Contains(m.Emoji, item):
		return ErrUnknownEmoji
	case !m.emoji.Spend(r.lobby.emoji, now):
		return ErrEmojiThrottled
	}
	r.deliver(Event{Account: account, Emoji: item})
	return nil
}

func (r *Room) inLobby(account string) (*member, error) {
	m := r.find(account)
	switch {
	case m == nil:
		return nil, ErrNotInRoom
	case r.phase != PhaseLobby:
		return nil, ErrWrongPhase
	}
	return m, nil
}

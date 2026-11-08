package session

import (
	"context"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/Alievs-corp/hypedomino/server/internal/gateway"
	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
	"github.com/Alievs-corp/hypedomino/server/internal/replay"
	"github.com/Alievs-corp/hypedomino/server/internal/room"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

var (
	ErrUnknownSession = errors.New("session: unknown token")
	ErrAlreadySeated  = errors.New("session: already seated in a room")
)

type Authenticator interface {
	Authenticate(token string) (account, session string, err error)
}

type Config struct {
	ReconnectGrace time.Duration
}

func DefaultConfig() Config {
	return Config{ReconnectGrace: 60 * time.Second}
}

type Clock interface {
	Now() time.Time
	After(d time.Duration) <-chan time.Time
}

type Random interface {
	Uint64() uint64
}

type Session struct {
	token      string
	account    string
	room       *room.Room
	seat       rules.Seat
	joined     bool
	acked      uint64
	active     *connection
	generation uint64
	routed     bool
}

type Router interface {
	Claim(ctx context.Context, token string) error
	Release(ctx context.Context, token string) error
}

type Manager struct {
	config   Config
	clock    Clock
	random   Random
	router   Router
	auth     Authenticator
	clocks   ClockObserver
	replays  ReplayObserver
	checker  IntegrityChecker
	queue    Queue
	mu       sync.Mutex
	sessions map[string]*Session
	accounts map[string]map[string]*Session
}

func New(config Config, clock Clock, random Random) *Manager {
	return &Manager{config: config, clock: clock, random: random, sessions: map[string]*Session{}, accounts: map[string]map[string]*Session{}}
}

func (m *Manager) Route(router Router) {
	m.router = router
}

func (m *Manager) Authenticate(auth Authenticator) {
	m.auth = auth
}

type ClockObserver interface {
	Observe(ctx context.Context, account string, clientUnixMs uint64)
}

type ReplayObserver interface {
	Observe(ctx context.Context, account string)
}

type IntegrityChecker interface {
	SessionStart(ctx context.Context, account, session string, proof *pb.IntegrityProof) bool
}

func (m *Manager) CheckIntegrity(checker IntegrityChecker) {
	m.checker = checker
}

func (m *Manager) ObserveClocks(observer ClockObserver) {
	m.clocks = observer
}

func (m *Manager) ObserveReplays(observer ReplayObserver) {
	m.replays = observer
}

func (c *connection) Replayed() {
	if c.manager.replays != nil {
		c.manager.replays.Observe(context.Background(), c.session.account)
	}
}

func (m *Manager) claim(token string) error {
	if m.router == nil {
		return nil
	}
	return m.router.Claim(context.Background(), token)
}

func (m *Manager) release(token string) {
	if m.router != nil {
		m.router.Release(context.Background(), token)
	}
}

func (m *Manager) Create(account string) string {
	var raw [16]byte
	for offset := 0; offset < len(raw); offset += 8 {
		value := m.random.Uint64()
		for index := 0; index < 8; index++ {
			raw[offset+index] = byte(value >> (8 * index))
		}
	}
	token := hex.EncodeToString(raw[:])
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensure(token, account)
	return token
}

func (m *Manager) ensure(token, account string) *Session {
	if s, ok := m.sessions[token]; ok {
		return s
	}
	s := &Session{token: token, account: account}
	m.sessions[token] = s
	if m.accounts[account] == nil {
		m.accounts[account] = map[string]*Session{}
	}
	m.accounts[account][token] = s
	return s
}

func (m *Manager) RevokeAccount(account string) int {
	m.mu.Lock()
	var tokens []string
	for token := range m.accounts[account] {
		tokens = append(tokens, token)
	}
	m.mu.Unlock()
	for _, token := range tokens {
		m.Revoke(token)
	}
	return len(tokens)
}

func (m *Manager) InMatch(account string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.accounts[account] {
		if s.room != nil {
			return true
		}
	}
	return false
}

func (m *Manager) finish(s *Session, r *room.Room) {
	<-r.Finished()
	m.mu.Lock()
	if s.room != r {
		m.mu.Unlock()
		return
	}
	s.room, s.joined, s.acked = nil, false, 0
	routed := s.routed
	s.routed = false
	m.mu.Unlock()
	if routed {
		m.release(s.token)
	}
}

func (m *Manager) Seat(token string, r *room.Room, seat rules.Seat) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[token]
	switch {
	case !ok:
		return ErrUnknownSession
	case s.room != nil:
		return ErrAlreadySeated
	}
	if err := m.claim(token); err != nil {
		return err
	}
	s.room, s.seat, s.joined, s.acked, s.routed = r, seat, false, 0, true
	go m.finish(s, r)
	if s.active != nil {
		s.active.attach(s)
	}
	return nil
}

func (m *Manager) Revoke(token string) {
	m.mu.Lock()
	s, ok := m.sessions[token]
	delete(m.sessions, token)
	routed := ok && s.routed
	if ok {
		s.routed = false
		delete(m.accounts[s.account], token)
		if len(m.accounts[s.account]) == 0 {
			delete(m.accounts, s.account)
		}
	}
	m.mu.Unlock()
	if routed {
		m.release(token)
	}
	if ok && s.active != nil {
		s.active.conn.Close(websocket.StatusPolicyViolation, "session revoked")
	}
}

func (m *Manager) Presence() map[string]bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	connected := make(map[string]bool, len(m.sessions))
	for _, s := range m.sessions {
		if s.active != nil {
			connected[s.account] = connected[s.account] || s.room != nil
		}
	}
	return connected
}

func (m *Manager) Connect(ctx context.Context, conn *gateway.Conn, hello *pb.ClientHello) (gateway.Session, error) {
	if !m.startAllowed(ctx, hello) {
		return nil, gateway.ErrIntegrityFailed
	}
	c, err := m.connect(conn, hello)
	if err != nil {
		return nil, err
	}
	if m.clocks != nil {
		m.clocks.Observe(ctx, c.session.account, hello.GetClientTimeUnixMs())
	}
	return c, nil
}

func (m *Manager) connect(conn *gateway.Conn, hello *pb.ClientHello) (*connection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[hello.GetSessionToken()]
	if !ok && m.auth != nil {
		account, id, err := m.auth.Authenticate(hello.GetSessionToken())
		if err != nil || id == "" || account == "" {
			return nil, gateway.ErrUnauthenticated
		}
		s, ok = m.ensure(id, account), true
	}
	if !ok {
		return nil, gateway.ErrUnauthenticated
	}
	if previous := s.active; previous != nil {
		previous.conn.Close(websocket.StatusPolicyViolation, "session replaced")
	}
	s.generation++
	c := &connection{manager: m, session: s, conn: conn}
	s.active = c
	return c, nil
}

type connection struct {
	manager *Manager
	session *Session
	conn    *gateway.Conn
	mu      sync.Mutex
	seat    *gateway.RoomSession
}

func (c *connection) Acked() uint64 {
	c.manager.mu.Lock()
	defer c.manager.mu.Unlock()
	return c.session.acked
}

func (c *connection) InMatch() bool {
	c.manager.mu.Lock()
	defer c.manager.mu.Unlock()
	return c.session.room != nil
}

func (c *connection) Loading(progress int, ready bool) {
	c.mu.Lock()
	seat := c.seat
	c.mu.Unlock()
	if seat != nil {
		seat.Loading(progress, ready)
	}
}

func (c *connection) Emoji(item string) {
	c.mu.Lock()
	seat := c.seat
	c.mu.Unlock()
	if seat != nil {
		seat.Emoji(item)
	}
}

func (c *connection) QuickMessage(key string, teamOnly bool) {
	c.mu.Lock()
	seat := c.seat
	c.mu.Unlock()
	if seat != nil {
		seat.QuickMessage(key, teamOnly)
	}
}

func (c *connection) Attach() {
	c.manager.mu.Lock()
	defer c.manager.mu.Unlock()
	if c.session.room != nil {
		c.attach(c.session)
	}
}

func (c *connection) attach(s *Session) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if s.joined {
		c.seat = gateway.ResumeRoom(s.room, s.seat, c.conn, s.acked)
	} else {
		c.seat = gateway.JoinRoom(s.room, s.seat, c.conn)
		s.joined = true
	}
	c.seat.Attach()
}

func (c *connection) Intent(sequence uint64, intent replay.Intent) {
	c.mu.Lock()
	seat := c.seat
	c.mu.Unlock()
	if seat == nil {
		c.conn.Send(gateway.ErrorFrame(pb.ErrorCode_ERROR_CODE_ILLEGAL_INTENT, sequence, c.Acked()))
		return
	}
	seat.Intent(sequence, intent)
}

func (c *connection) Close() {
	c.mu.Lock()
	seat := c.seat
	c.mu.Unlock()
	if seat != nil {
		seat.Close()
	}
	c.manager.mu.Lock()
	defer c.manager.mu.Unlock()
	s := c.session
	if s.active != c {
		return
	}
	s.active = nil
	go c.leaveOnClose()
	if seat != nil {
		s.acked = seat.Acked()
	}
	if s.room != nil {
		go c.manager.expire(s, s.generation, c.manager.clock.After(c.manager.config.ReconnectGrace))
	}
}

func (m *Manager) Connected(token string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[token]
	return ok && s.active != nil
}

func (m *Manager) expire(s *Session, generation uint64, due <-chan time.Time) {
	<-due
	m.mu.Lock()
	defer m.mu.Unlock()
	if s.generation != generation || s.active != nil || s.room == nil {
		return
	}
	s.room.Leave(s.seat)
	if s.routed {
		s.routed = false
		m.release(s.token)
	}
}

func (m *Manager) startAllowed(ctx context.Context, hello *pb.ClientHello) bool {
	if m.checker == nil || m.auth == nil {
		return true
	}
	account, id, err := m.auth.Authenticate(hello.GetSessionToken())
	if err != nil || m.known(id) {
		return true
	}
	return m.checker.SessionStart(ctx, account, id, hello.GetIntegrity())
}

func (m *Manager) known(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.sessions[id]
	return ok
}

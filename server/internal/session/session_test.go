package session

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"google.golang.org/protobuf/proto"

	"github.com/Alievs-corp/hypedomino/server/internal/bot"
	"github.com/Alievs-corp/hypedomino/server/internal/gateway"
	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
	"github.com/Alievs-corp/hypedomino/server/internal/room"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
	"github.com/Alievs-corp/hypedomino/server/internal/secrand"
)

type timer struct {
	due time.Time
	ch  chan time.Time
}

type fakeClock struct {
	mu     sync.Mutex
	now    time.Time
	timers []timer
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) After(d time.Duration) <-chan time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := timer{due: c.now.Add(d), ch: make(chan time.Time, 1)}
	c.timers = append(c.timers, t)
	return t.ch
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	kept := c.timers[:0]
	for _, t := range c.timers {
		if t.due.After(c.now) {
			kept = append(kept, t)
			continue
		}
		t.ch <- c.now
	}
	c.timers = kept
	c.mu.Unlock()
}

type table struct {
	t       *testing.T
	clock   *fakeClock
	manager *Manager
	room    *room.Room
	server  *httptest.Server
	tokens  map[rules.Seat]string
}

func newTable(t *testing.T, mode rules.Mode) *table {
	t.Helper()
	clock := &fakeClock{now: time.UnixMilli(1761480000000)}
	manager := New(DefaultConfig(), clock, secrand.Source{})
	r := room.New(room.Options{Mode: mode, Config: rules.DefaultConfig(), Bot: bot.DefaultConfig(), BotLevel: bot.Hard, Seed: 11, Clock: clock})
	ctx, cancel := context.WithCancel(context.Background())
	go r.Run(ctx)
	t.Cleanup(func() { cancel(); <-r.Done() })
	config := gateway.DefaultConfig()
	config.IntentsPerSecond = 1000
	config.IntentBurst = 1000
	config.CloseTimeout = 500 * time.Millisecond
	server := httptest.NewServer(gateway.New(Options(config, manager)))
	t.Cleanup(server.Close)
	tab := &table{t: t, clock: clock, manager: manager, room: r, server: server, tokens: map[rules.Seat]string{}}
	for _, seat := range mode.Seats() {
		token := manager.Create("account-" + seat.String())
		if err := manager.Seat(token, r, seat); err != nil {
			t.Fatal(err)
		}
		tab.tokens[seat] = token
	}
	return tab
}

func Options(config gateway.Config, manager *Manager) gateway.Options {
	return gateway.Options{Config: config, Handler: manager}
}

type player struct {
	t        *testing.T
	seat     pb.Seat
	socket   *websocket.Conn
	ctx      context.Context
	sequence uint64
	acked    uint64
	hello    *pb.Envelope
	last     *pb.MatchView
}

func (tab *table) connect(token string, expectAccepted bool) *player {
	tab.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	tab.t.Cleanup(cancel)
	socket, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(tab.server.URL, "http"), nil)
	if err != nil {
		tab.t.Fatal(err)
	}
	tab.t.Cleanup(func() { socket.CloseNow() })
	p := &player{t: tab.t, socket: socket, ctx: ctx}
	p.write(&pb.Envelope{
		ProtocolVersion: gateway.ProtocolVersion,
		Sequence:        1,
		Payload:         &pb.Envelope_ClientHello{ClientHello: &pb.ClientHello{AppVersion: "1.0.0", SessionToken: token}},
	})
	p.hello = p.read()
	if accepted := p.hello.GetServerHello().GetStatus() == pb.HelloStatus_HELLO_STATUS_ACCEPTED; accepted != expectAccepted {
		tab.t.Fatalf("hello answered %v", p.hello)
	}
	p.acked = p.hello.GetAckedSequence()
	p.sequence = p.acked
	if expectAccepted {
		p.write(&pb.Envelope{ProtocolVersion: gateway.ProtocolVersion, Payload: &pb.Envelope_LoadingIntent{LoadingIntent: &pb.LoadingIntent{Progress: 100, Ready: true}}})
	}
	return p
}

func (tab *table) join(seat rules.Seat) *player {
	tab.t.Helper()
	p := tab.connect(tab.tokens[seat], true)
	p.seat = pb.Seat(seat) + 1
	return p
}

func (p *player) write(frame *pb.Envelope) {
	p.t.Helper()
	data, err := proto.Marshal(frame)
	if err != nil {
		p.t.Fatal(err)
	}
	if err := p.socket.Write(p.ctx, websocket.MessageBinary, data); err != nil {
		p.t.Fatal(err)
	}
}

func (p *player) read() *pb.Envelope {
	p.t.Helper()
	frame, err := p.tryRead()
	if err != nil {
		p.t.Fatalf("%v: read: %v", p.seat, err)
	}
	return frame
}

func (p *player) tryRead() (*pb.Envelope, error) {
	_, data, err := p.socket.Read(p.ctx)
	if err != nil {
		return nil, err
	}
	frame := &pb.Envelope{}
	if err := proto.Unmarshal(data, frame); err != nil {
		return nil, err
	}
	return frame, nil
}

func (p *player) closeReason() error {
	for {
		if _, err := p.tryRead(); err != nil {
			return err
		}
	}
}

func (p *player) view() *pb.MatchView {
	p.t.Helper()
	for {
		frame := p.read()
		if frame.GetError() != nil {
			p.t.Fatalf("%v: rejected %v", p.seat, frame.GetError())
		}
		if view := frame.GetMatchView(); view != nil {
			p.acked = frame.GetAckedSequence()
			p.last = view
			return view
		}
	}
}

func (p *player) intent(payload *pb.Envelope) {
	p.sequence++
	payload.ProtocolVersion = gateway.ProtocolVersion
	payload.Sequence = p.sequence
	p.write(payload)
}

func (p *player) act(view *pb.MatchView) {
	switch {
	case view.GetState() == pb.MatchState_MATCH_STATE_OPENING && view.GetForcedOpening() != nil:
		p.intent(&pb.Envelope{Payload: &pb.Envelope_PlaceTileIntent{PlaceTileIntent: &pb.PlaceTileIntent{Tile: view.GetForcedOpening().GetTile()}}})
	case view.GetState() == pb.MatchState_MATCH_STATE_OPENING:
		p.intent(&pb.Envelope{Payload: &pb.Envelope_PlaceTileIntent{PlaceTileIntent: &pb.PlaceTileIntent{Tile: view.GetHand()[0]}}})
	case len(view.GetLegalMoves()) > 0:
		move := view.GetLegalMoves()[0]
		p.intent(&pb.Envelope{Payload: &pb.Envelope_PlaceTileIntent{PlaceTileIntent: &pb.PlaceTileIntent{Tile: move.GetTile(), End: move.GetEnd()}}})
	case view.GetTurnPlacements() > 0:
		p.intent(&pb.Envelope{Payload: &pb.Envelope_EndTurnIntent{EndTurnIntent: &pb.EndTurnIntent{}}})
	case view.GetMode() == pb.Mode_MODE_1V1 && view.GetBoneyardSize() > 1:
		p.intent(&pb.Envelope{Payload: &pb.Envelope_DrawIntent{DrawIntent: &pb.DrawIntent{}}})
	default:
		p.intent(&pb.Envelope{Payload: &pb.Envelope_PassIntent{PassIntent: &pb.PassIntent{}}})
	}
}

func (p *player) untilMyTurn() *pb.MatchView {
	p.t.Helper()
	for {
		view := p.view()
		if !pregame(view) && view.GetCurrent() == p.seat {
			return view
		}
	}
}

func (p *player) playUntil(stop func(*pb.MatchView) bool) *pb.MatchView {
	p.t.Helper()
	for {
		frame := p.read()
		view := frame.GetMatchView()
		if view == nil {
			continue
		}
		p.acked = frame.GetAckedSequence()
		p.last = view
		if stop(view) {
			return view
		}
		if view.GetCurrent() == p.seat {
			p.act(view)
		}
	}
}

func finished(view *pb.MatchView) bool {
	return view.GetState() == pb.MatchState_MATCH_STATE_FINISHED
}

func controllerOf(view *pb.MatchView, seat pb.Seat) pb.Controller {
	for _, s := range view.GetSeats() {
		if s.GetSeat() == seat {
			return s.GetController()
		}
	}
	return pb.Controller_CONTROLLER_UNSPECIFIED
}

func TestUnknownTokensAreRefused(t *testing.T) {
	tab := newTable(t, rules.Mode1v1)
	p := tab.connect("not-a-token", false)
	if p.hello.GetError().GetCode() != pb.ErrorCode_ERROR_CODE_UNAUTHENTICATED {
		t.Fatalf("unknown token answered %v", p.hello)
	}
	if err := tab.manager.Seat("not-a-token", tab.room, rules.SeatX); !errors.Is(err, ErrUnknownSession) {
		t.Fatalf("seating an unknown token: %v", err)
	}
	if err := tab.manager.Seat(tab.tokens[rules.SeatX], tab.room, rules.SeatY); !errors.Is(err, ErrAlreadySeated) {
		t.Fatalf("seating twice: %v", err)
	}
}

func TestTokensAreUniqueAndOpaque(t *testing.T) {
	manager := New(DefaultConfig(), &fakeClock{}, secrand.Source{})
	seen := map[string]bool{}
	for index := 0; index < 1000; index++ {
		token := manager.Create("account")
		if len(token) != 32 || seen[token] {
			t.Fatalf("token %q repeated or malformed", token)
		}
		seen[token] = true
	}
}

func TestDisconnectWithinTheGraceWindowResumesTheMatch(t *testing.T) {
	tab := newTable(t, rules.Mode1v1)
	x, y := tab.firstToAct()
	view := x.untilMyTurn()
	x.act(view)
	after := x.view()
	if after.GetState() != pb.MatchState_MATCH_STATE_PLAYING || x.acked != x.sequence {
		t.Fatalf("after the first intent: state %v acked %d sent %d", after.GetState(), x.acked, x.sequence)
	}
	x.socket.CloseNow()
	tab.awayFor(x, 30*time.Second)
	back := tab.rejoin(x)
	if back.hello.GetAckedSequence() != x.sequence {
		t.Fatalf("resume acked %d, want %d", back.hello.GetAckedSequence(), x.sequence)
	}
	back.sequence = x.sequence
	resumed := back.view()
	if resumed.GetState() != pb.MatchState_MATCH_STATE_PLAYING || len(resumed.GetHand()) != len(after.GetHand()) || resumed.GetHandsPlayed() != after.GetHandsPlayed() {
		t.Fatalf("resumed view %v\nbefore %v", resumed, after)
	}
	if controllerOf(resumed, back.seat) != pb.Controller_CONTROLLER_PLAYER {
		t.Fatal("a resumed seat is not player controlled")
	}
	if resumed.GetCurrent() == back.seat {
		back.act(resumed)
	}
	done := make(chan *pb.MatchView, 2)
	for _, p := range []*player{back, y} {
		go func() { done <- p.playUntil(finished) }()
	}
	for range 2 {
		select {
		case final := <-done:
			if final.GetWinner() == pb.Team_TEAM_UNSPECIFIED {
				t.Fatalf("final view without a winner: %v", final)
			}
		case <-time.After(30 * time.Second):
			t.Fatal("match did not finish after the resume")
		}
	}
}

func TestStayingAwayPastTheGraceHandsTheSeatToTheBotUntilReturn(t *testing.T) {
	tab := newTable(t, rules.Mode1v1)
	x, y := tab.firstToAct()
	x.untilMyTurn()
	x.socket.CloseNow()
	tab.awayFor(x, DefaultConfig().ReconnectGrace)
	view := y.playUntil(func(v *pb.MatchView) bool { return controllerOf(v, x.seat) == pb.Controller_CONTROLLER_BOT })
	if view.GetState() == pb.MatchState_MATCH_STATE_FINISHED {
		t.Fatal("match ended before the bot took over")
	}
	back := tab.rejoin(x)
	resumed := back.view()
	if controllerOf(resumed, x.seat) != pb.Controller_CONTROLLER_PLAYER {
		t.Fatalf("a returning leaver did not take the seat back: %v", resumed)
	}
}

func TestASecondDeviceReplacesTheFirstWithoutStartingTheGrace(t *testing.T) {
	tab := newTable(t, rules.Mode1v1)
	first := tab.join(rules.SeatX)
	first.awaitReady()
	second := tab.join(rules.SeatX)
	err := first.closeReason()
	if websocket.CloseStatus(err) != websocket.StatusPolicyViolation {
		t.Fatalf("first device saw %v", err)
	}
	if view := second.view(); view.GetSeat() != pb.Seat_SEAT_X || controllerOf(view, pb.Seat_SEAT_X) != pb.Controller_CONTROLLER_PLAYER {
		t.Fatalf("second device view %v", view)
	}
	y := tab.join(rules.SeatY)
	y.awaitReady()
	tab.clock.advance(2 * DefaultConfig().ReconnectGrace)
	view := second.untilMyTurn()
	if controllerOf(view, pb.Seat_SEAT_X) != pb.Controller_CONTROLLER_PLAYER {
		t.Fatal("the replaced device's disconnect started the grace timer")
	}
	second.act(view)
	if next := second.view(); next.GetCurrent() == pb.Seat_SEAT_X && next.GetTurnPlacements() == 0 {
		t.Fatalf("intent from the second device was not applied: %v", next)
	}
}

func TestIntentsBeforeASeatAreIllegalAndRevokeCloses(t *testing.T) {
	tab := newTable(t, rules.Mode1v1)
	token := tab.manager.Create("lobby")
	p := tab.connect(token, true)
	p.intent(&pb.Envelope{Payload: &pb.Envelope_PassIntent{PassIntent: &pb.PassIntent{}}})
	frame := p.read()
	if frame.GetError().GetCode() != pb.ErrorCode_ERROR_CODE_ILLEGAL_INTENT || frame.GetError().GetRejectedSequence() != p.sequence {
		t.Fatalf("intent without a room answered %v", frame)
	}
	tab.manager.Revoke(token)
	if _, err := p.tryRead(); websocket.CloseStatus(err) != websocket.StatusPolicyViolation {
		t.Fatalf("after revoke: %v", err)
	}
	if again := tab.connect(token, false); again.hello.GetError().GetCode() != pb.ErrorCode_ERROR_CODE_UNAUTHENTICATED {
		t.Fatalf("revoked token answered %v", again.hello)
	}
}

func TestSeatingAConnectedSessionJoinsAtOnce(t *testing.T) {
	tab := newTable(t, rules.Mode2v2)
	token := tab.manager.Create("late")
	p := tab.connect(token, true)
	other := room.New(room.Options{Mode: rules.Mode1v1, Config: rules.DefaultConfig(), Bot: bot.DefaultConfig(), Seed: 2, Clock: tab.clock})
	ctx, cancel := context.WithCancel(context.Background())
	go other.Run(ctx)
	t.Cleanup(func() { cancel(); <-other.Done() })
	if err := tab.manager.Seat(token, other, rules.SeatY); err != nil {
		t.Fatal(err)
	}
	if view := p.view(); view.GetState() != pb.MatchState_MATCH_STATE_LOADING || view.GetSeat() != pb.Seat_SEAT_Y || view.GetMode() != pb.Mode_MODE_1V1 {
		t.Fatalf("view after seating %v", view)
	}
}

func (tab *table) firstToAct() (*player, *player) {
	tab.t.Helper()
	x, y := tab.join(rules.SeatX), tab.join(rules.SeatY)
	for {
		view := x.view()
		if pregame(view) {
			continue
		}
		if view.GetCurrent() == x.seat {
			return x, y
		}
		return y, x
	}
}

func (tab *table) rejoin(p *player) *player {
	tab.t.Helper()
	back := tab.join(rules.Seat(p.seat - 1))
	return back
}

func (tab *table) awayFor(p *player, d time.Duration) {
	tab.t.Helper()
	token := tab.tokens[rules.Seat(p.seat-1)]
	deadline := time.Now().Add(10 * time.Second)
	for tab.manager.Connected(token) {
		if time.Now().After(deadline) {
			tab.t.Fatalf("%v: the server did not notice the disconnect", p.seat)
		}
		time.Sleep(5 * time.Millisecond)
	}
	tab.clock.advance(d)
}

func pregame(view *pb.MatchView) bool {
	return view.GetState() == pb.MatchState_MATCH_STATE_WAITING || view.GetState() == pb.MatchState_MATCH_STATE_LOADING
}

func (p *player) awaitReady() *pb.MatchView {
	p.t.Helper()
	for {
		view := p.view()
		for _, seat := range view.GetSeats() {
			if seat.GetSeat() == p.seat && seat.GetReady() {
				return view
			}
		}
	}
}

type fakeAuthenticator map[string][2]string

func (a fakeAuthenticator) Authenticate(token string) (string, string, error) {
	identity, ok := a[token]
	if !ok {
		return "", "", errors.New("bad token")
	}
	return identity[0], identity[1], nil
}

func TestAccessTokensOpenSessionsKeyedByTheirSessionIdAndLogoutAllClosesThem(t *testing.T) {
	tab := newTable(t, rules.Mode1v1)
	tab.manager.Authenticate(fakeAuthenticator{"jwt-1": {"ayla", "sid-1"}, "jwt-2": {"ayla", "sid-1"}, "jwt-3": {"ayla", "sid-2"}})
	tab.connect("jwt-bad", false)
	first := tab.connect("jwt-1", true)
	if tab.manager.InMatch("ayla") || !tab.manager.Connected("sid-1") {
		t.Fatal("the authenticated session was not created under its session id")
	}
	if err := tab.manager.Seat("sid-1", tab.room, rules.SeatX); err != nil {
		t.Fatal(err)
	}
	first.view()
	first.socket.CloseNow()
	for tab.manager.Connected("sid-1") {
		time.Sleep(time.Millisecond)
	}
	refreshed := tab.connect("jwt-2", true)
	if view := refreshed.view(); view.GetSeat() != pb.Seat_SEAT_X {
		t.Fatalf("a refreshed access token did not resume the seat: %v", view)
	}
	other := tab.connect("jwt-3", true)
	if n := tab.manager.RevokeAccount("ayla"); n != 2 {
		t.Fatalf("revoked %d sessions", n)
	}
	for _, p := range []*player{refreshed, other} {
		if err := p.closeReason(); websocket.CloseStatus(err) != websocket.StatusPolicyViolation {
			t.Fatalf("after logout-all the device saw %v", err)
		}
	}
	if tab.manager.InMatch("ayla") || tab.manager.RevokeAccount("ayla") != 0 {
		t.Fatal("sessions survived logout-all")
	}
}

package gateway

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"google.golang.org/protobuf/proto"

	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
	"github.com/Alievs-corp/hypedomino/server/internal/replay"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

type received struct {
	sequence uint64
	intent   replay.Intent
}

type loadingReport struct {
	progress int
	ready    bool
}

type recordingSession struct {
	acked    uint64
	conn     *Conn
	mu       sync.Mutex
	intents  []received
	loading  []loadingReport
	emojis   []string
	quick    []string
	inMatch  bool
	attached bool
	replays  int
	closed   chan struct{}
}

func (s *recordingSession) Acked() uint64 { return s.acked }

func (s *recordingSession) InMatch() bool { return s.inMatch }

func (s *recordingSession) Loading(progress int, ready bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loading = append(s.loading, loadingReport{progress: progress, ready: ready})
}

func (s *recordingSession) Emoji(item string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.emojis = append(s.emojis, item)
}

func (s *recordingSession) QuickMessage(key string, teamOnly bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.quick = append(s.quick, fmt.Sprint(key, " ", teamOnly))
}

func (s *recordingSession) loadingReports() []loadingReport {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]loadingReport(nil), s.loading...)
}

func (s *recordingSession) Attach() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attached = true
}

func (s *recordingSession) isAttached() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.attached
}

func (s *recordingSession) awaitAttached(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !s.isAttached() {
		if time.Now().After(deadline) {
			t.Fatal("session was not attached after the hello")
		}
		time.Sleep(time.Millisecond)
	}
}

func (s *recordingSession) Intent(sequence uint64, intent replay.Intent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.intents = append(s.intents, received{sequence, intent})
}

func (s *recordingSession) Close() { close(s.closed) }

func (s *recordingSession) Replayed() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.replays++
}

func (s *recordingSession) replayCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.replays
}

func (s *recordingSession) received() []received {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]received(nil), s.intents...)
}

func (s *recordingSession) awaitIntents(t *testing.T, count int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for len(s.received()) < count {
		if time.Now().After(deadline) {
			t.Fatalf("session received %d intents, want %d", len(s.received()), count)
		}
		time.Sleep(time.Millisecond)
	}
}

type harness struct {
	t        *testing.T
	server   *httptest.Server
	clock    *fakeClock
	config   Config
	mu       sync.Mutex
	sessions map[string]*recordingSession
}

func newHarness(t *testing.T, config Config) *harness {
	t.Helper()
	h := &harness{t: t, clock: &fakeClock{now: time.UnixMilli(1761480000000)}, config: config, sessions: map[string]*recordingSession{}}
	gateway := New(Options{Config: config, Handler: HandlerFunc(h.connect), Clock: h.clock, Strings: activeStrings})
	h.server = httptest.NewServer(gateway)
	t.Cleanup(h.server.Close)
	return h
}

func (h *harness) connect(ctx context.Context, conn *Conn, hello *pb.ClientHello) (Session, error) {
	if !strings.HasPrefix(hello.GetSessionToken(), "ok") {
		return nil, ErrUnauthenticated
	}
	if hello.GetSessionToken() == "ok-broken" {
		return nil, errors.New("storage down")
	}
	if hello.GetSessionToken() == "ok-tampered" {
		return nil, ErrIntegrityFailed
	}
	session := &recordingSession{acked: hello.GetLastAckedSequence(), conn: conn, closed: make(chan struct{}), inMatch: hello.GetSessionToken() == "ok-playing"}
	h.mu.Lock()
	h.sessions[hello.GetSessionToken()] = session
	h.mu.Unlock()
	return session, nil
}

func (h *harness) session(token string) *recordingSession {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.sessions[token]
}

type client struct {
	t      *testing.T
	socket *websocket.Conn
	ctx    context.Context
	nonce  uint64
}

func (h *harness) dial() *client {
	h.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	h.t.Cleanup(cancel)
	socket, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(h.server.URL, "http"), &websocket.DialOptions{CompressionMode: websocket.CompressionNoContextTakeover})
	if err != nil {
		h.t.Fatal(err)
	}
	h.t.Cleanup(func() { socket.CloseNow() })
	return &client{t: h.t, socket: socket, ctx: ctx}
}

func (c *client) send(frame *pb.Envelope) {
	c.t.Helper()
	if frame.GetClientHello() == nil && frame.SessionNonce == 0 {
		frame.SessionNonce = c.nonce
	}
	data, err := proto.Marshal(frame)
	if err != nil {
		c.t.Fatal(err)
	}
	c.sendBytes(data)
}

func (c *client) sendBytes(data []byte) {
	c.t.Helper()
	if err := c.socket.Write(c.ctx, websocket.MessageBinary, data); err != nil {
		c.t.Fatal(err)
	}
}

func (c *client) recv() *pb.Envelope {
	c.t.Helper()
	frame, err := c.tryRecv()
	if err != nil {
		c.t.Fatalf("receive: %v", err)
	}
	return frame
}

func (c *client) tryRecv() (*pb.Envelope, error) {
	kind, data, err := c.socket.Read(c.ctx)
	if err != nil {
		return nil, err
	}
	if kind != websocket.MessageBinary {
		return nil, errors.New("text frame")
	}
	frame := &pb.Envelope{}
	if err := proto.Unmarshal(data, frame); err != nil {
		return nil, err
	}
	return frame, nil
}

func (c *client) expectClose(status websocket.StatusCode) {
	c.t.Helper()
	_, err := c.tryRecv()
	if err == nil {
		c.t.Fatal("expected the connection to be closed")
	}
	if got := websocket.CloseStatus(err); got != status {
		c.t.Fatalf("close status %v, want %v (%v)", got, status, err)
	}
}

func hello(token, version string, sequence uint64) *pb.Envelope {
	return &pb.Envelope{
		ProtocolVersion: ProtocolVersion,
		Sequence:        sequence,
		Payload: &pb.Envelope_ClientHello{ClientHello: &pb.ClientHello{
			AppVersion:   version,
			Platform:     pb.Platform_PLATFORM_ANDROID,
			SessionToken: token,
		}},
	}
}

func (c *client) handshake(token string) *pb.ServerHello {
	c.t.Helper()
	c.send(hello(token, "1.2.3", 1))
	frame := c.recv()
	reply := frame.GetServerHello()
	if reply == nil || reply.GetStatus() != pb.HelloStatus_HELLO_STATUS_ACCEPTED {
		c.t.Fatalf("handshake answered %v", frame)
	}
	c.nonce = reply.GetSessionNonce()
	return reply
}

func intentFrame(sequence uint64, frame *pb.Envelope) *pb.Envelope {
	frame.ProtocolVersion = ProtocolVersion
	frame.Sequence = sequence
	return frame
}

func place(sequence uint64, low, high uint32, end pb.BoardEnd) *pb.Envelope {
	return intentFrame(sequence, &pb.Envelope{Payload: &pb.Envelope_PlaceTileIntent{PlaceTileIntent: &pb.PlaceTileIntent{Tile: &pb.Tile{Low: low, High: high}, End: end}}})
}

func activeStrings() Strings {
	return Strings{
		Bundles: []*pb.StringBundle{{Language: "en", Version: 4, Hash: "e4"}},
		Chains:  []*pb.FallbackChain{{Language: "en", Languages: []string{"en"}}},
	}
}

func testConfig() Config {
	config := DefaultConfig()
	config.CatalogVersion = "2026.10.26-1"
	config.ContentBaseURL = "https://content.example.com/staging"
	config.HelloTimeout = 200 * time.Millisecond
	config.CloseTimeout = 500 * time.Millisecond
	return config
}

func TestHelloIsAcceptedWithServerTimeAndAckedSequence(t *testing.T) {
	h := newHarness(t, testConfig())
	c := h.dial()
	c.send(&pb.Envelope{
		ProtocolVersion: ProtocolVersion,
		Sequence:        42,
		Payload: &pb.Envelope_ClientHello{ClientHello: &pb.ClientHello{
			AppVersion: "1.0.0", SessionToken: "ok-resume", LastAckedSequence: 41,
		}},
	})
	frame := c.recv()
	reply := frame.GetServerHello()
	if reply.GetStatus() != pb.HelloStatus_HELLO_STATUS_ACCEPTED || frame.GetAckedSequence() != 41 {
		t.Fatalf("accepted hello: %v", frame)
	}
	if reply.GetServerTimeUnixMs() != 1761480000000 || reply.GetCatalogVersion() != "2026.10.26-1" || reply.GetContentBaseUrl() != "https://content.example.com/staging" || reply.GetMinimumAppVersion() != "1.0.0" {
		t.Fatalf("server hello carries %v", reply)
	}
	if frame.GetProtocolVersion() != ProtocolVersion {
		t.Fatalf("protocol version %d", frame.GetProtocolVersion())
	}
	if want := activeStrings(); !proto.Equal(reply, &pb.ServerHello{
		Status: reply.GetStatus(), MinimumAppVersion: "1.0.0", ServerTimeUnixMs: 1761480000000, CatalogVersion: "2026.10.26-1", ContentBaseUrl: "https://content.example.com/staging",
		StringBundles: want.Bundles, FallbackChains: want.Chains, SessionNonce: reply.GetSessionNonce(),
	}) || reply.GetSessionNonce() == 0 {
		t.Fatalf("server hello carries the active string bundles: %v", reply)
	}
	h.session("ok-resume").awaitAttached(t)
}

func TestOldClientGetsUpdateRequiredAndIsClosed(t *testing.T) {
	config := testConfig()
	config.MinimumAppVersion = "1.4.0"
	h := newHarness(t, config)
	c := h.dial()
	c.send(hello("ok", "1.3.9", 1))
	frame := c.recv()
	if frame.GetServerHello().GetStatus() != pb.HelloStatus_HELLO_STATUS_UPDATE_REQUIRED || frame.GetServerHello().GetMinimumAppVersion() != "1.4.0" {
		t.Fatalf("old client answered %v", frame)
	}
	if len(frame.GetServerHello().GetStringBundles()) != 0 {
		t.Fatalf("an old client gets no string bundles: %v", frame)
	}
	c.expectClose(websocket.StatusNormalClosure)
	if h.session("ok") != nil {
		t.Fatal("an old client reached the handler")
	}
}

func TestUnsupportedProtocolIsRefused(t *testing.T) {
	h := newHarness(t, testConfig())
	c := h.dial()
	frame := hello("ok", "1.2.3", 5)
	frame.ProtocolVersion = 2
	c.send(frame)
	reply := c.recv()
	if reply.GetError().GetCode() != pb.ErrorCode_ERROR_CODE_UNSUPPORTED_PROTOCOL || reply.GetError().GetRejectedSequence() != 5 {
		t.Fatalf("protocol 2 answered %v", reply)
	}
	c.expectClose(websocket.StatusProtocolError)
}

func TestBadTokenIsUnauthenticated(t *testing.T) {
	h := newHarness(t, testConfig())
	c := h.dial()
	c.send(hello("expired", "1.2.3", 3))
	reply := c.recv()
	if reply.GetError().GetCode() != pb.ErrorCode_ERROR_CODE_UNAUTHENTICATED || reply.GetError().GetRejectedSequence() != 3 {
		t.Fatalf("bad token answered %v", reply)
	}
	c.expectClose(websocket.StatusPolicyViolation)
}

func TestHandlerFailureClosesWithInternalError(t *testing.T) {
	h := newHarness(t, testConfig())
	c := h.dial()
	c.send(hello("ok-broken", "1.2.3", 3))
	c.expectClose(websocket.StatusInternalError)
}

func TestFirstFrameMustBeTheHello(t *testing.T) {
	h := newHarness(t, testConfig())
	c := h.dial()
	c.send(place(1, 2, 3, pb.BoardEnd_BOARD_END_LEFT))
	c.expectClose(websocket.StatusPolicyViolation)
}

func TestSilentClientIsClosedAfterTheHelloTimeout(t *testing.T) {
	h := newHarness(t, testConfig())
	c := h.dial()
	c.expectClose(websocket.StatusPolicyViolation)
}

func TestOversizedFrameIsClosed(t *testing.T) {
	config := testConfig()
	config.MaxFrameBytes = 64
	h := newHarness(t, config)
	c := h.dial()
	c.handshake("ok")
	c.sendBytes(make([]byte, 65))
	c.expectClose(websocket.StatusMessageTooBig)
}

func TestMalformedAndTextFramesAreClosed(t *testing.T) {
	h := newHarness(t, testConfig())
	c := h.dial()
	c.handshake("ok")
	c.sendBytes([]byte{0xff, 0xff, 0xff})
	c.expectClose(websocket.StatusPolicyViolation)

	c = h.dial()
	c.handshake("ok")
	if err := c.socket.Write(c.ctx, websocket.MessageText, []byte("{}")); err != nil {
		t.Fatal(err)
	}
	c.expectClose(websocket.StatusUnsupportedData)
}

func TestIntentsReachTheSessionWithTheirSequence(t *testing.T) {
	h := newHarness(t, testConfig())
	c := h.dial()
	c.handshake("ok")
	c.send(place(1, 2, 3, pb.BoardEnd_BOARD_END_UNSPECIFIED))
	c.send(place(2, 3, 6, pb.BoardEnd_BOARD_END_RIGHT))
	c.send(intentFrame(3, &pb.Envelope{Payload: &pb.Envelope_EndTurnIntent{EndTurnIntent: &pb.EndTurnIntent{}}}))
	c.send(intentFrame(4, &pb.Envelope{Payload: &pb.Envelope_DrawIntent{DrawIntent: &pb.DrawIntent{}}}))
	c.send(intentFrame(5, &pb.Envelope{Payload: &pb.Envelope_PassIntent{PassIntent: &pb.PassIntent{}}}))
	c.send(place(6, 7, 3, pb.BoardEnd_BOARD_END_LEFT))
	reply := c.recv()
	if reply.GetError().GetCode() != pb.ErrorCode_ERROR_CODE_ILLEGAL_INTENT || reply.GetError().GetRejectedSequence() != 6 {
		t.Fatalf("pip 7 answered %v", reply)
	}
	c.send(place(7, 1, 3, pb.BoardEnd(9)))
	if reply := c.recv(); reply.GetError().GetRejectedSequence() != 7 {
		t.Fatalf("end 9 answered %v", reply)
	}
	c.send(intentFrame(8, &pb.Envelope{Payload: &pb.Envelope_ServerHello{ServerHello: &pb.ServerHello{}}}))
	if reply := c.recv(); reply.GetError().GetCode() != pb.ErrorCode_ERROR_CODE_ILLEGAL_INTENT || reply.GetError().GetRejectedSequence() != 8 {
		t.Fatalf("server payload from the client answered %v", reply)
	}
	want := []received{
		{1, replay.Intent{Kind: replay.KindOpen, Tile: rules.MustTile(2, 3)}},
		{2, replay.Intent{Kind: replay.KindPlace, Tile: rules.MustTile(3, 6), End: rules.EndRight}},
		{3, replay.Intent{Kind: replay.KindEndTurn}},
		{4, replay.Intent{Kind: replay.KindDraw}},
		{5, replay.Intent{Kind: replay.KindPass}},
	}
	got := h.session("ok").received()
	if len(got) != len(want) {
		t.Fatalf("session received %+v", got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("intent %d: got %+v want %+v", index, got[index], want[index])
		}
	}
}

func TestClosingTheClientClosesTheSession(t *testing.T) {
	h := newHarness(t, testConfig())
	c := h.dial()
	c.handshake("ok")
	c.socket.Close(websocket.StatusNormalClosure, "bye")
	select {
	case <-h.session("ok").closed:
	case <-time.After(5 * time.Second):
		t.Fatal("session was not closed")
	}
}

func TestServerFramesReachTheClient(t *testing.T) {
	h := newHarness(t, testConfig())
	c := h.dial()
	c.handshake("ok")
	session := h.session("ok")
	big := &pb.MatchView{State: pb.MatchState_MATCH_STATE_PLAYING}
	for pip := uint32(0); pip < 200; pip++ {
		big.Hand = append(big.Hand, &pb.Tile{Low: pip % 7, High: pip % 7})
	}
	if err := session.conn.Send(&pb.Envelope{AckedSequence: 9, Payload: &pb.Envelope_MatchView{MatchView: big}}); err != nil {
		t.Fatal(err)
	}
	frame := c.recv()
	if frame.GetAckedSequence() != 9 || len(frame.GetMatchView().GetHand()) != 200 || frame.GetProtocolVersion() != ProtocolVersion {
		t.Fatalf("view arrived as %v", frame)
	}
}

func TestRateLimitedIntentsAreAnsweredAndDropped(t *testing.T) {
	config := testConfig()
	config.IntentsPerSecond = 1
	config.IntentBurst = 2
	h := newHarness(t, config)
	c := h.dial()
	c.handshake("ok")
	for sequence := uint64(1); sequence <= 3; sequence++ {
		c.send(intentFrame(sequence, &pb.Envelope{Payload: &pb.Envelope_PassIntent{PassIntent: &pb.PassIntent{}}}))
	}
	reply := c.recv()
	if reply.GetError().GetCode() != pb.ErrorCode_ERROR_CODE_RATE_LIMITED || reply.GetError().GetRejectedSequence() != 3 {
		t.Fatalf("third intent answered %v", reply)
	}
	h.clock.advance(time.Second)
	c.send(intentFrame(4, &pb.Envelope{Payload: &pb.Envelope_PassIntent{PassIntent: &pb.PassIntent{}}}))
	c.send(intentFrame(5, &pb.Envelope{Payload: &pb.Envelope_PassIntent{PassIntent: &pb.PassIntent{}}}))
	if reply := c.recv(); reply.GetError().GetRejectedSequence() != 5 {
		t.Fatalf("after a second: %v", reply)
	}
	got := h.session("ok").received()
	if len(got) != 3 || got[2].sequence != 4 {
		t.Fatalf("session received %+v", got)
	}
}

func TestSlowConsumerIsDisconnected(t *testing.T) {
	config := testConfig()
	config.WriteQueue = 2
	accepted := make(chan *Conn, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		socket, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		conn := newConn(socket, Options{Config: config, Clock: systemClock{}})
		accepted <- conn
		<-conn.Done()
		conn.closeSocket()
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	socket, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer socket.CloseNow()
	conn := <-accepted
	for sequence := uint64(1); sequence <= 2; sequence++ {
		if err := conn.Send(ErrorFrame(pb.ErrorCode_ERROR_CODE_RATE_LIMITED, sequence, 0)); err != nil {
			t.Fatal(err)
		}
	}
	if err := conn.Send(ErrorFrame(pb.ErrorCode_ERROR_CODE_RATE_LIMITED, 3, 0)); !errors.Is(err, ErrSlowConsumer) {
		t.Fatalf("third frame: %v", err)
	}
	if err := conn.Send(ErrorFrame(pb.ErrorCode_ERROR_CODE_RATE_LIMITED, 4, 0)); !errors.Is(err, ErrClosed) {
		t.Fatalf("after closing: %v", err)
	}
	_, _, err = socket.Read(ctx)
	if websocket.CloseStatus(err) != websocket.StatusPolicyViolation {
		t.Fatalf("client saw %v", err)
	}
}

func TestVersionAtLeast(t *testing.T) {
	cases := []struct {
		version, minimum string
		want             bool
	}{
		{"1.0.0", "1.0.0", true},
		{"1.0.1", "1.0.0", true},
		{"1.10.0", "1.9.9", true},
		{"2.0.0", "1.99.99", true},
		{"0.9.9", "1.0.0", false},
		{"1.0.0", "1.0.1", false},
		{"v1.2.3", "1.2.3", true},
		{"1.2", "1.2.0", true},
		{"1.2", "1.2.1", false},
		{"", "1.0.0", false},
		{"beta", "1.0.0", false},
		{"1.2.3.4", "1.0.0", false},
		{"1.0.0", "", true},
		{"1.0.0", "latest", true},
	}
	for _, tc := range cases {
		if got := VersionAtLeast(tc.version, tc.minimum); got != tc.want {
			t.Errorf("VersionAtLeast(%q, %q) = %v, want %v", tc.version, tc.minimum, got, tc.want)
		}
	}
}

func TestAClientThatStopsAnsweringPingsIsClosed(t *testing.T) {
	config := testConfig()
	config.PingInterval = 50 * time.Millisecond
	config.PongTimeout = 100 * time.Millisecond
	h := newHarness(t, config)
	c := h.dial()
	c.handshake("ok")
	select {
	case <-h.session("ok").closed:
	case <-time.After(5 * time.Second):
		t.Fatal("a client that never reads was not disconnected")
	}
	c.expectClose(websocket.StatusGoingAway)

	alive := h.dial()
	alive.handshake("ok-alive")
	alive.socket.CloseRead(alive.ctx)
	time.Sleep(400 * time.Millisecond)
	alive.send(place(1, 2, 3, pb.BoardEnd_BOARD_END_UNSPECIFIED))
	alive.socket.Close(websocket.StatusNormalClosure, "")
	<-h.session("ok-alive").closed
	if got := h.session("ok-alive").received(); len(got) != 1 {
		t.Fatalf("a responsive client was dropped: %+v", got)
	}
}

func TestHelloTellsAResumingClientWhetherItIsInAMatch(t *testing.T) {
	h := newHarness(t, testConfig())
	idle, playing := h.dial(), h.dial()
	if idle.handshake("ok-idle").GetInMatch() {
		t.Fatal("an idle session was reported in a match")
	}
	if !playing.handshake("ok-playing").GetInMatch() {
		t.Fatal("a seated session was not reported in a match")
	}
}

func TestLoadingReportsEmojiAndQuickMessagesReachTheSessionWithoutASequence(t *testing.T) {
	h := newHarness(t, testConfig())
	c := h.dial()
	c.handshake("ok")
	c.send(intentFrame(0, &pb.Envelope{Payload: &pb.Envelope_LoadingIntent{LoadingIntent: &pb.LoadingIntent{Progress: 40}}}))
	c.send(intentFrame(0, &pb.Envelope{Payload: &pb.Envelope_EmojiIntent{EmojiIntent: &pb.EmojiIntent{ItemId: "emoji.laugh"}}}))
	c.send(intentFrame(0, &pb.Envelope{Payload: &pb.Envelope_QuickMessageIntent{QuickMessageIntent: &pb.QuickMessageIntent{Key: "quick.hello", TeamOnly: true}}}))
	c.send(intentFrame(0, &pb.Envelope{Payload: &pb.Envelope_LoadingIntent{LoadingIntent: &pb.LoadingIntent{Progress: 100, Ready: true}}}))
	c.send(place(1, 2, 3, pb.BoardEnd_BOARD_END_UNSPECIFIED))
	h.session("ok").awaitIntents(t, 1)
	want := []loadingReport{{40, false}, {100, true}}
	if got := h.session("ok").loadingReports(); !reflect.DeepEqual(got, want) {
		t.Fatalf("session received loading reports %+v", got)
	}
	h.session("ok").mu.Lock()
	emojis := h.session("ok").emojis
	quick := h.session("ok").quick
	h.session("ok").mu.Unlock()
	if !reflect.DeepEqual(emojis, []string{"emoji.laugh"}) {
		t.Fatalf("session received emoji %v", emojis)
	}
	if !reflect.DeepEqual(quick, []string{"quick.hello true"}) {
		t.Fatalf("session received quick messages %v", quick)
	}
	if got := h.session("ok").received(); len(got) != 1 || got[0].sequence != 1 {
		t.Fatalf("session received intents %+v", got)
	}
}

func TestFramesWithoutTheConnectionNonceAreRejectedAndCounted(t *testing.T) {
	h := newHarness(t, testConfig())
	first := h.dial()
	firstNonce := first.handshake("ok-a").GetSessionNonce()
	second := h.dial()
	secondNonce := second.handshake("ok-b").GetSessionNonce()
	if firstNonce == secondNonce {
		t.Fatalf("two connections share the nonce %d", firstNonce)
	}
	cases := []struct {
		name  string
		nonce uint64
	}{
		{"another connection", secondNonce},
		{"missing", 0},
		{"guessed", firstNonce + 1},
	}
	for i, c := range cases {
		frame := place(uint64(i+1), 0, 1, pb.BoardEnd_BOARD_END_LEFT)
		frame.SessionNonce = c.nonce
		if c.nonce == 0 {
			data, _ := proto.Marshal(frame)
			first.sendBytes(data)
		} else {
			first.send(frame)
		}
		reply := first.recv()
		if reply.GetError().GetCode() != pb.ErrorCode_ERROR_CODE_REPLAY_REJECTED || reply.GetError().GetRejectedSequence() != uint64(i+1) {
			t.Fatalf("%s: answered %v", c.name, reply)
		}
	}
	first.send(&pb.Envelope{SessionNonce: secondNonce, Payload: &pb.Envelope_LoadingIntent{LoadingIntent: &pb.LoadingIntent{Progress: 50}}})
	if reply := first.recv(); reply.GetError().GetCode() != pb.ErrorCode_ERROR_CODE_REPLAY_REJECTED {
		t.Fatalf("loading report with a foreign nonce answered %v", reply)
	}
	first.send(place(4, 0, 1, pb.BoardEnd_BOARD_END_LEFT))
	session := h.session("ok-a")
	session.awaitIntents(t, 1)
	if got := session.received(); got[0].sequence != 4 || session.replayCount() != 4 || len(session.loadingReports()) != 0 {
		t.Fatalf("session got %+v after %d replays", got, session.replayCount())
	}
	if h.session("ok-b").replayCount() != 0 {
		t.Fatal("the other session was counted")
	}
}

func TestTheHelloMayCarryAnIntegrityTokenLargerThanAnIntent(t *testing.T) {
	config := testConfig()
	config.MaxFrameBytes = 256
	config.MaxHelloBytes = 4096
	h := newHarness(t, config)
	withToken := func(token string, size int) *pb.Envelope {
		frame := hello(token, "1.2.3", 1)
		frame.GetClientHello().Integrity = &pb.IntegrityProof{Proof: &pb.IntegrityProof_PlayIntegrityToken{PlayIntegrityToken: strings.Repeat("t", size)}}
		return frame
	}
	c := h.dial()
	c.send(withToken("ok", 3000))
	if reply := c.recv().GetServerHello(); reply.GetStatus() != pb.HelloStatus_HELLO_STATUS_ACCEPTED {
		t.Fatalf("hello with a token answered %v", reply)
	}
	c.sendBytes(make([]byte, 257))
	c.expectClose(websocket.StatusMessageTooBig)
	big := h.dial()
	big.send(withToken("ok", 5000))
	big.expectClose(websocket.StatusMessageTooBig)
	tampered := h.dial()
	tampered.send(withToken("ok-tampered", 10))
	if code := tampered.recv().GetError().GetCode(); code != pb.ErrorCode_ERROR_CODE_INTEGRITY_FAILED {
		t.Fatalf("tampered device answered %v", code)
	}
	tampered.expectClose(websocket.StatusPolicyViolation)
}

func TestAQueueIntentToASessionWithoutMatchmakingIsRefusedAsUnavailable(t *testing.T) {
	h := newHarness(t, testConfig())
	c := h.dial()
	c.handshake("ok")
	c.send(intentFrame(0, &pb.Envelope{Payload: &pb.Envelope_JoinQueueIntent{JoinQueueIntent: &pb.JoinQueueIntent{Mode: pb.Mode_MODE_1V1, Queue: pb.QueueKind_QUEUE_KIND_CASUAL}}}))
	state := c.recv().GetQueueState()
	if state.GetStatus() != pb.QueueStatus_QUEUE_STATUS_REFUSED || state.GetRefusal() != pb.QueueRefusal_QUEUE_REFUSAL_UNAVAILABLE {
		t.Fatalf("queue state %v", state)
	}
}

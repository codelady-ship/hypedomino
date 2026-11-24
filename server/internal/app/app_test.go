package app

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/Alievs-corp/hypedomino/server/internal/bot"
	"github.com/Alievs-corp/hypedomino/server/internal/botclient"
	"github.com/Alievs-corp/hypedomino/server/internal/gateway"
	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
	"github.com/Alievs-corp/hypedomino/server/internal/room"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

type realClock struct{}

func (realClock) Now() time.Time                         { return time.Now() }
func (realClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

type seatHandler struct {
	room *room.Room
}

func (h *seatHandler) Connect(ctx context.Context, conn *gateway.Conn, hello *pb.ClientHello) (gateway.Session, error) {
	seat, err := rules.ParseSeat(hello.GetSessionToken())
	if err != nil || h.room == nil {
		return nil, gateway.ErrUnauthenticated
	}
	return gateway.JoinRoom(h.room, seat, conn), nil
}

type syncBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

type running struct {
	t       *testing.T
	app     *App
	handler *seatHandler
	logs    *syncBuffer
	url     string
	stop    context.CancelFunc
	done    chan error
	ended   bool
}

func start(t *testing.T, config Config, setups ...func(*App)) *running {
	t.Helper()
	config.ListenAddr = "127.0.0.1:0"
	config.Gateway.IntentsPerSecond = 1000
	config.Gateway.IntentBurst = 1000
	logs := &syncBuffer{}
	handler := &seatHandler{}
	a := New(config, NewLogger(logs, "info"), handler)
	for _, setup := range setups {
		setup(a)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- a.Run(ctx) }()
	var addr string
	select {
	case addr = <-a.Addr():
	case err := <-done:
		t.Fatal(err)
	}
	r := &running{t: t, app: a, handler: handler, logs: logs, url: "http://" + addr, stop: cancel, done: done}
	t.Cleanup(func() {
		cancel()
		if !r.ended {
			r.wait()
		}
	})
	return r
}

func (r *running) get(path string) (int, string) {
	r.t.Helper()
	resp, err := http.Get(r.url + path)
	if err != nil {
		r.t.Fatal(err)
	}
	defer resp.Body.Close()
	body := &bytes.Buffer{}
	body.ReadFrom(resp.Body)
	return resp.StatusCode, strings.TrimSpace(body.String())
}

func (r *running) openRoom(mode rules.Mode) *room.Room {
	r.t.Helper()
	rm, err := r.app.Rooms().Open(room.Options{Mode: mode, Config: rules.DefaultConfig(), Bot: bot.DefaultConfig(), BotLevel: bot.Hard, Seed: 9, Clock: realClock{}})
	if err != nil {
		r.t.Fatal(err)
	}
	r.handler.room = rm
	return rm
}

func (r *running) dial(seat rules.Seat) *botclient.Client {
	r.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	r.t.Cleanup(cancel)
	c, err := botclient.Dial(ctx, "ws"+strings.TrimPrefix(r.url, "http")+"/ws", botclient.Options{Token: seat.String(), AppVersion: "1.0.0"})
	if err != nil {
		r.t.Fatal(err)
	}
	r.t.Cleanup(c.Drop)
	return c
}

func (r *running) wait() error {
	r.t.Helper()
	select {
	case err := <-r.done:
		r.ended = true
		return err
	case <-time.After(30 * time.Second):
		r.t.Fatal("Run did not return")
		return nil
	}
}

func TestHealthEndpointsBeforeAndAfterStart(t *testing.T) {
	a := New(DefaultConfig(), NewLogger(&bytes.Buffer{}, "info"), &seatHandler{})
	server := httptest.NewServer(a.Handler())
	defer server.Close()
	resp, err := http.Get(server.URL + "/readyz")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("readyz before Run answered %d", resp.StatusCode)
	}
	resp, err = http.Get(server.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz answered %d", resp.StatusCode)
	}
	resp, err = http.Post(server.URL+"/healthz", "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("POST healthz answered %d", resp.StatusCode)
	}

	r := start(t, DefaultConfig())
	if status, body := r.get("/readyz"); status != http.StatusOK || body != "ready" {
		t.Fatalf("readyz while running: %d %q", status, body)
	}
	if status, body := r.get("/healthz"); status != http.StatusOK || body != "ok" {
		t.Fatalf("healthz while running: %d %q", status, body)
	}
	if !strings.Contains(r.logs.String(), `"msg":"listening"`) {
		t.Fatalf("no structured listening log: %s", r.logs.String())
	}
}

func TestShutdownDrainsLiveMatchesAndRefusesNewWork(t *testing.T) {
	r := start(t, DefaultConfig())
	r.openRoom(rules.Mode1v1)
	x, y := r.dial(rules.SeatX), r.dial(rules.SeatY)
	for _, c := range []*botclient.Client{x, y} {
		view, err := c.PlayUntil(func(v *pb.MatchView) bool { return v.GetState() != pb.MatchState_MATCH_STATE_WAITING })
		if err != nil {
			t.Fatal(err)
		}
		if view.GetCurrent() == view.GetSeat() {
			if err := c.Act(view); err != nil {
				t.Fatal(err)
			}
		}
	}
	r.stop()
	deadline := time.Now().Add(5 * time.Second)
	for status, _ := r.get("/readyz"); status != http.StatusServiceUnavailable; status, _ = r.get("/readyz") {
		if time.Now().After(deadline) {
			t.Fatal("readyz stayed ready after the stop signal")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := r.app.Rooms().Open(room.Options{Mode: rules.Mode1v1}); !errors.Is(err, ErrDraining) {
		t.Fatalf("opening a room while draining: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(r.url, "http")+"/ws", nil); err == nil {
		t.Fatal("a new connection was accepted while draining")
	}
	select {
	case err := <-r.done:
		t.Fatalf("Run returned %v with a live match", err)
	case <-time.After(200 * time.Millisecond):
	}
	finals := make(chan *pb.MatchView, 2)
	for _, c := range []*botclient.Client{x, y} {
		go func() {
			final, err := c.PlayUntil(botclient.Finished)
			if err != nil {
				t.Error(err)
			}
			finals <- final
		}()
	}
	for range 2 {
		select {
		case final := <-finals:
			if final.GetWinner() == pb.Team_TEAM_UNSPECIFIED {
				t.Fatalf("match ended without a winner: %v", final)
			}
		case <-time.After(30 * time.Second):
			t.Fatal("the match did not finish during the drain")
		}
	}
	if err := r.wait(); err != nil {
		t.Fatalf("Run returned %v", err)
	}
	if _, err := x.Read(); websocket.CloseStatus(err) != websocket.StatusGoingAway {
		t.Fatalf("after the drain the client saw %v", err)
	}
	if r.app.Rooms().Len() != 0 || !strings.Contains(r.logs.String(), `"msg":"stopped"`) {
		t.Fatalf("rooms %d, logs %s", r.app.Rooms().Len(), r.logs.String())
	}
}

func TestDrainTimeoutCancelsStuckRooms(t *testing.T) {
	config := DefaultConfig()
	config.DrainTimeout = 200 * time.Millisecond
	config.ShutdownTimeout = 2 * time.Second
	r := start(t, config)
	rm := r.openRoom(rules.Mode2v2)
	c := r.dial(rules.SeatNorth)
	if _, err := c.View(); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	r.stop()
	if err := r.wait(); err != nil {
		t.Fatalf("Run returned %v", err)
	}
	if elapsed := time.Since(started); elapsed > 3*time.Second {
		t.Fatalf("shutdown took %v", elapsed)
	}
	select {
	case <-rm.Done():
	default:
		t.Fatal("the stuck room was not cancelled")
	}
	if _, err := c.Read(); websocket.CloseStatus(err) != websocket.StatusGoingAway {
		t.Fatalf("client saw %v", err)
	}
	if !strings.Contains(r.logs.String(), "drain timeout") {
		t.Fatalf("logs %s", r.logs.String())
	}
}

func TestRunFailsOnABadListenAddress(t *testing.T) {
	config := DefaultConfig()
	config.ListenAddr = "256.256.256.256:1"
	err := New(config, NewLogger(&bytes.Buffer{}, "info"), &seatHandler{}).Run(context.Background())
	if err == nil {
		t.Fatal("Run succeeded on an unusable address")
	}
}

func TestMountedHandlersServeNextToTheBuiltInRoutes(t *testing.T) {
	config := DefaultConfig()
	config.ListenAddr = "127.0.0.1:0"
	logs := &bytes.Buffer{}
	a := New(config, NewLogger(logs, "info"), &seatHandler{})
	a.Handle("POST /auth/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(r.URL.Path))
	}))
	server := httptest.NewServer(a.Handler())
	t.Cleanup(server.Close)
	resp, err := http.Post(server.URL+"/auth/nonce", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	body := &bytes.Buffer{}
	body.ReadFrom(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated || body.String() != "/auth/nonce" {
		t.Fatalf("mounted handler answered %d %q", resp.StatusCode, body.String())
	}
	if resp, err := http.Get(server.URL + "/healthz"); err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz next to the mount: %v %v", resp, err)
	}
}

func TestReadyzWaitsForEveryAwaitedLoad(t *testing.T) {
	a := New(DefaultConfig(), NewLogger(&bytes.Buffer{}, "info"), &seatHandler{})
	var pass, shop atomic.Bool
	a.AwaitBeforeReady(pass.Load)
	a.AwaitBeforeReady(shop.Load)
	a.ready.Store(true)
	readyz := func() (int, string) {
		recorder := httptest.NewRecorder()
		a.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", nil))
		return recorder.Code, recorder.Body.String()
	}
	pass.Store(true)
	if status, body := readyz(); status != http.StatusServiceUnavailable || body != "loading config\n" {
		t.Fatalf("readyz with one config still loading: %d %q", status, body)
	}
	shop.Store(true)
	if status, body := readyz(); status != http.StatusOK || body != "ready\n" {
		t.Fatalf("readyz after every config loaded: %d %q", status, body)
	}
	a.ready.Store(false)
	if status, body := readyz(); status != http.StatusServiceUnavailable || body != "draining\n" {
		t.Fatalf("readyz while draining: %d %q", status, body)
	}
}

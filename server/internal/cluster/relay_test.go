package cluster_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Alievs-corp/hypedomino/server/internal/bot"
	"github.com/Alievs-corp/hypedomino/server/internal/botclient"
	"github.com/Alievs-corp/hypedomino/server/internal/cluster"
	"github.com/Alievs-corp/hypedomino/server/internal/cluster/redistest"
	"github.com/Alievs-corp/hypedomino/server/internal/gateway"
	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
	"github.com/Alievs-corp/hypedomino/server/internal/room"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
	"github.com/Alievs-corp/hypedomino/server/internal/secrand"
	"github.com/Alievs-corp/hypedomino/server/internal/session"
)

type realClock struct{}

func (realClock) Now() time.Time                         { return time.Now() }
func (realClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

type gameNode struct {
	cluster  *cluster.Cluster
	sessions *session.Manager
	relay    *cluster.Relay
	server   *httptest.Server
	draining atomic.Bool
}

func gatewayConfig() gateway.Config {
	config := gateway.DefaultConfig()
	config.MaxFrameBytes = 64 * 1024
	config.IntentsPerSecond = 1000
	config.IntentBurst = 1000
	return config
}

func startNode(t *testing.T, client *redis.Client, prefix, id string) *gameNode {
	t.Helper()
	listener := httptest.NewUnstartedServer(nil)
	config := cluster.DefaultConfig()
	config.NodeID, config.KeyPrefix = id, prefix
	config.Addr = listener.Listener.Addr().String()
	config.Heartbeat, config.NodeTTL = 50*time.Millisecond, 2*time.Second
	c, err := cluster.New(config, client, nil)
	if err != nil {
		t.Fatal(err)
	}
	n := &gameNode{cluster: c, server: listener}
	n.sessions = session.New(session.DefaultConfig(), realClock{}, secrand.Source{})
	n.sessions.Route(c)
	n.relay = cluster.NewRelay(c, n.sessions, cluster.DefaultRelayConfig())
	gw := gateway.New(gateway.Options{Config: gatewayConfig(), Handler: n.relay})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ws", func(w http.ResponseWriter, r *http.Request) {
		relayed := r.Header.Get(cluster.RelayHeader) != ""
		if n.draining.Load() && !relayed {
			http.Error(w, "draining", http.StatusServiceUnavailable)
			return
		}
		if relayed {
			r = r.WithContext(cluster.MarkRelayed(r.Context()))
		}
		gw.ServeHTTP(w, r)
	})
	listener.Config.Handler = mux
	listener.Start()
	t.Cleanup(listener.Close)
	c.Observe(func() cluster.Load { return cluster.Load{Connections: gw.Connections(), Draining: n.draining.Load()} })
	if err := c.Announce(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Leave(context.Background()) })
	return n
}

func (n *gameNode) url() string {
	return "ws" + strings.TrimPrefix(n.server.URL, "http") + "/ws"
}

func (n *gameNode) host(t *testing.T) (*room.Room, map[rules.Seat]string) {
	t.Helper()
	r := room.New(room.Options{Mode: rules.Mode1v1, Config: rules.DefaultConfig(), Bot: bot.DefaultConfig(), BotLevel: bot.Hard, Seed: 11, Clock: realClock{}})
	ctx, cancel := context.WithCancel(context.Background())
	go r.Run(ctx)
	t.Cleanup(func() { cancel(); <-r.Done() })
	tokens := map[rules.Seat]string{}
	for _, seat := range rules.Mode1v1.Seats() {
		token := n.sessions.Create("acct-" + seat.String())
		if err := n.sessions.Seat(token, r, seat); err != nil {
			t.Fatal(err)
		}
		tokens[seat] = token
	}
	return r, tokens
}

func dial(t *testing.T, url, token string, acked uint64) *botclient.Client {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	c, err := botclient.Dial(ctx, url, botclient.Options{Token: token, AppVersion: "1.0.0", LastAcked: acked})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Drop)
	return c
}

func TestAResumeOnAnotherNodeIsRelayedToTheHostingNode(t *testing.T) {
	client := redistest.Client(t)
	prefix := redistest.Prefix(t)
	a := startNode(t, client, prefix, "a")
	b := startNode(t, client, prefix, "b")
	_, tokens := a.host(t)
	north := dial(t, b.url(), tokens[rules.SeatX], 0)
	south := dial(t, a.url(), tokens[rules.SeatY], 0)
	if a.relay.Relayed() != 0 || b.relay.Relayed() != 1 {
		t.Fatalf("relayed counts a=%d b=%d", a.relay.Relayed(), b.relay.Relayed())
	}
	view, err := north.View()
	if err != nil || view.GetSeat() != pb.Seat_SEAT_X {
		t.Fatalf("north's first view through the relay: %v, %v", view, err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := south.PlayUntil(botclient.Finished)
		done <- err
	}()
	final, err := north.PlayUntil(botclient.Finished)
	if err != nil {
		t.Fatalf("north through the relay: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("south on the host: %v", err)
	}
	if final.GetState() != pb.MatchState_MATCH_STATE_FINISHED || north.Sent == 0 {
		t.Fatalf("relayed match ended in %s after %d intents", final.GetState(), north.Sent)
	}
	if !a.sessions.Connected(tokens[rules.SeatX]) {
		t.Fatal("the host does not see the relayed session as connected")
	}
}

func TestRelayFollowsTheHostIntoItsDrain(t *testing.T) {
	client := redistest.Client(t)
	prefix := redistest.Prefix(t)
	a := startNode(t, client, prefix, "a")
	b := startNode(t, client, prefix, "b")
	_, tokens := a.host(t)
	first := dial(t, a.url(), tokens[rules.SeatX], 0)
	if _, err := first.View(); err != nil {
		t.Fatal(err)
	}
	first.Drop()
	a.draining.Store(true)
	if err := a.cluster.Announce(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := botclient.Dial(ctx, a.url(), botclient.Options{Token: tokens[rules.SeatX], AppVersion: "1.0.0"}); err == nil {
		t.Fatal("a draining node accepted a public connection")
	}
	second := dial(t, b.url(), tokens[rules.SeatX], first.Acked)
	view, err := second.View()
	if err != nil || view.GetSeat() != pb.Seat_SEAT_X {
		t.Fatalf("resume through the relay into a draining host: %v, %v", view, err)
	}
	if b.relay.Relayed() != 1 {
		t.Fatalf("relayed count %d", b.relay.Relayed())
	}
}

func TestRelayRefusesUnknownRoutesAndGoneNodes(t *testing.T) {
	client := redistest.Client(t)
	prefix := redistest.Prefix(t)
	a := startNode(t, client, prefix, "a")
	b := startNode(t, client, prefix, "b")
	_, tokens := a.host(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := botclient.Dial(ctx, b.url(), botclient.Options{Token: "no-such-token", AppVersion: "1.0.0"})
	if !errors.Is(err, botclient.ErrRefused) {
		t.Fatalf("unknown token through b: %v", err)
	}
	if err := a.cluster.Leave(ctx); err != nil {
		t.Fatal(err)
	}
	_, err = botclient.Dial(ctx, b.url(), botclient.Options{Token: tokens[rules.SeatX], AppVersion: "1.0.0"})
	if !errors.Is(err, botclient.ErrRefused) {
		t.Fatalf("token of a gone node through b: %v", err)
	}
	if _, err := a.cluster.Lookup(ctx, tokens[rules.SeatX]); !errors.Is(err, cluster.ErrNoRoute) {
		t.Fatalf("route of a gone node kept: %v", err)
	}
	if b.relay.Relayed() != 0 {
		t.Fatalf("relayed count %d", b.relay.Relayed())
	}
}

func TestARelayedConnectionIsNeverRelayedAgain(t *testing.T) {
	client := redistest.Client(t)
	prefix := redistest.Prefix(t)
	a := startNode(t, client, prefix, "a")
	b := startNode(t, client, prefix, "b")
	_, tokens := a.host(t)
	if err := b.cluster.Claim(context.Background(), "foreign"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	header := http.Header{cluster.RelayHeader: []string{"b"}}
	socket, _, err := websocketDial(ctx, a.url(), header)
	if err != nil {
		t.Fatal(err)
	}
	defer socket.CloseNow()
	hello := &pb.Envelope{ProtocolVersion: gateway.ProtocolVersion, Payload: &pb.Envelope_ClientHello{ClientHello: &pb.ClientHello{AppVersion: "1.0.0", SessionToken: "foreign"}}}
	if err := writeFrame(ctx, socket, hello); err != nil {
		t.Fatal(err)
	}
	reply, err := readFrame(ctx, socket)
	if err != nil || reply.GetError().GetCode() != pb.ErrorCode_ERROR_CODE_UNAUTHENTICATED {
		t.Fatalf("a relayed hello with a foreign route was relayed again: %v, %v", reply, err)
	}
	if a.relay.Relayed() != 0 {
		t.Fatalf("relayed count %d", a.relay.Relayed())
	}
	_ = tokens
}

type tokenAuth map[string][2]string

func (a tokenAuth) Authenticate(token string) (string, string, error) {
	identity, ok := a[token]
	if !ok {
		return "", "", errors.New("bad token")
	}
	return identity[0], identity[1], nil
}

func TestAnAuthenticatedSessionIsRelayedToTheNodeThatSeatedIt(t *testing.T) {
	client := redistest.Client(t)
	prefix := redistest.Prefix(t)
	a := startNode(t, client, prefix, "a")
	b := startNode(t, client, prefix, "b")
	auth := tokenAuth{"jwt-x": {"acct-x", "sid-x"}, "jwt-y": {"acct-y", "sid-y"}}
	a.sessions.Authenticate(auth)
	b.sessions.Authenticate(auth)
	r := room.New(room.Options{Mode: rules.Mode1v1, Config: rules.DefaultConfig(), Bot: bot.DefaultConfig(), BotLevel: bot.Hard, Seed: 11, Clock: realClock{}})
	ctx, cancel := context.WithCancel(context.Background())
	go r.Run(ctx)
	t.Cleanup(func() { cancel(); <-r.Done() })
	if err := a.sessions.Place("sid-x", "acct-x", r, rules.SeatX); err != nil {
		t.Fatal(err)
	}
	if err := a.sessions.Place("sid-y", "acct-y", r, rules.SeatY); err != nil {
		t.Fatal(err)
	}
	north := dial(t, b.url(), "jwt-x", 0)
	if b.relay.Relayed() != 1 {
		t.Fatalf("relayed %d", b.relay.Relayed())
	}
	if view, err := north.View(); err != nil || view.GetSeat() != pb.Seat_SEAT_X {
		t.Fatalf("view through the relay: %v, %v", view, err)
	}
	if !a.sessions.Connected("sid-x") {
		t.Fatal("the host does not see the relayed session")
	}
}

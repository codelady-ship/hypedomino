package gateway

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"

	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
	"github.com/Alievs-corp/hypedomino/server/internal/replay"
)

const ProtocolVersion = 1

var (
	ErrUnauthenticated = errors.New("gateway: session token rejected")
	ErrIntegrityFailed = errors.New("gateway: device integrity check failed")
	ErrSlowConsumer    = errors.New("gateway: write queue full")
	ErrClosed          = errors.New("gateway: connection closed")
)

type RoutedElsewhere struct {
	Key string
}

func (e *RoutedElsewhere) Error() string {
	return "gateway: session hosted on another node"
}

func (e *RoutedElsewhere) Unwrap() error {
	return ErrUnauthenticated
}

type Config struct {
	MinimumAppVersion    string
	CatalogVersion       string
	ContentBaseURL       string
	MaxFrameBytes        int
	MaxHelloBytes        int
	WriteQueue           int
	HelloTimeout         time.Duration
	WriteTimeout         time.Duration
	CloseTimeout         time.Duration
	PingInterval         time.Duration
	PongTimeout          time.Duration
	CompressionThreshold int
	IntentsPerSecond     int
	IntentBurst          int
}

func DefaultConfig() Config {
	return Config{
		MinimumAppVersion:    "1.0.0",
		MaxFrameBytes:        1024,
		MaxHelloBytes:        8 * 1024,
		WriteQueue:           32,
		HelloTimeout:         5 * time.Second,
		WriteTimeout:         5 * time.Second,
		CloseTimeout:         2 * time.Second,
		PingInterval:         15 * time.Second,
		PongTimeout:          5 * time.Second,
		CompressionThreshold: 256,
		IntentsPerSecond:     10,
		IntentBurst:          20,
	}
}

type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

type Session interface {
	Acked() uint64
	InMatch() bool
	Attach()
	Loading(progress int, ready bool)
	Emoji(item string)
	QuickMessage(key string, teamOnly bool)
	Intent(sequence uint64, intent replay.Intent)
	Close()
}

type Handler interface {
	Connect(ctx context.Context, conn *Conn, hello *pb.ClientHello) (Session, error)
}

type HandlerFunc func(ctx context.Context, conn *Conn, hello *pb.ClientHello) (Session, error)

func (f HandlerFunc) Connect(ctx context.Context, conn *Conn, hello *pb.ClientHello) (Session, error) {
	return f(ctx, conn, hello)
}

type Strings struct {
	Bundles []*pb.StringBundle
	Chains  []*pb.FallbackChain
}

type Options struct {
	Config    Config
	Handler   Handler
	Clock     Clock
	Origins   []string
	Strings   func() Strings
	Admission func() Admission
}

type Gateway struct {
	options Options
	mu      sync.Mutex
	conns   map[*Conn]struct{}
}

func New(options Options) *Gateway {
	if options.Clock == nil {
		options.Clock = systemClock{}
	}
	return &Gateway{options: options, conns: map[*Conn]struct{}{}}
}

func (g *Gateway) track(conn *Conn) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.conns[conn] = struct{}{}
}

func (g *Gateway) forget(conn *Conn) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.conns, conn)
}

func (g *Gateway) Connections() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.conns)
}

func (g *Gateway) Enforce() {
	g.mu.Lock()
	conns := make([]*Conn, 0, len(g.conns))
	for conn := range g.conns {
		conns = append(conns, conn)
	}
	g.mu.Unlock()
	for _, conn := range conns {
		conn.enforce()
	}
}

func (g *Gateway) Shutdown(ctx context.Context) error {
	g.mu.Lock()
	conns := make([]*Conn, 0, len(g.conns))
	for conn := range g.conns {
		conns = append(conns, conn)
	}
	g.mu.Unlock()
	for _, conn := range conns {
		conn.Close(websocket.StatusGoingAway, "server shutdown")
	}
	for _, conn := range conns {
		select {
		case <-conn.served:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	socket, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns:       g.options.Origins,
		CompressionMode:      websocket.CompressionNoContextTakeover,
		CompressionThreshold: g.options.Config.CompressionThreshold,
	})
	if err != nil {
		return
	}
	conn := newConn(socket, g.options)
	g.track(conn)
	defer g.forget(conn)
	conn.serve(r.Context())
}

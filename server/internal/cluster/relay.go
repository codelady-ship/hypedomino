package cluster

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"google.golang.org/protobuf/proto"

	"github.com/Alievs-corp/hypedomino/server/internal/gateway"
	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
	"github.com/Alievs-corp/hypedomino/server/internal/replay"
)

const RelayHeader = "X-HD-Relay"

var ErrRelayRefused = errors.New("cluster: relay refused by the hosting node")

type relayedKey struct{}

func MarkRelayed(ctx context.Context) context.Context {
	return context.WithValue(ctx, relayedKey{}, true)
}

func Relayed(ctx context.Context) bool {
	relayed, _ := ctx.Value(relayedKey{}).(bool)
	return relayed
}

type RelayConfig struct {
	DialTimeout   time.Duration
	MaxFrameBytes int
}

func DefaultRelayConfig() RelayConfig {
	return RelayConfig{DialTimeout: 5 * time.Second, MaxFrameBytes: 64 * 1024}
}

type Relay struct {
	cluster *Cluster
	local   gateway.Handler
	config  RelayConfig
	scheme  string
	relayed atomic.Int64
}

func NewRelay(cluster *Cluster, local gateway.Handler, config RelayConfig) *Relay {
	return &Relay{cluster: cluster, local: local, config: config, scheme: "ws"}
}

func (r *Relay) Relayed() int {
	return int(r.relayed.Load())
}

func (r *Relay) Connect(ctx context.Context, conn *gateway.Conn, hello *pb.ClientHello) (gateway.Session, error) {
	session, err := r.local.Connect(ctx, conn, hello)
	if !errors.Is(err, gateway.ErrUnauthenticated) || Relayed(ctx) {
		return session, err
	}
	key := hello.GetSessionToken()
	var routed *gateway.RoutedElsewhere
	if errors.As(err, &routed) {
		key = routed.Key
	}
	node, lookupErr := r.cluster.Lookup(ctx, key)
	if lookupErr != nil || node.ID == r.cluster.NodeID() || node.Addr == "" {
		return nil, gateway.ErrUnauthenticated
	}
	upstream, err := r.dial(ctx, node, conn, hello)
	if err != nil {
		r.cluster.logger.Warn("relay dial failed", "node", node.ID, "addr", node.Addr, "error", err)
		return nil, gateway.ErrUnauthenticated
	}
	r.relayed.Add(1)
	return upstream, nil
}

func (r *Relay) dial(ctx context.Context, node Node, conn *gateway.Conn, hello *pb.ClientHello) (*relaySession, error) {
	dialCtx, cancel := context.WithTimeout(ctx, r.config.DialTimeout)
	defer cancel()
	header := http.Header{}
	header.Set(RelayHeader, r.cluster.NodeID())
	socket, _, err := websocket.Dial(dialCtx, r.scheme+"://"+node.Addr+"/ws", &websocket.DialOptions{HTTPHeader: header})
	if err != nil {
		return nil, err
	}
	socket.SetReadLimit(int64(r.config.MaxFrameBytes))
	s := &relaySession{socket: socket, conn: conn, done: make(chan struct{})}
	if err := s.handshake(dialCtx, hello); err != nil {
		socket.CloseNow()
		return nil, err
	}
	return s, nil
}

type relaySession struct {
	socket  *websocket.Conn
	conn    *gateway.Conn
	acked   atomic.Uint64
	inMatch bool
	nonce   uint64
	mu      sync.Mutex
	done    chan struct{}
	once    sync.Once
}

func (s *relaySession) handshake(ctx context.Context, hello *pb.ClientHello) error {
	frame := &pb.Envelope{
		ProtocolVersion: gateway.ProtocolVersion,
		Sequence:        hello.GetLastAckedSequence(),
		Payload:         &pb.Envelope_ClientHello{ClientHello: proto.Clone(hello).(*pb.ClientHello)},
	}
	if err := s.write(ctx, frame); err != nil {
		return err
	}
	reply, err := s.read(ctx)
	if err != nil {
		return err
	}
	if reply.GetServerHello().GetStatus() != pb.HelloStatus_HELLO_STATUS_ACCEPTED {
		return ErrRelayRefused
	}
	s.acked.Store(reply.GetAckedSequence())
	s.inMatch = reply.GetServerHello().GetInMatch()
	s.nonce = reply.GetServerHello().GetSessionNonce()
	return nil
}

func (s *relaySession) write(ctx context.Context, frame *pb.Envelope) error {
	data, err := proto.Marshal(frame)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.socket.Write(ctx, websocket.MessageBinary, data)
}

func (s *relaySession) read(ctx context.Context) (*pb.Envelope, error) {
	kind, data, err := s.socket.Read(ctx)
	if err != nil {
		return nil, err
	}
	if kind != websocket.MessageBinary {
		return nil, errors.New("cluster: text frame from the hosting node")
	}
	frame := &pb.Envelope{}
	if err := proto.Unmarshal(data, frame); err != nil {
		return nil, err
	}
	return frame, nil
}

func (s *relaySession) Acked() uint64 {
	return s.acked.Load()
}

func (s *relaySession) InMatch() bool {
	return s.inMatch
}

func (s *relaySession) Loading(progress int, ready bool) {
	frame := &pb.Envelope{ProtocolVersion: gateway.ProtocolVersion, SessionNonce: s.nonce, Payload: &pb.Envelope_LoadingIntent{LoadingIntent: &pb.LoadingIntent{Progress: uint32(progress), Ready: ready}}}
	if err := s.write(context.Background(), frame); err != nil {
		s.conn.Close(websocket.StatusGoingAway, "hosting node unreachable")
	}
}

func (s *relaySession) Emoji(item string) {
	s.forward(&pb.Envelope{Payload: &pb.Envelope_EmojiIntent{EmojiIntent: &pb.EmojiIntent{ItemId: item}}})
}

func (s *relaySession) QuickMessage(key string, teamOnly bool) {
	s.forward(&pb.Envelope{Payload: &pb.Envelope_QuickMessageIntent{QuickMessageIntent: &pb.QuickMessageIntent{Key: key, TeamOnly: teamOnly}}})
}

func (s *relaySession) JoinQueue(intent *pb.JoinQueueIntent) {
	s.forward(&pb.Envelope{Payload: &pb.Envelope_JoinQueueIntent{JoinQueueIntent: intent}})
}

func (s *relaySession) LeaveQueue() {
	s.forward(&pb.Envelope{Payload: &pb.Envelope_LeaveQueueIntent{LeaveQueueIntent: &pb.LeaveQueueIntent{}}})
}

func (s *relaySession) forward(frame *pb.Envelope) {
	frame.ProtocolVersion, frame.SessionNonce = gateway.ProtocolVersion, s.nonce
	if err := s.write(context.Background(), frame); err != nil {
		s.conn.Close(websocket.StatusGoingAway, "hosting node unreachable")
	}
}

func (s *relaySession) Attach() {
	go s.pump()
}

func (s *relaySession) pump() {
	defer s.Close()
	for {
		frame, err := s.read(context.Background())
		if err != nil {
			s.conn.Close(websocket.StatusGoingAway, "hosting node closed")
			return
		}
		if view := frame.GetMatchView(); view != nil {
			s.acked.Store(frame.GetAckedSequence())
		}
		if s.conn.Send(frame) != nil {
			return
		}
	}
}

func (s *relaySession) Intent(sequence uint64, intent replay.Intent) {
	frame, ok := gateway.EncodeIntent(intent)
	if !ok {
		return
	}
	frame.ProtocolVersion = gateway.ProtocolVersion
	frame.Sequence = sequence
	frame.SessionNonce = s.nonce
	if err := s.write(context.Background(), frame); err != nil {
		s.conn.Close(websocket.StatusGoingAway, "hosting node unreachable")
	}
}

func (s *relaySession) Close() {
	s.once.Do(func() {
		close(s.done)
		s.socket.Close(websocket.StatusNormalClosure, "")
	})
}

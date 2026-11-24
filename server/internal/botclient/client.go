package botclient

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/coder/websocket"
	"google.golang.org/protobuf/proto"

	"github.com/Alievs-corp/hypedomino/server/internal/gateway"
	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
)

var (
	ErrUpdateRequired = errors.New("botclient: update required")
	ErrRefused        = errors.New("botclient: hello refused")
)

type Client struct {
	socket        *websocket.Conn
	ctx           context.Context
	buffer        []byte
	nonce         uint64
	Hello         *pb.ServerHello
	Acked         uint64
	Sequence      uint64
	Sent          int
	SentBytes     int
	ReceivedBytes int
	Last          *pb.MatchView
}

type Options struct {
	Token       string
	AppVersion  string
	Platform    pb.Platform
	LastAcked   uint64
	Compression bool
}

func Dial(ctx context.Context, url string, options Options) (*Client, error) {
	dial := &websocket.DialOptions{}
	if options.Compression {
		dial.CompressionMode = websocket.CompressionNoContextTakeover
	}
	socket, _, err := websocket.Dial(ctx, url, dial)
	if err != nil {
		return nil, err
	}
	c := &Client{socket: socket, ctx: ctx, Sequence: options.LastAcked}
	if err := c.handshake(options); err != nil {
		socket.CloseNow()
		return nil, err
	}
	return c, nil
}

func (c *Client) handshake(options Options) error {
	frame := &pb.Envelope{
		ProtocolVersion: gateway.ProtocolVersion,
		Sequence:        c.Sequence,
		Payload: &pb.Envelope_ClientHello{ClientHello: &pb.ClientHello{
			AppVersion:        options.AppVersion,
			Platform:          options.Platform,
			SessionToken:      options.Token,
			LastAckedSequence: options.LastAcked,
		}},
	}
	if err := c.write(frame); err != nil {
		return err
	}
	reply, err := c.Read()
	if err != nil {
		return err
	}
	switch payload := reply.GetPayload().(type) {
	case *pb.Envelope_ServerHello:
		c.Hello = payload.ServerHello
		c.nonce = payload.ServerHello.GetSessionNonce()
		c.Acked = reply.GetAckedSequence()
		c.Sequence = c.Acked
		if payload.ServerHello.GetStatus() != pb.HelloStatus_HELLO_STATUS_ACCEPTED {
			return fmt.Errorf("%w: minimum app version %s", ErrUpdateRequired, payload.ServerHello.GetMinimumAppVersion())
		}
		return nil
	case *pb.Envelope_Error:
		return fmt.Errorf("%w: %s", ErrRefused, payload.Error.GetCode())
	}
	return fmt.Errorf("%w: unexpected payload %T", ErrRefused, reply.GetPayload())
}

func (c *Client) write(frame *pb.Envelope) error {
	if frame.GetClientHello() == nil {
		frame.SessionNonce = c.nonce
	}
	data, err := proto.MarshalOptions{}.MarshalAppend(c.buffer[:0], frame)
	if err != nil {
		return err
	}
	c.buffer = data
	return c.socket.Write(c.ctx, websocket.MessageBinary, data)
}

func (c *Client) Ready() error {
	return c.write(&pb.Envelope{ProtocolVersion: gateway.ProtocolVersion, Payload: &pb.Envelope_LoadingIntent{LoadingIntent: &pb.LoadingIntent{Progress: 100, Ready: true}}})
}

func NeedsReady(view *pb.MatchView) bool {
	if view.GetState() != pb.MatchState_MATCH_STATE_LOADING {
		return false
	}
	for _, seat := range view.GetSeats() {
		if seat.GetSeat() == view.GetSeat() {
			return !seat.GetReady()
		}
	}
	return true
}

func Idle(view *pb.MatchView) bool {
	switch view.GetState() {
	case pb.MatchState_MATCH_STATE_WAITING, pb.MatchState_MATCH_STATE_LOADING, pb.MatchState_MATCH_STATE_FINISHED:
		return true
	}
	return view.GetCurrent() != view.GetSeat()
}

func (c *Client) Send(frame *pb.Envelope) error {
	c.Sequence++
	frame.ProtocolVersion = gateway.ProtocolVersion
	frame.Sequence = c.Sequence
	if err := c.write(frame); err != nil {
		return err
	}
	c.Sent++
	c.SentBytes += len(c.buffer)
	return nil
}

func (c *Client) Receive() (*pb.Envelope, error) {
	kind, data, err := c.socket.Read(c.ctx)
	if err != nil {
		return nil, err
	}
	if kind != websocket.MessageBinary {
		return nil, errors.New("botclient: text frame from the server")
	}
	c.ReceivedBytes += len(data)
	frame := &pb.Envelope{}
	if err := proto.Unmarshal(data, frame); err != nil {
		return nil, err
	}
	return frame, nil
}

func (c *Client) Observe(frame *pb.Envelope) {
	if view := frame.GetMatchView(); view != nil {
		if view.GetRoster() == nil && c.Last != nil {
			view.Roster, view.Table = c.Last.GetRoster(), c.Last.GetTable()
		}
		c.Acked = frame.GetAckedSequence()
		c.Sequence = max(c.Sequence, c.Acked)
		c.Last = view
	}
}

func (c *Client) Read() (*pb.Envelope, error) {
	frame, err := c.Receive()
	if err != nil {
		return nil, err
	}
	c.Observe(frame)
	return frame, nil
}

func (c *Client) View() (*pb.MatchView, error) {
	for {
		frame, err := c.Read()
		if err != nil {
			return nil, err
		}
		if rejected := frame.GetError(); rejected != nil {
			return nil, fmt.Errorf("botclient: intent %d rejected: %s", rejected.GetRejectedSequence(), rejected.GetCode())
		}
		if view := frame.GetMatchView(); view != nil {
			return view, nil
		}
	}
}

func (c *Client) Act(view *pb.MatchView) error {
	switch {
	case view.GetState() == pb.MatchState_MATCH_STATE_OPENING && view.GetForcedOpening() != nil:
		return c.Send(place(view.GetForcedOpening().GetTile(), pb.BoardEnd_BOARD_END_UNSPECIFIED))
	case view.GetState() == pb.MatchState_MATCH_STATE_OPENING && len(view.GetHand()) > 0:
		return c.Send(place(view.GetHand()[0], pb.BoardEnd_BOARD_END_UNSPECIFIED))
	case len(view.GetLegalMoves()) > 0:
		move := view.GetLegalMoves()[0]
		return c.Send(place(move.GetTile(), move.GetEnd()))
	case view.GetTurnPlacements() > 0:
		return c.Send(&pb.Envelope{Payload: &pb.Envelope_EndTurnIntent{EndTurnIntent: &pb.EndTurnIntent{}}})
	case view.GetMode() == pb.Mode_MODE_1V1 && view.GetBoneyardSize() > 1:
		return c.Send(&pb.Envelope{Payload: &pb.Envelope_DrawIntent{DrawIntent: &pb.DrawIntent{}}})
	}
	return c.Send(&pb.Envelope{Payload: &pb.Envelope_PassIntent{PassIntent: &pb.PassIntent{}}})
}

func place(tile *pb.Tile, end pb.BoardEnd) *pb.Envelope {
	return &pb.Envelope{Payload: &pb.Envelope_PlaceTileIntent{PlaceTileIntent: &pb.PlaceTileIntent{Tile: tile, End: end}}}
}

func (c *Client) PlayUntil(stop func(*pb.MatchView) bool) (*pb.MatchView, error) {
	for {
		frame, err := c.Read()
		if err != nil {
			return nil, err
		}
		view := frame.GetMatchView()
		if view == nil {
			continue
		}
		if stop(view) {
			return view, nil
		}
		if NeedsReady(view) {
			if err := c.Ready(); err != nil {
				return nil, err
			}
			continue
		}
		if !Idle(view) {
			if err := c.Act(view); err != nil {
				return nil, err
			}
		}
	}
}

func Finished(view *pb.MatchView) bool {
	return view.GetState() == pb.MatchState_MATCH_STATE_FINISHED
}

func (c *Client) Close() error {
	return c.socket.Close(websocket.StatusNormalClosure, "")
}

func (c *Client) Drop() {
	c.socket.CloseNow()
}

func (c *Client) Nonce() uint64 {
	return c.nonce
}

func (c *Client) LastSent() []byte {
	return slices.Clone(c.buffer)
}

func (c *Client) WriteRaw(kind websocket.MessageType, data []byte) error {
	return c.socket.Write(c.ctx, kind, data)
}

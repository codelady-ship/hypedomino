package cluster_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/redis/go-redis/v9"

	"github.com/Alievs-corp/hypedomino/server/internal/bot"
	"github.com/Alievs-corp/hypedomino/server/internal/cluster/redistest"
	"github.com/Alievs-corp/hypedomino/server/internal/custom"
	"github.com/Alievs-corp/hypedomino/server/internal/gateway"
	"github.com/Alievs-corp/hypedomino/server/internal/lobbynet"
	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
	"github.com/Alievs-corp/hypedomino/server/internal/room"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
	"github.com/Alievs-corp/hypedomino/server/internal/secrand"
)

type everyoneFriends struct{}

func (everyoneFriends) FriendIDs(ctx context.Context, account string) ([]string, error) {
	return []string{"ana", "ben"}, nil
}

func (n *gameNode) hostLobby(t *testing.T, client *redis.Client, prefix string) {
	t.Helper()
	lobby := custom.NewLobby(secrand.Source{}, func(options room.Options, seats []custom.SeatState) (*room.Room, error) {
		options.Config, options.Bot, options.BotLevel, options.Clock = rules.DefaultConfig(), bot.DefaultConfig(), bot.Hard, realClock{}
		r := room.New(options)
		ctx, cancel := context.WithCancel(context.Background())
		go r.Run(ctx)
		t.Cleanup(func() { cancel(); <-r.Done() })
		return r, nil
	})
	lobbies := lobbynet.New(client, n.cluster, prefix, 5*time.Second)
	lobby.Route(lobbies)
	members := func(ctx context.Context, account string) (custom.Member, error) {
		return custom.Member{Emoji: []string{"emoji.laugh"}}, nil
	}
	n.sessions.Host(lobby, members, everyoneFriends{})
	n.sessions.Reach(lobbies)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go lobbies.Listen(ctx, n.sessions.Adopt, n.sessions.Invited)
}

type lobbyClient struct {
	t      *testing.T
	ctx    context.Context
	socket *websocket.Conn
	nonce  uint64
}

func dialLobby(t *testing.T, url, token string) *lobbyClient {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	socket, _, err := websocketDial(ctx, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { socket.CloseNow() })
	c := &lobbyClient{t: t, ctx: ctx, socket: socket}
	if err := writeFrame(ctx, socket, &pb.Envelope{ProtocolVersion: gateway.ProtocolVersion, Payload: &pb.Envelope_ClientHello{ClientHello: &pb.ClientHello{AppVersion: "1.0.0", SessionToken: token}}}); err != nil {
		t.Fatal(err)
	}
	hello := c.until(func(f *pb.Envelope) bool { return f.GetServerHello() != nil }).GetServerHello()
	if hello.GetStatus() != pb.HelloStatus_HELLO_STATUS_ACCEPTED {
		t.Fatalf("hello %v", hello)
	}
	c.nonce = hello.GetSessionNonce()
	return c
}

func (c *lobbyClient) send(intent *pb.LobbyIntent) {
	c.t.Helper()
	c.write(&pb.Envelope{Payload: &pb.Envelope_LobbyIntent{LobbyIntent: intent}})
}

func (c *lobbyClient) emoji(item string) {
	c.t.Helper()
	c.write(&pb.Envelope{Payload: &pb.Envelope_EmojiIntent{EmojiIntent: &pb.EmojiIntent{ItemId: item}}})
}

func (c *lobbyClient) write(frame *pb.Envelope) {
	c.t.Helper()
	frame.ProtocolVersion, frame.SessionNonce = gateway.ProtocolVersion, c.nonce
	if err := writeFrame(c.ctx, c.socket, frame); err != nil {
		c.t.Fatal(err)
	}
}

func (c *lobbyClient) until(match func(*pb.Envelope) bool) *pb.Envelope {
	c.t.Helper()
	for {
		frame, err := readFrame(c.ctx, c.socket)
		if err != nil {
			c.t.Fatal(err)
		}
		if match(frame) {
			return frame
		}
	}
}

func (c *lobbyClient) state() *pb.LobbyState {
	c.t.Helper()
	return c.until(func(f *pb.Envelope) bool { return f.GetLobbyState() != nil }).GetLobbyState()
}

func (c *lobbyClient) closed() websocket.StatusCode {
	for {
		if _, err := readFrame(c.ctx, c.socket); err != nil {
			return websocket.CloseStatus(err)
		}
	}
}

func listening(t *testing.T, client *redis.Client, prefix string, nodes int64) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		counts, err := client.PubSubNumSub(context.Background(), prefix+"lobby:invite").Result()
		if err == nil && counts[prefix+"lobby:invite"] >= nodes {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("lobby listeners did not subscribe")
}

func TestAFriendOnAnotherNodeIsInvitedJoinsByCodeAndPlaysTheCustomMatch(t *testing.T) {
	client := redistest.Client(t)
	prefix := redistest.Prefix(t)
	a := startNode(t, client, prefix, "a")
	b := startNode(t, client, prefix, "b")
	a.hostLobby(t, client, prefix)
	b.hostLobby(t, client, prefix)
	listening(t, client, prefix, 2)
	ana := dialLobby(t, a.url(), a.sessions.Create("ana"))
	benToken := b.sessions.Create("ben")
	ben := dialLobby(t, b.url(), benToken)
	ana.send(&pb.LobbyIntent{Command: pb.LobbyCommand_LOBBY_COMMAND_CREATE})
	code := ana.state().GetCode()
	ana.send(&pb.LobbyIntent{Command: pb.LobbyCommand_LOBBY_COMMAND_INVITE, AccountId: "ben"})
	invite := ben.until(func(f *pb.Envelope) bool { return f.GetLobbyInvite() != nil }).GetLobbyInvite()
	if invite.GetCode() != code || invite.GetFromAccountId() != "ana" {
		t.Fatalf("cross-node invite %v", invite)
	}
	ben.send(&pb.LobbyIntent{Command: pb.LobbyCommand_LOBBY_COMMAND_JOIN, Code: "NOSUCH"})
	if refusal := ben.state().GetRefusal(); refusal != pb.LobbyRefusal_LOBBY_REFUSAL_UNKNOWN_CODE {
		t.Fatalf("unknown code refused with %v", refusal)
	}
	ben.send(&pb.LobbyIntent{Command: pb.LobbyCommand_LOBBY_COMMAND_JOIN, Code: code})
	if status := ben.closed(); status != websocket.StatusGoingAway {
		t.Fatalf("joiner closed with %v", status)
	}
	ben = dialLobby(t, b.url(), benToken)
	if b.relay.Relayed() != 1 {
		t.Fatalf("relayed %d", b.relay.Relayed())
	}
	for _, c := range []*lobbyClient{ana, ben} {
		if state := c.state(); state.GetCode() != code || !slices.Equal(state.GetMemberAccountIds(), []string{"ana", "ben"}) {
			t.Fatalf("members after the cross-node join %v", state)
		}
	}
	ben.emoji("emoji.laugh")
	if relay := ana.until(func(f *pb.Envelope) bool { return f.GetLobbyRelay() != nil }).GetLobbyRelay(); relay.GetAccountId() != "ben" || relay.GetItemId() != "emoji.laugh" {
		t.Fatalf("relayed emoji %v", relay)
	}
	ana.send(&pb.LobbyIntent{Command: pb.LobbyCommand_LOBBY_COMMAND_TEAMS, Mode: pb.Mode_MODE_1V1})
	ana.send(&pb.LobbyIntent{Command: pb.LobbyCommand_LOBBY_COMMAND_PLACE, AccountId: "ana", Seat: pb.Seat_SEAT_X})
	ana.send(&pb.LobbyIntent{Command: pb.LobbyCommand_LOBBY_COMMAND_PLACE, AccountId: "ben", Seat: pb.Seat_SEAT_Y})
	ana.send(&pb.LobbyIntent{Command: pb.LobbyCommand_LOBBY_COMMAND_START})
	for seat, c := range map[pb.Seat]*lobbyClient{pb.Seat_SEAT_X: ana, pb.Seat_SEAT_Y: ben} {
		if view := c.until(func(f *pb.Envelope) bool { return f.GetMatchView() != nil }).GetMatchView(); view.GetSeat() != seat {
			t.Fatalf("custom match view %v", view)
		}
	}
	if !a.sessions.InMatch("ben") {
		t.Fatal("the joiner from b was not seated on a")
	}
}

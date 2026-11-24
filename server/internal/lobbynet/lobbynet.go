package lobbynet

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Alievs-corp/hypedomino/server/internal/cluster"
	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
)

type Net struct {
	client  redis.UniversalClient
	members *cluster.Cluster
	prefix  string
	timeout time.Duration
}

type join struct {
	Code    string `json:"code"`
	Account string `json:"account"`
	Session string `json:"session"`
	Reply   string `json:"reply"`
}

type invite struct {
	To   string `json:"to"`
	Code string `json:"code"`
	From string `json:"from"`
}

type Joiner func(ctx context.Context, code, account, session string) pb.LobbyRefusal

type Invited func(to string, invite *pb.LobbyInvite)

func New(client redis.UniversalClient, members *cluster.Cluster, prefix string, timeout time.Duration) *Net {
	return &Net{client: client, members: members, prefix: prefix, timeout: timeout}
}

func route(code string) string {
	return "lobby:" + code
}

func (n *Net) joinChannel(node string) string {
	return n.prefix + "lobby:join:" + node
}

func (n *Net) inviteChannel() string {
	return n.prefix + "lobby:invite"
}

func (n *Net) Claim(ctx context.Context, code string) error {
	return n.members.Claim(ctx, route(code))
}

func (n *Net) Release(ctx context.Context, code string) error {
	return n.members.Release(ctx, route(code))
}

func (n *Net) Join(ctx context.Context, code, account, session string) (pb.LobbyRefusal, error) {
	node, err := n.members.Lookup(ctx, route(code))
	switch {
	case errors.Is(err, cluster.ErrNoRoute), errors.Is(err, cluster.ErrNodeGone):
		return pb.LobbyRefusal_LOBBY_REFUSAL_UNKNOWN_CODE, nil
	case err != nil:
		return 0, err
	case node.ID == n.members.NodeID():
		return pb.LobbyRefusal_LOBBY_REFUSAL_UNKNOWN_CODE, nil
	}
	request := join{Code: code, Account: account, Session: session, Reply: n.prefix + "lobby:reply:" + rand.Text()}
	raw, err := json.Marshal(request)
	if err != nil {
		return 0, err
	}
	receivers, err := n.client.Publish(ctx, n.joinChannel(node.ID), raw).Result()
	if err != nil || receivers == 0 {
		return pb.LobbyRefusal_LOBBY_REFUSAL_UNAVAILABLE, err
	}
	values, err := n.client.BLPop(ctx, n.timeout, request.Reply).Result()
	if errors.Is(err, redis.Nil) {
		return pb.LobbyRefusal_LOBBY_REFUSAL_UNAVAILABLE, nil
	}
	if err != nil {
		return 0, err
	}
	refusal, err := strconv.Atoi(values[1])
	return pb.LobbyRefusal(refusal), err
}

func (n *Net) Invite(ctx context.Context, to string, message *pb.LobbyInvite) error {
	raw, err := json.Marshal(invite{To: to, Code: message.GetCode(), From: message.GetFromAccountId()})
	if err != nil {
		return err
	}
	return n.client.Publish(ctx, n.inviteChannel(), raw).Err()
}

func (n *Net) Listen(ctx context.Context, joiner Joiner, invited Invited) error {
	sub := n.client.Subscribe(ctx, n.joinChannel(n.members.NodeID()), n.inviteChannel())
	defer sub.Close()
	if _, err := sub.Receive(ctx); err != nil {
		return err
	}
	for {
		select {
		case message, ok := <-sub.Channel():
			if !ok {
				return nil
			}
			n.receive(ctx, message, joiner, invited)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (n *Net) receive(ctx context.Context, message *redis.Message, joiner Joiner, invited Invited) {
	if message.Channel == n.inviteChannel() {
		var to invite
		if json.Unmarshal([]byte(message.Payload), &to) == nil {
			invited(to.To, &pb.LobbyInvite{Code: to.Code, FromAccountId: to.From})
		}
		return
	}
	var request join
	if json.Unmarshal([]byte(message.Payload), &request) == nil {
		go n.answer(ctx, request, joiner)
	}
}

func (n *Net) answer(ctx context.Context, request join, joiner Joiner) {
	refusal := joiner(ctx, request.Code, request.Account, request.Session)
	pipe := n.client.TxPipeline()
	pipe.RPush(ctx, request.Reply, strconv.Itoa(int(refusal)))
	pipe.Expire(ctx, request.Reply, n.timeout)
	pipe.Exec(ctx)
}

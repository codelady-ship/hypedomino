package session

import (
	"context"

	"github.com/coder/websocket"

	"github.com/Alievs-corp/hypedomino/server/internal/gateway"
	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
)

type Queue interface {
	Join(ctx context.Context, account, session string, intent *pb.JoinQueueIntent) *pb.QueueState
	Leave(ctx context.Context, account string) *pb.QueueState
}

func (m *Manager) Queue(queue Queue) {
	m.queue = queue
}

func refused(refusal pb.QueueRefusal) *pb.QueueState {
	return &pb.QueueState{Status: pb.QueueStatus_QUEUE_STATUS_REFUSED, Refusal: refusal}
}

func (c *connection) JoinQueue(intent *pb.JoinQueueIntent) {
	state := c.join(intent)
	c.conn.Send(gateway.QueueStateFrame(state, c.Acked()))
}

func (c *connection) join(intent *pb.JoinQueueIntent) *pb.QueueState {
	switch {
	case c.InMatch():
		return refused(pb.QueueRefusal_QUEUE_REFUSAL_IN_MATCH)
	case c.manager.queue == nil:
		return refused(pb.QueueRefusal_QUEUE_REFUSAL_UNAVAILABLE)
	}
	if refusal := c.matchIntegrity(intent); refusal != pb.QueueRefusal_QUEUE_REFUSAL_UNSPECIFIED {
		return refused(refusal)
	}
	return c.manager.queue.Join(context.Background(), c.session.account, c.session.token, intent)
}

func (c *connection) matchIntegrity(intent *pb.JoinQueueIntent) pb.QueueRefusal {
	if c.manager.checker == nil {
		return pb.QueueRefusal_QUEUE_REFUSAL_UNSPECIFIED
	}
	return c.manager.checker.MatchStart(context.Background(), c.session.account, c.session.token, intent.GetQueue(), intent.GetIntegrity())
}

func (c *connection) LeaveQueue() {
	if c.manager.queue == nil {
		c.conn.Send(gateway.QueueStateFrame(refused(pb.QueueRefusal_QUEUE_REFUSAL_UNAVAILABLE), c.Acked()))
		return
	}
	c.conn.Send(gateway.QueueStateFrame(c.manager.queue.Leave(context.Background(), c.session.account), c.Acked()))
}

func (c *connection) leaveOnClose() {
	if c.manager.queue != nil {
		c.manager.queue.Leave(context.Background(), c.session.account)
	}
}

func (m *Manager) MatchFound(id string) {
	m.mu.Lock()
	s, ok := m.sessions[id]
	if !ok || s.active == nil {
		m.mu.Unlock()
		return
	}
	c, hosted := s.active, s.room != nil
	m.mu.Unlock()
	c.conn.Send(gateway.QueueStateFrame(&pb.QueueState{Status: pb.QueueStatus_QUEUE_STATUS_MATCH_FOUND}, c.Acked()))
	if !hosted {
		c.conn.Close(websocket.StatusGoingAway, "match hosted on another node")
	}
}

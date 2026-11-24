package gateway

import (
	"sync/atomic"

	"github.com/coder/websocket"

	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
	"github.com/Alievs-corp/hypedomino/server/internal/replay"
	"github.com/Alievs-corp/hypedomino/server/internal/room"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

type RoomSession struct {
	room       *room.Room
	seat       rules.Seat
	conn       *Conn
	resume     bool
	staticSent bool
	leftShown  int
	acked      atomic.Uint64
}

func JoinRoom(r *room.Room, seat rules.Seat, conn *Conn) *RoomSession {
	return &RoomSession{room: r, seat: seat, conn: conn}
}

func ResumeRoom(r *room.Room, seat rules.Seat, conn *Conn, acked uint64) *RoomSession {
	s := &RoomSession{room: r, seat: seat, conn: conn, resume: true}
	s.acked.Store(acked)
	return s
}

func (s *RoomSession) Acked() uint64 {
	return s.acked.Load()
}

func (s *RoomSession) InMatch() bool {
	select {
	case <-s.room.Finished():
		return false
	default:
		return true
	}
}

func (s *RoomSession) Loading(progress int, ready bool) {
	s.room.Loading(s.seat, progress, ready)
}

func (s *RoomSession) Emoji(item string) {
	s.room.Emoji(s.seat, item)
}

func (s *RoomSession) QuickMessage(key string, teamOnly bool) {
	s.room.QuickMessage(s.seat, key, teamOnly)
}

func (s *RoomSession) Attach() {
	if s.resume {
		s.room.Reconnect(s.seat, s)
		return
	}
	s.room.Join(s.seat, s)
}

func (s *RoomSession) Intent(sequence uint64, intent replay.Intent) {
	s.room.Intent(s.seat, sequence, intent)
}

func (s *RoomSession) Close() {
	s.room.Disconnect(s.seat, s)
}

func (s *RoomSession) Deliver(message room.Message) {
	if message.Emoji != nil {
		s.conn.Send(&pb.Envelope{
			AckedSequence: s.acked.Load(),
			Payload:       &pb.Envelope_EmojiShown{EmojiShown: &pb.EmojiShown{Seat: seatMessage(message.Emoji.Seat), ItemId: message.Emoji.Item}},
		})
		return
	}
	if message.Quick != nil {
		s.conn.Send(&pb.Envelope{
			AckedSequence: s.acked.Load(),
			Payload:       &pb.Envelope_QuickMessageShown{QuickMessageShown: &pb.QuickMessageShown{Seat: seatMessage(message.Quick.Seat), Key: message.Quick.Key, TeamOnly: message.Quick.TeamOnly}},
		})
		return
	}
	if message.Rejected && message.Sequence == 0 {
		s.conn.Close(websocket.StatusPolicyViolation, message.Reason.Error())
		return
	}
	s.acked.Store(message.View.Acked)
	if message.Rejected {
		s.conn.Send(ErrorFrame(pb.ErrorCode_ERROR_CODE_ILLEGAL_INTENT, message.Sequence, message.View.Acked))
	}
	view := ViewMessage(message.View, !s.staticSent, message.View.HandsPlayed != s.leftShown)
	s.conn.Send(&pb.Envelope{
		AckedSequence: message.View.Acked,
		Payload:       &pb.Envelope_MatchView{MatchView: view},
	})
	s.staticSent = true
	if len(view.GetLastResult().GetLeft()) > 0 {
		s.leftShown = message.View.HandsPlayed
	}
}

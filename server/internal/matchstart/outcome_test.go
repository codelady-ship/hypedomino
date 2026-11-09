package matchstart

import (
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/matchmaking"
	"github.com/Alievs-corp/hypedomino/server/internal/matchstore"
	"github.com/Alievs-corp/hypedomino/server/internal/penalties"
	"github.com/Alievs-corp/hypedomino/server/internal/replay"
	"github.com/Alievs-corp/hypedomino/server/internal/room"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func TestAMatchThatNeverDealtIsArchivedCancelledWithOneOutcomePerPlayer(t *testing.T) {
	now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	match := matchmaking.Match{Mode: rules.Mode2v2, Queue: matchmaking.QueueRanked, Seats: []matchmaking.Seat{{Seat: rules.SeatNorth, Account: "a"}, {Seat: rules.SeatEast, Bot: true}, {Seat: rules.SeatSouth, Account: "b"}, {Seat: rules.SeatWest, Account: "c"}}}
	s := started(match, []room.Player{{Seat: rules.SeatNorth, Character: "char.a"}}, 7, now)
	if s.Kind != matchstore.KindRanked || s.Participants[0].Loadout.Character != "char.a" || !s.Participants[1].Bot {
		t.Fatalf("started %+v", s)
	}
	f := finished(s, replay.Record{}, false, map[rules.Seat]penalties.Kind{rules.SeatNorth: penalties.KindLeave, rules.SeatSouth: penalties.KindAfkReturned}, now)
	if !f.Cancelled || len(f.Outbox) != 4 || f.Outbox[0].Topic != matchstore.TopicRating || f.Outbox[0].Key != "match:"+s.ID {
		t.Fatalf("finished %+v", f)
	}
	outcome := f.Outbox[0].Payload.(matchstore.Outcome)
	if !outcome.Cancelled || !outcome.BotSeat || len(outcome.Players) != 3 || outcome.Players[2].Account != "c" || outcome.Players[2].Team != int(rules.SeatWest.Team()) || outcome.Players[0].Won {
		t.Fatalf("outcome %+v", outcome)
	}
	if a, b, c := outcome.Players[0], outcome.Players[1], outcome.Players[2]; !a.Offender || a.Returned || !b.Offender || !b.Returned || c.Offender {
		t.Fatalf("offenders %+v", outcome.Players)
	}
}

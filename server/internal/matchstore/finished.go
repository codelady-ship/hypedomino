package matchstore

import (
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/replay"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func FinishedFrom(started Started, record replay.Record, endedAt time.Time) Finished {
	finished := Finished{
		ID:          started.ID,
		StartedAt:   started.StartedAt,
		EndedAt:     endedAt,
		HandsPlayed: record.Result.HandsPlayed,
		Scores:      record.Result.Scores,
		Winner:      record.Result.Winner,
		HasWinner:   record.Result.HasWinner,
		Cancelled:   !record.HasResult,
		Results:     map[rules.Seat]string{},
	}
	if data, err := record.MarshalBinary(); err == nil {
		finished.Replay = data
	}
	for _, p := range started.Participants {
		finished.Results[p.Seat] = seatResult(finished, p.Seat)
	}
	return finished
}

func seatResult(f Finished, seat rules.Seat) string {
	switch {
	case f.Cancelled || !f.HasWinner:
		return ResultCancelled
	case seat.Team() == f.Winner:
		return ResultWin
	default:
		return ResultLoss
	}
}

package partnerfit

import (
	"errors"
	"fmt"

	"github.com/Alievs-corp/hypedomino/server/internal/replay"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

var ErrNotTwoVsTwo = errors.New("partnerfit: only 2v2 matches have partners")

type Count struct {
	Decisions int
	Fits      int
}

func Measure(record replay.Record) (map[rules.Seat]Count, error) {
	if record.Mode != rules.Mode2v2 {
		return nil, ErrNotTwoVsTwo
	}
	m := rules.NewMatch(record.Mode, record.Config, rules.NewSeededRandom(record.Seed))
	counts := map[rules.Seat]Count{}
	moves := make([]rules.Move, 0, 4*rules.SetSize)
	for index, intent := range record.Intents {
		if intent.Kind == replay.KindPlace {
			moves = m.LegalMoves(intent.Seat, moves[:0])
			judge(counts, m, intent, moves)
		}
		if err := replay.Apply(m, intent); err != nil {
			return nil, fmt.Errorf("%w: intent %d: %v", replay.ErrIntentRefused, index, err)
		}
	}
	return counts, nil
}

func judge(counts map[rules.Seat]Count, m *rules.Match, intent replay.Intent, moves []rules.Move) {
	partner, ok := intent.Seat.Partner()
	if !ok || len(moves) < 2 {
		return
	}
	hand := m.Hand(partner)
	best, worst, chosen := -1, len(hand)+1, -1
	for _, move := range moves {
		fit := Fitting(m.Board(), hand, move)
		best, worst = max(best, fit), min(worst, fit)
		if move.Tile == intent.Tile && move.End == intent.End {
			chosen = fit
		}
	}
	if best == worst || chosen < 0 {
		return
	}
	count := counts[intent.Seat]
	count.Decisions++
	if chosen == best {
		count.Fits++
	}
	counts[intent.Seat] = count
}

func Fitting(board *rules.Board, hand rules.Hand, move rules.Move) int {
	trial := *board
	if trial.Place(move.Tile, move.End) != nil {
		return 0
	}
	fitting := 0
	for _, tile := range hand {
		if trial.HasLegalMove(rules.Hand{tile}) {
			fitting++
		}
	}
	return fitting
}

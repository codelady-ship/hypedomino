package replay

import "github.com/Alievs-corp/hypedomino/server/internal/rules"

type SeatStats struct {
	HandsWon     int
	Dominoes     int
	BoardPoints  int
	DoubleChains int
}

func (t Timeline) Stats() map[rules.Seat]SeatStats {
	stats := map[rules.Seat]SeatStats{}
	for _, hand := range t.Hands {
		left := handStats(hand, stats)
		if hand.Outcome == nil {
			continue
		}
		if last := len(hand.Steps) - 1; last >= 0 && placed(hand.Steps[last]) && left[hand.Steps[last].Seat] == 0 {
			s := stats[hand.Steps[last].Seat]
			s.Dominoes++
			stats[hand.Steps[last].Seat] = s
		}
		if winner := hand.Outcome.Winner; winner != nil {
			for seat := range hand.Dealt {
				if seat.Team() == *winner {
					s := stats[seat]
					s.HandsWon++
					stats[seat] = s
				}
			}
		}
	}
	return stats
}

func placed(step Step) bool {
	return step.Kind == kindNames[KindOpen] || step.Kind == kindNames[KindPlace]
}

func handStats(hand Hand, stats map[rules.Seat]SeatStats) map[rules.Seat]int {
	left := make(map[rules.Seat]int, len(hand.Dealt))
	for seat, tiles := range hand.Dealt {
		left[seat] = len(tiles)
		stats[seat] = stats[seat]
	}
	var turnSeat rules.Seat
	turnOpen, doubles := false, 0
	for _, step := range hand.Steps {
		if !turnOpen || turnSeat != step.Seat {
			turnSeat, turnOpen, doubles = step.Seat, true, 0
		}
		s := stats[step.Seat]
		switch {
		case step.Kind == kindNames[KindDraw]:
			left[step.Seat]++
		case placed(step):
			left[step.Seat]--
			s.BoardPoints += step.Points
			if step.Tile.IsDouble() {
				doubles++
				if doubles == 2 {
					s.DoubleChains++
				}
			}
		default:
			turnOpen = false
		}
		stats[step.Seat] = s
	}
	return left
}

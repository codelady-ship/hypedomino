package rules

import "fmt"

type HandStatus uint8

const (
	HandContinues HandStatus = iota
	HandDomino
	HandBlocked
	handStatusCount
)

var handStatusNames = [handStatusCount]string{"continue", "domino", "block"}

func (s HandStatus) String() string {
	if s >= handStatusCount {
		return "invalid"
	}
	return handStatusNames[s]
}

func (s HandStatus) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

func (s *HandStatus) UnmarshalText(text []byte) error {
	for status, name := range handStatusNames {
		if name == string(text) {
			*s = HandStatus(status)
			return nil
		}
	}
	return fmt.Errorf("rules: invalid hand status %q", text)
}

func (m Mode) HandStatus(board *Board, hands map[Seat]Hand, boneyard Boneyard) HandStatus {
	for _, seat := range m.Seats() {
		if len(hands[seat]) == 0 {
			return HandDomino
		}
	}
	if m == Mode1v1 && len(boneyard) > 1 {
		return HandContinues
	}
	for _, seat := range m.Seats() {
		if board.HasLegalMove(hands[seat]) {
			return HandContinues
		}
	}
	return HandBlocked
}

func HandPoints(pips int, config Config) int {
	if pips <= 0 {
		return 0
	}
	multiple := config.ScoringMultiple
	return (pips + multiple - 1) / multiple * multiple
}

type TeamPoints [teamCount]int

func (m Mode) TeamPips(hands map[Seat]Hand) TeamPoints {
	var pips TeamPoints
	for _, seat := range m.Seats() {
		pips[seat.Team()] += hands[seat].Pips()
	}
	return pips
}

type HandScores struct {
	AtHandStart TeamPoints
	Current     TeamPoints
	CarryOver   int
}

type HandResult struct {
	Pips       TeamPoints
	Winner     Team
	HasWinner  bool
	Awarded    TeamPoints
	CarryOver  int
	Scores     TeamPoints
	NextOpener Team
}

type HandEnd struct {
	Status    HandStatus
	Locked    bool
	CarriedIn int
	Left      map[Seat]Hand
}

func DominoResult(mode Mode, config Config, lastPlacedBy Seat, hands map[Seat]Hand, scores HandScores) HandResult {
	winner := lastPlacedBy.Team()
	result := HandResult{
		Pips:       mode.TeamPips(hands),
		Winner:     winner,
		HasWinner:  true,
		Scores:     scores.Current,
		NextOpener: winner,
	}
	result.award(winner, HandPoints(result.Pips[winner.Other()], config), scores, config)
	return result
}

func BlockResult(mode Mode, config Config, lastPlacedBy Seat, hands map[Seat]Hand, scores HandScores) HandResult {
	result := HandResult{
		Pips:       mode.TeamPips(hands),
		Scores:     scores.Current,
		NextOpener: lastPlacedBy.Team(),
	}
	if result.Pips[TeamA] == result.Pips[TeamB] {
		result.CarryOver = scores.CarryOver + HandPoints(result.Pips[TeamA]+result.Pips[TeamB], config)
		return result
	}
	winner := TeamA
	if result.Pips[TeamB] < result.Pips[TeamA] {
		winner = TeamB
	}
	result.Winner = winner
	result.HasWinner = true
	result.award(winner, HandPoints(result.Pips[winner.Other()], config), scores, config)
	return result
}
func (r *HandResult) award(winner Team, points int, scores HandScores, config Config) {
	if scores.AtHandStart[winner] >= config.HandPointsLockThreshold {
		return
	}
	r.Awarded[winner] = points + scores.CarryOver
	r.Scores[winner] += r.Awarded[winner]
}

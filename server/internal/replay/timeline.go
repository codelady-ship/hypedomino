package replay

import (
	"fmt"
	"slices"

	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

var kindNames = map[Kind]string{KindOpen: "open", KindPlace: "place", KindEndTurn: "end_turn", KindDraw: "draw", KindPass: "pass"}

type Step struct {
	Seat   rules.Seat  `json:"seat"`
	Kind   string      `json:"kind"`
	Tile   *rules.Tile `json:"tile,omitempty"`
	End    *rules.End  `json:"end,omitempty"`
	Points int         `json:"points,omitempty"`
}

type HandOutcome struct {
	Pips      rules.TeamPoints `json:"pips"`
	Awarded   rules.TeamPoints `json:"awarded"`
	Scores    rules.TeamPoints `json:"scores"`
	CarryOver int              `json:"carry_over"`
	Winner    *rules.Team      `json:"winner"`
}

type Hand struct {
	Number  int                       `json:"number"`
	Dealt   map[rules.Seat]rules.Hand `json:"dealt"`
	Steps   []Step                    `json:"steps"`
	Outcome *HandOutcome              `json:"outcome"`
}

type Timeline struct {
	Hands   []Hand `json:"hands"`
	Problem string `json:"problem,omitempty"`
}

func NewTimeline(record Record) Timeline {
	m := rules.NewMatch(record.Mode, record.Config, rules.NewSeededRandom(record.Seed))
	timeline := Timeline{Hands: []Hand{}}
	hand := dealt(m, 1)
	for index, intent := range record.Intents {
		step, err := apply(m, intent)
		if err != nil {
			timeline.Problem = fmt.Sprintf("%v: intent %d: %v", ErrIntentRefused, index, err)
			break
		}
		hand.Steps = append(hand.Steps, step)
		if m.HandsPlayed() < hand.Number {
			continue
		}
		result, _ := m.LastResult()
		hand.Outcome = outcomeOf(result)
		timeline.Hands = append(timeline.Hands, hand)
		hand = dealt(m, hand.Number+1)
	}
	if len(hand.Steps) > 0 {
		timeline.Hands = append(timeline.Hands, hand)
	}
	if timeline.Problem == "" {
		if err := Verify(record); err != nil {
			timeline.Problem = err.Error()
		}
	}
	return timeline
}

func dealt(m *rules.Match, number int) Hand {
	hands := make(map[rules.Seat]rules.Hand, len(m.Mode().Seats()))
	for _, seat := range m.Mode().Seats() {
		hands[seat] = slices.Clone(m.Hand(seat))
	}
	return Hand{Number: number, Dealt: hands, Steps: []Step{}}
}

func apply(m *rules.Match, intent Intent) (Step, error) {
	step := Step{Seat: intent.Seat, Kind: kindNames[intent.Kind]}
	switch intent.Kind {
	case KindOpen:
		points, err := m.Open(intent.Seat, intent.Tile)
		step.Tile, step.Points = &intent.Tile, points
		return step, err
	case KindPlace:
		placement, err := m.Place(intent.Seat, intent.Tile, intent.End)
		step.Tile, step.End, step.Points = &intent.Tile, &intent.End, placement.Score
		return step, err
	case KindDraw:
		tile, err := m.Draw(intent.Seat)
		step.Tile = &tile
		return step, err
	}
	return step, Apply(m, intent)
}

func outcomeOf(result rules.HandResult) *HandOutcome {
	outcome := &HandOutcome{Pips: result.Pips, Awarded: result.Awarded, Scores: result.Scores, CarryOver: result.CarryOver}
	if result.HasWinner {
		outcome.Winner = &result.Winner
	}
	return outcome
}

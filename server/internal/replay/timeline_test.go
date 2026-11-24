package replay

import (
	"strconv"
	"strings"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func TestATimelineShowsEveryHandDealtPlayedAndScored(t *testing.T) {
	for _, mode := range []rules.Mode{rules.Mode1v1, rules.Mode2v2} {
		record := recordMatch(t, mode, 41)
		timeline := NewTimeline(record)
		if timeline.Problem != "" || len(timeline.Hands) != record.Result.HandsPlayed {
			t.Fatalf("%s: problem %q, %d hands for %d played", mode, timeline.Problem, len(timeline.Hands), record.Result.HandsPlayed)
		}
		steps := 0
		for index, hand := range timeline.Hands {
			if hand.Number != index+1 || len(hand.Dealt) != len(mode.Seats()) || hand.Outcome == nil {
				t.Fatalf("%s hand %d: %+v", mode, index, hand)
			}
			for _, step := range hand.Steps {
				if step.Kind == "" || (step.Kind == "draw" || step.Kind == "place" || step.Kind == "open") != (step.Tile != nil) || (step.Kind == "place") != (step.End != nil) {
					t.Fatalf("%s hand %d step %+v", mode, hand.Number, step)
				}
			}
			if hand.Steps[0].Kind != "open" || !hand.Dealt[hand.Steps[0].Seat].Contains(*hand.Steps[0].Tile) {
				t.Fatalf("%s hand %d opens with %+v from %v", mode, hand.Number, hand.Steps[0], hand.Dealt)
			}
			steps += len(hand.Steps)
		}
		last := timeline.Hands[len(timeline.Hands)-1].Outcome
		if steps != len(record.Intents) || last.Scores != record.Result.Scores {
			t.Fatalf("%s: %d steps for %d intents, last %+v, result %+v", mode, steps, len(record.Intents), last, record.Result)
		}
	}
}

func TestATimelineNamesWhereARecordStopsMakingSense(t *testing.T) {
	record := recordMatch(t, rules.Mode2v2, 7)
	record.Result.Scores[rules.TeamA]++
	if timeline := NewTimeline(record); !strings.Contains(timeline.Problem, ErrResultDiffers.Error()) || len(timeline.Hands) != record.Result.HandsPlayed {
		t.Fatalf("result: %q %d", timeline.Problem, len(timeline.Hands))
	}

	record = recordMatch(t, rules.Mode2v2, 7)
	broken := len(record.Intents) / 2
	for record.Intents[broken].Kind != KindPlace {
		broken++
	}
	record.Intents[broken] = Intent{Kind: KindPass, Seat: record.Intents[broken].Seat}
	timeline := NewTimeline(record)
	steps := 0
	for _, hand := range timeline.Hands {
		steps += len(hand.Steps)
	}
	if !strings.Contains(timeline.Problem, "intent "+strconv.Itoa(broken)+":") || steps != broken {
		t.Fatalf("intent: %q, %d steps before intent %d", timeline.Problem, steps, broken)
	}
}

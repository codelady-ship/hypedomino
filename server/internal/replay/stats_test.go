package replay

import (
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func step(seat rules.Seat, kind string, tile string, points int) Step {
	s := Step{Seat: seat, Kind: kind, Points: points}
	if tile != "" {
		parsed, _ := rules.ParseTile(tile)
		s.Tile = &parsed
	}
	return s
}

func dealtTiles(texts ...string) rules.Hand {
	hand := rules.Hand{}
	for _, text := range texts {
		tile, _ := rules.ParseTile(text)
		hand = append(hand, tile)
	}
	return hand
}

func TestSeatStatsCountHandsWonDominoesBoardPointsAndDoubleChains(t *testing.T) {
	x, y := rules.SeatX, rules.SeatY
	winner := x.Team()
	timeline := Timeline{Hands: []Hand{
		{
			Number: 1,
			Dealt:  map[rules.Seat]rules.Hand{x: dealtTiles("5-5", "6-6", "0-1"), y: dealtTiles("1-2", "3-4")},
			Steps: []Step{
				step(x, "open", "5-5", 0), step(x, "place", "6-6", 10), step(x, "end_turn", "", 0),
				step(y, "draw", "2-3", 0), step(y, "place", "1-2", 5), step(y, "end_turn", "", 0),
				step(x, "place", "0-1", 0),
			},
			Outcome: &HandOutcome{Winner: &winner},
		},
		{
			Number: 2,
			Dealt:  map[rules.Seat]rules.Hand{x: dealtTiles("0-0"), y: dealtTiles("3-3", "4-4")},
			Steps:  []Step{step(y, "open", "3-3", 0), step(y, "place", "4-4", 15)},
		},
	}}
	want := map[rules.Seat]SeatStats{
		x: {HandsWon: 1, Dominoes: 1, BoardPoints: 10, DoubleChains: 1},
		y: {BoardPoints: 20, DoubleChains: 1},
	}
	got := timeline.Stats()
	for seat, expected := range want {
		if got[seat] != expected {
			t.Fatalf("seat %s: got %+v, want %+v", seat, got[seat], expected)
		}
	}
}

func TestSeatStatsOfARecordedMatchAddUpToItsResult(t *testing.T) {
	record := recordMatch(t, rules.Mode2v2, 41)
	timeline := NewTimeline(record)
	stats := timeline.Stats()
	hands := 0
	for _, seat := range rules.Mode2v2.Seats() {
		if seat.Team() == rules.TeamA {
			hands += stats[seat].HandsWon
		}
	}
	if len(stats) != 4 || hands > record.Result.HandsPlayed {
		t.Fatalf("stats %+v for %d hands", stats, record.Result.HandsPlayed)
	}
}

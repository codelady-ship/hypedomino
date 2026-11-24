package partnerfit

import (
	"errors"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/replay"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func TestFittingCountsThePartnerTilesLeftPlaceable(t *testing.T) {
	board := rules.NewBoard(rules.MustTile(3, 5))
	partner := rules.Hand{rules.MustTile(6, 6), rules.MustTile(1, 6), rules.MustTile(0, 3), rules.MustTile(2, 4)}
	cases := []struct {
		move rules.Move
		want int
	}{
		{rules.Move{Tile: rules.MustTile(5, 6), End: rules.EndRight}, 3},
		{rules.Move{Tile: rules.MustTile(3, 4), End: rules.EndLeft}, 1},
		{rules.Move{Tile: rules.MustTile(1, 1), End: rules.EndLeft}, 0},
	}
	for _, c := range cases {
		if got := Fitting(&board, partner, c.move); got != c.want {
			t.Errorf("%v on %v leaves %d partner tiles placeable, want %d", c.move.Tile, c.move.End, got, c.want)
		}
	}
	if board.TileCount() != 1 {
		t.Fatal("measuring changed the board")
	}
}

func playMatch(t *testing.T, seed uint64, colluder rules.Seat) replay.Record {
	t.Helper()
	recorder := replay.NewRecorder(rules.Mode2v2, rules.DefaultConfig(), seed, [32]byte{})
	m := recorder.Match
	random := rules.NewSeededRandom(seed + 7)
	moves := make([]rules.Move, 0, 4*rules.SetSize)
	for step := 0; m.Phase() != rules.PhaseFinished; step++ {
		if step > 20000 {
			t.Fatalf("seed %d: match does not end", seed)
		}
		if err := act(recorder, random, moves, colluder); err != nil {
			t.Fatalf("seed %d step %d: %v", seed, step, err)
		}
	}
	return recorder.Record()
}

func act(r *replay.Recorder, random rules.Random, buffer []rules.Move, colluder rules.Seat) error {
	m := r.Match
	if m.Phase() == rules.PhaseOpening {
		opening, forced := m.ForcedOpening()
		if !forced {
			seats := m.Mode().TeamSeats(m.OpeningTeam(), make([]rules.Seat, 0, 2))
			opening.Seat = seats[random.IntN(len(seats))]
			hand := m.Hand(opening.Seat)
			opening.Tile = hand[random.IntN(len(hand))]
		}
		_, err := r.Open(opening.Seat, opening.Tile)
		return err
	}
	seat := m.Current()
	moves := m.LegalMoves(seat, buffer)
	if len(moves) == 0 {
		if m.Turn().Placements() > 0 {
			return r.EndTurn(seat)
		}
		return r.Pass(seat)
	}
	move := moves[random.IntN(len(moves))]
	if seat == colluder {
		partner, _ := seat.Partner()
		for _, candidate := range moves {
			if Fitting(m.Board(), m.Hand(partner), candidate) > Fitting(m.Board(), m.Hand(partner), move) {
				move = candidate
			}
		}
	}
	_, err := r.Place(seat, move.Tile, move.End)
	return err
}

func TestAPlayerWhoSeesThePartnerHandAlwaysFits(t *testing.T) {
	var colluder, honest Count
	for seed := uint64(1); seed <= 20; seed++ {
		counts, err := Measure(playMatch(t, seed, rules.SeatNorth))
		if err != nil {
			t.Fatal(err)
		}
		colluder.Decisions += counts[rules.SeatNorth].Decisions
		colluder.Fits += counts[rules.SeatNorth].Fits
		honest.Decisions += counts[rules.SeatEast].Decisions
		honest.Fits += counts[rules.SeatEast].Fits
	}
	if colluder.Decisions < 100 || colluder.Fits != colluder.Decisions {
		t.Fatalf("colluder %+v", colluder)
	}
	if honest.Decisions < 100 || honest.Fits*100/honest.Decisions > 80 {
		t.Fatalf("random player fits like a colluder: %+v", honest)
	}
}

func TestOnlyReplayableTwoVsTwoRecordsAreMeasured(t *testing.T) {
	if _, err := Measure(replay.Record{Mode: rules.Mode1v1}); !errors.Is(err, ErrNotTwoVsTwo) {
		t.Fatalf("1v1: %v", err)
	}
	record := playMatch(t, 3, rules.SeatNorth)
	record.Intents = append(record.Intents[:5:5], replay.Intent{Kind: replay.KindPlace, Seat: rules.SeatNorth, Tile: rules.MustTile(6, 6), End: rules.EndLeft})
	if _, err := Measure(record); !errors.Is(err, replay.ErrIntentRefused) {
		t.Fatalf("tampered record: %v", err)
	}
}

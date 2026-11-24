package rules

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

const vectorsDir = "../../../docs/rules-vectors"

type vector struct {
	ID     string
	Kind   string
	Mode   Mode
	Given  json.RawMessage
	Expect json.RawMessage
}

type vectorFile struct {
	Vectors []vector
}

var vectorRunners = map[string]func(*testing.T, vector){
	"deal":           runDealVector,
	"seat":           runSeatVector,
	"place":          runPlaceVector,
	"draw":           runDrawVector,
	"opening-score":  runOpeningScoreVector,
	"turn":           runTurnVector,
	"hand-status":    runHandStatusVector,
	"round":          runRoundVector,
	"hand-end":       runHandEndVector,
	"first-opener":   runFirstOpenerVector,
	"match-end":      runMatchEndVector,
	"turn-timer":     runTurnTimerVector,
	"afk":            runAfkVector,
	"doubles-redeal": runDoublesRedealVector,
}

func TestRuleVectors(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join(vectorsDir, "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	ran := 0
	for _, path := range paths {
		if filepath.Base(path) == "schema.json" {
			continue
		}
		for _, v := range loadVectors(t, path) {
			t.Run(v.ID, func(t *testing.T) {
				run, ok := vectorRunners[v.Kind]
				if !ok {
					t.Fatalf("unknown vector kind %q", v.Kind)
				}
				run(t, v)
			})
			ran++
		}
	}
	if ran == 0 {
		t.Fatalf("no vectors found in %s", vectorsDir)
	}
}

func loadVectors(t *testing.T, path string) []vector {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var file vectorFile
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return file.Vectors
}

func decodeVector(t *testing.T, raw json.RawMessage, into any) {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(into); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
}

func runDealVector(t *testing.T, v vector) {
	var given struct {
		Hands    map[Seat]Hand
		Boneyard Boneyard
	}
	var expect struct {
		Valid bool
	}
	decodeVector(t, v.Given, &given)
	decodeVector(t, v.Expect, &expect)
	err := ValidateDeal(v.Mode, DefaultConfig(), Deal{Hands: given.Hands, Boneyard: given.Boneyard})
	if (err == nil) != expect.Valid {
		t.Fatalf("valid = %v (%v), want %v", err == nil, err, expect.Valid)
	}
}

type vectorBoard struct {
	Line    []Oriented
	Spinner *Oriented
	Top     []Oriented
	Bottom  []Oriented
}

func boardFromVector(t *testing.T, vb vectorBoard) Board {
	t.Helper()
	board := Board{head: SetSize, tail: SetSize}
	for _, tile := range vb.Line {
		board.line[board.tail] = tile
		board.tail++
	}
	for _, tile := range vb.Top {
		board.top.push(tile)
	}
	for _, tile := range vb.Bottom {
		board.bottom.push(tile)
	}
	if vb.Spinner != nil {
		board.spinner = *vb.Spinner
		board.hasSpinner = true
	}
	return board
}

func boardToVector(board *Board) vectorBoard {
	vb := vectorBoard{
		Line:   append([]Oriented{}, board.Line()...),
		Top:    append([]Oriented{}, board.Top()...),
		Bottom: append([]Oriented{}, board.Bottom()...),
	}
	if board.hasSpinner {
		spinner := board.spinner
		vb.Spinner = &spinner
	}
	return vb
}

func openEndsToVector(ends OpenEnds) map[End]int {
	values := map[End]int{}
	for end := EndLeft; end < endCount; end++ {
		if value, open := ends.Value(end); open {
			values[end] = value
		}
	}
	return values
}

func runPlaceVector(t *testing.T, v vector) {
	var given struct {
		Board                vectorBoard
		Tile                 Tile
		End                  End
		TeamScoreAtHandStart int
	}
	var expect struct {
		Legal    bool
		Board    vectorBoard
		OpenEnds map[End]int
		Total    int
		Score    int
	}
	decodeVector(t, v.Given, &given)
	decodeVector(t, v.Expect, &expect)
	board := boardFromVector(t, given.Board)
	before := board
	err := board.Place(given.Tile, given.End)
	if (err == nil) != expect.Legal {
		t.Fatalf("legal = %v (%v), want %v", err == nil, err, expect.Legal)
	}
	if !expect.Legal {
		if board != before {
			t.Fatal("a refused placement changed the board")
		}
		return
	}
	if got := boardToVector(&board); !reflect.DeepEqual(got, expect.Board) {
		t.Fatalf("board = %+v, want %+v", got, expect.Board)
	}
	ends := board.OpenEnds()
	if got := openEndsToVector(ends); !reflect.DeepEqual(got, expect.OpenEnds) {
		t.Fatalf("openEnds = %v, want %v", got, expect.OpenEnds)
	}
	if ends.Total() != expect.Total {
		t.Fatalf("total = %d, want %d", ends.Total(), expect.Total)
	}
	if score := PlacementScore(ends.Total(), DefaultConfig()); score != expect.Score {
		t.Fatalf("score = %d, want %d", score, expect.Score)
	}
}

func runDrawVector(t *testing.T, v vector) {
	var given struct {
		Board    vectorBoard
		Hand     Hand
		Boneyard Boneyard
	}
	var expect struct {
		Drawn   []Tile
		Outcome TurnOutcome
	}
	decodeVector(t, v.Given, &given)
	decodeVector(t, v.Expect, &expect)
	board := boardFromVector(t, given.Board)
	var buffer [SetSize]Tile
	drawn, outcome := DrawUntilPlayable(v.Mode, &board, &given.Hand, &given.Boneyard, buffer[:0])
	if outcome != expect.Outcome {
		t.Fatalf("outcome = %v, want %v", outcome, expect.Outcome)
	}
	if !sameTiles(drawn, expect.Drawn) {
		t.Fatalf("drawn = %v, want %v", drawn, expect.Drawn)
	}
	if (outcome == OutcomePlace) != board.HasLegalMove(given.Hand) {
		t.Fatalf("outcome %v but HasLegalMove %v", outcome, board.HasLegalMove(given.Hand))
	}
}

func runOpeningScoreVector(t *testing.T, v vector) {
	var given struct {
		FirstHand            bool
		Tile                 Tile
		TeamScoreAtHandStart int
	}
	var expect struct {
		Score int
	}
	decodeVector(t, v.Given, &given)
	decodeVector(t, v.Expect, &expect)
	score := OpeningScore(DefaultConfig(), given.FirstHand, given.Tile, given.TeamScoreAtHandStart)
	if score != expect.Score {
		t.Fatalf("score = %d, want %d", score, expect.Score)
	}
}

func runTurnVector(t *testing.T, v vector) {
	var given struct {
		Board      vectorBoard
		Placements []struct {
			Tile Tile
			End  End
		}
	}
	var expect struct {
		Steps []struct {
			Placed bool
			Total  int
			Score  int
		}
		TurnScore int
	}
	decodeVector(t, v.Given, &given)
	decodeVector(t, v.Expect, &expect)
	board := boardFromVector(t, given.Board)
	turn := BeginTurn(&board, DefaultConfig())
	if len(expect.Steps) != len(given.Placements) {
		t.Fatalf("%d steps for %d placements", len(expect.Steps), len(given.Placements))
	}
	for i, placement := range given.Placements {
		step := expect.Steps[i]
		result, err := turn.Place(placement.Tile, placement.End)
		if (err == nil) != step.Placed {
			t.Fatalf("step %d placed = %v (%v), want %v", i+1, err == nil, err, step.Placed)
		}
		if !step.Placed {
			if i != len(given.Placements)-1 {
				t.Fatalf("step %d refused before the last placement", i+1)
			}
			break
		}
		if result.Total != step.Total || result.Score != step.Score {
			t.Fatalf("step %d = %+v, want total %d score %d", i+1, result, step.Total, step.Score)
		}
	}
	if turn.Score() != expect.TurnScore {
		t.Fatalf("turnScore = %d, want %d", turn.Score(), expect.TurnScore)
	}
}

func runSeatVector(t *testing.T, v vector) {
	var given struct {
		Seat Seat
	}
	var expect struct {
		Team Team
		Next Seat
	}
	decodeVector(t, v.Given, &given)
	decodeVector(t, v.Expect, &expect)
	if given.Seat.Team() != expect.Team {
		t.Fatalf("team = %s, want %s", given.Seat.Team(), expect.Team)
	}
	if given.Seat.Next() != expect.Next {
		t.Fatalf("next = %s, want %s", given.Seat.Next(), expect.Next)
	}
}

func runHandStatusVector(t *testing.T, v vector) {
	var given struct {
		Board    vectorBoard
		Hands    map[Seat]Hand
		Boneyard Boneyard
	}
	var expect struct {
		Status HandStatus
	}
	decodeVector(t, v.Given, &given)
	decodeVector(t, v.Expect, &expect)
	board := boardFromVector(t, given.Board)
	if got := v.Mode.HandStatus(&board, given.Hands, given.Boneyard); got != expect.Status {
		t.Fatalf("status = %s, want %s", got, expect.Status)
	}
}

func runRoundVector(t *testing.T, v vector) {
	var given struct {
		Pips int
	}
	var expect struct {
		Points int
	}
	decodeVector(t, v.Given, &given)
	decodeVector(t, v.Expect, &expect)
	if got := HandPoints(given.Pips, DefaultConfig()); got != expect.Points {
		t.Fatalf("HandPoints(%d) = %d, want %d", given.Pips, got, expect.Points)
	}
}

type vectorTeamPoints struct {
	A int
	B int
}

func (p vectorTeamPoints) points() TeamPoints {
	return TeamPoints{p.A, p.B}
}

func runHandEndVector(t *testing.T, v vector) {
	var given struct {
		End               HandStatus
		LastPlacedBy      Seat
		Hands             map[Seat]Hand
		Boneyard          Boneyard
		ScoresAtHandStart vectorTeamPoints
		Scores            vectorTeamPoints
		CarryOver         int
	}
	var expect struct {
		Pips            vectorTeamPoints
		Winner          *Team
		Awarded         vectorTeamPoints
		CarryOver       int
		Scores          vectorTeamPoints
		NextOpeningTeam Team
	}
	decodeVector(t, v.Given, &given)
	decodeVector(t, v.Expect, &expect)
	scores := HandScores{
		AtHandStart: given.ScoresAtHandStart.points(),
		Current:     given.Scores.points(),
		CarryOver:   given.CarryOver,
	}
	settle := DominoResult
	if given.End == HandBlocked {
		settle = BlockResult
	}
	got := settle(v.Mode, DefaultConfig(), given.LastPlacedBy, given.Hands, scores)
	want := HandResult{
		Pips:       expect.Pips.points(),
		HasWinner:  expect.Winner != nil,
		Awarded:    expect.Awarded.points(),
		CarryOver:  expect.CarryOver,
		Scores:     expect.Scores.points(),
		NextOpener: expect.NextOpeningTeam,
	}
	if expect.Winner != nil {
		want.Winner = *expect.Winner
	}
	if got != want {
		t.Fatalf("result = %+v, want %+v", got, want)
	}
}

func runFirstOpenerVector(t *testing.T, v vector) {
	var given struct {
		Hands    map[Seat]Hand
		Boneyard Boneyard
	}
	var expect struct {
		Redeal bool
		Opener *Seat
		Tile   *Tile
		Next   *Seat
	}
	decodeVector(t, v.Given, &given)
	decodeVector(t, v.Expect, &expect)
	opening, ok := FirstOpener(v.Mode, DefaultConfig(), given.Hands)
	if ok == expect.Redeal {
		t.Fatalf("redeal = %v, want %v", !ok, expect.Redeal)
	}
	if !ok {
		return
	}
	if opening.Seat != *expect.Opener || opening.Tile != *expect.Tile || opening.Seat.Next() != *expect.Next {
		t.Fatalf("opening = %s %s next %s, want %s %s next %s", opening.Seat, opening.Tile, opening.Seat.Next(), *expect.Opener, *expect.Tile, *expect.Next)
	}
}

func runDoublesRedealVector(t *testing.T, v vector) {
	var given struct {
		Hands    map[Seat]Hand
		Boneyard Boneyard
	}
	var expect struct {
		Redeal bool
	}
	decodeVector(t, v.Given, &given)
	decodeVector(t, v.Expect, &expect)
	if err := ValidateDeal(v.Mode, DefaultConfig(), Deal{Hands: given.Hands, Boneyard: given.Boneyard}); err != nil {
		t.Fatalf("deal: %v", err)
	}
	if got := NeedsDoublesRedeal(v.Mode, DefaultConfig(), given.Hands); got != expect.Redeal {
		t.Fatalf("redeal = %v, want %v", got, expect.Redeal)
	}
}

func runMatchEndVector(t *testing.T, v vector) {
	var given struct {
		End    string
		Scores vectorTeamPoints
	}
	var expect struct {
		Outcome MatchOutcome
		Winner  *Team
	}
	decodeVector(t, v.Given, &given)
	decodeVector(t, v.Expect, &expect)
	end := HandContinues
	if given.End != "none" {
		if err := end.UnmarshalText([]byte(given.End)); err != nil {
			t.Fatal(err)
		}
	}
	outcome, winner := MatchEnd(DefaultConfig(), end, given.Scores.points())
	if outcome != expect.Outcome {
		t.Fatalf("outcome = %s, want %s", outcome, expect.Outcome)
	}
	if (outcome == MatchWon) != (expect.Winner != nil) {
		t.Fatalf("winner presence does not match outcome %s", outcome)
	}
	if expect.Winner != nil && winner != *expect.Winner {
		t.Fatalf("winner = %s, want %s", winner, *expect.Winner)
	}
}

func runTurnTimerVector(t *testing.T, v vector) {
	var given struct {
		TurnSeconds    float64
		ActedAtSeconds *float64
	}
	var expect struct {
		MovedBy Controller
	}
	decodeVector(t, v.Given, &given)
	decodeVector(t, v.Expect, &expect)
	config := DefaultConfig()
	config.TurnSeconds = int(given.TurnSeconds)
	if float64(config.TurnSeconds) != given.TurnSeconds {
		t.Fatalf("turnSeconds %v is not a whole number of seconds", given.TurnSeconds)
	}
	actedAfter := time.Duration(0)
	if given.ActedAtSeconds != nil {
		actedAfter = time.Duration(*given.ActedAtSeconds * float64(time.Second))
	}
	if got := MovedBy(config, given.ActedAtSeconds != nil, actedAfter); got != expect.MovedBy {
		t.Fatalf("movedBy = %s, want %s", got, expect.MovedBy)
	}
}

func runAfkVector(t *testing.T, v vector) {
	var given struct {
		AfkTimeouts int
		Events      []struct {
			Seat Seat
			Type string
		}
	}
	var expect struct {
		Controllers map[Seat]Controller
	}
	decodeVector(t, v.Given, &given)
	decodeVector(t, v.Expect, &expect)
	config := DefaultConfig()
	config.AfkConsecutiveTimeouts = given.AfkTimeouts
	presence := NewPresence(config)
	for _, event := range given.Events {
		switch event.Type {
		case "timeout":
			presence.Timeout(event.Seat)
		case "move":
			presence.Move(event.Seat)
		case "reconnect":
			presence.Reconnect(event.Seat)
		case "leave":
			presence.Leave(event.Seat)
		default:
			t.Fatalf("unknown event %q", event.Type)
		}
	}
	for seat, want := range expect.Controllers {
		if got := presence.Controller(seat); got != want {
			t.Fatalf("%s controlled by %s, want %s", seat, got, want)
		}
	}
}

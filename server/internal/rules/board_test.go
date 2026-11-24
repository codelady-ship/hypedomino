package rules

import (
	"errors"
	"testing"
)

func mustPlace(t *testing.T, board *Board, text string, end End) {
	t.Helper()
	tile, err := ParseTile(text)
	if err != nil {
		t.Fatal(err)
	}
	if !board.CanPlace(tile, end) {
		t.Fatalf("CanPlace(%s, %s) = false", tile, end)
	}
	if err := board.Place(tile, end); err != nil {
		t.Fatalf("Place(%s, %s): %v", tile, end, err)
	}
}

func refuse(t *testing.T, board *Board, text string, end End) {
	t.Helper()
	tile, err := ParseTile(text)
	if err != nil {
		t.Fatal(err)
	}
	before := *board
	if board.CanPlace(tile, end) {
		t.Fatalf("CanPlace(%s, %s) = true", tile, end)
	}
	if err := board.Place(tile, end); !errors.Is(err, ErrIllegalPlacement) {
		t.Fatalf("Place(%s, %s) = %v", tile, end, err)
	}
	if *board != before {
		t.Fatalf("a refused placement changed the board")
	}
}

func lineText(tiles []Oriented) string {
	text := ""
	for _, tile := range tiles {
		text += tile.String() + " "
	}
	return text
}

func TestOrientedParseKeepsOrder(t *testing.T) {
	for _, c := range []struct {
		text string
		want Oriented
	}{
		{"4-2", Oriented{4, 2}},
		{"2-4", Oriented{2, 4}},
		{"3-3", Oriented{3, 3}},
		{"0-6", Oriented{0, 6}},
		{"6-0", Oriented{6, 0}},
	} {
		got, err := ParseOriented(c.text)
		if err != nil || got != c.want || got.String() != c.text {
			t.Fatalf("ParseOriented(%q) = %v, %v", c.text, got, err)
		}
		if got.Tile() != MustTile(c.want.First, c.want.Second) {
			t.Fatalf("%v.Tile() = %v", got, got.Tile())
		}
	}
	for _, text := range []string{"", "7-1", "1", "a-2"} {
		if _, err := ParseOriented(text); !errors.Is(err, ErrInvalidOrientation) {
			t.Errorf("ParseOriented(%q) = %v", text, err)
		}
	}
}

func TestEndText(t *testing.T) {
	for _, text := range []string{"left", "right", "top", "bottom"} {
		end, err := ParseEnd(text)
		if err != nil || end.String() != text {
			t.Fatalf("ParseEnd(%q) = %v, %v", text, end, err)
		}
	}
	if _, err := ParseEnd("middle"); !errors.Is(err, ErrInvalidEnd) {
		t.Errorf("ParseEnd(middle) = %v", err)
	}
	if End(9).String() != "invalid" {
		t.Error("End(9) has a name")
	}
}

func TestOpeningHasTwoEndsAndNoArms(t *testing.T) {
	board := NewBoard(MustTile(2, 3))
	ends := board.OpenEnds()
	left, _ := ends.Value(EndLeft)
	right, _ := ends.Value(EndRight)
	if left != 2 || right != 3 || ends.Total() != 5 {
		t.Fatalf("ends %v total %d", ends, ends.Total())
	}
	if _, open := ends.Value(EndTop); open {
		t.Fatal("top is open without a spinner")
	}
	if _, open := board.EndPip(EndBottom); open {
		t.Fatal("bottom accepts tiles without a spinner")
	}
	refuse(t, &board, "3-5", EndTop)
	refuse(t, &board, "4-5", EndLeft)
	if _, ok := board.Spinner(); ok {
		t.Fatal("an opening tile is a spinner")
	}
}

func TestOpeningDoubleCountsBothHalvesOnBothEnds(t *testing.T) {
	board := NewBoard(MustTile(5, 5))
	if board.OpenEnds().Total() != 20 {
		t.Fatalf("total %d", board.OpenEnds().Total())
	}
	mustPlace(t, &board, "1-5", EndRight)
	left, _ := board.OpenEnds().Value(EndLeft)
	right, _ := board.OpenEnds().Value(EndRight)
	if left != 10 || right != 1 {
		t.Fatalf("ends %d %d", left, right)
	}
}

func TestSpinnerFormsWhenBothSidesAreTaken(t *testing.T) {
	board := NewBoard(MustTile(2, 3))
	mustPlace(t, &board, "3-3", EndRight)
	if _, ok := board.Spinner(); ok {
		t.Fatal("3-3 became the spinner with one side taken")
	}
	refuse(t, &board, "3-5", EndTop)
	mustPlace(t, &board, "2-4", EndLeft)
	if got := lineText(board.Line()); got != "4-2 2-3 3-3 " {
		t.Fatalf("line %q", got)
	}
	mustPlace(t, &board, "1-3", EndRight)
	spinner, ok := board.Spinner()
	if !ok || spinner != MustTile(3, 3) {
		t.Fatalf("spinner %v %v", spinner, ok)
	}
	if board.OpenEnds().Total() != 5 {
		t.Fatalf("empty arms count: total %d", board.OpenEnds().Total())
	}
	refuse(t, &board, "5-6", EndTop)
	mustPlace(t, &board, "3-5", EndTop)
	mustPlace(t, &board, "0-3", EndBottom)
	ends := board.OpenEnds()
	top, _ := ends.Value(EndTop)
	bottom, bottomOpen := ends.Value(EndBottom)
	if top != 5 || bottom != 0 || !bottomOpen || ends.Total() != 10 {
		t.Fatalf("ends %v", ends)
	}
	if got := lineText(board.Top()); got != "3-5 " {
		t.Fatalf("top %q", got)
	}
	if got := lineText(board.Bottom()); got != "3-0 " {
		t.Fatalf("bottom %q", got)
	}
	mustPlace(t, &board, "5-5", EndTop)
	top, _ = board.OpenEnds().Value(EndTop)
	if top != 10 {
		t.Fatalf("double on the arm counts %d", top)
	}
	if board.TileCount() != 7 {
		t.Fatalf("tile count %d", board.TileCount())
	}
}

func TestOnlyTheFirstClosedDoubleIsTheSpinner(t *testing.T) {
	board := NewBoard(MustTile(3, 3))
	mustPlace(t, &board, "3-5", EndRight)
	mustPlace(t, &board, "5-5", EndRight)
	mustPlace(t, &board, "4-5", EndRight)
	spinner, _ := board.Spinner()
	if spinner != MustTile(5, 5) {
		t.Fatalf("spinner %v", spinner)
	}
	mustPlace(t, &board, "1-3", EndLeft)
	if spinner, _ = board.Spinner(); spinner != MustTile(5, 5) {
		t.Fatalf("closing 3-3 changed the spinner to %v", spinner)
	}
	left, _ := board.OpenEnds().Value(EndLeft)
	right, _ := board.OpenEnds().Value(EndRight)
	if left != 1 || right != 4 || board.OpenEnds().Total() != 5 {
		t.Fatalf("ends %d %d", left, right)
	}
}

func TestOpeningDoubleBecomesSpinner(t *testing.T) {
	board := NewBoard(MustTile(6, 6))
	refuse(t, &board, "1-6", EndTop)
	mustPlace(t, &board, "6-1", EndRight)
	mustPlace(t, &board, "4-6", EndLeft)
	spinner, ok := board.Spinner()
	if !ok || spinner != MustTile(6, 6) {
		t.Fatalf("spinner %v %v", spinner, ok)
	}
	if got := lineText(board.Line()); got != "4-6 6-6 6-1 " {
		t.Fatalf("line %q", got)
	}
}

func TestOpeningIndexFollowsLeftPlacements(t *testing.T) {
	type placement struct {
		tile string
		end  End
	}
	cases := []struct {
		name       string
		opening    Tile
		placements []placement
		want       int
	}{
		{"opening only", MustTile(2, 3), nil, 0},
		{"right placements keep it first", MustTile(2, 3), []placement{{"3-4", EndRight}, {"4-6", EndRight}}, 0},
		{"each left placement moves it right", MustTile(2, 3), []placement{{"2-5", EndLeft}, {"3-3", EndRight}, {"1-5", EndLeft}}, 2},
		{"arm placements leave it", MustTile(6, 6), []placement{{"6-1", EndRight}, {"4-6", EndLeft}, {"3-6", EndTop}, {"0-6", EndBottom}}, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			board := NewBoard(c.opening)
			for _, p := range c.placements {
				mustPlace(t, &board, p.tile, p.end)
			}
			if got := board.OpeningIndex(); got != c.want {
				t.Fatalf("opening index %d, want %d", got, c.want)
			}
			if line := board.Line(); line[c.want].Tile() != c.opening {
				t.Fatalf("line %q has %v at the opening index", lineText(line), line[c.want])
			}
		})
	}
}

func TestBoardCopiesAreIndependent(t *testing.T) {
	board := NewBoard(MustTile(2, 3))
	snapshot := board
	mustPlace(t, &board, "3-3", EndRight)
	if snapshot.TileCount() != 1 || board.TileCount() != 2 {
		t.Fatal("the snapshot shares state with the board")
	}
}

func TestPlaceDoesNotAllocate(t *testing.T) {
	board := NewBoard(MustTile(2, 3))
	tile := MustTile(3, 4)
	allocations := testing.AllocsPerRun(100, func() {
		copy := board
		copy.CanPlace(tile, EndRight)
		_ = copy.Place(tile, EndRight)
		copy.OpenEnds()
	})
	if allocations != 0 {
		t.Fatalf("placement allocates %.0f times", allocations)
	}
}

func FuzzBoardPlace(f *testing.F) {
	f.Add(uint64(1), []byte{0, 1, 2, 3})
	f.Add(uint64(9), []byte{27, 26, 25, 3, 2, 1})
	f.Fuzz(func(t *testing.T, seed uint64, moves []byte) {
		random := seeded(seed)
		set := FullSet()
		Shuffle(set[:], random)
		board := NewBoard(set[0])
		remaining := Hand(set[1:])
		for _, move := range moves {
			if len(remaining) == 0 {
				break
			}
			tile := remaining[int(move>>2)%len(remaining)]
			end := End(move & 3)
			before := board
			legal := board.CanPlace(tile, end)
			err := board.Place(tile, end)
			if legal != (err == nil) {
				t.Fatalf("CanPlace %v but Place %v", legal, err)
			}
			if !legal {
				if board != before {
					t.Fatal("a refused placement changed the board")
				}
				continue
			}
			remaining.Remove(tile)
			if board.TileCount()+len(remaining) != SetSize {
				t.Fatalf("tiles are not conserved: %d on board, %d left", board.TileCount(), len(remaining))
			}
			ends := board.OpenEnds()
			if _, open := ends.Value(EndTop); open && board.top.count == 0 {
				t.Fatal("an empty arm counts")
			}
			if _, ok := board.Spinner(); !ok && board.top.count+board.bottom.count > 0 {
				t.Fatal("arms without a spinner")
			}
		}
	})
}

func BenchmarkPlace(b *testing.B) {
	board := NewBoard(MustTile(2, 3))
	tile := MustTile(3, 4)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		copy := board
		_ = copy.Place(tile, EndRight)
	}
}

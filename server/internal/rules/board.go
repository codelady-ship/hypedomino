package rules

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	ErrInvalidEnd         = errors.New("rules: invalid end")
	ErrIllegalPlacement   = errors.New("rules: illegal placement")
	ErrInvalidOrientation = errors.New("rules: invalid oriented tile")
)

type Oriented struct {
	First  uint8
	Second uint8
}

func ParseOriented(text string) (Oriented, error) {
	halves := strings.Split(text, "-")
	if len(halves) != 2 {
		return Oriented{}, fmt.Errorf("%w: %q", ErrInvalidOrientation, text)
	}
	first, err := parsePip(halves[0])
	if err != nil {
		return Oriented{}, fmt.Errorf("%w: %q", ErrInvalidOrientation, text)
	}
	second, err := parsePip(halves[1])
	if err != nil {
		return Oriented{}, fmt.Errorf("%w: %q", ErrInvalidOrientation, text)
	}
	return Oriented{First: first, Second: second}, nil
}

func (o Oriented) Tile() Tile {
	return MustTile(o.First, o.Second)
}

func (o Oriented) IsDouble() bool {
	return o.First == o.Second
}

func (o Oriented) String() string {
	return strconv.Itoa(int(o.First)) + "-" + strconv.Itoa(int(o.Second))
}

func (o Oriented) MarshalText() ([]byte, error) {
	return []byte(o.String()), nil
}

func (o *Oriented) UnmarshalText(text []byte) error {
	parsed, err := ParseOriented(string(text))
	if err != nil {
		return err
	}
	*o = parsed
	return nil
}

type End uint8

const (
	EndLeft End = iota
	EndRight
	EndTop
	EndBottom
	endCount
)

var endNames = [endCount]string{"left", "right", "top", "bottom"}

func ParseEnd(text string) (End, error) {
	for end, name := range endNames {
		if name == text {
			return End(end), nil
		}
	}
	return 0, fmt.Errorf("%w: %q", ErrInvalidEnd, text)
}

func (e End) String() string {
	if e >= endCount {
		return "invalid"
	}
	return endNames[e]
}

func (e End) MarshalText() ([]byte, error) {
	return []byte(e.String()), nil
}

func (e *End) UnmarshalText(text []byte) error {
	parsed, err := ParseEnd(string(text))
	if err != nil {
		return err
	}
	*e = parsed
	return nil
}

type OpenEnds struct {
	values [endCount]int
	open   [endCount]bool
}

func (o OpenEnds) Value(end End) (int, bool) {
	if end >= endCount {
		return 0, false
	}
	return o.values[end], o.open[end]
}

func (o OpenEnds) Total() int {
	total := 0
	for end, open := range o.open {
		if open {
			total += o.values[end]
		}
	}
	return total
}

func (o *OpenEnds) set(end End, value int) {
	o.values[end] = value
	o.open[end] = true
}

type arm struct {
	tiles [SetSize]Oriented
	count int
}

func (a *arm) tail() []Oriented {
	return a.tiles[:a.count]
}

func (a *arm) push(tile Oriented) {
	a.tiles[a.count] = tile
	a.count++
}

type Board struct {
	line       [2 * SetSize]Oriented
	head       int
	tail       int
	spinner    Oriented
	hasSpinner bool
	top        arm
	bottom     arm
}

func NewBoard(opening Tile) Board {
	board := Board{head: SetSize, tail: SetSize}
	board.line[board.tail] = Oriented{First: opening.Low, Second: opening.High}
	board.tail++
	return board
}

func (b *Board) Line() []Oriented {
	return b.line[b.head:b.tail]
}

func (b *Board) Spinner() (Tile, bool) {
	if !b.hasSpinner {
		return Tile{}, false
	}
	return b.spinner.Tile(), true
}

func (b *Board) Top() []Oriented {
	return b.top.tail()
}

func (b *Board) Bottom() []Oriented {
	return b.bottom.tail()
}

func (b *Board) OpeningIndex() int {
	return SetSize - b.head
}

func (b *Board) TileCount() int {
	return b.tail - b.head + b.top.count + b.bottom.count
}

func (b *Board) EndPip(end End) (uint8, bool) {
	switch end {
	case EndLeft:
		if b.tail == b.head {
			return 0, false
		}
		return b.line[b.head].First, true
	case EndRight:
		if b.tail == b.head {
			return 0, false
		}
		return b.line[b.tail-1].Second, true
	case EndTop:
		return b.armPip(&b.top)
	case EndBottom:
		return b.armPip(&b.bottom)
	}
	return 0, false
}

func (b *Board) armPip(a *arm) (uint8, bool) {
	if !b.hasSpinner {
		return 0, false
	}
	if a.count == 0 {
		return b.spinner.First, true
	}
	return a.tiles[a.count-1].Second, true
}

func (b *Board) OpenEnds() OpenEnds {
	var ends OpenEnds
	if b.tail > b.head {
		ends.set(EndLeft, endValue(b.line[b.head], b.line[b.head].First))
		ends.set(EndRight, endValue(b.line[b.tail-1], b.line[b.tail-1].Second))
	}
	if b.top.count > 0 {
		last := b.top.tiles[b.top.count-1]
		ends.set(EndTop, endValue(last, last.Second))
	}
	if b.bottom.count > 0 {
		last := b.bottom.tiles[b.bottom.count-1]
		ends.set(EndBottom, endValue(last, last.Second))
	}
	return ends
}

func endValue(outermost Oriented, exposed uint8) int {
	if outermost.IsDouble() {
		return 2 * int(exposed)
	}
	return int(exposed)
}

func (b *Board) CanPlace(tile Tile, end End) bool {
	pip, open := b.EndPip(end)
	return open && tile.Valid() && tile.Has(pip)
}

func (b *Board) Place(tile Tile, end End) error {
	pip, open := b.EndPip(end)
	if !open || !tile.Valid() || !tile.Has(pip) {
		return fmt.Errorf("%w: %s on %s", ErrIllegalPlacement, tile, end)
	}
	other := tile.High
	if tile.High == pip {
		other = tile.Low
	}
	switch end {
	case EndLeft:
		b.head--
		b.line[b.head] = Oriented{First: other, Second: pip}
		b.adoptSpinner(b.head + 1)
	case EndRight:
		b.line[b.tail] = Oriented{First: pip, Second: other}
		b.tail++
		b.adoptSpinner(b.tail - 2)
	case EndTop:
		b.top.push(Oriented{First: pip, Second: other})
	case EndBottom:
		b.bottom.push(Oriented{First: pip, Second: other})
	}
	return nil
}

func (b *Board) adoptSpinner(index int) {
	if b.hasSpinner {
		return
	}
	candidate := b.line[index]
	if !candidate.IsDouble() || index == b.head || index == b.tail-1 {
		return
	}
	b.spinner = candidate
	b.hasSpinner = true
}

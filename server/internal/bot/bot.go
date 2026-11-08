package bot

import (
	"errors"
	"fmt"

	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

var ErrNotActing = errors.New("bot: nothing for this seat to do")

type Table interface {
	Mode() rules.Mode
	Config() rules.Config
	Phase() rules.Phase
	Scores() rules.TeamPoints
	HandsPlayed() int
	Current() rules.Seat
	OpeningTeam() rules.Team
	ForcedOpening() (rules.Opening, bool)
	Hand(seat rules.Seat) rules.Hand
	Board() *rules.Board
	Turn() *rules.Turn
	LegalMoves(seat rules.Seat, moves []rules.Move) []rules.Move
	Open(seat rules.Seat, tile rules.Tile) (int, error)
	Place(seat rules.Seat, tile rules.Tile, end rules.End) (rules.Placement, error)
	EndTurn(seat rules.Seat) error
	Draw(seat rules.Seat) (rules.Tile, error)
	Pass(seat rules.Seat) error
}
type Level uint8

const (
	Easy Level = iota
	Medium
	Hard
	levelCount
)

var levelNames = [levelCount]string{"easy", "medium", "hard"}

func (l Level) String() string {
	if l >= levelCount {
		return "invalid"
	}
	return levelNames[l]
}

func (l Level) MarshalText() ([]byte, error) {
	if l >= levelCount {
		return nil, fmt.Errorf("bot: invalid level %d", l)
	}
	return []byte(levelNames[l]), nil
}

func (l *Level) UnmarshalText(text []byte) error {
	for level, name := range levelNames {
		if name == string(text) {
			*l = Level(level)
			return nil
		}
	}
	return fmt.Errorf("bot: unknown level %q", text)
}

type Weights struct {
	Score       int
	Pips        int
	Flexibility int
	Block       int
}

type Config struct {
	Medium Weights
	Hard   Weights
}

func DefaultConfig() Config {
	return Config{
		Medium: Weights{Score: 10, Pips: 1},
		Hard:   Weights{Score: 10, Pips: 1, Flexibility: 2, Block: 1},
	}
}

func (c Config) weights(level Level) (Weights, bool) {
	switch level {
	case Medium:
		return c.Medium, true
	case Hard:
		return c.Hard, true
	}
	return Weights{}, false
}

type Action uint8

const (
	ActionNone Action = iota
	ActionOpen
	ActionPlace
	ActionEndTurn
	ActionDraw
	ActionPass
)

type Player struct {
	level   Level
	weights Weights
	weighed bool
	random  rules.Random
	moves   []rules.Move
}

func New(level Level, config Config, random rules.Random) *Player {
	weights, weighed := config.weights(level)
	return &Player{
		level:   level,
		weights: weights,
		weighed: weighed,
		random:  random,
		moves:   make([]rules.Move, 0, 4*rules.SetSize),
	}
}

func (p *Player) Level() Level {
	return p.level
}

func (p *Player) Act(m Table, seat rules.Seat) (Action, error) {
	switch m.Phase() {
	case rules.PhaseOpening:
		return p.open(m, seat)
	case rules.PhasePlaying:
		if m.Current() != seat {
			return ActionNone, ErrNotActing
		}
		return p.play(m, seat)
	}
	return ActionNone, ErrNotActing
}

func (p *Player) open(m Table, seat rules.Seat) (Action, error) {
	if opening, forced := m.ForcedOpening(); forced {
		if opening.Seat != seat {
			return ActionNone, ErrNotActing
		}
		_, err := m.Open(seat, opening.Tile)
		return ActionOpen, err
	}
	if seat.Team() != m.OpeningTeam() {
		return ActionNone, ErrNotActing
	}
	hand := m.Hand(seat)
	tile := hand[p.random.IntN(len(hand))]
	if p.weighed {
		tile = p.bestOpening(m, seat, hand)
	}
	_, err := m.Open(seat, tile)
	return ActionOpen, err
}

func (p *Player) bestOpening(m Table, seat rules.Seat, hand rules.Hand) rules.Tile {
	config := m.Config()
	scores := m.Scores()
	best, bestValue := hand[0], -1
	for _, tile := range hand {
		score := rules.OpeningScore(config, m.HandsPlayed() == 0, tile, scores[seat.Team()])
		board := rules.NewBoard(tile)
		value := p.weights.Score*score + p.weights.Pips*tile.Pips() + p.weights.Flexibility*flexibility(&board, hand, tile)
		if value > bestValue {
			best, bestValue = tile, value
		}
	}
	return best
}

func (p *Player) play(m Table, seat rules.Seat) (Action, error) {
	moves := m.LegalMoves(seat, p.moves[:0])
	if len(moves) == 0 {
		if m.Turn().Placements() > 0 {
			return ActionEndTurn, m.EndTurn(seat)
		}
		if _, err := m.Draw(seat); err == nil {
			return ActionDraw, nil
		} else if !errors.Is(err, rules.ErrCannotDraw) {
			return ActionDraw, err
		}
		return ActionPass, m.Pass(seat)
	}
	move := moves[p.random.IntN(len(moves))]
	if p.weighed {
		move = p.bestMove(m, seat, moves)
	}
	if _, err := m.Place(seat, move.Tile, move.End); err != nil {
		return ActionPlace, fmt.Errorf("%s %s on %s: %w", seat, move.Tile, move.End, err)
	}
	return ActionPlace, nil
}

func (p *Player) bestMove(m Table, seat rules.Seat, moves []rules.Move) rules.Move {
	config := m.Config()
	hand := m.Hand(seat)
	best, bestValue := moves[0], -1
	for _, move := range moves {
		trial := *m.Board()
		if err := trial.Place(move.Tile, move.End); err != nil {
			continue
		}
		score := rules.PlacementScore(trial.OpenEnds().Total(), config)
		value := p.weights.Score*score + p.weights.Pips*move.Tile.Pips()
		if p.weights.Flexibility != 0 {
			value += p.weights.Flexibility * flexibility(&trial, hand, move.Tile)
		}
		if p.weights.Block != 0 {
			value += p.weights.Block * blocking(&trial, hand, move.Tile)
		}
		if value > bestValue {
			best, bestValue = move, value
		}
	}
	return best
}

func flexibility(board *rules.Board, hand rules.Hand, placed rules.Tile) int {
	playable := 0
	for _, tile := range hand {
		if tile != placed && board.CanPlaceAnywhere(tile) {
			playable++
		}
	}
	return playable
}

func blocking(board *rules.Board, hand rules.Hand, placed rules.Tile) int {
	var seen [rules.MaxPip + 1]int
	countPips(&seen, board.Line())
	countPips(&seen, board.Top())
	countPips(&seen, board.Bottom())
	for _, tile := range hand {
		if tile != placed {
			seen[tile.Low]++
			if !tile.IsDouble() {
				seen[tile.High]++
			}
		}
	}
	ends := board.OpenEnds()
	hidden := 0
	for end := rules.EndLeft; end <= rules.EndBottom; end++ {
		pip, open := board.EndPip(end)
		if _, counted := ends.Value(end); open && counted {
			hidden += seen[pip]
		}
	}
	return hidden
}

func countPips(seen *[rules.MaxPip + 1]int, tiles []rules.Oriented) {
	for _, tile := range tiles {
		seen[tile.First]++
		if tile.First != tile.Second {
			seen[tile.Second]++
		}
	}
}

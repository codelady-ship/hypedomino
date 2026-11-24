package rules

import (
	"errors"
	"fmt"
)

var (
	ErrWrongPhase  = errors.New("rules: not allowed in this phase")
	ErrNotYourTurn = errors.New("rules: not this seat's turn")
	ErrWrongOpener = errors.New("rules: this seat may not open")
	ErrMustPlace   = errors.New("rules: a legal placement exists")
	ErrMustDraw    = errors.New("rules: the boneyard must be drawn from first")
	ErrCannotDraw  = errors.New("rules: nothing can be drawn")
)

type MatchOutcome uint8

const (
	MatchContinues MatchOutcome = iota
	MatchNextHand
	MatchWon
	matchOutcomeCount
)

var matchOutcomeNames = [matchOutcomeCount]string{"continue", "next-hand", "won"}

func (o MatchOutcome) String() string {
	if o >= matchOutcomeCount {
		return "invalid"
	}
	return matchOutcomeNames[o]
}

func (o MatchOutcome) MarshalText() ([]byte, error) {
	return []byte(o.String()), nil
}

func (o *MatchOutcome) UnmarshalText(text []byte) error {
	for outcome, name := range matchOutcomeNames {
		if name == string(text) {
			*o = MatchOutcome(outcome)
			return nil
		}
	}
	return fmt.Errorf("rules: invalid match outcome %q", text)
}

func MatchEnd(config Config, end HandStatus, scores TeamPoints) (MatchOutcome, Team) {
	switch end {
	case HandContinues:
		return MatchContinues, TeamA
	case HandBlocked:
		return MatchNextHand, TeamA
	}
	leader := TeamA
	if scores[TeamB] > scores[TeamA] {
		leader = TeamB
	}
	if scores[leader] < config.TargetScore || scores[TeamA] == scores[TeamB] {
		return MatchNextHand, TeamA
	}
	return MatchWon, leader
}

type Phase uint8

const (
	PhaseOpening Phase = iota
	PhasePlaying
	PhaseFinished
)

type Match struct {
	mode              Mode
	config            Config
	random            Random
	scores            TeamPoints
	scoresAtHandStart TeamPoints
	carryOver         int
	handsPlayed       int
	phase             Phase
	winner            Team
	hands             map[Seat]Hand
	boneyard          Boneyard
	board             Board
	turn              Turn
	current           Seat
	lastPlacedBy      Seat
	openingTeam       Team
	forcedOpening     Opening
	hasForcedOpening  bool
	lastResult        HandResult
	lastEnd           HandEnd
	hasLastResult     bool
	moves             []Move
}

func NewMatch(mode Mode, config Config, random Random) *Match {
	m := &Match{mode: mode, config: config, random: random, moves: make([]Move, 0, 4*SetSize)}
	m.dealHand()
	return m
}

func (m *Match) Mode() Mode          { return m.mode }
func (m *Match) Config() Config      { return m.config }
func (m *Match) Phase() Phase        { return m.phase }
func (m *Match) Scores() TeamPoints  { return m.scores }
func (m *Match) CarryOver() int      { return m.carryOver }
func (m *Match) HandsPlayed() int    { return m.handsPlayed }
func (m *Match) Current() Seat       { return m.current }
func (m *Match) OpeningTeam() Team   { return m.openingTeam }
func (m *Match) Hand(seat Seat) Hand { return m.hands[seat] }
func (m *Match) BoneyardSize() int   { return len(m.boneyard) }
func (m *Match) Board() *Board       { return &m.board }
func (m *Match) Turn() *Turn         { return &m.turn }

func (m *Match) Winner() (Team, bool) {
	return m.winner, m.phase == PhaseFinished
}

func (m *Match) ForcedOpening() (Opening, bool) {
	return m.forcedOpening, m.hasForcedOpening && m.phase == PhaseOpening
}

func (m *Match) LastResult() (HandResult, bool) {
	return m.lastResult, m.hasLastResult
}

func (m *Match) HandPointsLocked(team Team) bool {
	return m.scoresAtHandStart[team] >= m.config.HandPointsLockThreshold
}

func (m *Match) LastHandEnd() (HandEnd, bool) {
	return m.lastEnd, m.hasLastResult
}

func (m *Match) LegalMoves(seat Seat, moves []Move) []Move {
	if m.phase != PhasePlaying || seat != m.current {
		return moves[:0]
	}
	return m.turn.LegalMoves(m.hands[seat], moves)
}

func (m *Match) dealHand() {
	for {
		deal := NewDeal(m.mode, m.config, m.random)
		m.hands = deal.Hands
		m.boneyard = deal.Boneyard
		if NeedsDoublesRedeal(m.mode, m.config, m.hands) {
			continue
		}
		if m.handsPlayed > 0 {
			break
		}
		opening, ok := FirstOpener(m.mode, m.config, m.hands)
		if ok {
			m.forcedOpening = opening
			m.hasForcedOpening = true
			m.openingTeam = opening.Seat.Team()
			break
		}
	}
	m.scoresAtHandStart = m.scores
	m.phase = PhaseOpening
}

func (m *Match) Open(seat Seat, tile Tile) (int, error) {
	if m.phase != PhaseOpening {
		return 0, ErrWrongPhase
	}
	if m.hasForcedOpening && (seat != m.forcedOpening.Seat || tile != m.forcedOpening.Tile) {
		return 0, fmt.Errorf("%w: %s must open with %s", ErrWrongOpener, m.forcedOpening.Seat, m.forcedOpening.Tile)
	}
	if seat.Team() != m.openingTeam {
		return 0, fmt.Errorf("%w: team %s opens", ErrWrongOpener, m.openingTeam)
	}
	hand := m.hands[seat]
	if !hand.Remove(tile) {
		return 0, fmt.Errorf("%w: %s", ErrTileNotHeld, tile)
	}
	m.hands[seat] = hand
	m.board = NewBoard(tile)
	score := OpeningScore(m.config, m.handsPlayed == 0, tile, m.scoresAtHandStart[seat.Team()])
	m.scores[seat.Team()] += score
	m.lastPlacedBy = seat
	m.hasForcedOpening = false
	m.phase = PhasePlaying
	m.current = seat
	m.advance()
	return score, nil
}

func (m *Match) Place(seat Seat, tile Tile, end End) (Placement, error) {
	if err := m.checkTurn(seat); err != nil {
		return Placement{}, err
	}
	hand := m.hands[seat]
	if !hand.Contains(tile) {
		return Placement{}, fmt.Errorf("%w: %s", ErrTileNotHeld, tile)
	}
	placement, err := m.turn.Place(tile, end)
	if err != nil {
		return Placement{}, err
	}
	hand.Remove(tile)
	m.hands[seat] = hand
	m.scores[seat.Team()] += placement.Score
	m.lastPlacedBy = seat
	if len(hand) == 0 {
		m.endHand(HandDomino)
		return placement, nil
	}
	if !m.turn.CanContinue() || len(m.turn.LegalMoves(hand, m.moves)) == 0 {
		m.advance()
	}
	return placement, nil
}

func (m *Match) EndTurn(seat Seat) error {
	if err := m.checkTurn(seat); err != nil {
		return err
	}
	if m.turn.Placements() == 0 {
		return ErrMustPlace
	}
	m.advance()
	return nil
}

func (m *Match) Draw(seat Seat) (Tile, error) {
	if err := m.checkTurn(seat); err != nil {
		return Tile{}, err
	}
	if m.turn.Placements() > 0 || m.board.HasLegalMove(m.hands[seat]) {
		return Tile{}, ErrMustPlace
	}
	if !m.canDraw() {
		return Tile{}, ErrCannotDraw
	}
	tile, _ := m.boneyard.Draw()
	hand := m.hands[seat]
	hand.Add(tile)
	m.hands[seat] = hand
	return tile, nil
}

func (m *Match) Pass(seat Seat) error {
	if err := m.checkTurn(seat); err != nil {
		return err
	}
	if m.turn.Placements() > 0 || m.board.HasLegalMove(m.hands[seat]) {
		return ErrMustPlace
	}
	if m.canDraw() {
		return ErrMustDraw
	}
	if m.mode.HandStatus(&m.board, m.hands, m.boneyard) == HandBlocked {
		m.endHand(HandBlocked)
		return nil
	}
	m.advance()
	return nil
}

func (m *Match) canDraw() bool {
	return m.mode == Mode1v1 && len(m.boneyard) > 1
}

func (m *Match) checkTurn(seat Seat) error {
	if m.phase != PhasePlaying {
		return ErrWrongPhase
	}
	if seat != m.current {
		return fmt.Errorf("%w: %s moves", ErrNotYourTurn, m.current)
	}
	return nil
}

func (m *Match) advance() {
	m.current = m.current.Next()
	m.turn = BeginTurn(&m.board, m.config)
}

func (m *Match) recordEnd(status HandStatus, result HandResult, scores HandScores) {
	locked := result.HasWinner && scores.AtHandStart[result.Winner] >= m.config.HandPointsLockThreshold
	carriedIn := 0
	if result.HasWinner && !locked {
		carriedIn = scores.CarryOver
	}
	if m.lastEnd.Left == nil {
		m.lastEnd.Left = make(map[Seat]Hand, len(m.mode.Seats()))
	}
	for _, seat := range m.mode.Seats() {
		m.lastEnd.Left[seat] = append(m.lastEnd.Left[seat][:0], m.hands[seat]...)
	}
	m.lastEnd.Status = status
	m.lastEnd.Locked = locked
	m.lastEnd.CarriedIn = carriedIn
}

func (m *Match) endHand(status HandStatus) {
	settle := DominoResult
	if status == HandBlocked {
		settle = BlockResult
	}
	scores := HandScores{AtHandStart: m.scoresAtHandStart, Current: m.scores, CarryOver: m.carryOver}
	result := settle(m.mode, m.config, m.lastPlacedBy, m.hands, scores)
	m.recordEnd(status, result, scores)
	m.scores = result.Scores
	m.carryOver = result.CarryOver
	m.lastResult = result
	m.hasLastResult = true
	m.handsPlayed++
	outcome, winner := MatchEnd(m.config, status, m.scores)
	if outcome == MatchWon {
		m.winner = winner
		m.phase = PhaseFinished
		return
	}
	m.openingTeam = result.NextOpener
	m.dealHand()
}

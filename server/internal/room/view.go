package room

import (
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

type BoardView struct {
	Line       []rules.Oriented
	Opening    int
	Spinner    rules.Tile
	HasSpinner bool
	Top        []rules.Oriented
	Bottom     []rules.Oriented
	OpenEnds   rules.OpenEnds
}

type View struct {
	State           State
	Seat            rules.Seat
	Mode            rules.Mode
	Acked           uint64
	HandsPlayed     int
	Scores          rules.TeamPoints
	CarryOver       int
	Hand            []rules.Tile
	HandSizes       [6]int
	BoneyardSize    int
	Board           BoardView
	Current         rules.Seat
	OpeningTeam     rules.Team
	ForcedOpening   rules.Opening
	HasForced       bool
	TurnPlacements  int
	TurnDeadline    time.Time
	TurnSeconds     int
	TurnWarning     int
	Controllers     [6]rules.Controller
	Disconnected    [6]bool
	Timeouts        [6]int
	AfkTimeouts     int
	Presence        [6]rules.PresenceReport
	LegalMoves      []rules.Move
	LastResult      rules.HandResult
	LastEnd         rules.HandEnd
	HasLastResult   bool
	TargetScore     int
	Locked          [2]bool
	LockThreshold   int
	Winner          rules.Team
	HasWinner       bool
	Loading         [6]int
	Ready           [6]bool
	LoadingDeadline time.Time
	Roster          []Player
	Table           string
}

func (r *Room) view(seat rules.Seat) View {
	config := r.options.Config
	v := View{State: r.state, Seat: seat, Mode: r.options.Mode, Acked: r.seats[seat].acked, AfkTimeouts: config.AfkConsecutiveTimeouts, TargetScore: config.TargetScore, LockThreshold: config.HandPointsLockThreshold, TurnSeconds: config.TurnSeconds, TurnWarning: config.TurnWarningSeconds}
	for _, s := range r.options.Mode.Seats() {
		v.Controllers[s] = r.presence.Controller(s)
		v.Presence[s] = r.presence.Report(s)
		v.Disconnected[s] = r.seats[s].sink == nil && !r.botSeat(s)
		v.Timeouts[s] = r.presence.ConsecutiveTimeouts(s)
		v.Loading[s] = r.seats[s].loading
		v.Ready[s] = r.seats[s].ready
	}
	v.Roster = r.options.Roster
	v.Table = r.options.Table
	if r.state == StateLoading {
		v.LoadingDeadline = r.deadline
	}
	if r.recorder == nil {
		return v
	}
	m := r.match()
	v.HandsPlayed = m.HandsPlayed()
	for _, team := range []rules.Team{rules.TeamA, rules.TeamB} {
		v.Locked[team] = m.HandPointsLocked(team)
	}
	v.Scores = m.Scores()
	v.CarryOver = m.CarryOver()
	v.Hand = append([]rules.Tile(nil), m.Hand(seat)...)
	for _, s := range r.options.Mode.Seats() {
		v.HandSizes[s] = len(m.Hand(s))
	}
	v.BoneyardSize = m.BoneyardSize()
	v.Current = m.Current()
	if acting, ok := r.actingSeat(); ok {
		v.Current = acting
	}
	v.OpeningTeam = m.OpeningTeam()
	v.ForcedOpening, v.HasForced = m.ForcedOpening()
	v.TurnDeadline = r.deadline
	v.LastResult, v.HasLastResult = m.LastResult()
	if v.HasLastResult {
		v.LastEnd = r.lastEnd(m)
	}
	v.Winner, v.HasWinner = m.Winner()
	if m.Phase() == rules.PhasePlaying {
		board := m.Board()
		v.Board = BoardView{
			Line:     append([]rules.Oriented(nil), board.Line()...),
			Opening:  board.OpeningIndex(),
			Top:      append([]rules.Oriented(nil), board.Top()...),
			Bottom:   append([]rules.Oriented(nil), board.Bottom()...),
			OpenEnds: board.OpenEnds(),
		}
		v.Board.Spinner, v.Board.HasSpinner = board.Spinner()
		v.TurnPlacements = m.Turn().Placements()
		if m.Current() == seat {
			v.LegalMoves = append([]rules.Move(nil), m.LegalMoves(seat, r.moves[:0])...)
		}
	}
	return v
}

func (r *Room) lastEnd(m *rules.Match) rules.HandEnd {
	end, _ := m.LastHandEnd()
	shown := rules.HandEnd{Status: end.Status, Locked: end.Locked, CarriedIn: end.CarriedIn}
	if m.Phase() == rules.PhasePlaying {
		return shown
	}
	shown.Left = make(map[rules.Seat]rules.Hand, len(r.options.Mode.Seats()))
	for _, s := range r.options.Mode.Seats() {
		shown.Left[s] = append(rules.Hand(nil), end.Left[s]...)
	}
	return shown
}

func (r *Room) botSeat(seat rules.Seat) bool {
	for _, player := range r.options.Roster {
		if player.Seat == seat {
			return player.Bot
		}
	}
	return false
}

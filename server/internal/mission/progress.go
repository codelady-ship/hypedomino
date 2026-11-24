package mission

import (
	"github.com/Alievs-corp/hypedomino/server/internal/matchstore"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

type Match struct {
	Kind         string
	Mode         rules.Mode
	Won          bool
	BotSeat      bool
	Offender     bool
	Cancelled    bool
	HandsWon     int64
	Dominoes     int64
	BoardPoints  int64
	DoubleChains int64
}

func (m Match) CountsFor(missionType string) bool {
	if m.Cancelled || m.Offender || (m.Kind != matchstore.KindRanked && m.Kind != matchstore.KindCasual) {
		return false
	}
	botTable := m.Kind == matchstore.KindCasual && m.BotSeat
	return !botTable || missionType == Daily || missionType == Weekly
}

func (m Mission) Progress(match Match) int64 {
	if !match.CountsFor(m.Type) || !m.Filters.Fit(match) {
		return 0
	}
	switch m.Event {
	case MatchCompleted:
		return 1
	case MatchWon:
		if match.Won {
			return 1
		}
	case HandWon:
		return match.HandsWon
	case Domino:
		return match.Dominoes
	case BoardPoints:
		return match.BoardPoints
	case DoubleChain:
		return match.DoubleChains
	}
	return 0
}

func (f Filters) Fit(match Match) bool {
	return (f.Queue == "" || f.Queue == match.Kind) && (f.Mode == "" || f.Mode == match.Mode.String())
}

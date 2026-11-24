package pass

import (
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/matchstore"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

type Match struct {
	ID        string
	Account   string
	Mode      rules.Mode
	Kind      string
	Won       bool
	BotSeat   bool
	Offender  bool
	Cancelled bool
	EndedAt   time.Time
}

func (s *Snapshot) MatchXp(match Match) int {
	if match.Cancelled || match.Offender || (match.Kind != matchstore.KindRanked && match.Kind != matchstore.KindCasual) {
		return 0
	}
	table := s.MatchXp1v1
	if match.Mode == rules.Mode2v2 {
		table = s.MatchXp2v2
	}
	xp := table.Loss
	if match.Won {
		xp = table.Win
	}
	if match.Kind == matchstore.KindCasual && match.BotSeat {
		return percentHalfAwayFromZero(xp, s.BotSeatRewardPercent)
	}
	return xp
}

func percentHalfAwayFromZero(value, percent int) int {
	return (value*percent*2 + 100) / 200
}

func capToRemainder(xp, earnedToday, limit int) int {
	return max(0, min(xp, limit-earnedToday))
}

func EconomyDay(at time.Time) time.Time {
	utc := at.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}

package rating

import (
	"errors"
	"fmt"
	"math"
)

var ErrMatch = errors.New("rating: invalid match")

type Player struct {
	Account   string
	MMR       int
	Matches   int
	Placement bool
	Rank      Rank
	Offender  bool
}

type Change struct {
	Account    string  `json:"account_id"`
	Won        bool    `json:"won"`
	Offender   bool    `json:"offender"`
	Placement  bool    `json:"placement"`
	K          int     `json:"k"`
	Expected   float64 `json:"expected"`
	MMRBefore  int     `json:"mmr_before"`
	MMR        int     `json:"mmr"`
	LPDelta    int     `json:"lp_delta"`
	Penalty    int     `json:"penalty"`
	RankBefore Rank    `json:"rank_before"`
	Rank       Rank    `json:"rank"`
}

type Match struct {
	Teams  [2][]Player
	Winner int
}

func Expected(team, opponent float64, scale int) float64 {
	return 1 / (1 + math.Pow(10, (opponent-team)/float64(scale)))
}

func round(value float64) int {
	return int(math.Round(value))
}

func mean(players []Player) float64 {
	var sum float64
	for _, p := range players {
		sum += float64(p.MMR)
	}
	return sum / float64(len(players))
}

func (c Config) Resolve(m Match) ([2][]Change, error) {
	var changes [2][]Change
	switch {
	case m.Winner != 0 && m.Winner != 1:
		return changes, fmt.Errorf("%w: winner %d", ErrMatch, m.Winner)
	case len(m.Teams[0]) == 0 || len(m.Teams[0]) != len(m.Teams[1]):
		return changes, fmt.Errorf("%w: teams of %d and %d players", ErrMatch, len(m.Teams[0]), len(m.Teams[1]))
	}
	for team := range m.Teams {
		won := team == m.Winner
		expected := Expected(mean(m.Teams[team]), mean(m.Teams[1-team]), c.MmrScale)
		offenders := 0
		for _, p := range m.Teams[team] {
			if p.Offender {
				offenders++
			}
		}
		changes[team] = make([]Change, 0, len(m.Teams[team]))
		for _, p := range m.Teams[team] {
			partner := !won && !p.Offender && offenders > 0
			changes[team] = append(changes[team], c.resolve(p, won, expected, partner))
		}
	}
	return changes, nil
}

func (c Config) resolve(p Player, won bool, expected float64, partner bool) Change {
	change := Change{Account: p.Account, Won: won && !p.Offender, Offender: p.Offender, Placement: p.Placement, Expected: expected, MMRBefore: p.MMR, RankBefore: p.Rank, Rank: p.Rank}
	change.K = c.MmrK
	if p.Placement || p.Matches < c.MmrProvisionalMatches {
		change.K = c.MmrKProvisional
	}
	score := 0.0
	if won && !p.Offender {
		score = 1
	}
	delta := round(float64(change.K) * (score - expected))
	if partner {
		delta = c.reduced(delta)
	}
	change.MMR = p.MMR + delta
	if p.Placement {
		if p.Offender {
			change.Penalty = c.OffenseRatingPenalty
		}
		return change
	}
	change.LPDelta = c.lpDelta(p, won, expected, partner)
	change.Rank = c.Move(p.Rank, change.LPDelta)
	return change
}

func (c Config) lpDelta(p Player, won bool, expected float64, partner bool) int {
	gap := float64(p.MMR) - c.RankMmr(p.Rank)
	factor := func(sign float64) float64 {
		return min(max(1+sign*gap/float64(c.LpGapScale), c.LpFactorMin), c.LpFactorMax)
	}
	clamp := func(value int) int {
		return min(max(value, c.LpChangeMin), c.LpChangeMax)
	}
	if won && !p.Offender {
		return clamp(round(float64(c.LpBase) * 2 * (1 - expected) * factor(1)))
	}
	loss := clamp(round(float64(c.LpBase) * 2 * expected * factor(-1)))
	switch {
	case p.Offender:
		loss += c.OffenseRatingPenalty
	case partner:
		loss = c.reduced(loss)
	}
	return -loss
}

func (c Config) reduced(delta int) int {
	return round(float64(delta) * float64(100-c.PartnerLossReductionPercent) / 100)
}

func (c Config) Place(mmr, penalty int) Rank {
	index := min(c.MmrRankIndex(mmr), c.placementCap())
	return Rank{Index: index, LP: max(0, -penalty)}
}

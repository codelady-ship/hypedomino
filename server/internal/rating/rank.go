package rating

import (
	"errors"
	"fmt"
	"strings"
)

const (
	MaxDivisionIndex = 27
	ApexIndex        = 28
	TierMaster       = "Master"
	TierGrandmaster  = "Grandmaster"
	TierChallenger   = "Challenger"

	ApexMaster      = "master"
	ApexGrandmaster = "grandmaster"
	ApexChallenger  = "challenger"
)

var (
	ErrRank = errors.New("rating: invalid rank")

	tiers     = [...]string{"Iron", "Bronze", "Silver", "Gold", "Platinum", "Emerald", "Diamond"}
	divisions = [...]string{"IV", "III", "II", "I"}
)

type Rank struct {
	Index int `json:"index"`
	LP    int `json:"lp"`
}

func (r Rank) Tier() string {
	if r.Index >= ApexIndex {
		return TierMaster
	}
	return tiers[r.Index/len(divisions)]
}

func (r Rank) Division() string {
	if r.Index >= ApexIndex {
		return ""
	}
	return divisions[r.Index%len(divisions)]
}

func (r Rank) Apex() bool {
	return r.Index >= ApexIndex
}

func (r Rank) String() string {
	if r.Apex() {
		return TierMaster
	}
	return r.Tier() + " " + r.Division()
}

func (r Rank) MarshalText() ([]byte, error) {
	return []byte(r.String()), nil
}

func KnownTier(name string) bool {
	for _, tier := range tiers {
		if tier == name {
			return true
		}
	}
	return name == TierMaster || name == TierGrandmaster || name == TierChallenger
}

func TierName(apex string) string {
	switch apex {
	case ApexChallenger:
		return TierChallenger
	case ApexGrandmaster:
		return TierGrandmaster
	}
	return TierMaster
}

func ParseRankIndex(name string) (int, error) {
	if strings.EqualFold(strings.TrimSpace(name), TierMaster) {
		return ApexIndex, nil
	}
	tier, division, ok := strings.Cut(strings.TrimSpace(name), " ")
	if !ok {
		return 0, fmt.Errorf("%w: %q", ErrRank, name)
	}
	for t, tierName := range tiers {
		if !strings.EqualFold(tierName, tier) {
			continue
		}
		for d, divisionName := range divisions {
			if strings.EqualFold(divisionName, division) {
				return t*len(divisions) + d, nil
			}
		}
	}
	return 0, fmt.Errorf("%w: %q", ErrRank, name)
}

func (c Config) Move(rank Rank, delta int) Rank {
	lp := rank.LP + delta
	switch {
	case lp >= c.DivisionLp && !rank.Apex():
		return Rank{Index: rank.Index + 1, LP: lp - c.DivisionLp}
	case lp < 0 && rank.Index == 0:
		return Rank{Index: 0, LP: 0}
	case lp < 0 && rank.Apex():
		return Rank{Index: MaxDivisionIndex, LP: c.DemotionLp}
	case lp < 0:
		return Rank{Index: rank.Index - 1, LP: c.DemotionLp}
	}
	return Rank{Index: rank.Index, LP: lp}
}

func (c Config) RankMmr(rank Rank) float64 {
	return float64(c.RankMmrBase) + float64(c.RankMmrStep)*(float64(rank.Index)+float64(rank.LP)/float64(c.DivisionLp))
}

func (c Config) MmrRankIndex(mmr int) int {
	index := (mmr - c.RankMmrBase) / c.RankMmrStep
	if mmr < c.RankMmrBase {
		index = 0
	}
	return min(max(index, 0), MaxDivisionIndex)
}

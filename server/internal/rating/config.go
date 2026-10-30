package rating

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/Alievs-corp/hypedomino/server/internal/reward"

	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

const (
	Domain = "rating"
	Schema = 1
)

var ErrInvalidConfig = errors.New("rating: invalid config")

type Config struct {
	RankRewards                 map[string]reward.Bundle `json:"rankRewards,omitempty"`
	DivisionLp                  int                      `json:"divisionLp"`
	DemotionLp                  int                      `json:"demotionLp"`
	ApexRecalcHourUtc           int                      `json:"apexRecalcHourUtc"`
	ChallengerSlots             int                      `json:"challengerSlots"`
	ChallengerMinLp             int                      `json:"challengerMinLp"`
	GrandmasterSlots            int                      `json:"grandmasterSlots"`
	GrandmasterMinLp            int                      `json:"grandmasterMinLp"`
	InitialMmr                  int                      `json:"initialMmr"`
	MmrScale                    int                      `json:"mmrScale"`
	MmrK                        int                      `json:"mmrK"`
	MmrKProvisional             int                      `json:"mmrKProvisional"`
	MmrProvisionalMatches       int                      `json:"mmrProvisionalMatches"`
	RankMmrBase                 int                      `json:"rankMmrBase"`
	RankMmrStep                 int                      `json:"rankMmrStep"`
	LpBase                      int                      `json:"lpBase"`
	LpGapScale                  int                      `json:"lpGapScale"`
	LpFactorMin                 float64                  `json:"lpFactorMin"`
	LpFactorMax                 float64                  `json:"lpFactorMax"`
	LpChangeMin                 int                      `json:"lpChangeMin"`
	LpChangeMax                 int                      `json:"lpChangeMax"`
	PlacementMatches            int                      `json:"placementMatches"`
	PlacementMaxDropDivisions   int                      `json:"placementMaxDropDivisions"`
	PlacementMaxClimbDivisions  int                      `json:"placementMaxClimbDivisions"`
	NewPlayerPlacementCap       string                   `json:"newPlayerPlacementCap"`
	SeasonLengthMonths          int                      `json:"seasonLengthMonths"`
	ResetTierDrop               int                      `json:"resetTierDrop"`
	ResetFloor                  string                   `json:"resetFloor"`
	ResetCeiling                string                   `json:"resetCeiling"`
	SeasonResetTable            map[string]string        `json:"seasonResetTable,omitempty"`
	OffenseRatingPenalty        int                      `json:"offenseRatingPenalty"`
	PartnerLossReductionPercent int                      `json:"partnerLossReductionPercent"`
}

func DefaultConfig() Config {
	return Config{
		DivisionLp:                  100,
		DemotionLp:                  75,
		ApexRecalcHourUtc:           0,
		ChallengerSlots:             50,
		ChallengerMinLp:             500,
		GrandmasterSlots:            200,
		GrandmasterMinLp:            200,
		InitialMmr:                  1200,
		MmrScale:                    400,
		MmrK:                        20,
		MmrKProvisional:             40,
		MmrProvisionalMatches:       20,
		RankMmrBase:                 800,
		RankMmrStep:                 40,
		LpBase:                      20,
		LpGapScale:                  200,
		LpFactorMin:                 0.5,
		LpFactorMax:                 1.5,
		LpChangeMin:                 5,
		LpChangeMax:                 40,
		PlacementMatches:            5,
		PlacementMaxDropDivisions:   4,
		PlacementMaxClimbDivisions:  4,
		NewPlayerPlacementCap:       "Gold IV",
		SeasonLengthMonths:          3,
		ResetTierDrop:               1,
		ResetFloor:                  "Bronze IV",
		ResetCeiling:                "Diamond IV",
		OffenseRatingPenalty:        10,
		PartnerLossReductionPercent: 50,
	}
}

func (c Config) Validate() error {
	fail := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s", ErrInvalidConfig, fmt.Sprintf(format, args...))
	}
	for tier, bundle := range c.RankRewards {
		if !KnownTier(tier) {
			return fail("rankRewards tier %q", tier)
		}
		if err := bundle.Validate(); err != nil {
			return fail("rankRewards %s: %v", tier, err)
		}
	}
	switch {
	case c.DivisionLp < 1:
		return fail("divisionLp must be at least 1")
	case c.DemotionLp < 0 || c.DemotionLp >= c.DivisionLp:
		return fail("demotionLp must be between 0 and divisionLp - 1")
	case c.ApexRecalcHourUtc < 0 || c.ApexRecalcHourUtc > 23:
		return fail("apexRecalcHourUtc must be an hour of the day")
	case c.ChallengerSlots < 0 || c.GrandmasterSlots < 0 || c.ChallengerMinLp < 0 || c.GrandmasterMinLp < 0:
		return fail("apex slots and minimum LP must not be negative")
	case c.MmrScale < 1 || c.MmrK < 1 || c.MmrKProvisional < 1 || c.MmrProvisionalMatches < 0:
		return fail("mmrScale, mmrK and mmrKProvisional must be at least 1 and mmrProvisionalMatches at least 0")
	case c.RankMmrStep < 1:
		return fail("rankMmrStep must be at least 1")
	case c.LpBase < 1 || c.LpGapScale < 1:
		return fail("lpBase and lpGapScale must be at least 1")
	case c.LpFactorMin <= 0 || c.LpFactorMin > c.LpFactorMax:
		return fail("lpFactorMin must be positive and at most lpFactorMax")
	case c.LpChangeMin < 0 || c.LpChangeMin > c.LpChangeMax:
		return fail("lpChangeMin must be between 0 and lpChangeMax")
	case c.OffenseRatingPenalty < 0 || c.PartnerLossReductionPercent < 0 || c.PartnerLossReductionPercent > 100:
		return fail("offenseRatingPenalty must not be negative and partnerLossReductionPercent must be a percentage")
	case c.LpChangeMax+c.OffenseRatingPenalty >= c.DivisionLp:
		return fail("lpChangeMax + offenseRatingPenalty must stay below divisionLp")
	case c.PlacementMatches < 1:
		return fail("placementMatches must be at least 1")
	case c.PlacementMaxDropDivisions < 0 || c.PlacementMaxClimbDivisions < 0:
		return fail("placement drop and climb limits must not be negative")
	}
	cap, err := ParseRankIndex(c.NewPlayerPlacementCap)
	if err != nil {
		return fail("newPlayerPlacementCap: %v", err)
	}
	if cap > MaxDivisionIndex {
		return fail("newPlayerPlacementCap must be a division rank")
	}
	if c.SeasonLengthMonths < 1 || c.ResetTierDrop < 0 {
		return fail("seasonLengthMonths must be at least 1 and resetTierDrop not negative")
	}
	if _, err := c.ResetTable(); err != nil {
		return fail("%v", err)
	}
	return nil
}

func (c Config) placementCap() int {
	cap, _ := ParseRankIndex(c.NewPlayerPlacementCap)
	return cap
}

type Modes struct {
	OneVsOne Config `json:"1v1"`
	TwoVsTwo Config `json:"2v2"`
}

type Snapshot struct {
	Schema             int                 `json:"schema"`
	LeaderboardTopSize int                 `json:"leaderboardTopSize"`
	LeaderboardRegions map[string][]string `json:"leaderboardRegions,omitempty"`
	RankRewardCoins    map[string]int64    `json:"rankRewardCoins"`
	EmblemBrackets     []int               `json:"emblemBrackets"`
	Abuse              Abuse               `json:"abuse"`
	Modes              Modes               `json:"modes"`
}

var regionID = regexp.MustCompile(`^[a-z0-9_]+$`)

func (s *Snapshot) Region(country string) (string, bool) {
	for region, countries := range s.LeaderboardRegions {
		for _, c := range countries {
			if c == country {
				return region, true
			}
		}
	}
	return "", false
}

func (s *Snapshot) KnownRegion(region string) bool {
	_, ok := s.LeaderboardRegions[region]
	return ok
}

func Default() *Snapshot {
	return &Snapshot{
		Schema:             Schema,
		LeaderboardTopSize: 200,
		RankRewardCoins:    map[string]int64{"Iron": 100, "Bronze": 200, "Silver": 400, "Gold": 700, "Platinum": 1000, "Emerald": 1500, "Diamond": 2000, TierMaster: 3000, TierGrandmaster: 4000, TierChallenger: 5000},
		EmblemBrackets:     []int{1, 50, 100, 200},
		Abuse:              DefaultAbuse(),
		Modes:              Modes{OneVsOne: DefaultConfig(), TwoVsTwo: DefaultConfig()},
	}
}

func (s *Snapshot) Mode(mode rules.Mode) (Config, bool) {
	switch mode {
	case rules.Mode1v1:
		return s.Modes.OneVsOne, true
	case rules.Mode2v2:
		return s.Modes.TwoVsTwo, true
	}
	return Config{}, false
}

func (s *Snapshot) Validate() error {
	if s.Schema != Schema {
		return fmt.Errorf("%w: schema %d, want %d", ErrInvalidConfig, s.Schema, Schema)
	}
	if s.LeaderboardTopSize < 1 {
		return fmt.Errorf("%w: leaderboardTopSize must be at least 1", ErrInvalidConfig)
	}
	seen := map[string]string{}
	for region, countries := range s.LeaderboardRegions {
		if !regionID.MatchString(region) {
			return fmt.Errorf("%w: region id %q", ErrInvalidConfig, region)
		}
		for _, country := range countries {
			if len(country) != 2 || strings.ToUpper(country) != country {
				return fmt.Errorf("%w: country %q of region %s", ErrInvalidConfig, country, region)
			}
			if other, dup := seen[country]; dup && other != region {
				return fmt.Errorf("%w: %s belongs to %s and %s", ErrInvalidConfig, country, other, region)
			}
			seen[country] = region
		}
	}
	for tier, coins := range s.RankRewardCoins {
		if !KnownTier(tier) || coins < 0 {
			return fmt.Errorf("%w: rankRewardCoins %s = %d", ErrInvalidConfig, tier, coins)
		}
	}
	for i, bracket := range s.EmblemBrackets {
		if bracket < 1 || (i > 0 && bracket <= s.EmblemBrackets[i-1]) {
			return fmt.Errorf("%w: emblemBrackets must be increasing positions", ErrInvalidConfig)
		}
	}
	if err := s.Abuse.Validate(); err != nil {
		return err
	}
	if err := s.Modes.OneVsOne.Validate(); err != nil {
		return fmt.Errorf("1v1: %w", err)
	}
	if err := s.Modes.TwoVsTwo.Validate(); err != nil {
		return fmt.Errorf("2v2: %w", err)
	}
	return nil
}

func Parse(raw []byte) (*Snapshot, error) {
	snapshot := Default()
	if err := json.Unmarshal(raw, snapshot); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	if err := snapshot.Validate(); err != nil {
		return nil, err
	}
	return snapshot, nil
}

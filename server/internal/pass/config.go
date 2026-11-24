package pass

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

const (
	Domain  = "pass"
	Schema  = 1
	Free    = "free"
	Premium = "premium"
)

var (
	ErrInvalidConfig = errors.New("pass: invalid config")
	ErrFrozen        = errors.New("pass: running season changed")
	seasonID         = regexp.MustCompile(`^season_[1-9][0-9]*$`)
)

type Season struct {
	ID      string                `json:"id"`
	Start   time.Time             `json:"start"`
	End     time.Time             `json:"end"`
	Curve   Curve                 `json:"curve"`
	Free    map[int]reward.Bundle `json:"free"`
	Premium map[int]reward.Bundle `json:"premium"`
}

func (s Season) Running(at time.Time) bool {
	return !at.Before(s.Start) && at.Before(s.End)
}

func (s Season) Track(track string) map[int]reward.Bundle {
	if track == Premium {
		return s.Premium
	}
	return s.Free
}

type MatchXp struct {
	Win  int `json:"win"`
	Loss int `json:"loss"`
}

type Snapshot struct {
	Schema               int      `json:"schema"`
	FreeTrackCoins       int64    `json:"freeTrackCoins"`
	PremiumTrackCrystals int64    `json:"premiumTrackCrystals"`
	PremiumPlusLevels    int      `json:"premiumPlusLevels"`
	MatchXp1v1           MatchXp  `json:"matchXp1v1"`
	MatchXp2v2           MatchXp  `json:"matchXp2v2"`
	DailyMatchXpCap      int      `json:"dailyMatchXpCap"`
	BotSeatRewardPercent int      `json:"botSeatRewardPercent"`
	LevelCoinPrice       int64    `json:"levelCoinPrice"`
	LevelCrystalPrice    int64    `json:"levelCrystalPrice"`
	Seasons              []Season `json:"seasons"`
}

func Default() *Snapshot {
	return &Snapshot{
		Schema:               Schema,
		FreeTrackCoins:       1500,
		PremiumTrackCrystals: 600,
		PremiumPlusLevels:    20,
		MatchXp1v1:           MatchXp{Win: 100, Loss: 60},
		MatchXp2v2:           MatchXp{Win: 140, Loss: 80},
		DailyMatchXpCap:      2000,
		BotSeatRewardPercent: 50,
		LevelCoinPrice:       200,
		LevelCrystalPrice:    40,
	}
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

func (s *Snapshot) Active(at time.Time) (Season, bool) {
	for _, season := range s.Seasons {
		if season.Running(at) {
			return season, true
		}
	}
	return Season{}, false
}

func (s *Snapshot) Season(id string) (Season, bool) {
	for _, season := range s.Seasons {
		if season.ID == id {
			return season, true
		}
	}
	return Season{}, false
}

func (s *Snapshot) Validate() error {
	fail := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s", ErrInvalidConfig, fmt.Sprintf(format, args...))
	}
	if s.Schema != Schema {
		return fail("schema %d", s.Schema)
	}
	if s.FreeTrackCoins < 0 || s.PremiumTrackCrystals < 0 || s.PremiumPlusLevels < 0 {
		return fail("track budgets and premiumPlusLevels must not be negative")
	}
	if s.MatchXp1v1.Win < 0 || s.MatchXp1v1.Loss < 0 || s.MatchXp2v2.Win < 0 || s.MatchXp2v2.Loss < 0 || s.DailyMatchXpCap < 0 {
		return fail("match XP and dailyMatchXpCap must not be negative")
	}
	if s.BotSeatRewardPercent < 0 || s.BotSeatRewardPercent > 100 {
		return fail("botSeatRewardPercent must be between 0 and 100")
	}
	if s.LevelCoinPrice < 1 || s.LevelCrystalPrice < 1 {
		return fail("levelCoinPrice and levelCrystalPrice must be at least 1")
	}
	ids := map[string]bool{}
	for _, season := range s.Seasons {
		if ids[season.ID] {
			return fail("season %s listed twice", season.ID)
		}
		ids[season.ID] = true
		if err := s.validateSeason(season); err != nil {
			return fail("season %s: %v", season.ID, err)
		}
	}
	return s.validateWindows(fail)
}

func (s *Snapshot) validateWindows(fail func(string, ...any) error) error {
	ordered := slices.Clone(s.Seasons)
	slices.SortFunc(ordered, func(a, b Season) int { return a.Start.Compare(b.Start) })
	for i := 1; i < len(ordered); i++ {
		if ordered[i].Start.Before(ordered[i-1].End) {
			return fail("seasons %s and %s overlap", ordered[i-1].ID, ordered[i].ID)
		}
	}
	return nil
}

func (s *Snapshot) validateSeason(season Season) error {
	curve := season.Curve
	switch {
	case !seasonID.MatchString(season.ID):
		return errors.New("id must be season_<n>")
	case !season.End.After(season.Start):
		return errors.New("end must be after start")
	case curve.Levels < 1 || curve.XpBase < 1 || curve.XpStep < 1 || curve.XpMax < 1:
		return errors.New("curve values must be at least 1")
	case curve.XpMax < curve.XpBase:
		return errors.New("xpMax must be at least xpBase")
	case curve.Levels <= s.PremiumPlusLevels:
		return errors.New("levels must be above premiumPlusLevels")
	}
	for level := 1; level <= curve.Levels; level++ {
		if _, ok := season.Premium[level]; !ok {
			return fmt.Errorf("premium level %d has no bundle", level)
		}
	}
	if err := validateTrack(season.Free, curve.Levels, wallet.Coin, s.FreeTrackCoins); err != nil {
		return fmt.Errorf("free track: %w", err)
	}
	if err := validateTrack(season.Premium, curve.Levels, wallet.Crystal, s.PremiumTrackCrystals); err != nil {
		return fmt.Errorf("premium track: %w", err)
	}
	return nil
}

func validateTrack(track map[int]reward.Bundle, levels int, currency wallet.Currency, budget int64) error {
	var paid int64
	for level, bundle := range track {
		if level < 1 || level > levels {
			return fmt.Errorf("level %d is outside 1 to %d", level, levels)
		}
		if err := bundle.Validate(); err != nil {
			return fmt.Errorf("level %d: %v", level, err)
		}
		if len(bundle.Medals) > 0 || bundle.PassXP != 0 {
			return fmt.Errorf("level %d grants medals or pass XP", level)
		}
		amounts := map[wallet.Currency]int64{wallet.Coin: bundle.Coins, wallet.Crystal: bundle.Crystals}
		for other, amount := range amounts {
			if other != currency && amount > 0 {
				return fmt.Errorf("level %d grants %s", level, other)
			}
		}
		paid += amounts[currency]
	}
	if paid > budget {
		return fmt.Errorf("%d %s exceed the budget of %d", paid, currency, budget)
	}
	return nil
}

type Replaces func(disabled, replacement string) bool

func CheckFrozen(published, next *Snapshot, now time.Time, replaces Replaces) (bool, error) {
	replaced := false
	for _, running := range published.Seasons {
		if !running.Running(now) {
			continue
		}
		changed, ok := next.Season(running.ID)
		switch {
		case !ok:
			return false, fmt.Errorf("%w: season %s removed", ErrFrozen, running.ID)
		case changed.Curve != running.Curve:
			return false, fmt.Errorf("%w: season %s levels or curve", ErrFrozen, running.ID)
		}
		for _, track := range []string{Free, Premium} {
			swapped, ok := onlyReplaced(running.Track(track), changed.Track(track), replaces)
			if !ok {
				return false, fmt.Errorf("%w: season %s track bundles", ErrFrozen, running.ID)
			}
			replaced = replaced || swapped
		}
	}
	return replaced, nil
}

func onlyReplaced(before, after map[int]reward.Bundle, replaces Replaces) (bool, bool) {
	if len(before) != len(after) {
		return false, false
	}
	replaced := false
	for level, old := range before {
		next, ok := after[level]
		if !ok {
			return false, false
		}
		if reflect.DeepEqual(old, next) {
			continue
		}
		if !bundleReplaces(old, next, replaces) {
			return false, false
		}
		replaced = true
	}
	return replaced, true
}

func bundleReplaces(old, next reward.Bundle, replaces Replaces) bool {
	if replaces == nil || next.Revision != old.Revision+1 || len(next.Cosmetics) != len(old.Cosmetics) {
		return false
	}
	for i, id := range old.Cosmetics {
		if next.Cosmetics[i] != id && !replaces(id, next.Cosmetics[i]) {
			return false
		}
	}
	old.Revision, old.Cosmetics, next.Revision, next.Cosmetics = 0, nil, 0, nil
	return reflect.DeepEqual(old, next)
}

func TrackSource(season, track string, level int) cosmetics.Source {
	return cosmetics.Source{Type: SourcePass, Ref: fmt.Sprintf("pass.%s.%s.level_%d", season, track, level)}
}

func (s *Snapshot) CheckSources(grantable func(id string, source cosmetics.Source) bool, mythic func(id string) bool) error {
	for _, season := range s.Seasons {
		for _, track := range []string{Free, Premium} {
			for level := 1; level <= season.Curve.Levels; level++ {
				source := TrackSource(season.ID, track, level)
				for _, id := range season.Track(track)[level].Cosmetics {
					if mythic(id) {
						return fmt.Errorf("%w: %s grants the mythic item %s", ErrInvalidConfig, source.Ref, id)
					}
					if !grantable(id, source) {
						return fmt.Errorf("%w: %s grants %s, whose sources do not list it", ErrInvalidConfig, source.Ref, id)
					}
				}
			}
		}
	}
	return nil
}

func (s *Snapshot) LevelPrice(currency wallet.Currency) int64 {
	if currency == wallet.Crystal {
		return s.LevelCrystalPrice
	}
	return s.LevelCoinPrice
}

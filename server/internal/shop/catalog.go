package shop

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/reward"
)

const (
	Domain     = "shop"
	Schema     = 1
	SourceType = "shop"

	KindCosmetic   = "cosmetic"
	KindBundle     = "bundle"
	KindCoins      = "coins"
	KindPass       = "pass"
	KindPassLevels = "pass_levels"

	SectionFeatured = "featured"
	SectionDaily    = "daily_rotation"
	SectionWeekly   = "weekly_rotation"
	SectionEvent    = "event"
	SectionPass     = "pass"
	SectionCoins    = "coins"

	PeriodDaily  = "daily"
	PeriodWeekly = "weekly"
)

var (
	Sections = []string{SectionFeatured, SectionDaily, SectionWeekly, SectionEvent, SectionPass, SectionCoins}
	Kinds    = []string{KindCosmetic, KindBundle, KindCoins, KindPass, KindPassLevels}

	ErrInvalid       = errors.New("shop: invalid catalog")
	ErrInvalidConfig = errors.New("shop: invalid config")

	offerID   = regexp.MustCompile(`^offer\.[a-z0-9_]+$`)
	productID = regexp.MustCompile(`^[a-z0-9_]+$`)

	rotationSection = map[string]string{PeriodDaily: SectionDaily, PeriodWeekly: SectionWeekly}
)

type Config struct {
	DailySlots   int
	WeeklySlots  int
	ResetHourUTC int
}

func DefaultConfig() Config {
	return Config{DailySlots: 4, WeeklySlots: 2}
}

func (c Config) Validate() error {
	if c.DailySlots < 1 || c.WeeklySlots < 1 {
		return fmt.Errorf("%w: rotation slots must be at least 1", ErrInvalidConfig)
	}
	if c.ResetHourUTC < 0 || c.ResetHourUTC > 23 {
		return fmt.Errorf("%w: reset hour must be between 0 and 23", ErrInvalidConfig)
	}
	return nil
}

func (c Config) Slots(period string) int {
	if period == PeriodWeekly {
		return c.WeeklySlots
	}
	return c.DailySlots
}

func (c Config) Period(period string, now time.Time) (time.Time, time.Time) {
	utc := now.UTC()
	start := time.Date(utc.Year(), utc.Month(), utc.Day(), c.ResetHourUTC, 0, 0, 0, time.UTC)
	if start.After(utc) {
		start = start.AddDate(0, 0, -1)
	}
	if period != PeriodWeekly {
		return start, start.AddDate(0, 0, 1)
	}
	for start.Weekday() != time.Monday {
		start = start.AddDate(0, 0, -1)
	}
	return start, start.AddDate(0, 0, 7)
}

type Window struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end,omitzero"`
}

func (w Window) Open(now time.Time) bool {
	return !now.Before(w.Start) && (w.End.IsZero() || now.Before(w.End))
}

type Eligibility struct {
	MaxAccountAgeHours int  `json:"maxAccountAgeHours,omitempty"`
	FirstPurchaseOnly  bool `json:"firstPurchaseOnly,omitempty"`
}

type Offer struct {
	ID                   string        `json:"id"`
	Kind                 string        `json:"kind"`
	Section              string        `json:"section"`
	PriceCoins           int64         `json:"priceCoins,omitempty"`
	ProductID            string        `json:"productId,omitempty"`
	Bundle               reward.Bundle `json:"bundle"`
	Window               Window        `json:"window"`
	LimitPerAccount      int           `json:"limitPerAccount,omitempty"`
	Eligibility          Eligibility   `json:"eligibility,omitzero"`
	NameKey              string        `json:"nameKey"`
	ReferencePriceCoins  int64         `json:"referencePriceCoins,omitempty"`
	RegionalOverrideNote string        `json:"regionalOverrideNote,omitempty"`
}

func (o Offer) Validate() error {
	fail := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s: %s", ErrInvalid, o.ID, fmt.Sprintf(format, args...))
	}
	switch {
	case !offerID.MatchString(o.ID):
		return fmt.Errorf("%w: offer id %q", ErrInvalid, o.ID)
	case !contains(Kinds, o.Kind):
		return fail("kind %q", o.Kind)
	case !contains(Sections, o.Section):
		return fail("section %q", o.Section)
	case o.NameKey == "":
		return fail("nameKey is required")
	case o.PriceCoins < 0:
		return fail("negative price")
	case (o.PriceCoins > 0) == (o.ProductID != ""):
		return fail("price is coins or a store product, never both or neither")
	case o.ProductID != "" && !productID.MatchString(o.ProductID):
		return fail("product id %q", o.ProductID)
	case o.Window.Start.IsZero():
		return fail("window start is required")
	case !o.Window.End.IsZero() && !o.Window.End.After(o.Window.Start):
		return fail("window ends before it starts")
	case o.LimitPerAccount < 0 || o.Eligibility.MaxAccountAgeHours < 0 || o.ReferencePriceCoins < 0:
		return fail("negative limit, eligibility or reference price")
	case o.ReferencePriceCoins > 0 && o.ReferencePriceCoins <= o.PriceCoins:
		return fail("reference price must be above the coin price")
	}
	if err := o.validateBundle(); err != nil {
		return fail("%v", err)
	}
	return o.validateSection()
}

func (o Offer) validateBundle() error {
	if o.Kind == KindPass || o.Kind == KindPassLevels {
		if o.Bundle.ID == "" {
			return nil
		}
	}
	if err := o.Bundle.Validate(); err != nil {
		return err
	}
	switch {
	case len(o.Bundle.Medals) > 0:
		return errors.New("an offer never carries a medal")
	case o.PriceCoins > 0 && o.Bundle.Coins > 0:
		return errors.New("a coin-priced offer grants no coins")
	case o.Kind == KindCosmetic && (len(o.Bundle.Cosmetics) != 1 || o.Bundle.Coins > 0 || o.Bundle.PassXP > 0):
		return errors.New("a cosmetic offer grants exactly one cosmetic")
	case o.Kind == KindCosmetic && o.LimitPerAccount != 1:
		return errors.New("a cosmetic offer is limited to one per account")
	case o.Kind == KindBundle && len(o.Bundle.Cosmetics) == 0:
		return errors.New("a bundle offer grants at least one cosmetic")
	case o.Kind == KindCoins && (o.Bundle.Coins == 0 || o.ProductID == ""):
		return errors.New("a coin pack grants coins and is a store product")
	}
	return nil
}

func (o Offer) validateSection() error {
	fail := func(message string) error {
		return fmt.Errorf("%w: %s: section %s %s", ErrInvalid, o.ID, o.Section, message)
	}
	switch o.Section {
	case SectionDaily, SectionWeekly:
		if o.PriceCoins == 0 || (o.Kind != KindCosmetic && o.Kind != KindBundle) {
			return fail("holds cosmetic offers paid with coins")
		}
	case SectionCoins:
		if o.Kind != KindCoins {
			return fail("holds coin packs")
		}
	case SectionPass:
		if o.Kind != KindPass && o.Kind != KindPassLevels {
			return fail("holds pass products")
		}
	}
	return nil
}

type Rotation struct {
	Period string    `json:"period"`
	Start  time.Time `json:"start"`
	Offers []string  `json:"offers"`
}

type Catalog struct {
	Schema    int        `json:"schema"`
	Offers    []Offer    `json:"offers"`
	Rotations []Rotation `json:"rotations"`
	index     map[string]Offer
}

func Parser(config Config) func(raw []byte) (*Catalog, error) {
	return func(raw []byte) (*Catalog, error) {
		return Parse(raw, config)
	}
}

func Parse(raw []byte, config Config) (*Catalog, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	var catalog Catalog
	if err := json.Unmarshal(raw, &catalog); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	if catalog.Schema != Schema {
		return nil, fmt.Errorf("%w: schema %d", ErrInvalid, catalog.Schema)
	}
	catalog.index = make(map[string]Offer, len(catalog.Offers))
	for _, offer := range catalog.Offers {
		if err := offer.Validate(); err != nil {
			return nil, err
		}
		if _, dup := catalog.index[offer.ID]; dup {
			return nil, fmt.Errorf("%w: offer %s listed twice", ErrInvalid, offer.ID)
		}
		catalog.index[offer.ID] = offer
	}
	if err := catalog.validateRotations(config); err != nil {
		return nil, err
	}
	sort.SliceStable(catalog.Rotations, func(i, j int) bool {
		return catalog.Rotations[i].Start.Before(catalog.Rotations[j].Start)
	})
	return &catalog, nil
}

func (c *Catalog) validateRotations(config Config) error {
	type slot struct {
		period string
		start  time.Time
	}
	seen := make(map[slot]bool, len(c.Rotations))
	for _, rotation := range c.Rotations {
		section, known := rotationSection[rotation.Period]
		start, _ := config.Period(rotation.Period, rotation.Start)
		switch {
		case !known:
			return fmt.Errorf("%w: rotation period %q", ErrInvalid, rotation.Period)
		case rotation.Start.IsZero() || !start.Equal(rotation.Start):
			return fmt.Errorf("%w: %s rotation start %s is not a period start", ErrInvalid, rotation.Period, rotation.Start.Format(time.RFC3339))
		case len(rotation.Offers) == 0 || len(rotation.Offers) > config.Slots(rotation.Period):
			return fmt.Errorf("%w: %s rotation at %s has %d offers, slots 1 to %d", ErrInvalid, rotation.Period, rotation.Start.Format(time.RFC3339), len(rotation.Offers), config.Slots(rotation.Period))
		case seen[slot{rotation.Period, rotation.Start.UTC()}]:
			return fmt.Errorf("%w: %s rotation at %s listed twice", ErrInvalid, rotation.Period, rotation.Start.Format(time.RFC3339))
		}
		seen[slot{rotation.Period, rotation.Start.UTC()}] = true
		if err := c.validateRotationOffers(rotation, section); err != nil {
			return err
		}
	}
	return nil
}

func (c *Catalog) validateRotationOffers(rotation Rotation, section string) error {
	listed := make(map[string]bool, len(rotation.Offers))
	for _, id := range rotation.Offers {
		offer, ok := c.index[id]
		switch {
		case !ok:
			return fmt.Errorf("%w: %s rotation lists unknown offer %s", ErrInvalid, rotation.Period, id)
		case offer.Section != section:
			return fmt.Errorf("%w: %s rotation lists %s of section %s", ErrInvalid, rotation.Period, id, offer.Section)
		case listed[id]:
			return fmt.Errorf("%w: %s rotation lists %s twice", ErrInvalid, rotation.Period, id)
		}
		listed[id] = true
	}
	return nil
}

func (c *Catalog) Offer(id string) (Offer, bool) {
	offer, ok := c.index[id]
	return offer, ok
}

func (c *Catalog) Rotation(period string, periodStart time.Time) (Rotation, bool) {
	var current Rotation
	found := false
	for _, rotation := range c.Rotations {
		if rotation.Period == period && !rotation.Start.After(periodStart) {
			current, found = rotation, true
		}
	}
	return current, found
}

type Section struct {
	Name   string
	EndsAt time.Time
	Offers []Offer
}

func (c *Catalog) Sections(config Config, now time.Time) []Section {
	sections := make([]Section, 0, len(Sections))
	for _, name := range Sections {
		section := Section{Name: name, Offers: []Offer{}}
		if period, rotates := rotationPeriod(name); rotates {
			start, end := config.Period(period, now)
			section.EndsAt = end
			if rotation, ok := c.Rotation(period, start); ok {
				for _, id := range rotation.Offers {
					if offer := c.index[id]; offer.Window.Open(now) {
						section.Offers = append(section.Offers, offer)
					}
				}
			}
			sections = append(sections, section)
			continue
		}
		for _, offer := range c.Offers {
			if offer.Section == name && offer.Window.Open(now) {
				section.Offers = append(section.Offers, offer)
			}
		}
		sections = append(sections, section)
	}
	return sections
}

func (c *Catalog) Available(config Config, offer Offer, now time.Time) bool {
	if !offer.Window.Open(now) {
		return false
	}
	period, rotates := rotationPeriod(offer.Section)
	if !rotates {
		return true
	}
	start, _ := config.Period(period, now)
	rotation, ok := c.Rotation(period, start)
	return ok && contains(rotation.Offers, offer.ID)
}

func rotationPeriod(section string) (string, bool) {
	for period, s := range rotationSection {
		if s == section {
			return period, true
		}
	}
	return "", false
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

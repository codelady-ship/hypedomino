package storeprice

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"
)

const (
	Domain = "store_prices"
	Schema = 1

	TierFull = "full"
)

var (
	ErrInvalid = errors.New("storeprice: invalid store prices")

	Products = []string{"crystals_tier1", "crystals_tier2", "crystals_tier3", "crystals_tier4", "crystals_tier5", "pass_premium", "pass_premium_plus", "offer_starter"}

	tierName = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`)

	notStoreCountries = []string{"RU"}

	isoCountries = strings.Fields(`AD AE AF AG AI AL AM AO AQ AR AS AT AU AW AX AZ BA BB BD BE BF BG BH BI BJ BL BM BN BO BQ BR BS BT BV BW BY BZ
		CA CC CD CF CG CH CI CK CL CM CN CO CR CU CV CW CX CY CZ DE DJ DK DM DO DZ EC EE EG EH ER ES ET FI FJ FK FM FO FR
		GA GB GD GE GF GG GH GI GL GM GN GP GQ GR GS GT GU GW GY HK HM HN HR HT HU ID IE IL IM IN IO IQ IR IS IT JE JM JO JP
		KE KG KH KI KM KN KP KR KW KY KZ LA LB LC LI LK LR LS LT LU LV LY MA MC MD ME MF MG MH MK ML MM MN MO MP MQ MR MS MT MU MV MW MX MY MZ
		NA NC NE NF NG NI NL NO NP NR NU NZ OM PA PE PF PG PH PK PL PM PN PR PS PT PW PY QA RE RO RS RU RW
		SA SB SC SD SE SG SH SI SJ SK SL SM SN SO SR SS ST SV SX SY SZ TC TD TF TG TH TJ TK TL TM TN TO TR TT TV TW TZ
		UA UG UM US UY UZ VA VC VE VG VI VN VU WF WS YE YT ZA ZM ZW`)
)

type Config struct {
	Schema       int                           `json:"schema"`
	ReferenceUsd map[string]float64            `json:"referenceUsd"`
	Tiers        map[string]int                `json:"tiers"`
	Countries    map[string]string             `json:"countries"`
	Overrides    map[string]map[string]float64 `json:"overrides"`
}

type Target struct {
	Product  string `json:"product"`
	Country  string `json:"country"`
	Tier     string `json:"tier,omitempty"`
	Cents    int64  `json:"cents"`
	Override bool   `json:"override,omitempty"`
}

func StoreCountries() []string {
	return slices.DeleteFunc(slices.Clone(isoCountries), func(code string) bool { return slices.Contains(notStoreCountries, code) })
}

func Default() *Config {
	config := &Config{
		Schema: Schema,
		ReferenceUsd: map[string]float64{
			"crystals_tier1": 0.99, "crystals_tier2": 4.99, "crystals_tier3": 9.99, "crystals_tier4": 19.99, "crystals_tier5": 49.99,
			"pass_premium": 4.99, "pass_premium_plus": 9.99, "offer_starter": 1.99,
		},
		Tiers:     map[string]int{TierFull: 100, "high": 80, "mid": 60, "low": 45},
		Countries: map[string]string{},
		Overrides: map[string]map[string]float64{},
	}
	defaults := map[string]string{
		"high": "PT GR CY MT SI CZ SK EE LV LT PL HU HR CL UY PA BN",
		"mid":  "AZ TR GE AM KZ UZ KG TJ TM BY UA MD RO BG RS BA ME MK AL MX BR AR CO PE EC CR DO MY TH CN ZA MA TN JO LB IQ MN",
		"low":  "IN PK BD LK NP ID PH VN KH LA MM EG NG KE GH TZ UG ET DZ BO PY VE HN NI GT SV",
	}
	for tier, codes := range defaults {
		for _, code := range strings.Fields(codes) {
			config.Countries[code] = tier
		}
	}
	return config
}

func Parse(raw []byte) (*Config, error) {
	config := &Config{}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(config); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return config, nil
}

func fail(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
}

func (c *Config) Validate() error {
	if c.Schema != Schema {
		return fail("schema %d", c.Schema)
	}
	for _, check := range []func() error{c.validateReferences, c.validateTiers, c.validateCountries, c.validateOverrides} {
		if err := check(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) validateReferences() error {
	for product, usd := range c.ReferenceUsd {
		if !slices.Contains(Products, product) {
			return fail("unknown product %q", product)
		}
		if _, ok := cents(usd); !ok {
			return fail("reference price of %s is %v, a positive USD amount with at most two decimals", product, usd)
		}
	}
	for _, product := range Products {
		if _, set := c.ReferenceUsd[product]; !set {
			return fail("product %s has no reference price", product)
		}
	}
	return nil
}

func (c *Config) validateTiers() error {
	for tier, percent := range c.Tiers {
		if !tierName.MatchString(tier) {
			return fail("tier name %q", tier)
		}
		if percent < 1 || percent > 100 {
			return fail("tier %s is %d%%, outside 1 to 100", tier, percent)
		}
	}
	return nil
}

func (c *Config) validateCountries() error {
	for country, tier := range c.Countries {
		if err := checkCountry(country); err != nil {
			return err
		}
		if _, known := c.Tiers[tier]; !known {
			return fail("country %s has unknown tier %q", country, tier)
		}
	}
	return nil
}

func (c *Config) validateOverrides() error {
	for product, prices := range c.Overrides {
		reference, known := c.ReferenceUsd[product]
		if !known {
			return fail("override for unknown product %q", product)
		}
		for country, usd := range prices {
			if err := checkCountry(country); err != nil {
				return err
			}
			amount, ok := cents(usd)
			if !ok {
				return fail("override of %s in %s is %v, a positive USD amount with at most two decimals", product, country, usd)
			}
			if limit, _ := cents(reference); amount > limit {
				return fail("override of %s in %s is %v, above its reference price %v", product, country, usd, reference)
			}
		}
	}
	return nil
}

func checkCountry(country string) error {
	if slices.Contains(notStoreCountries, country) {
		return fail("country %s is not a store country", country)
	}
	if !slices.Contains(isoCountries, country) {
		return fail("country code %q is not ISO 3166-1 alpha-2", country)
	}
	return nil
}

func cents(usd float64) (int64, bool) {
	scaled := usd * 100
	rounded := math.Round(scaled)
	if rounded < 1 || math.Abs(scaled-rounded) > 1e-6 {
		return 0, false
	}
	return int64(rounded), true
}

func (c *Config) Target(product, country string) (Target, bool) {
	if usd, set := c.Overrides[product][country]; set {
		amount, _ := cents(usd)
		return Target{Product: product, Country: country, Cents: amount, Override: true}, true
	}
	tier, listed := c.Countries[country]
	reference, known := c.ReferenceUsd[product]
	if !listed || !known {
		return Target{}, false
	}
	base, _ := cents(reference)
	return Target{Product: product, Country: country, Tier: tier, Cents: (base*int64(c.Tiers[tier]) + 50) / 100}, true
}

func (c *Config) Targets() []Target {
	targets, countries := []Target{}, StoreCountries()
	for _, product := range Products {
		for _, country := range countries {
			if target, ok := c.Target(product, country); ok {
				targets = append(targets, target)
			}
		}
	}
	return targets
}

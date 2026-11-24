package storeprice

import (
	"encoding/csv"
	"errors"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestTheExampleIsTheDefault(t *testing.T) {
	raw, err := os.ReadFile("../../../docs/economy/examples/store_prices.json")
	if err != nil {
		t.Fatal(err)
	}
	example, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(example, Default()) {
		t.Fatalf("the example differs from the default:\n%+v\n%+v", example, Default())
	}
	if err := Default().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestTheReferencePricesAreTheBalanceSheetPrices(t *testing.T) {
	file, err := os.Open("../../../docs/economy/balance.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	sheet := map[string][]string{}
	for _, row := range rows {
		sheet[row[0]] = strings.Split(row[1], ";")
	}
	keys := map[string]string{"pass_premium": "PassPremiumUsd", "pass_premium_plus": "PassPremiumPlusUsd", "offer_starter": "StarterOfferUsd"}
	for tier := 1; tier <= 5; tier++ {
		keys["crystals_tier"+strconv.Itoa(tier)] = "CrystalPackUsd"
	}
	for product, key := range keys {
		index := 0
		if strings.HasPrefix(product, "crystals_tier") {
			index, _ = strconv.Atoi(strings.TrimPrefix(product, "crystals_tier"))
			index--
		}
		want, err := strconv.ParseFloat(sheet[key][index], 64)
		if err != nil || Default().ReferenceUsd[product] != want {
			t.Errorf("%s: default %v, %s says %v (%v)", product, Default().ReferenceUsd[product], key, want, err)
		}
	}
}

func TestBrokenStorePricesAreRefused(t *testing.T) {
	cases := []struct {
		name   string
		change func(*Config)
	}{
		{"wrong schema", func(c *Config) { c.Schema = 2 }},
		{"unknown product reference", func(c *Config) { c.ReferenceUsd["coins_tier1"] = 0.99 }},
		{"missing product reference", func(c *Config) { delete(c.ReferenceUsd, "offer_starter") }},
		{"zero reference", func(c *Config) { c.ReferenceUsd["pass_premium"] = 0 }},
		{"reference with three decimals", func(c *Config) { c.ReferenceUsd["pass_premium"] = 4.995 }},
		{"percent zero", func(c *Config) { c.Tiers["low"] = 0 }},
		{"percent above 100", func(c *Config) { c.Tiers["high"] = 101 }},
		{"bad tier name", func(c *Config) { c.Tiers["Mid Tier"] = 60 }},
		{"unknown tier", func(c *Config) { c.Countries["AZ"] = "cheap" }},
		{"lowercase country", func(c *Config) { c.Countries["az"] = "mid" }},
		{"not an ISO country", func(c *Config) { c.Countries["UK"] = "high" }},
		{"three letter country", func(c *Config) { c.Countries["AZE"] = "mid" }},
		{"Russia in a tier", func(c *Config) { c.Countries["RU"] = "mid" }},
		{"Russia overridden", func(c *Config) { c.Overrides["pass_premium"] = map[string]float64{"RU": 2.99} }},
		{"override of unknown product", func(c *Config) { c.Overrides["coins_tier1"] = map[string]float64{"AZ": 0.49} }},
		{"override in a bad country", func(c *Config) { c.Overrides["pass_premium"] = map[string]float64{"XX": 2.99} }},
		{"override above reference", func(c *Config) { c.Overrides["pass_premium"] = map[string]float64{"AZ": 5.99} }},
		{"zero override", func(c *Config) { c.Overrides["pass_premium"] = map[string]float64{"AZ": 0} }},
		{"override with three decimals", func(c *Config) { c.Overrides["pass_premium"] = map[string]float64{"AZ": 2.999} }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			config := Default()
			c.change(config)
			if err := config.Validate(); !errors.Is(err, ErrInvalid) {
				t.Fatalf("Validate() = %v", err)
			}
		})
	}
}

func TestUnknownKeysAreRefused(t *testing.T) {
	for _, raw := range []string{
		`{"schema":1,"referenceUsd":{},"tiers":{},"countries":{},"overrides":{},"currency":"AZN"}`,
		`not json`,
	} {
		if _, err := Parse([]byte(raw)); !errors.Is(err, ErrInvalid) {
			t.Fatalf("Parse(%s) = %v", raw, err)
		}
	}
}

func TestTargets(t *testing.T) {
	config := Default()
	config.Overrides["pass_premium"] = map[string]float64{"AZ": 2.49, "DE": 3.99}
	cases := []struct {
		product, country string
		want             Target
		ok               bool
	}{
		{"pass_premium", "AZ", Target{Product: "pass_premium", Country: "AZ", Cents: 249, Override: true}, true},
		{"pass_premium", "DE", Target{Product: "pass_premium", Country: "DE", Cents: 399, Override: true}, true},
		{"pass_premium", "TR", Target{Product: "pass_premium", Country: "TR", Tier: "mid", Cents: 299}, true},
		{"crystals_tier1", "AZ", Target{Product: "crystals_tier1", Country: "AZ", Tier: "mid", Cents: 59}, true},
		{"crystals_tier5", "PL", Target{Product: "crystals_tier5", Country: "PL", Tier: "high", Cents: 3999}, true},
		{"offer_starter", "IN", Target{Product: "offer_starter", Country: "IN", Tier: "low", Cents: 90}, true},
		{"crystals_tier3", "US", Target{}, false},
		{"crystals_tier3", "RU", Target{}, false},
	}
	for _, c := range cases {
		got, ok := config.Target(c.product, c.country)
		if ok != c.ok || got != c.want {
			t.Errorf("Target(%s, %s) = %+v %v, want %+v %v", c.product, c.country, got, ok, c.want, c.ok)
		}
	}
}

func TestAzerbaijanAndTurkeyStayAtSixtyPercent(t *testing.T) {
	want := map[string]int64{"crystals_tier1": 59, "crystals_tier2": 299, "crystals_tier3": 599, "crystals_tier4": 1199, "crystals_tier5": 2999, "pass_premium": 299, "pass_premium_plus": 599, "offer_starter": 119}
	for _, country := range []string{"AZ", "TR"} {
		for product, cents := range want {
			if got, ok := Default().Target(product, country); !ok || got.Cents != cents {
				t.Errorf("%s in %s: %+v, want %d cents", product, country, got, cents)
			}
		}
	}
}

func TestAnExplicitFullTierKeepsTheReferencePrice(t *testing.T) {
	config := Default()
	config.Countries["US"] = TierFull
	if got, ok := config.Target("crystals_tier5", "US"); !ok || got.Cents != 4999 {
		t.Fatalf("full tier %+v", got)
	}
}

func TestTargetsListEveryListedCountryOncePerProduct(t *testing.T) {
	config := Default()
	targets := config.Targets()
	if len(targets) != len(Products)*len(config.Countries) {
		t.Fatalf("%d targets for %d countries", len(targets), len(config.Countries))
	}
	if slices.ContainsFunc(targets, func(target Target) bool { return target.Country == "US" || target.Country == "RU" }) {
		t.Fatal("an unlisted country has a target")
	}
	if slices.Contains(StoreCountries(), "RU") || !slices.Contains(StoreCountries(), "AZ") || len(StoreCountries()) != 248 {
		t.Fatalf("store countries %d", len(StoreCountries()))
	}
}

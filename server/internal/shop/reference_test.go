package shop

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
)

func TestTheShopDefaultsAreTheBalanceSheetValues(t *testing.T) {
	file, err := os.Open("../../../docs/economy/balance.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	sheet := map[string]string{}
	for _, row := range rows {
		sheet[row[0]] = row[1]
	}
	defaults := DefaultConfig()
	for key, value := range map[string]int{
		"ShopDailyRotationSlots":   defaults.DailySlots,
		"ShopWeeklyRotationSlots":  defaults.WeeklySlots,
		"ShopRotationResetHourUtc": defaults.ResetHourUTC,
		"ReferencePriceDays":       defaults.ReferencePriceDays,
	} {
		if want, err := strconv.Atoi(sheet[key]); err != nil || value != want {
			t.Errorf("%s: default %d, balance sheet %q (%v)", key, value, sheet[key], err)
		}
	}
}

func publishedPricing(t *testing.T, offers ...Offer) snapshot.Published {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"offers": offers})
	if err != nil {
		t.Fatal(err)
	}
	return snapshot.Published{Content: raw}
}

func TestAReferencePriceIsTheLowestEarlierPriceOfItsOffer(t *testing.T) {
	coins := func(price, reference int64) Offer {
		return Offer{ID: "offer.x", PriceCoins: price, ReferencePriceCoins: reference}
	}
	crystals := func(price, reference int64) Offer {
		return Offer{ID: "offer.x", PriceCrystals: price, ReferencePriceCrystals: reference}
	}
	cases := []struct {
		name     string
		offer    Offer
		inEffect [][]Offer
		refusal  string
	}{
		{"no reference price", coins(3000, 0), nil, ""},
		{"the only earlier price", coins(3000, 3500), [][]Offer{{coins(3500, 0)}}, ""},
		{"the lowest of several earlier prices", coins(3000, 3200), [][]Offer{{coins(3500, 0)}, {coins(3200, 0)}, {coins(4000, 0)}}, ""},
		{"above the lowest earlier price", coins(3000, 3500), [][]Offer{{coins(3500, 0)}, {coins(3200, 0)}}, "offer.x reference price 3500 is not its lowest price 3200"},
		{"below every earlier price", coins(3000, 3400), [][]Offer{{coins(3500, 0)}}, "offer.x reference price 3400 is not its lowest price 3500"},
		{"no earlier version", coins(3000, 3500), nil, "offer.x has no earlier price"},
		{"offer absent from the window", coins(3000, 3500), [][]Offer{{}}, "offer.x has no earlier price"},
		{"earlier price in another currency", coins(3000, 3500), [][]Offer{{crystals(400, 0)}}, "offer.x has no earlier price"},
		{"running discount republished unchanged", coins(3000, 3500), [][]Offer{{coins(3000, 3500)}, {coins(3500, 0)}}, ""},
		{"running discount lowered again", coins(2500, 3500), [][]Offer{{coins(3000, 3500)}, {coins(3500, 0)}}, "offer.x reference price 3500 is not its lowest price 3000"},
		{"crystal reference", crystals(400, 500), [][]Offer{{crystals(500, 0)}}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inEffect := make([]snapshot.Published, 0, len(tc.inEffect))
			for _, offers := range tc.inEffect {
				inEffect = append(inEffect, publishedPricing(t, offers...))
			}
			catalog := &Catalog{Offers: []Offer{tc.offer}}
			err := catalog.CheckReferencePrices(inEffect)
			switch {
			case tc.refusal == "" && err != nil:
				t.Fatalf("refused: %v", err)
			case tc.refusal != "" && (!errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), tc.refusal)):
				t.Fatalf("got %v, want %q", err, tc.refusal)
			}
		})
	}
}

func TestAnUnreadableEarlierVersionIsRefused(t *testing.T) {
	catalog := &Catalog{Offers: []Offer{{ID: "offer.x", PriceCoins: 3000, ReferencePriceCoins: 3500}}}
	if err := catalog.CheckReferencePrices([]snapshot.Published{{Version: 7, Content: []byte(`{`)}}); !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "shop version 7") {
		t.Fatalf("got %v", err)
	}
}

package shop

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
)

func TestOffersAndTheFirstPurchaseGrantOnlyItemsListingThem(t *testing.T) {
	items, err := cosmetics.Load("../../../docs/cosmetics/examples/catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(*Catalog){
		"shop offer.starter grants avatar.paper_kite": func(c *Catalog) {
			for i := range c.Offers {
				if c.Offers[i].ID == "offer.starter" {
					c.Offers[i].Bundle.Cosmetics = []string{"avatar.comet_fox", "avatar.paper_kite"}
				}
			}
		},
		"shop offer.table_night_market grants avatar.missing": func(c *Catalog) {
			for i := range c.Offers {
				if c.Offers[i].ID == "offer.table_night_market" {
					c.Offers[i].Bundle.Cosmetics = []string{"avatar.missing"}
				}
			}
		},
		"first_purchase bundle.first_purchase grants avatar.comet_fox": func(c *Catalog) {
			bonus := *c.FirstPurchase
			bonus.Cosmetics = []string{"avatar.comet_fox"}
			c.FirstPurchase = &bonus
		},
	}
	for reason, change := range cases {
		catalog := example(t)
		change(catalog)
		if err := catalog.CheckSources(items); err == nil || !strings.Contains(err.Error(), reason) {
			t.Fatalf("%s: %v", reason, err)
		}
	}
	catalog := example(t)
	catalog.FirstPurchase = &reward.Bundle{ID: "bundle.coins_only", Crystals: 100}
	if err := catalog.CheckSources(items); err != nil {
		t.Fatalf("a bonus without items: %v", err)
	}
}

func TestAMythicItemIsSoldOnlyInATimeLimitedOffer(t *testing.T) {
	items, err := cosmetics.Load("../../../docs/cosmetics/examples/catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	end := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name    string
		section string
		end     time.Time
		allowed bool
	}{
		{"weekly rotation", SectionWeekly, end, true},
		{"open-ended daily rotation", SectionDaily, time.Time{}, true},
		{"featured with an end", SectionFeatured, end, true},
		{"event with an end", SectionEvent, end, true},
		{"open-ended featured", SectionFeatured, time.Time{}, false},
		{"open-ended event", SectionEvent, time.Time{}, false},
		{"pass", SectionPass, end, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			catalog := example(t)
			for i := range catalog.Offers {
				if catalog.Offers[i].ID == "offer.table_night_market" {
					catalog.Offers[i].Section = tc.section
					catalog.Offers[i].Window.End = tc.end
				}
			}
			err := catalog.CheckSources(items)
			switch {
			case tc.allowed && err != nil:
				t.Fatalf("refused: %v", err)
			case !tc.allowed && (!errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "offer.table_night_market grants the mythic item table.night_market")):
				t.Fatalf("accepted or wrong refusal: %v", err)
			}
		})
	}
}

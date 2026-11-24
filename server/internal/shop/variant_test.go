package shop

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

const variantOffer = `"id":"offer.x","kind":"cosmetic","section":"event","priceCoins":1,"bundle":{"id":"bundle.x","revision":1,"cosmetics":["avatar.x"]},` + okWindow + `,"limitPerAccount":1,"nameKey":"k"`

func TestInvalidVariantsAreRefused(t *testing.T) {
	cases := map[string]struct {
		variants string
		reason   string
	}{
		"id":        {`[{"id":"B","nameKey":"k2"}]`, "variant id"},
		"twice":     {`[{"id":"b","nameKey":"k2"},{"id":"b","position":1}]`, "listed twice"},
		"position":  {`[{"id":"b","position":-1}]`, "position -1"},
		"no change": {`[{"id":"b"}]`, "changes neither"},
		"same name": {`[{"id":"b","nameKey":"k"}]`, "changes neither"},
	}
	for name, c := range cases {
		_, err := Parse([]byte(offerJSON(variantOffer+`,"variants":`+c.variants)), DefaultConfig())
		if err == nil || !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), c.reason) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	catalog, err := Parse([]byte(offerJSON(variantOffer+`,"variants":[{"id":"b","nameKey":"k2"},{"id":"c","position":1}]`)), DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	if offer, _ := catalog.Offer("offer.x"); len(offer.Variants) != 2 || offer.Variants[0].NameKey != "k2" || offer.Variants[1].Position != 1 {
		t.Fatalf("variants %+v", offer.Variants)
	}
}

func TestAVariantChangesNothingButItsNameKeyAndPosition(t *testing.T) {
	catalog, err := Parse([]byte(offerJSON(variantOffer+`,"variants":[{"id":"b","nameKey":"k2","priceCoins":5,"window":{"start":"2030-01-01T00:00:00Z"}}]`)), DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	offer, _ := catalog.Offer("offer.x")
	for account := range 50 {
		shown := Section{Offers: []Offer{offer}}.For(fmt.Sprint("account-", account)).Offers[0]
		shown.NameKey = offer.NameKey
		if fmt.Sprint(shown) != fmt.Sprint(offer) {
			t.Fatalf("variant changed more than the name key: %+v", shown)
		}
	}
}

func TestPlayersSplitStablyAndEvenlyAcrossTheOfferAndItsVariants(t *testing.T) {
	offer := Offer{ID: "offer.x", NameKey: "k", Variants: []Variant{{ID: "b", NameKey: "k2"}, {ID: "c", NameKey: "k3"}}}
	groups := map[string]int{}
	for account := range 3000 {
		id := fmt.Sprintf("acc-%d", account)
		variant, ok := offer.VariantFor(id)
		again, okAgain := offer.VariantFor(id)
		if ok != okAgain || variant != again {
			t.Fatalf("%s moved from %v to %v", id, variant, again)
		}
		groups[variant.ID]++
	}
	for _, group := range []string{"", "b", "c"} {
		if groups[group] < 800 || groups[group] > 1200 {
			t.Fatalf("uneven split %v", groups)
		}
	}
	if _, ok := (Offer{ID: "offer.y", NameKey: "k"}).VariantFor("acc-1"); ok {
		t.Fatal("an offer without variants has no variant")
	}
}

func accountIn(t *testing.T, offer Offer, variant string) string {
	t.Helper()
	for account := range 1000 {
		id := fmt.Sprintf("acc-%d", account)
		if found, ok := offer.VariantFor(id); ok && found.ID == variant || !ok && variant == "" {
			return id
		}
	}
	t.Fatalf("no account in variant %q", variant)
	return ""
}

func TestAVariantMovesTheOfferToItsPositionInTheSection(t *testing.T) {
	moved := Offer{ID: "offer.c", NameKey: "c", Variants: []Variant{{ID: "first", NameKey: "c2", Position: 1}, {ID: "last", Position: 9}}}
	section := Section{Name: SectionEvent, Offers: []Offer{{ID: "offer.a", NameKey: "a"}, {ID: "offer.b", NameKey: "b"}, moved}}

	control := section.For(accountIn(t, moved, ""))
	if !same(ids(control.Offers), []string{"offer.a", "offer.b", "offer.c"}) || control.Offers[2].NameKey != "c" {
		t.Fatalf("control %+v", control.Offers)
	}
	first := section.For(accountIn(t, moved, "first"))
	if !same(ids(first.Offers), []string{"offer.c", "offer.a", "offer.b"}) || first.Offers[0].NameKey != "c2" {
		t.Fatalf("first %+v", first.Offers)
	}
	last := section.For(accountIn(t, moved, "last"))
	if !same(ids(last.Offers), []string{"offer.a", "offer.b", "offer.c"}) || last.Offers[2].NameKey != "c" {
		t.Fatalf("last %+v", last.Offers)
	}
	if !same(ids(section.Offers), []string{"offer.a", "offer.b", "offer.c"}) {
		t.Fatalf("the published section changed: %v", ids(section.Offers))
	}
}

func TestAChosenVariantIsShownWhateverTheAccount(t *testing.T) {
	moved := Offer{ID: "offer.c", NameKey: "c", Variants: []Variant{{ID: "first", NameKey: "c2", Position: 1}}}
	section := Section{Name: SectionEvent, Offers: []Offer{{ID: "offer.a", NameKey: "a"}, {ID: "offer.b", NameKey: "b"}, moved}}

	chosen := section.WithVariants(map[string]string{"offer.c": "first"})
	if !same(ids(chosen.Offers), []string{"offer.c", "offer.a", "offer.b"}) || chosen.Offers[0].NameKey != "c2" {
		t.Fatalf("chosen %+v", chosen.Offers)
	}
	for name, variants := range map[string]map[string]string{"none": nil, "unknown": {"offer.c": "last"}, "other offer": {"offer.a": "first"}} {
		written := section.WithVariants(variants)
		if !same(ids(written.Offers), []string{"offer.a", "offer.b", "offer.c"}) || written.Offers[2].NameKey != "c" {
			t.Fatalf("%s %+v", name, written.Offers)
		}
	}
	if _, ok := moved.Variant(""); ok {
		t.Fatal("an empty variant id names a variant")
	}
}

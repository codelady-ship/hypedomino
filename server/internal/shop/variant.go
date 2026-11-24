package shop

import (
	"fmt"
	"hash/fnv"
	"regexp"
	"slices"
)

var variantID = regexp.MustCompile(`^[a-z0-9_]+$`)

type Variant struct {
	ID       string `json:"id"`
	NameKey  string `json:"nameKey,omitempty"`
	Position int    `json:"position,omitempty"`
}

func (o Offer) validateVariants() error {
	seen := make(map[string]bool, len(o.Variants))
	for _, variant := range o.Variants {
		switch {
		case !variantID.MatchString(variant.ID):
			return fmt.Errorf("variant id %q", variant.ID)
		case seen[variant.ID]:
			return fmt.Errorf("variant %s listed twice", variant.ID)
		case variant.Position < 0:
			return fmt.Errorf("variant %s position %d", variant.ID, variant.Position)
		case (variant.NameKey == "" || variant.NameKey == o.NameKey) && variant.Position == 0:
			return fmt.Errorf("variant %s changes neither the name key nor the position", variant.ID)
		}
		seen[variant.ID] = true
	}
	return nil
}

func (o Offer) VariantFor(account string) (Variant, bool) {
	if len(o.Variants) == 0 {
		return Variant{}, false
	}
	hash := fnv.New32a()
	hash.Write([]byte(o.ID + "\x00" + account))
	group := int(hash.Sum32() % uint32(len(o.Variants)+1))
	if group == 0 {
		return Variant{}, false
	}
	return o.Variants[group-1], true
}

type placedOffer struct {
	offer    Offer
	position int
}

func (o Offer) Variant(id string) (Variant, bool) {
	for _, variant := range o.Variants {
		if variant.ID == id {
			return variant, true
		}
	}
	return Variant{}, false
}

func (s Section) For(account string) Section {
	return s.shown(func(offer Offer) (Variant, bool) { return offer.VariantFor(account) })
}

func (s Section) WithVariants(chosen map[string]string) Section {
	return s.shown(func(offer Offer) (Variant, bool) { return offer.Variant(chosen[offer.ID]) })
}

func (s Section) shown(variantOf func(Offer) (Variant, bool)) Section {
	offers := make([]Offer, 0, len(s.Offers))
	var placed []placedOffer
	for _, offer := range s.Offers {
		variant, ok := variantOf(offer)
		if ok && variant.NameKey != "" {
			offer.NameKey = variant.NameKey
		}
		if !ok || variant.Position == 0 {
			offers = append(offers, offer)
			continue
		}
		placed = append(placed, placedOffer{offer: offer, position: variant.Position})
	}
	slices.SortStableFunc(placed, func(a, b placedOffer) int { return a.position - b.position })
	for _, moved := range placed {
		offers = slices.Insert(offers, min(moved.position-1, len(offers)), moved.offer)
	}
	s.Offers = offers
	return s
}

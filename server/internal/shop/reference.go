package shop

import (
	"encoding/json"
	"fmt"

	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
)

type publishedOffers struct {
	Offers []Offer `json:"offers"`
}

func (c *Catalog) CheckReferencePrices(inEffect []snapshot.Published) error {
	versions := make([]map[string]Offer, 0, len(inEffect))
	for _, published := range inEffect {
		var earlier publishedOffers
		if err := json.Unmarshal(published.Content, &earlier); err != nil {
			return fmt.Errorf("%w: shop version %d: %v", ErrInvalid, published.Version, err)
		}
		offers := make(map[string]Offer, len(earlier.Offers))
		for _, offer := range earlier.Offers {
			offers[offer.ID] = offer
		}
		versions = append(versions, offers)
	}
	for _, offer := range c.Offers {
		if err := offer.referenceIsLowestEarlierPrice(versions); err != nil {
			return err
		}
	}
	return nil
}

func (o Offer) referenceIsLowestEarlierPrice(versions []map[string]Offer) error {
	reference := o.ReferencePriceCoins + o.ReferencePriceCrystals
	if reference == 0 || (len(versions) > 0 && o.samePricing(versions[0][o.ID])) {
		return nil
	}
	lowest := int64(0)
	for _, offers := range versions {
		price := o.earlierPriceIn(offers[o.ID])
		if price > 0 && (lowest == 0 || price < lowest) {
			lowest = price
		}
	}
	switch {
	case lowest == 0:
		return fmt.Errorf("%w: %s has no earlier price in the reference window for its reference price %d", ErrInvalid, o.ID, reference)
	case lowest != reference:
		return fmt.Errorf("%w: %s reference price %d is not its lowest price %d in the reference window", ErrInvalid, o.ID, reference, lowest)
	}
	return nil
}

func (o Offer) earlierPriceIn(earlier Offer) int64 {
	if o.ReferencePriceCoins > 0 {
		return earlier.PriceCoins
	}
	return earlier.PriceCrystals
}

func (o Offer) samePricing(earlier Offer) bool {
	return o.PriceCoins == earlier.PriceCoins && o.PriceCrystals == earlier.PriceCrystals && o.ReferencePriceCoins == earlier.ReferencePriceCoins && o.ReferencePriceCrystals == earlier.ReferencePriceCrystals
}

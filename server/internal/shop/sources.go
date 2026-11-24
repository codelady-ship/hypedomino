package shop

import (
	"fmt"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
)

func (c *Catalog) CheckSources(items *cosmetics.Catalog) error {
	if err := c.checkChestPools(items); err != nil {
		return err
	}
	for _, offer := range c.Offers {
		if offer.Kind == KindChest {
			continue
		}
		if err := grantsListed(items, offer.Bundle.Cosmetics, cosmetics.Source{Type: SourceType, Ref: offer.ID}); err != nil {
			return err
		}
		if err := mythicOnlyWhenTimeLimited(items, offer); err != nil {
			return err
		}
	}
	if c.FirstPurchase == nil {
		return nil
	}
	return grantsListed(items, c.FirstPurchase.Cosmetics, cosmetics.Source{Type: SourceFirstPurchase, Ref: c.FirstPurchase.ID})
}

func grantsListed(items *cosmetics.Catalog, ids []string, source cosmetics.Source) error {
	for _, id := range ids {
		if item, ok := items.Item(id); !ok || !item.Grantable(source) {
			return fmt.Errorf("%w: %s %s grants %s, whose sources do not list it", ErrInvalid, source.Type, source.Ref, id)
		}
	}
	return nil
}

func mythicOnlyWhenTimeLimited(items *cosmetics.Catalog, offer Offer) error {
	if offer.TimeLimited() {
		return nil
	}
	for _, id := range offer.Bundle.Cosmetics {
		if item, ok := items.Item(id); ok && item.Rarity == RarityMythic {
			return fmt.Errorf("%w: %s grants the mythic item %s outside a time-limited offer", ErrInvalid, offer.ID, id)
		}
	}
	return nil
}

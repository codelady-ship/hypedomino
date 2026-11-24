package shop

import (
	"fmt"
	"slices"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
)

type Random interface {
	Uint64() uint64
}

type ChestRules struct {
	RarityPercent map[string]int `json:"rarityPercent"`
	RarePityCount int            `json:"rarePityCount"`
}

func (r ChestRules) Validate() error {
	total := 0
	for rarity, percent := range r.RarityPercent {
		if !slices.Contains(ChestRarities, rarity) || percent < 0 {
			return fmt.Errorf("%w: chest odds %s %d", ErrInvalid, rarity, percent)
		}
		total += percent
	}
	switch {
	case len(r.RarityPercent) != len(ChestRarities) || total != 100:
		return fmt.Errorf("%w: chest odds list common and rare and add up to 100", ErrInvalid)
	case r.RarePityCount < 1:
		return fmt.Errorf("%w: chest rare pity count must be at least 1", ErrInvalid)
	}
	return nil
}

func (r ChestRules) OpensToPity(pity int) int {
	return max(1, r.RarePityCount-pity)
}

type ChestDraw struct {
	Item       string
	Rarity     string
	Roll       int
	PityBefore int
	PityAfter  int
}

func (r ChestRules) Draw(unowned []cosmetics.Item, pity int, random Random) (ChestDraw, bool) {
	roll := int(random.Uint64() % 100)
	rarity := RarityRare
	if roll < r.RarityPercent[RarityCommon] && pity < r.RarePityCount-1 {
		rarity = RarityCommon
	}
	candidates := ofRarity(unowned, rarity)
	if len(candidates) == 0 {
		candidates = ofRarity(unowned, otherRarity(rarity))
	}
	if len(candidates) == 0 {
		return ChestDraw{}, false
	}
	item := candidates[random.Uint64()%uint64(len(candidates))]
	draw := ChestDraw{Item: item.ID, Rarity: item.Rarity, Roll: roll, PityBefore: pity, PityAfter: pity + 1}
	if item.Rarity == RarityRare {
		draw.PityAfter = 0
	}
	return draw, true
}

func ofRarity(items []cosmetics.Item, rarity string) []cosmetics.Item {
	var matching []cosmetics.Item
	for _, item := range items {
		if item.Rarity == rarity {
			matching = append(matching, item)
		}
	}
	return matching
}

func otherRarity(rarity string) string {
	if rarity == RarityRare {
		return RarityCommon
	}
	return RarityRare
}

func chestPool(offer Offer, items *cosmetics.Catalog, owns func(id string) (bool, error)) ([]cosmetics.Item, error) {
	var unowned []cosmetics.Item
	for _, id := range offer.Bundle.Cosmetics {
		item, ok := items.Item(id)
		if !ok || item.Availability != cosmetics.Live || !slices.Contains(ChestRarities, item.Rarity) || !item.Grantable(cosmetics.Source{Type: SourceChest, Ref: offer.ID}) {
			continue
		}
		held, err := owns(id)
		if err != nil {
			return nil, err
		}
		if !held {
			unowned = append(unowned, item)
		}
	}
	return unowned, nil
}

func (c *Catalog) checkChestPools(items *cosmetics.Catalog) error {
	for _, offer := range c.Offers {
		if offer.Kind != KindChest {
			continue
		}
		source := cosmetics.Source{Type: SourceChest, Ref: offer.ID}
		for _, id := range offer.Bundle.Cosmetics {
			item, ok := items.Item(id)
			switch {
			case !ok:
				return fmt.Errorf("%w: %s pool lists unknown item %s", ErrInvalid, offer.ID, id)
			case !slices.Contains(ChestRarities, item.Rarity):
				return fmt.Errorf("%w: %s pool lists %s of rarity %s", ErrInvalid, offer.ID, id, item.Rarity)
			case !item.Grantable(source):
				return fmt.Errorf("%w: %s pool lists %s whose sources lack the chest", ErrInvalid, offer.ID, id)
			}
		}
	}
	return nil
}

type ChestOdds struct {
	Rarity  string `json:"rarity"`
	Percent int    `json:"percent"`
}

func (r ChestRules) Odds() []ChestOdds {
	odds := make([]ChestOdds, 0, len(ChestRarities))
	for _, rarity := range ChestRarities {
		odds = append(odds, ChestOdds{Rarity: rarity, Percent: r.RarityPercent[rarity]})
	}
	return odds
}

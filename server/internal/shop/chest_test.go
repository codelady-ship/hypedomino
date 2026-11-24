package shop

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
)

type script []uint64

func (s *script) Uint64() uint64 {
	value := (*s)[0]
	*s = (*s)[1:]
	return value
}

var defaultChests = ChestRules{RarityPercent: map[string]int{RarityCommon: 85, RarityRare: 15}, RarePityCount: 10}

func poolOf(rarities ...string) []cosmetics.Item {
	out := make([]cosmetics.Item, 0, len(rarities))
	for i, rarity := range rarities {
		out = append(out, cosmetics.Item{ID: fmt.Sprintf("avatar.%s_%d", rarity, i), Rarity: rarity})
	}
	return out
}

func TestADrawFollowsTheRollThePityAndTheUnownedItems(t *testing.T) {
	cases := []struct {
		name    string
		unowned []cosmetics.Item
		pity    int
		values  script
		item    string
		after   int
		ok      bool
	}{
		{"lowest roll is common", poolOf("common", "rare"), 0, script{0, 0}, "avatar.common_0", 1, true},
		{"last common roll", poolOf("common", "rare"), 3, script{84, 0}, "avatar.common_0", 4, true},
		{"first rare roll", poolOf("common", "rare"), 3, script{85, 0}, "avatar.rare_1", 0, true},
		{"roll wraps at 100", poolOf("common", "rare"), 0, script{199, 0}, "avatar.rare_1", 0, true},
		{"second value picks the item", poolOf("common", "common", "rare"), 0, script{10, 4}, "avatar.common_0", 1, true},
		{"odd second value picks the other", poolOf("common", "common", "rare"), 0, script{10, 5}, "avatar.common_1", 1, true},
		{"tenth open without a rare is rare", poolOf("common", "rare"), 9, script{0, 0}, "avatar.rare_1", 0, true},
		{"ninth open follows the roll", poolOf("common", "rare"), 8, script{0, 0}, "avatar.common_0", 9, true},
		{"no unowned rare falls back to common", poolOf("common"), 9, script{99, 0}, "avatar.common_0", 10, true},
		{"no unowned common falls back to rare", poolOf("rare"), 0, script{0, 0}, "avatar.rare_0", 0, true},
		{"fully owned pool", nil, 0, script{0}, "", 0, false},
	}
	for _, c := range cases {
		values := append(script{}, c.values...)
		draw, ok := defaultChests.Draw(c.unowned, c.pity, &values)
		if ok != c.ok || draw.Item != c.item || (ok && (draw.PityAfter != c.after || draw.PityBefore != c.pity || draw.Roll != int(c.values[0]%100))) {
			t.Fatalf("%s: %+v %v", c.name, draw, ok)
		}
	}
}

func TestDrawsFollowTheOddsAndNeverPassThePity(t *testing.T) {
	random := rand.New(rand.NewPCG(357, 2026))
	pool := poolOf("common", "common", "rare", "rare")
	const opens = 200000
	counts := map[string]int{}
	pity, longest := 0, 0
	for range opens {
		draw, ok := defaultChests.Draw(pool, pity, random)
		if !ok {
			t.Fatal("a draw from a full pool failed")
		}
		counts[draw.Item]++
		pity = draw.PityAfter
		longest = max(longest, pity)
	}
	if longest > defaultChests.RarePityCount-1 {
		t.Fatalf("%d opens in a row without a rare", longest)
	}
	if share := float64(counts["avatar.rare_2"]+counts["avatar.rare_3"]) / opens; share < 0.17 || share > 0.19 {
		t.Fatalf("rare share %.4f with the pity", share)
	}
	for _, pair := range [][2]string{{"avatar.common_0", "avatar.common_1"}, {"avatar.rare_2", "avatar.rare_3"}} {
		if ratio := float64(counts[pair[0]]) / float64(counts[pair[1]]); ratio < 0.97 || ratio > 1.03 {
			t.Fatalf("%s and %s are not uniform: %v", pair[0], pair[1], counts)
		}
	}
	noPity := ChestRules{RarityPercent: defaultChests.RarityPercent, RarePityCount: opens + 1}
	rare := 0
	for range opens {
		if draw, _ := noPity.Draw(pool, 0, random); draw.Rarity == RarityRare {
			rare++
		}
	}
	if share := float64(rare) / opens; share < 0.145 || share > 0.155 {
		t.Fatalf("rare share %.4f without the pity", share)
	}
}

func TestOpensToPityCountsDownToTheGuaranteedRare(t *testing.T) {
	for pity, want := range map[int]int{0: 10, 1: 9, 9: 1, 10: 1} {
		if got := defaultChests.OpensToPity(pity); got != want {
			t.Fatalf("pity %d: %d opens left, want %d", pity, got, want)
		}
	}
}

const (
	chestFields = `"id":"offer.k","kind":"chest","section":"chests","priceCoins":500,"bundle":{"id":"bundle.k","revision":1,"cosmetics":["avatar.x","avatar.y"]},` + okWindow + `,"nameKey":"k"`
	okChests    = `"chests":{"rarityPercent":{"common":85,"rare":15},"rarePityCount":10}`
)

func chestJSON(fields, chests string) string {
	return `{"schema":1,"offers":[{` + fields + `}],"rotations":[],` + chests + `}`
}

func TestInvalidChestsAreRefused(t *testing.T) {
	cases := map[string]struct {
		raw    string
		reason string
	}{
		"crystal price":    {chestJSON(strings.Replace(chestFields, `"priceCoins":500`, `"priceCrystals":50`, 1), okChests), "priced in coins"},
		"store product":    {chestJSON(strings.Replace(chestFields, `"priceCoins":500`, `"productId":"chest_tier1"`, 1), okChests), "priced in coins"},
		"coins in bundle":  {chestJSON(strings.Replace(chestFields, `"revision":1,`, `"revision":1,"coins":5,`, 1), okChests), "grants no currency"},
		"pass xp":          {chestJSON(strings.Replace(chestFields, `"revision":1,`, `"revision":1,"passXp":5,`, 1), okChests), "only its pool"},
		"empty pool":       {chestJSON(strings.Replace(chestFields, `"cosmetics":["avatar.x","avatar.y"]`, `"medals":[]`, 1), okChests), "grants nothing"},
		"limit":            {chestJSON(chestFields+`,"limitPerAccount":1`, okChests), "no purchase limit"},
		"eligibility":      {chestJSON(chestFields+`,"eligibility":{"firstPurchaseOnly":true}`, okChests), "no purchase limit"},
		"event section":    {chestJSON(strings.Replace(chestFields, `"section":"chests"`, `"section":"event"`, 1), okChests), "holds cosmetic offers"},
		"cosmetic section": {chestJSON(`"id":"offer.x","kind":"cosmetic","section":"chests","priceCoins":1,"bundle":{"id":"bundle.x","revision":1,"cosmetics":["avatar.x"]},`+okWindow+`,"limitPerAccount":1,"nameKey":"k"`, okChests), "holds chests"},
		"no rules":         {chestJSON(chestFields, `"chests":{}`), "chest odds"},
		"odds total":       {chestJSON(chestFields, `"chests":{"rarityPercent":{"common":80,"rare":15},"rarePityCount":10}`), "add up to 100"},
		"epic odds":        {chestJSON(chestFields, `"chests":{"rarityPercent":{"common":85,"epic":15},"rarePityCount":10}`), "chest odds epic"},
		"negative odds":    {chestJSON(chestFields, `"chests":{"rarityPercent":{"common":115,"rare":-15},"rarePityCount":10}`), "chest odds rare -15"},
		"pity":             {chestJSON(chestFields, `"chests":{"rarityPercent":{"common":85,"rare":15},"rarePityCount":0}`), "pity"},
		"rules alone":      {`{"schema":1,"offers":[],"rotations":[],"chests":{"rarePityCount":10}}`, "chest odds"},
	}
	for name, c := range cases {
		if _, err := Parse([]byte(c.raw), DefaultConfig()); err == nil || !strings.Contains(err.Error(), c.reason) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if _, err := Parse([]byte(chestJSON(chestFields, okChests)), DefaultConfig()); err != nil {
		t.Fatalf("valid chest: %v", err)
	}
}

func TestChestPoolsMustHoldCommonOrRareItemsListingTheChest(t *testing.T) {
	items, err := cosmetics.Load("../../../docs/cosmetics/examples/catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	catalog := example(t)
	if err := catalog.CheckSources(items); err != nil {
		t.Fatalf("example sources: %v", err)
	}
	cases := map[string]string{
		"avatar.comet_fox":   "lack the chest",
		"frame.season_1_top": "rarity epic",
		"avatar.missing":     "unknown item",
	}
	for id, reason := range cases {
		changed := *catalog
		changed.Offers = append([]Offer{}, catalog.Offers...)
		for i, offer := range changed.Offers {
			if offer.Kind == KindChest {
				changed.Offers[i].Bundle.Cosmetics = append(append([]string{}, offer.Bundle.Cosmetics...), id)
			}
		}
		if err := changed.CheckSources(items); err == nil || !strings.Contains(err.Error(), reason) {
			t.Fatalf("%s: %v", id, err)
		}
	}
	chest, _ := catalog.Offer("offer.chest_avatar")
	if chest.PriceCoins != 500 || catalog.Chests.RarePityCount != 10 || catalog.Chests.RarityPercent[RarityRare] != 15 {
		t.Fatalf("example chest %+v rules %+v", chest, catalog.Chests)
	}
}

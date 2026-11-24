package reward

import (
	"errors"
	"strings"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

func TestBundlesParseAndValidate(t *testing.T) {
	bundle, err := ParseBundle([]byte(`{"id":"bundle.season1_rank_gold","revision":1,"coins":700,"cosmetics":["frame.season1_gold"],"medals":["medal.season1_gold"],"passXp":0}`))
	if err != nil || bundle.Coins != 700 || len(bundle.Cosmetics) != 1 || len(bundle.Medals) != 1 || bundle.PassXP != 0 {
		t.Fatalf("parsed %+v %v", bundle, err)
	}
	cases := map[string]struct {
		raw    string
		reason string
	}{
		"not json":     {`{`, "invalid bundle"},
		"id":           {`{"id":"season1","revision":1,"coins":1}`, "id"},
		"revision":     {`{"id":"bundle.a","revision":0,"coins":1}`, "revision 0"},
		"negative":     {`{"id":"bundle.a","revision":1,"coins":-1}`, "negative"},
		"empty":        {`{"id":"bundle.a","revision":1}`, "grants nothing"},
		"cosmetic id":  {`{"id":"bundle.a","revision":1,"cosmetics":["hat.red"]}`, "cosmetic id"},
		"cosmetic dup": {`{"id":"bundle.a","revision":1,"cosmetics":["avatar.fox","avatar.fox"]}`, "twice"},
		"medal id":     {`{"id":"bundle.a","revision":1,"medals":["avatar.fox"]}`, "medal id"},
		"medal dup":    {`{"id":"bundle.a","revision":1,"medals":["medal.x","medal.x"]}`, "twice"},
		"crystals":     {`{"id":"bundle.a","revision":1,"crystals":-5}`, "negative"},
	}
	for name, c := range cases {
		if _, err := ParseBundle([]byte(c.raw)); !errors.Is(err, ErrInvalidBundle) || !strings.Contains(err.Error(), c.reason) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if err := (Bundle{ID: "bundle.xp", Revision: 2, PassXP: 50}).Validate(); err != nil {
		t.Fatalf("xp only bundle: %v", err)
	}
}

func TestDuplicateCompensationFollowsTheGuidePrice(t *testing.T) {
	config := DefaultConfig()
	cases := []struct {
		item cosmetics.Item
		want int64
	}{
		{cosmetics.Item{Type: cosmetics.TypeAvatar, Rarity: "common"}, 80},
		{cosmetics.Item{Type: cosmetics.TypeFrame, Rarity: "rare"}, 300},
		{cosmetics.Item{Type: cosmetics.TypeTable, Rarity: "epic"}, 1050},
		{cosmetics.Item{Type: cosmetics.TypeTile, Rarity: "legendary"}, 3000},
		{cosmetics.Item{Type: cosmetics.TypeEmoji, Rarity: "legendary"}, 1200},
		{cosmetics.Item{Type: cosmetics.TypeEmoji, Rarity: "mythic"}, 3000},
		{cosmetics.Item{Type: "character", Rarity: "rare"}, 0},
		{cosmetics.Item{Type: cosmetics.TypeFrame, Rarity: "unknown"}, 0},
		{cosmetics.Item{Type: "hat", Rarity: "common"}, 0},
	}
	for _, c := range cases {
		item, want := c.item, c.want
		if got := config.Compensation(item); got != want {
			t.Fatalf("compensation for %s %s = %d, want %d", item.Type, item.Rarity, got, want)
		}
	}
}

func TestABundleGrantsOnlyTheCurrencyOfItsSource(t *testing.T) {
	coins := Bundle{ID: "bundle.coins", Revision: 1, Coins: 100}
	crystals := Bundle{ID: "bundle.crystals", Revision: 1, Crystals: 100}
	for _, c := range []struct {
		bundle Bundle
		reason wallet.Reason
		valid  bool
	}{
		{coins, wallet.MatchReward, true},
		{coins, wallet.Mission, true},
		{coins, wallet.Promo, true},
		{coins, wallet.RankReward, true},
		{crystals, wallet.MatchReward, false},
		{crystals, wallet.DailyLogin, false},
		{crystals, wallet.Mission, false},
		{crystals, wallet.Promo, false},
		{crystals, wallet.RankReward, false},
		{crystals, wallet.EventReward, false},
		{crystals, wallet.Purchase, true},
		{crystals, wallet.FirstPurchaseBonus, true},
		{crystals, wallet.PassReward, true},
	} {
		err := c.bundle.ValidateFor(c.reason)
		if (err == nil) != c.valid || (err != nil && !errors.Is(err, ErrInvalidBundle)) {
			t.Errorf("%s from %s: %v", c.bundle.ID, c.reason, err)
		}
	}
	if err := (Bundle{ID: "bundle.crystals", Revision: 1, Crystals: 10}).Validate(); err != nil {
		t.Fatalf("crystals only bundle: %v", err)
	}
}

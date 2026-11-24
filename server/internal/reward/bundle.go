package reward

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

var ErrInvalidBundle = errors.New("reward: invalid bundle")

var (
	bundlePattern = regexp.MustCompile(`^bundle\.[a-z0-9_]+$`)
	medalPattern  = regexp.MustCompile(`^medal\.[a-z0-9_]+$`)
)

type Bundle struct {
	ID        string   `json:"id"`
	Revision  int      `json:"revision"`
	Coins     int64    `json:"coins"`
	Crystals  int64    `json:"crystals,omitempty"`
	Cosmetics []string `json:"cosmetics"`
	Medals    []string `json:"medals"`
	PassXP    int      `json:"passXp"`
}

func ParseBundle(raw []byte) (Bundle, error) {
	var b Bundle
	if err := json.Unmarshal(raw, &b); err != nil {
		return Bundle{}, fmt.Errorf("%w: %v", ErrInvalidBundle, err)
	}
	return b, b.Validate()
}

func (b Bundle) Validate() error {
	switch {
	case !bundlePattern.MatchString(b.ID):
		return fmt.Errorf("%w: id %q", ErrInvalidBundle, b.ID)
	case b.Revision < 1:
		return fmt.Errorf("%w: %s revision %d", ErrInvalidBundle, b.ID, b.Revision)
	case b.Coins < 0 || b.Crystals < 0 || b.PassXP < 0:
		return fmt.Errorf("%w: %s negative coins, crystals or pass xp", ErrInvalidBundle, b.ID)
	case b.Coins == 0 && b.Crystals == 0 && len(b.Cosmetics) == 0 && len(b.Medals) == 0 && b.PassXP == 0:
		return fmt.Errorf("%w: %s grants nothing", ErrInvalidBundle, b.ID)
	}
	if err := unique(b.ID, "cosmetic", b.Cosmetics, func(id string) bool {
		return slices.Contains(cosmetics.Types, cosmetics.TypeOf(id)) && len(id) > len(cosmetics.TypeOf(id))+1
	}); err != nil {
		return err
	}
	return unique(b.ID, "medal", b.Medals, medalPattern.MatchString)
}

func unique(bundle, kind string, ids []string, valid func(string) bool) error {
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if !valid(id) {
			return fmt.Errorf("%w: %s %s id %q", ErrInvalidBundle, bundle, kind, id)
		}
		if seen[id] {
			return fmt.Errorf("%w: %s lists %s %s twice", ErrInvalidBundle, bundle, kind, id)
		}
		seen[id] = true
	}
	return nil
}

func (b Bundle) ValidateFor(reason wallet.Reason) error {
	if err := b.Validate(); err != nil {
		return err
	}
	if b.Coins > 0 && !wallet.Moves(reason, wallet.Coin) {
		return fmt.Errorf("%w: %s grants coins from a %s source", ErrInvalidBundle, b.ID, reason)
	}
	if b.Crystals > 0 && !wallet.Moves(reason, wallet.Crystal) {
		return fmt.Errorf("%w: %s grants crystals from a %s source", ErrInvalidBundle, b.ID, reason)
	}
	return nil
}

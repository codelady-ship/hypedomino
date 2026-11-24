package medal

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
)

const (
	Domain           = "medals"
	Schema           = 1
	SourceMission    = "mission"
	SourceRankReward = "rank_reward"
	GroupAchievement = "achievement"
	GroupEvent       = "event"
	GroupRankReward  = "rank_reward"
)

var (
	ErrInvalidCatalog = errors.New("medal: invalid catalog")
	idPattern         = regexp.MustCompile(`^medal\.([a-z0-9_]+)$`)
)

type Medal struct {
	ID             string             `json:"id"`
	Revision       int                `json:"revision"`
	NameKey        string             `json:"nameKey"`
	DescriptionKey string             `json:"descriptionKey"`
	RarityID       string             `json:"rarityId"`
	StaticAddress  string             `json:"staticAddress"`
	Sources        []cosmetics.Source `json:"sources"`
	Availability   string             `json:"availability"`
}

func (m Medal) Lists(source cosmetics.Source) bool {
	return slices.Contains(m.Sources, source)
}

func (m Medal) Group(missionType func(id string) string) string {
	for _, source := range m.Sources {
		if source.Type == SourceRankReward {
			return GroupRankReward
		}
		if source.Type != SourceMission || missionType == nil {
			continue
		}
		if group := missionType(source.Ref); group == GroupAchievement || group == GroupEvent {
			return group
		}
	}
	return ""
}

type Catalog struct {
	Schema            int     `json:"schema"`
	ProfileMedalSlots int     `json:"profileMedalSlots"`
	Medals            []Medal `json:"medals"`
}

func Default() *Catalog {
	return &Catalog{Schema: Schema, ProfileMedalSlots: 3}
}

func Parse(raw []byte) (*Catalog, error) {
	catalog := Default()
	if err := json.Unmarshal(raw, catalog); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidCatalog, err)
	}
	if err := catalog.Validate(cosmetics.DefaultRarities()); err != nil {
		return nil, err
	}
	return catalog, nil
}

func ValidID(id string) bool {
	return idPattern.MatchString(id)
}

func StaticAddress(id string) string {
	return "medal/" + strings.TrimPrefix(id, "medal.") + "/static"
}

func (c *Catalog) Medal(id string) (Medal, bool) {
	for _, medal := range c.Medals {
		if medal.ID == id {
			return medal, true
		}
	}
	return Medal{}, false
}

func (c *Catalog) Validate(rarities cosmetics.Rarities) error {
	fail := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s", ErrInvalidCatalog, fmt.Sprintf(format, args...))
	}
	if c.Schema != Schema {
		return fail("schema %d", c.Schema)
	}
	if c.ProfileMedalSlots < 0 {
		return fail("profileMedalSlots must not be negative")
	}
	seen := map[string]bool{}
	for _, medal := range c.Medals {
		if seen[medal.ID] {
			return fail("%s listed twice", medal.ID)
		}
		seen[medal.ID] = true
		if err := validateMedal(medal, rarities); err != nil {
			return fail("%s: %v", medal.ID, err)
		}
	}
	return nil
}

func validateMedal(medal Medal, rarities cosmetics.Rarities) error {
	match := idPattern.FindStringSubmatch(medal.ID)
	if match == nil {
		return errors.New("id must be medal.<slug>")
	}
	slug := match[1]
	switch {
	case medal.Revision < 1:
		return errors.New("revision must be at least 1")
	case medal.NameKey != "medal."+slug+".name" || medal.DescriptionKey != "medal."+slug+".description":
		return errors.New("nameKey and descriptionKey must be medal.<slug>.name and medal.<slug>.description")
	case medal.StaticAddress != StaticAddress(medal.ID):
		return errors.New("staticAddress must be medal/<slug>/static")
	case rarities[medal.RarityID] == 0:
		return fmt.Errorf("rarityId %q is not in the rarity table", medal.RarityID)
	case medal.Availability != cosmetics.Live && medal.Availability != cosmetics.Retired && medal.Availability != cosmetics.Disabled:
		return fmt.Errorf("availability %q", medal.Availability)
	case len(medal.Sources) == 0:
		return errors.New("at least one source is required")
	}
	for _, source := range medal.Sources {
		if (source.Type != SourceMission && source.Type != SourceRankReward) || strings.TrimSpace(source.Ref) == "" {
			return fmt.Errorf("source %s %q: only mission and rank_reward with a ref", source.Type, source.Ref)
		}
	}
	return nil
}

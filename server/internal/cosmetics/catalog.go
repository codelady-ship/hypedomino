package cosmetics

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

const SchemaVersion = 1

const (
	TypeAvatar    = "avatar"
	TypeCharacter = "character"
	TypeFrame     = "frame"
	TypeTable     = "table"
	TypeEmoji     = "emoji"
	TypeAction    = "action"
)

var Types = []string{TypeAvatar, TypeCharacter, TypeFrame, TypeTable, TypeEmoji, TypeAction}

const (
	Live     = "live"
	Retired  = "retired"
	Disabled = "disabled"
)

const SourceDefault = "default"

var (
	ErrSchema  = errors.New("cosmetics: unsupported manifest schema")
	ErrInvalid = errors.New("cosmetics: invalid manifest")
)

//go:embed builtin.json
var builtin []byte

type Source struct {
	Type string `json:"type"`
	Ref  string `json:"ref,omitempty"`
}

type Item struct {
	ID           string   `json:"id"`
	Type         string   `json:"type"`
	Revision     int      `json:"revision"`
	Rarity       string   `json:"rarityId"`
	Animated     bool     `json:"animated"`
	Availability string   `json:"availability"`
	Sources      []Source `json:"sources"`
}

func (i Item) Default() bool {
	for _, s := range i.Sources {
		if s.Type == SourceDefault {
			return true
		}
	}
	return false
}

func (i Item) Grantable(source Source) bool {
	for _, s := range i.Sources {
		if s == source {
			return true
		}
	}
	return false
}

func (i Item) Equippable() bool {
	return i.Availability != Disabled
}

type Catalog struct {
	Version int
	items   map[string]Item
	order   []string
}

type manifest struct {
	SchemaVersion   int    `json:"schemaVersion"`
	ManifestVersion int    `json:"manifestVersion"`
	Items           []Item `json:"items"`
}

var idPattern = regexp.MustCompile(`^(avatar|character|frame|table|emoji|action)\.[a-z0-9_]+$`)

func Builtin() *Catalog {
	catalog, err := Parse(builtin)
	if err != nil {
		panic(err)
	}
	return catalog
}

func Load(path string) (*Catalog, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(raw)
}

func Parse(raw []byte) (*Catalog, error) {
	var m manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	if m.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("%w: schemaVersion %d", ErrSchema, m.SchemaVersion)
	}
	if m.ManifestVersion < 1 || len(m.Items) == 0 {
		return nil, fmt.Errorf("%w: manifestVersion and items are required", ErrInvalid)
	}
	c := &Catalog{Version: m.ManifestVersion, items: make(map[string]Item, len(m.Items))}
	for _, item := range m.Items {
		if err := check(item); err != nil {
			return nil, err
		}
		if _, dup := c.items[item.ID]; dup {
			return nil, fmt.Errorf("%w: duplicate id %s", ErrInvalid, item.ID)
		}
		c.items[item.ID] = item
		c.order = append(c.order, item.ID)
	}
	for _, t := range Types {
		if _, ok := c.Default(t); !ok && t != TypeAction {
			return nil, fmt.Errorf("%w: no live default %s", ErrInvalid, t)
		}
	}
	return c, nil
}

func check(item Item) error {
	switch {
	case !idPattern.MatchString(item.ID):
		return fmt.Errorf("%w: id %q", ErrInvalid, item.ID)
	case TypeOf(item.ID) != item.Type:
		return fmt.Errorf("%w: %s is not of type %s", ErrInvalid, item.ID, item.Type)
	case item.Revision < 1:
		return fmt.Errorf("%w: %s revision %d", ErrInvalid, item.ID, item.Revision)
	case item.Availability != Live && item.Availability != Retired && item.Availability != Disabled:
		return fmt.Errorf("%w: %s availability %q", ErrInvalid, item.ID, item.Availability)
	case len(item.Sources) == 0:
		return fmt.Errorf("%w: %s has no sources", ErrInvalid, item.ID)
	}
	seen := make(map[Source]bool, len(item.Sources))
	for _, s := range item.Sources {
		switch {
		case seen[s]:
			return fmt.Errorf("%w: %s lists source %s %s twice", ErrInvalid, item.ID, s.Type, s.Ref)
		case s.Type == SourceDefault && s.Ref != "":
			return fmt.Errorf("%w: %s default source with a ref", ErrInvalid, item.ID)
		case s.Type != SourceDefault && s.Ref == "":
			return fmt.Errorf("%w: %s %s source without a ref", ErrInvalid, item.ID, s.Type)
		}
		seen[s] = true
	}
	if item.Default() && (item.Availability != Live || item.Animated || item.Type == TypeAction) {
		return fmt.Errorf("%w: default item %s must be live, static and not an action", ErrInvalid, item.ID)
	}
	return nil
}

func TypeOf(id string) string {
	t, _, _ := strings.Cut(id, ".")
	return t
}

func (c *Catalog) Item(id string) (Item, bool) {
	item, ok := c.items[id]
	return item, ok
}

func (c *Catalog) Items() []Item {
	items := make([]Item, 0, len(c.order))
	for _, id := range c.order {
		items = append(items, c.items[id])
	}
	return items
}

func (c *Catalog) Default(t string) (Item, bool) {
	for _, id := range c.order {
		item := c.items[id]
		if item.Type == t && item.Default() && item.Availability == Live {
			return item, true
		}
	}
	return Item{}, false
}

func (c *Catalog) Defaults() []Item {
	var items []Item
	for _, id := range c.order {
		if item := c.items[id]; item.Default() && item.Availability == Live {
			items = append(items, item)
		}
	}
	return items
}

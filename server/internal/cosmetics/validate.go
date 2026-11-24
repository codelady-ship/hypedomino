package cosmetics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

const reviewApproved = "approved"

var (
	SourceTypes = []string{SourceDefault, "shop", "pass", "mission", "promo", "rank_reward", "first_purchase", "event", "chest"}
	ArtSources  = []string{"handmade", "ai_assisted"}

	referencePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]*$`)
	layerNamePattern = regexp.MustCompile(`^[a-z0-9_]+$`)
)

type strictLayer struct {
	Name     string `json:"name"`
	Address  string `json:"address"`
	Animated *bool  `json:"animated"`
}

type strictItem struct {
	ID               string        `json:"id"`
	Type             string        `json:"type"`
	Revision         int           `json:"revision"`
	NameKey          string        `json:"nameKey"`
	RarityID         string        `json:"rarityId"`
	Animated         *bool         `json:"animated"`
	StaticAddress    string        `json:"staticAddress"`
	Availability     string        `json:"availability"`
	Sources          []Source      `json:"sources"`
	ArtSource        string        `json:"artSource"`
	CulturalReview   string        `json:"culturalReview"`
	AnimationAddress string        `json:"animationAddress"`
	Layers           []strictLayer `json:"layers"`
	Group            string        `json:"group"`
}

func BuiltinManifest() Manifest {
	m, err := ParseManifest(builtin)
	if err != nil {
		panic(err)
	}
	return m
}

func ValidateManifest(m Manifest, rarities Rarities) error {
	for _, raw := range m.Items {
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		var item strictItem
		if err := decoder.Decode(&item); err != nil {
			return fmt.Errorf("%w: V-1 %v", ErrInvalid, err)
		}
		if err := item.check(rarities); err != nil {
			return fmt.Errorf("%w: %s %v", ErrInvalid, item.ID, err)
		}
	}
	return nil
}

func (i strictItem) check(rarities Rarities) error {
	if err := i.checkShared(rarities); err != nil {
		return err
	}
	if err := i.checkTypeFields(); err != nil {
		return err
	}
	return i.checkAddresses()
}

func (i strictItem) checkShared(rarities Rarities) error {
	if _, known := rarities[i.RarityID]; !known {
		return fmt.Errorf("V-14 rarity %q", i.RarityID)
	}
	if i.NameKey != NameKeyOf(i.ID) {
		return fmt.Errorf("V-9 nameKey %q", i.NameKey)
	}
	if i.Animated == nil {
		return fmt.Errorf("V-1 animated is required")
	}
	if !slices.Contains(ArtSources, i.ArtSource) {
		return fmt.Errorf("V-1 artSource %q", i.ArtSource)
	}
	if i.CulturalReview != reviewApproved {
		return fmt.Errorf("V-1 culturalReview %q", i.CulturalReview)
	}
	for _, source := range i.Sources {
		if !slices.Contains(SourceTypes, source.Type) {
			return fmt.Errorf("V-1 source type %q", source.Type)
		}
		if source.Type != SourceDefault && !referencePattern.MatchString(source.Ref) {
			return fmt.Errorf("V-1 source ref %q", source.Ref)
		}
	}
	if (i.Type == TypeEmoji) != (i.Group != "") || (i.Group != "" && !referencePattern.MatchString(i.Group)) {
		return fmt.Errorf("V-6 group %q on a %s", i.Group, i.Type)
	}
	return nil
}

func (i strictItem) checkTypeFields() error {
	animated := *i.Animated
	if i.Type == TypeTable {
		if i.AnimationAddress != "" || animated != (len(i.Layers) > 0) {
			return fmt.Errorf("V-3 fields do not match a table with animated %t", animated)
		}
		return i.checkLayers()
	}
	if len(i.Layers) > 0 || animated != (i.AnimationAddress != "") {
		return fmt.Errorf("V-3 fields do not match a %s with animated %t", i.Type, animated)
	}
	return nil
}

func (i strictItem) checkLayers() error {
	names := make(map[string]bool, len(i.Layers))
	anyAnimated := len(i.Layers) == 0
	for _, layer := range i.Layers {
		if !layerNamePattern.MatchString(layer.Name) || names[layer.Name] || layer.Animated == nil {
			return fmt.Errorf("V-5 layer %q", layer.Name)
		}
		names[layer.Name] = true
		anyAnimated = anyAnimated || *layer.Animated
	}
	if !anyAnimated {
		return fmt.Errorf("V-5 an animated table needs an animated layer")
	}
	return nil
}

func (i strictItem) checkAddresses() error {
	expected := map[string]string{AddressOf(i.ID, "static"): i.StaticAddress}
	if i.AnimationAddress != "" {
		expected[AddressOf(i.ID, "anim")] = i.AnimationAddress
	}
	for _, layer := range i.Layers {
		expected[AddressOf(i.ID, "layer/"+layer.Name)] = layer.Address
	}
	for want, got := range expected {
		if want != got {
			return fmt.Errorf("V-10 address %q, want %q", got, want)
		}
	}
	return nil
}

func NameKeyOf(id string) string {
	return "cosmetic." + id + ".name"
}

func AddressOf(id, slot string) string {
	return "cosmetic/" + strings.Replace(id, ".", "/", 1) + "/" + slot
}

func Stamp(previous Manifest, raw []byte) (Manifest, error) {
	var submitted Manifest
	if err := json.Unmarshal(raw, &submitted); err != nil {
		return Manifest{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	_, before, err := previous.published()
	if err != nil {
		return Manifest{}, err
	}
	stamped := Manifest{SchemaVersion: SchemaVersion, ManifestVersion: previous.ManifestVersion + 1, Items: make([]json.RawMessage, 0, len(submitted.Items))}
	for _, rawItem := range submitted.Items {
		item, err := stampItem(rawItem, before)
		if err != nil {
			return Manifest{}, err
		}
		stamped.Items = append(stamped.Items, item)
	}
	return stamped, nil
}

func stampItem(raw json.RawMessage, before map[string]publishedItem) (json.RawMessage, error) {
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	id, _ := fields["id"].(string)
	if !idPattern.MatchString(id) {
		return nil, fmt.Errorf("%w: id %q", ErrInvalid, id)
	}
	deriveFields(fields, id)
	fields["revision"] = 1
	derived, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}
	item, err := readItem(derived)
	if err != nil {
		return nil, err
	}
	if old, existed := before[id]; existed {
		fields["revision"] = old.revision
		if old.content != item.content {
			fields["revision"] = old.revision + 1
		}
	}
	return json.Marshal(fields)
}

func deriveFields(fields map[string]any, id string) {
	fields["nameKey"] = NameKeyOf(id)
	fields["staticAddress"] = AddressOf(id, "static")
	if TypeOf(id) == TypeTable {
		deriveLayers(fields, id)
		return
	}
	if animated, _ := fields["animated"].(bool); animated {
		fields["animationAddress"] = AddressOf(id, "anim")
	}
}

func deriveLayers(fields map[string]any, id string) {
	layers, _ := fields["layers"].([]any)
	for _, entry := range layers {
		if layer, ok := entry.(map[string]any); ok {
			name, _ := layer["name"].(string)
			layer["address"] = AddressOf(id, "layer/"+name)
		}
	}
}

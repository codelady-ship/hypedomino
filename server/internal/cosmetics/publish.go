package cosmetics

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

const Domain = "cosmetics"

var (
	ErrSuccessor   = errors.New("cosmetics: manifest cannot follow the previous one")
	ErrUnknownItem = errors.New("cosmetics: no such item in the manifest")
)

type Manifest struct {
	SchemaVersion   int               `json:"schemaVersion"`
	ManifestVersion int               `json:"manifestVersion"`
	Items           []json.RawMessage `json:"items"`
}

type publishedItem struct {
	id       string
	itemType string
	revision int
	content  string
	fields   map[string]any
}

func ParseManifest(raw []byte) (Manifest, error) {
	if _, err := Parse(raw); err != nil {
		return Manifest{}, err
	}
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return Manifest{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	return m, nil
}

func (m Manifest) Encode() ([]byte, error) {
	return json.Marshal(m)
}

func (m Manifest) published() ([]publishedItem, map[string]publishedItem, error) {
	items := make([]publishedItem, 0, len(m.Items))
	byID := make(map[string]publishedItem, len(m.Items))
	for _, raw := range m.Items {
		item, err := readItem(raw)
		if err != nil {
			return nil, nil, err
		}
		items = append(items, item)
		byID[item.id] = item
	}
	return items, byID, nil
}

func readItem(raw json.RawMessage) (publishedItem, error) {
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		return publishedItem{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	var head Item
	if err := json.Unmarshal(raw, &head); err != nil {
		return publishedItem{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	withoutRevision := make(map[string]any, len(fields))
	for name, value := range fields {
		if name != "revision" {
			withoutRevision[name] = value
		}
	}
	content, err := json.Marshal(withoutRevision)
	if err != nil {
		return publishedItem{}, err
	}
	return publishedItem{id: head.ID, itemType: head.Type, revision: head.Revision, content: string(content), fields: fields}, nil
}

func CheckSuccessor(previous, next Manifest) error {
	if next.ManifestVersion != previous.ManifestVersion+1 {
		return fmt.Errorf("%w: manifestVersion %d after %d", ErrSuccessor, next.ManifestVersion, previous.ManifestVersion)
	}
	before, _, err := previous.published()
	if err != nil {
		return err
	}
	_, after, err := next.published()
	if err != nil {
		return err
	}
	for _, old := range before {
		if err := checkItemSuccessor(old, after); err != nil {
			return err
		}
	}
	return nil
}

func checkItemSuccessor(old publishedItem, after map[string]publishedItem) error {
	item, kept := after[old.id]
	switch {
	case !kept:
		return fmt.Errorf("%w: V-12 %s is missing", ErrSuccessor, old.id)
	case item.itemType != old.itemType:
		return fmt.Errorf("%w: V-12 %s changed type from %s to %s", ErrSuccessor, old.id, old.itemType, item.itemType)
	case item.revision < old.revision:
		return fmt.Errorf("%w: V-13 %s revision %d after %d", ErrSuccessor, old.id, item.revision, old.revision)
	case item.content != old.content && item.revision == old.revision:
		return fmt.Errorf("%w: V-13 %s changed without a new revision", ErrSuccessor, old.id)
	}
	return nil
}

func Restore(current, target Manifest) (Manifest, error) {
	items, _, err := current.published()
	if err != nil {
		return Manifest{}, err
	}
	_, earlier, err := target.published()
	if err != nil {
		return Manifest{}, err
	}
	restored := Manifest{SchemaVersion: current.SchemaVersion, ManifestVersion: current.ManifestVersion + 1, Items: make([]json.RawMessage, 0, len(items))}
	for i, item := range items {
		old, existed := earlier[item.id]
		if !existed || old.content == item.content {
			restored.Items = append(restored.Items, current.Items[i])
			continue
		}
		old.fields["revision"] = item.revision + 1
		raw, err := json.Marshal(old.fields)
		if err != nil {
			return Manifest{}, err
		}
		restored.Items = append(restored.Items, raw)
	}
	return restored, nil
}

func Added(previous, next Manifest) ([]string, error) {
	_, before, err := previous.published()
	if err != nil {
		return nil, err
	}
	items, _, err := next.published()
	if err != nil {
		return nil, err
	}
	var added []string
	for _, item := range items {
		if _, existed := before[item.id]; !existed {
			added = append(added, item.id)
		}
	}
	return added, nil
}

func (m Manifest) Disable(id string) (Manifest, error) {
	items, _, err := m.published()
	if err != nil {
		return Manifest{}, err
	}
	disabled := Manifest{SchemaVersion: m.SchemaVersion, ManifestVersion: m.ManifestVersion, Items: slices.Clone(m.Items)}
	for i, item := range items {
		if item.id != id {
			continue
		}
		item.fields["availability"] = Disabled
		disabled.Items[i], err = json.Marshal(item.fields)
		return disabled, err
	}
	return Manifest{}, fmt.Errorf("%w: %q", ErrUnknownItem, id)
}

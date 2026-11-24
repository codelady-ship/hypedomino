package cosmetics

import (
	"encoding/json"
	"errors"
	"os"
	"slices"
	"testing"
)

func exampleManifest(t *testing.T) Manifest {
	t.Helper()
	raw, err := os.ReadFile("../../../docs/cosmetics/examples/catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	m, err := ParseManifest(raw)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func itemFields(t *testing.T, m Manifest, id string) map[string]any {
	t.Helper()
	for _, raw := range m.Items {
		var fields map[string]any
		if err := json.Unmarshal(raw, &fields); err != nil {
			t.Fatal(err)
		}
		if fields["id"] == id {
			return fields
		}
	}
	t.Fatalf("no item %s", id)
	return nil
}

func edit(t *testing.T, m Manifest, id string, change func(fields map[string]any)) Manifest {
	t.Helper()
	edited := Manifest{SchemaVersion: m.SchemaVersion, ManifestVersion: m.ManifestVersion, Items: slices.Clone(m.Items)}
	for i, raw := range edited.Items {
		var fields map[string]any
		if err := json.Unmarshal(raw, &fields); err != nil {
			t.Fatal(err)
		}
		if fields["id"] != id {
			continue
		}
		change(fields)
		changed, err := json.Marshal(fields)
		if err != nil {
			t.Fatal(err)
		}
		edited.Items[i] = changed
		return edited
	}
	t.Fatalf("no item %s", id)
	return Manifest{}
}

func next(m Manifest) Manifest {
	return Manifest{SchemaVersion: m.SchemaVersion, ManifestVersion: m.ManifestVersion + 1, Items: slices.Clone(m.Items)}
}

func revision(fields map[string]any) int {
	return int(fields["revision"].(float64))
}

func TestAManifestEncodesAndParsesBackWithEveryField(t *testing.T) {
	m := exampleManifest(t)
	raw, err := m.Encode()
	if err != nil {
		t.Fatal(err)
	}
	again, err := ParseManifest(raw)
	if err != nil || again.ManifestVersion != m.ManifestVersion || len(again.Items) != len(m.Items) {
		t.Fatalf("parsed %+v %v", again, err)
	}
	if fox := itemFields(t, again, "avatar.comet_fox"); fox["nameKey"] != "cosmetic.avatar.comet_fox.name" || fox["staticAddress"] == nil {
		t.Fatalf("fields lost on the way: %v", fox)
	}
	if _, err := ParseManifest([]byte(`{"schemaVersion":1,"manifestVersion":1,"items":[]}`)); !errors.Is(err, ErrInvalid) {
		t.Fatalf("an empty manifest parsed: %v", err)
	}
}

func TestAPublishMustKeepEveryItemAndRaiseTheRevisionOfAChangedOne(t *testing.T) {
	previous := exampleManifest(t)
	foxRevision := revision(itemFields(t, previous, "avatar.comet_fox"))
	renamed := func(fields map[string]any) { fields["rarityId"] = "epic" }
	bumped := func(fields map[string]any) { renamed(fields); fields["revision"] = foxRevision + 1 }
	withoutFox := next(previous)
	withoutFox.Items = slices.DeleteFunc(withoutFox.Items, func(raw json.RawMessage) bool {
		var head Item
		return json.Unmarshal(raw, &head) == nil && head.ID == "avatar.comet_fox"
	})
	added := next(previous)
	added.Items = append(added.Items, json.RawMessage(`{"id":"frame.new_one","type":"frame","revision":1}`))
	tests := []struct {
		name string
		next Manifest
		want error
	}{
		{"unchanged", next(previous), nil},
		{"changed with a new revision", edit(t, next(previous), "avatar.comet_fox", bumped), nil},
		{"new item", added, nil},
		{"same manifest version", previous, ErrSuccessor},
		{"skipped manifest version", next(next(previous)), ErrSuccessor},
		{"V-12 item removed", withoutFox, ErrSuccessor},
		{"V-12 type changed", edit(t, next(previous), "avatar.comet_fox", func(fields map[string]any) { fields["type"] = "frame" }), ErrSuccessor},
		{"V-13 revision lowered", edit(t, next(previous), "avatar.comet_fox", func(fields map[string]any) { fields["revision"] = foxRevision - 1 }), ErrSuccessor},
		{"V-13 changed without a new revision", edit(t, next(previous), "avatar.comet_fox", renamed), ErrSuccessor},
	}
	for _, test := range tests {
		if err := CheckSuccessor(previous, test.next); !errors.Is(err, test.want) {
			t.Errorf("%s: %v", test.name, err)
		}
	}
}

func TestARollbackRestoresEarlierContentAsNewRevisionsAndKeepsLaterItems(t *testing.T) {
	target := exampleManifest(t)
	foxRevision := revision(itemFields(t, target, "avatar.comet_fox"))
	current := edit(t, next(target), "avatar.comet_fox", func(fields map[string]any) {
		fields["rarityId"] = "epic"
		fields["revision"] = foxRevision + 1
	})
	current = next(current)
	current.Items = append(current.Items, json.RawMessage(`{"id":"frame.new_one","type":"frame","revision":1}`))
	restored, err := Restore(current, target)
	if err != nil {
		t.Fatal(err)
	}
	if restored.ManifestVersion != current.ManifestVersion+1 || len(restored.Items) != len(current.Items) {
		t.Fatalf("restored version %d with %d items", restored.ManifestVersion, len(restored.Items))
	}
	fox := itemFields(t, restored, "avatar.comet_fox")
	if fox["rarityId"] != itemFields(t, target, "avatar.comet_fox")["rarityId"] || revision(fox) != foxRevision+2 {
		t.Fatalf("fox %v", fox)
	}
	if string(restored.Items[0]) != string(current.Items[0]) {
		t.Fatalf("an unchanged item was rewritten: %s", restored.Items[0])
	}
	if itemFields(t, restored, "frame.new_one")["revision"] != 1.0 {
		t.Fatal("an item added after the target was dropped or changed")
	}
	if err := CheckSuccessor(current, restored); err != nil {
		t.Fatalf("the rollback is not a valid publish: %v", err)
	}
}

func TestAddedListsOnlyTheItemsTheNextManifestIntroduces(t *testing.T) {
	m := exampleManifest(t)
	added, err := Added(BuiltinManifest(), m)
	if err != nil || !slices.Contains(added, "avatar.comet_fox") || slices.Contains(added, "avatar.starter") {
		t.Fatalf("added %v %v", added, err)
	}
	if added, err := Added(m, next(m)); err != nil || len(added) != 0 {
		t.Fatalf("nothing new %v %v", added, err)
	}
}

func TestDisableSwitchesOffOneItemAndLeavesTheRest(t *testing.T) {
	m := exampleManifest(t)
	disabled, err := m.Disable("avatar.comet_fox")
	if err != nil {
		t.Fatal(err)
	}
	if fox := itemFields(t, disabled, "avatar.comet_fox"); fox["availability"] != Disabled || revision(fox) != revision(itemFields(t, m, "avatar.comet_fox")) {
		t.Fatalf("disabled %v", fox)
	}
	if itemFields(t, m, "avatar.comet_fox")["availability"] == Disabled || itemFields(t, disabled, "avatar.starter")["availability"] != Live {
		t.Fatal("the switch-off reached the source manifest or another item")
	}
	if _, err := m.Disable("avatar.nobody"); !errors.Is(err, ErrUnknownItem) {
		t.Fatalf("unknown item %v", err)
	}
}

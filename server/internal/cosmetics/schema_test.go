package cosmetics

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"
)

const schemaPath = "../../../docs/cosmetics/schema.json"

type schemaEnum struct {
	Enum    []string `json:"enum"`
	Const   string   `json:"const"`
	Pattern string   `json:"pattern"`
}

type schemaTypeBranch struct {
	If struct {
		Properties struct {
			Type schemaEnum `json:"type"`
		} `json:"properties"`
	} `json:"if"`
	Then struct {
		Properties map[string]json.RawMessage `json:"properties"`
	} `json:"then"`
}

type cosmeticsSchema struct {
	Defs struct {
		ItemType schemaEnum `json:"itemType"`
		ItemID   schemaEnum `json:"itemId"`
		NameKey  schemaEnum `json:"nameKey"`
		Address  schemaEnum `json:"address"`
		Source   struct {
			Properties struct {
				Type schemaEnum `json:"type"`
			} `json:"properties"`
		} `json:"source"`
		Item struct {
			Properties struct {
				Availability schemaEnum `json:"availability"`
				ArtSource    schemaEnum `json:"artSource"`
			} `json:"properties"`
			AllOf []schemaTypeBranch `json:"allOf"`
		} `json:"item"`
	} `json:"$defs"`
}

func readSchema(t *testing.T) cosmeticsSchema {
	t.Helper()
	raw, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatal(err)
	}
	var schema cosmeticsSchema
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}
	return schema
}

func TestTheSchemaListsExactlyTheServerTypesSourcesAndStates(t *testing.T) {
	defs := readSchema(t).Defs
	lists := []struct {
		name         string
		schema, code []string
	}{
		{"itemType", defs.ItemType.Enum, Types},
		{"source type", defs.Source.Properties.Type.Enum, SourceTypes},
		{"availability", defs.Item.Properties.Availability.Enum, []string{Live, Retired, Disabled}},
		{"artSource", defs.Item.Properties.ArtSource.Enum, ArtSources},
	}
	for _, list := range lists {
		if !slices.Equal(list.schema, list.code) {
			t.Errorf("%s: schema %v, server %v", list.name, list.schema, list.code)
		}
	}
}

func TestEveryTypePatternOfTheSchemaIsBuiltFromTheServerTypes(t *testing.T) {
	defs := readSchema(t).Defs
	if defs.ItemID.Pattern != idPattern.String() {
		t.Errorf("itemId: schema %q, server %q", defs.ItemID.Pattern, idPattern.String())
	}
	alternatives := "(" + strings.Join(Types, "|") + ")"
	for name, pattern := range map[string]string{"nameKey": defs.NameKey.Pattern, "address": defs.Address.Pattern} {
		if !strings.Contains(pattern, alternatives) {
			t.Errorf("%s pattern %q does not list %s", name, pattern, alternatives)
		}
	}
}

func TestEveryTypeButTheTableHasTheOneAnimationShapeInTheSchema(t *testing.T) {
	var oneAnimation, layered []string
	for _, branch := range readSchema(t).Defs.Item.AllOf {
		if _, ok := branch.Then.Properties["layers"]; ok {
			oneAnimation = append(oneAnimation, branch.If.Properties.Type.Enum...)
		}
		if _, ok := branch.Then.Properties["animationAddress"]; ok {
			layered = append(layered, branch.If.Properties.Type.Const)
		}
	}
	want := slices.DeleteFunc(slices.Clone(Types), func(t string) bool { return t == TypeTable })
	if !slices.Equal(oneAnimation, want) {
		t.Errorf("one-animation types: schema %v, server %v", oneAnimation, want)
	}
	if !slices.Equal(layered, []string{TypeTable}) {
		t.Errorf("layered types: schema %v, server [%s]", layered, TypeTable)
	}
}

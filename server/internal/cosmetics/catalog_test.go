package cosmetics

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestTheBuiltinManifestIsTheDocumentedMinimalExample(t *testing.T) {
	documented, err := os.ReadFile("../../../docs/cosmetics/examples/minimal.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(documented, builtin) {
		t.Fatal("builtin.json differs from docs/cosmetics/examples/minimal.json")
	}
	catalog := Builtin()
	if catalog.Version != 1 || len(catalog.Items()) != 5 || len(catalog.Defaults()) != 5 {
		t.Fatalf("builtin catalog %+v", catalog.Items())
	}
}

func TestAManifestLivesOnTheCDNUnderTheHashOfItsBytes(t *testing.T) {
	cases := []struct {
		raw  []byte
		hash string
	}{
		{[]byte(""), "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{[]byte("abc"), "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"},
	}
	for _, c := range cases {
		if got := Hash(c.raw); got != c.hash {
			t.Errorf("%q: %s, want %s", c.raw, got, c.hash)
		}
		if got, want := File(c.hash), "cosmetics/"+c.hash+".json"; got != want {
			t.Errorf("file %s, want %s", got, want)
		}
	}
	catalog, err := Parse(builtin)
	if err != nil || catalog.Hash != Hash(builtin) {
		t.Fatalf("parsed hash %v %v", catalog, err)
	}
}

func TestTheExampleCatalogParsesWithItsAvailabilityAndSources(t *testing.T) {
	catalog, err := Load("../../../docs/cosmetics/examples/catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	if catalog.Version < 1 || len(catalog.Items()) != 16 {
		t.Fatalf("catalog %d %d", catalog.Version, len(catalog.Items()))
	}
	fox, ok := catalog.Item("avatar.comet_fox")
	if !ok || fox.Type != TypeAvatar || fox.Default() || !fox.Equippable() || !fox.Grantable(Source{Type: "shop", Ref: "offer.avatar_comet_fox"}) || fox.Grantable(Source{Type: "shop", Ref: "offer.other"}) || fox.Grantable(Source{Type: SourceDefault}) {
		t.Fatalf("comet fox %+v %v", fox, ok)
	}
	if withdrawn, ok := catalog.Item("frame.withdrawn"); !ok || withdrawn.Equippable() || withdrawn.Availability != Disabled {
		t.Fatalf("withdrawn frame %+v %v", withdrawn, ok)
	}
	if old, ok := catalog.Item("avatar.old_logo"); !ok || !old.Equippable() || old.Availability != Retired {
		t.Fatalf("old logo %+v %v", old, ok)
	}
	if def, ok := catalog.Default(TypeAvatar); !ok || def.ID != "avatar.starter" {
		t.Fatalf("default avatar %+v %v", def, ok)
	}
	if _, ok := catalog.Item("avatar.nope"); ok {
		t.Fatal("unknown item found")
	}
	if TypeOf("table.night_market") != TypeTable || TypeOf("bogus") != "bogus" {
		t.Fatal("type of id")
	}
}

func without(t *testing.T, id string) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(builtin, &m); err != nil {
		t.Fatal(err)
	}
	var kept []any
	for _, item := range m["items"].([]any) {
		if item.(map[string]any)["id"] != id {
			kept = append(kept, item)
		}
	}
	m["items"] = kept
	encoded, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func duplicated(t *testing.T) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(builtin, &m); err != nil {
		t.Fatal(err)
	}
	items := m["items"].([]any)
	m["items"] = append(items, items[0])
	encoded, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func TestInvalidManifestsAreRefusedWithTheirReason(t *testing.T) {
	base := string(builtin)
	cases := map[string]struct {
		manifest string
		err      error
		reason   string
	}{
		"not json":           {"{", ErrInvalid, ""},
		"schema":             {strings.Replace(base, `"schemaVersion": 1`, `"schemaVersion": 2`, 1), ErrSchema, "schemaVersion 2"},
		"no items":           {`{"schemaVersion":1,"manifestVersion":1,"items":[]}`, ErrInvalid, "required"},
		"bad id":             {strings.Replace(base, `"id": "avatar.starter"`, `"id": "Avatar.Starter"`, 1), ErrInvalid, "id"},
		"type mismatch":      {strings.Replace(base, `"id": "avatar.starter"`, `"id": "frame.starter2"`, 1), ErrInvalid, "not of type"},
		"duplicate":          {duplicated(t), ErrInvalid, "duplicate"},
		"missing default":    {without(t, "table.starter"), ErrInvalid, "no live default table"},
		"default with ref":   {strings.Replace(base, `"type": "default"`, `"type": "default", "ref": "x"`, 1), ErrInvalid, "default source with a ref"},
		"source without ref": {strings.Replace(base, `"type": "default"`, `"type": "shop"`, 1), ErrInvalid, "without a ref"},
		"animated default":   {strings.Replace(base, `"animated": false`, `"animated": true`, 1), ErrInvalid, "static"},
		"bad availability":   {strings.Replace(base, `"availability": "live"`, `"availability": "soon"`, 1), ErrInvalid, "availability"},
	}
	for name, c := range cases {
		_, err := Parse([]byte(c.manifest))
		if !errors.Is(err, c.err) || !strings.Contains(err.Error(), c.reason) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if _, err := Load("missing.json"); err == nil {
		t.Fatal("a missing file loaded")
	}
}

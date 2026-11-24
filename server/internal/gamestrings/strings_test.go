package gamestrings

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func coinsEntry() Entry {
	return Entry{
		Key:          "match.result.coins",
		Note:         "Match end screen, under the score. coins is the reward amount.",
		MaxLength:    24,
		Placeholders: []Placeholder{{Name: "coins", Type: Integer}},
	}
}

func TestKeysFollowTheDottedLowercaseFormat(t *testing.T) {
	tests := map[string]bool{
		"match.result.win":        true,
		"nickname.taken":          true,
		"common.text.unavailable": true,
		"shop.offer_2.title":      true,
		"match":                   false,
		"Match.result.win":        false,
		"match..win":              false,
		"match.result.":           false,
		".match.result":           false,
		"match.result-win":        false,
		"match.rəsult.win":        false,
	}
	for key, want := range tests {
		if got := ValidKey(key); got != want {
			t.Errorf("%q: %v, want %v", key, got, want)
		}
	}
}

func TestEntriesNeedAKeyANoteALimitAndTypedPlaceholders(t *testing.T) {
	tests := map[string]func(*Entry){
		"bad key":              func(e *Entry) { e.Key = "Match.Result" },
		"no note":              func(e *Entry) { e.Note = "  " },
		"no max length":        func(e *Entry) { e.MaxLength = 0 },
		"bad placeholder name": func(e *Entry) { e.Placeholders = []Placeholder{{Name: "1coins", Type: Integer}} },
		"duplicate name":       func(e *Entry) { e.Placeholders = append(e.Placeholders, Placeholder{Name: "coins", Type: String}) },
		"unknown type":         func(e *Entry) { e.Placeholders = []Placeholder{{Name: "coins", Type: "money"}} },
	}
	if err := coinsEntry().Validate(); err != nil {
		t.Fatal(err)
	}
	for name, change := range tests {
		entry := coinsEntry()
		change(&entry)
		if err := entry.Validate(); !errors.Is(err, ErrInvalidEntry) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestValuesAreCheckedPerLanguage(t *testing.T) {
	opponent := Entry{
		Key: "match.hud.opponent", Note: "HUD label, then the opponent nickname.", MaxLength: 20,
		Placeholders: []Placeholder{{Name: "opponentName", Type: String}},
	}
	kind := Entry{
		Key: "shop.offer.kind", Note: "Offer badge by kind.", MaxLength: 12,
		Placeholders: []Placeholder{{Name: "kind", Type: String}},
	}
	tests := []struct {
		name     string
		entry    Entry
		language Language
		text     string
		valid    bool
	}{
		{"plain plural en", coinsEntry(), English, "{coins, plural, one {# coin} other {# coins}}", true},
		{"az singular noun in other", coinsEntry(), Azerbaijani, "{coins, plural, one {# qızıl} other {# qızıl}}", true},
		{"ru needs few and many", coinsEntry(), Russian, "{coins, plural, one {# монета} other {# монеты}}", false},
		{"ru complete", coinsEntry(), Russian, "{coins, plural, one {# монета} few {# монеты} many {# монет} other {# монеты}}", true},
		{"exact number branch", coinsEntry(), English, "{coins, plural, =0 {No coins} one {# coin} other {# coins}}", true},
		{"category the language lacks", coinsEntry(), English, "{coins, plural, one {# coin} few {# coins} other {# coins}}", false},
		{"no other", coinsEntry(), English, "{coins, plural, one {# coin}}", false},
		{"repeated selector", coinsEntry(), English, "{coins, plural, one {a} one {b} other {c}}", false},
		{"missing placeholder", coinsEntry(), English, "Coins earned", false},
		{"extra placeholder", coinsEntry(), English, "{coins, plural, one {# coin} other {# coins}} {bonus}", false},
		{"misspelt placeholder", coinsEntry(), English, "{coin, plural, one {# coin} other {# coins}}", false},
		{"plural on a string", opponent, English, "{opponentName, plural, one {a} other {b}}", false},
		{"unclosed brace", coinsEntry(), English, "{coins, plural, one {# coin} other {# coins}", false},
		{"stray closing brace", opponent, English, "Opponent: {opponentName}}", false},
		{"unknown argument kind", coinsEntry(), English, "{coins, number}", false},
		{"too long", opponent, English, "Your opponent this round: {opponentName}", false},
		{"length ignores the placeholder", opponent, English, "Opponent: {opponentName}", true},
		{"combining marks do not count", opponent, Turkish, "Rakip i̇: {opponentName}", true},
		{"select with other", kind, English, "{kind, select, daily {Daily} other {Offer}}", true},
		{"select without other", kind, English, "{kind, select, daily {Daily} weekly {Weekly}}", false},
		{"unknown language", coinsEntry(), "de", "{coins, plural, one {# Münze} other {# Münzen}}", false},
	}
	for _, test := range tests {
		err := test.entry.CheckValue(test.language, test.text)
		if test.valid && err != nil {
			t.Errorf("%s: %v", test.name, err)
		}
		if !test.valid && !errors.Is(err, ErrInvalidValue) {
			t.Errorf("%s: accepted or wrong error: %v", test.name, err)
		}
	}
}

func TestTheLongestBranchCountsTowardsTheLimit(t *testing.T) {
	entry := coinsEntry()
	entry.MaxLength = 7
	if err := entry.CheckValue(English, "{coins, plural, one {# coin} other {# coins}}"); err != nil {
		t.Fatalf("' coins' is 6 long: %v", err)
	}
	if err := entry.CheckValue(English, "{coins, plural, one {# coin} other {# gold coins}}"); !errors.Is(err, ErrInvalidValue) {
		t.Fatalf("' gold coins' is 11 long: %v", err)
	}
}

func catalog() []Entry {
	win := Entry{Key: "match.result.win", Note: "Match end title.", MaxLength: 12, Values: map[Language]Value{
		English:     {Text: "Victory!", Status: Approved},
		Azerbaijani: {Text: "Qələbə!", Status: Approved},
		Russian:     {Text: "Победа!", Status: Draft},
	}}
	coins := coinsEntry()
	coins.Values = map[Language]Value{
		English:     {Text: "{coins, plural, one {# coin} other {# coins}}", Status: Approved},
		Azerbaijani: {Text: "{coins, plural, one {# qızıl} other {# qızıl}}", Status: Approved},
	}
	retired := Entry{Key: "lobby.banner.old", Note: "Retired banner.", MaxLength: 30, Deprecated: true, Values: map[Language]Value{
		English: {Text: "Old banner", Status: Approved},
	}}
	return []Entry{win, coins, retired}
}

func TestABundleHoldsOnlyApprovedCurrentValues(t *testing.T) {
	bundle, problems := BuildBundle(catalog(), English)
	if problems != nil {
		t.Fatalf("problems %v", problems)
	}
	want := map[string]string{"match.result.win": "Victory!", "match.result.coins": "{coins, plural, one {# coin} other {# coins}}"}
	if bundle.Language != English || !reflect.DeepEqual(bundle.Strings, want) {
		t.Fatalf("bundle %+v", bundle)
	}
	russian, problems := BuildBundle(catalog(), Russian)
	if problems != nil || len(russian.Strings) != 0 {
		t.Fatalf("a draft value entered the bundle: %+v %v", russian, problems)
	}
}

func TestAnInvalidApprovedValueBlocksTheBundle(t *testing.T) {
	entries := catalog()
	entries[0].Values[English] = Value{Text: "A victory far too long for the badge", Status: Approved}
	entries[1].Values[English] = Value{Text: "{coin} coins", Status: Approved}
	bundle, problems := BuildBundle(entries, English)
	if bundle.Strings != nil || len(problems) != 2 {
		t.Fatalf("bundle %+v problems %v", bundle, problems)
	}
	keys := []string{problems[0].Key, problems[1].Key}
	if !reflect.DeepEqual(keys, []string{"match.result.win", "match.result.coins"}) {
		t.Fatalf("problem keys %v", keys)
	}
	for _, problem := range problems {
		if !errors.Is(problem.Err, ErrInvalidValue) {
			t.Errorf("%s: %v", problem.Key, problem.Err)
		}
	}
}

func TestTheEncodedBundleIsStableAndHashed(t *testing.T) {
	first, _ := BuildBundle(catalog(), Azerbaijani)
	entries := catalog()
	entries[0], entries[1] = entries[1], entries[0]
	second, _ := BuildBundle(entries, Azerbaijani)
	a, hashA, err := first.Encode()
	if err != nil {
		t.Fatal(err)
	}
	b, hashB, err := second.Encode()
	if err != nil || string(a) != string(b) || hashA != hashB || len(hashA) != 64 {
		t.Fatalf("encodings differ:\n%s %s\n%s %s", a, hashA, b, hashB)
	}
	var decoded Bundle
	if err := json.Unmarshal(a, &decoded); err != nil || !reflect.DeepEqual(decoded, first) || !strings.Contains(string(a), `"language":"az"`) {
		t.Fatalf("decoded %+v %v from %s", decoded, err, a)
	}
}

func TestAPublishedBundleLivesAtItsLanguageAndContentHash(t *testing.T) {
	hash := strings.Repeat("ab", 32)
	cases := []struct {
		bundle Published
		want   string
	}{
		{Published{Language: Russian, Version: 3, Hash: hash}, "strings/ru/" + hash + ".json"},
		{Published{Language: English, Version: 1, Hash: "00ff"}, "strings/en/00ff.json"},
	}
	for _, c := range cases {
		if got := c.bundle.File(); got != c.want {
			t.Errorf("%s v%d: %q, want %q", c.bundle.Language, c.bundle.Version, got, c.want)
		}
	}
}

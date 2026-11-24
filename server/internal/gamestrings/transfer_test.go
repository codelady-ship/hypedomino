package gamestrings

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func transferEntries() []Entry {
	coins := coinsEntry()
	coins.Values = map[Language]Value{
		English: {Text: "{coins, plural, one {# coin} other {# coins}}", Status: Approved},
		Russian: {Text: "{coins} монет", Status: Draft},
	}
	win := Entry{Key: "match.result.win", Note: "Match end title.", MaxLength: 12,
		Values: map[Language]Value{English: {Text: "Victory", Status: Approved}}}
	retired := Entry{Key: "lobby.banner.old", Note: "Retired banner.", MaxLength: 30, Deprecated: true}
	return []Entry{win, retired, coins}
}

func TestATransferListsLiveKeysInOrderWithTheEnglishReference(t *testing.T) {
	transfer, err := BuildTransfer(transferEntries(), Russian)
	if err != nil {
		t.Fatal(err)
	}
	want := Transfer{Language: Russian, Rows: []TransferRow{
		{Key: "match.result.coins", Note: coinsEntry().Note, MaxLength: 24, Placeholders: coinsEntry().Placeholders,
			Reference: "{coins, plural, one {# coin} other {# coins}}", Text: "{coins} монет", Status: Draft},
		{Key: "match.result.win", Note: "Match end title.", MaxLength: 12, Placeholders: []Placeholder{},
			Reference: "Victory", Status: Missing},
	}}
	if !reflect.DeepEqual(transfer, want) {
		t.Fatalf("transfer\n%+v\nwant\n%+v", transfer, want)
	}
	if _, err := BuildTransfer(transferEntries(), "de"); !errors.Is(err, ErrInvalidValue) {
		t.Fatalf("unknown language: %v", err)
	}
}

func TestCSVExportHasOneColumnPerLanguageAndABOMForSpreadsheets(t *testing.T) {
	tests := map[Language]string{
		Russian: "key,note,max_length,placeholders,en,ru,status\n" +
			"match.result.coins,\"Match end screen, under the score. coins is the reward amount.\",24,coins:integer,\"{coins, plural, one {# coin} other {# coins}}\",{coins} монет,draft\n" +
			"match.result.win,Match end title.,12,,Victory,,missing\n",
		English: "key,note,max_length,placeholders,en,status\n" +
			"match.result.coins,\"Match end screen, under the score. coins is the reward amount.\",24,coins:integer,\"{coins, plural, one {# coin} other {# coins}}\",approved\n" +
			"match.result.win,Match end title.,12,,Victory,approved\n",
	}
	for language, want := range tests {
		transfer, err := BuildTransfer(transferEntries(), language)
		if err != nil {
			t.Fatal(err)
		}
		data, err := transfer.Encode(CSV)
		if err != nil {
			t.Fatal(err)
		}
		if got := string(data); got != byteOrderMark+want {
			t.Errorf("%s csv\n%s\nwant\n%s", language, got, want)
		}
	}
}

func TestJSONExportCarriesTheLanguage(t *testing.T) {
	transfer, _ := BuildTransfer(transferEntries(), Russian)
	data, err := transfer.Encode(JSON)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Transfer
	if err := json.Unmarshal(data, &decoded); err != nil || decoded.Language != Russian || len(decoded.Rows) != 2 {
		t.Fatalf("decoded %+v %v", decoded, err)
	}
	if _, err := transfer.Encode("xlsx"); !errors.Is(err, ErrInvalidImport) {
		t.Fatalf("unknown format: %v", err)
	}
}

func TestAnExportedFileReadsBackForItsLanguage(t *testing.T) {
	transfer, _ := BuildTransfer(transferEntries(), Russian)
	for _, format := range []Format{CSV, JSON} {
		data, err := transfer.Encode(format)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := ParseTransfer(data, format, Russian)
		if err != nil {
			t.Fatalf("%s: %v", format, err)
		}
		texts := map[string]string{}
		for _, row := range parsed.Rows {
			texts[row.Key] = row.Text
		}
		want := map[string]string{"match.result.coins": "{coins} монет", "match.result.win": ""}
		if parsed.Language != Russian || !reflect.DeepEqual(texts, want) {
			t.Errorf("%s: %+v", format, parsed)
		}
	}
}

func TestAnImportFileMustMatchItsLanguageAndShape(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		format   Format
		language Language
		want     error
	}{
		{"json for another language", `{"language":"tr","rows":[]}`, JSON, Russian, ErrInvalidImport},
		{"broken json", `{"language":`, JSON, Russian, ErrInvalidImport},
		{"csv without the language column", "key,en\nmatch.result.win,Victory\n", CSV, Russian, ErrInvalidImport},
		{"csv without a key column", "ru\nПобеда\n", CSV, Russian, ErrInvalidImport},
		{"empty csv", "", CSV, Russian, ErrInvalidImport},
		{"ragged csv", "key,ru\nmatch.result.win\n", CSV, Russian, ErrInvalidImport},
		{"unknown format", "", "xlsx", Russian, ErrInvalidImport},
		{"unknown language", "key,de\n", CSV, "de", ErrInvalidValue},
	}
	for _, test := range tests {
		if _, err := ParseTransfer([]byte(test.data), test.format, test.language); !errors.Is(err, test.want) {
			t.Errorf("%s: %v", test.name, err)
		}
	}
	parsed, err := ParseTransfer([]byte("ru,key\nПобеда,match.result.win\n"), CSV, Russian)
	if err != nil || !reflect.DeepEqual(parsed.Rows, []TransferRow{{Key: "match.result.win", Text: "Победа"}}) {
		t.Fatalf("columns are found by name: %+v %v", parsed, err)
	}
}

func TestAnImportChangesOnlyNewTexts(t *testing.T) {
	transfer := Transfer{Language: Russian, Rows: []TransferRow{
		{Key: "match.result.coins", Text: "{coins} монет"},
		{Key: "match.result.win", Text: "Победа"},
	}}
	changes, problems := PlanImport(transferEntries(), transfer)
	if problems != nil || !reflect.DeepEqual(changes, []Change{{Key: "match.result.win", Text: "Победа"}}) {
		t.Fatalf("changes %+v problems %+v", changes, problems)
	}
	transfer.Rows = []TransferRow{{Key: "match.result.win", Text: ""}}
	if changes, problems := PlanImport(transferEntries(), transfer); changes != nil || problems != nil {
		t.Fatalf("an empty cell leaves the value alone: %+v %+v", changes, problems)
	}
}

func TestAnyBadRowBlocksTheWholeImport(t *testing.T) {
	transfer := Transfer{Language: Russian, Rows: []TransferRow{
		{Key: "match.result.win", Text: "Победа"},
		{Key: "match.result.win", Text: "Выигрыш"},
		{Key: "match.result.lose", Text: "Поражение"},
		{Key: "lobby.banner.old", Text: "Старый баннер"},
		{Key: "match.result.coins", Text: "{coin} монет"},
	}}
	changes, problems := PlanImport(transferEntries(), transfer)
	if changes != nil {
		t.Fatalf("changes %+v", changes)
	}
	want := []error{ErrInvalidImport, ErrUnknownKey, ErrInvalidImport, ErrInvalidValue}
	if len(problems) != len(want) {
		t.Fatalf("problems %+v", problems)
	}
	for i, problem := range problems {
		if !errors.Is(problem.Err, want[i]) {
			t.Errorf("problem %d %s: %v, want %v", i, problem.Key, problem.Err, want[i])
		}
	}
	if !strings.Contains(problems[0].Err.Error(), "repeats") || problems[3].Key != "match.result.coins" {
		t.Errorf("problems name their row: %+v", problems)
	}
}

func TestATextTooLongForItsKeyBlocksTheImport(t *testing.T) {
	transfer := Transfer{Language: Russian, Rows: []TransferRow{{Key: "match.result.win", Text: string(bytes.Repeat([]byte("Я"), 13))}}}
	if _, problems := PlanImport(transferEntries(), transfer); len(problems) != 1 || !errors.Is(problems[0].Err, ErrInvalidValue) {
		t.Fatalf("problems %+v", problems)
	}
}

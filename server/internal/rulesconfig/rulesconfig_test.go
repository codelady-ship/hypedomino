package rulesconfig

import (
	"errors"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func documentedKeys(t *testing.T) map[string][]string {
	t.Helper()
	doc, err := os.ReadFile("../../../docs/rules.md")
	if err != nil {
		t.Fatal(err)
	}
	_, table, found := strings.Cut(string(doc), "| Key | Type | Unit | Default | Min | Max |\n|---|---|---|---|---|---|\n")
	if !found {
		t.Fatal("rules.md has no rules domain bounds table")
	}
	rows := map[string][]string{}
	for _, line := range strings.Split(table, "\n") {
		if !strings.HasPrefix(line, "| ") {
			break
		}
		cells := strings.Split(strings.Trim(line, "| "), " | ")
		rows[strings.Trim(cells[0], "`")] = cells[1:]
	}
	return rows
}

func TestTheKeysMatchTheBoundsTableInTheRulesSpec(t *testing.T) {
	documented := documentedKeys(t)
	if len(documented) != len(Keys()) {
		t.Fatalf("rules.md lists %d keys, code has %d", len(documented), len(Keys()))
	}
	for _, key := range Keys() {
		want := []string{"integer", key.Unit, strconv.Itoa(key.Default()), strconv.Itoa(key.Min), strconv.Itoa(key.Max)}
		if got := documented[key.Name]; !reflect.DeepEqual(got, want) {
			t.Errorf("%s: rules.md %v, code %v", key.Name, got, want)
		}
		if key.Default() < key.Min || key.Default() > key.Max {
			t.Errorf("%s: default %d outside its bounds", key.Name, key.Default())
		}
	}
}

func TestSnapshotsAreCheckedAgainstTheBounds(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want Tuning
	}{
		{"every key", `{"TurnSeconds": 25, "TurnWarningSeconds": 6, "AfkConsecutiveTimeouts": 2, "MatchLoadingSeconds": 15}`, Tuning{"TurnSeconds": 25, "TurnWarningSeconds": 6, "AfkConsecutiveTimeouts": 2, "MatchLoadingSeconds": 15}},
		{"one key", `{"TurnSeconds": 30}`, Tuning{"TurnSeconds": 30}},
		{"at the bounds", `{"TurnSeconds": 10, "TurnWarningSeconds": 10, "MatchLoadingSeconds": 60}`, Tuning{"TurnSeconds": 10, "TurnWarningSeconds": 10, "MatchLoadingSeconds": 60}},
		{"lowest warning", `{"TurnWarningSeconds": 1}`, Tuning{"TurnWarningSeconds": 1}},
		{"no keys", `{}`, Tuning{}},
	}
	for _, test := range tests {
		got, err := Parse([]byte(test.raw))
		if err != nil || !reflect.DeepEqual(got, test.want) {
			t.Errorf("%s: %v %v", test.name, got, err)
		}
	}
	for name, raw := range map[string]string{
		"unknown key": `{"HandSize": 5}`,
		"below min":   `{"TurnSeconds": 9}`,
		"above max":   `{"AfkConsecutiveTimeouts": 11}`,
		"no warning":  `{"TurnWarningSeconds": 0}`,
		"warn at 11":  `{"TurnWarningSeconds": 11}`,
		"fraction":    `{"TurnSeconds": 20.5}`,
		"string":      `{"TurnSeconds": "20"}`,
		"null value":  `{"TurnSeconds": null}`,
		"array":       `[20]`,
		"null":        `null`,
		"not JSON":    `TurnSeconds=20`,
	} {
		if _, err := Parse([]byte(raw)); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s accepted: %v", name, err)
		}
	}
}

func TestATuningChangesOnlyTheKeysItSets(t *testing.T) {
	base := rules.DefaultConfig()
	if got := Tuning(nil).Apply(base); got != base {
		t.Fatalf("an empty tuning changed the config: %+v", got)
	}
	want := base
	want.TurnSeconds = 30
	want.TurnWarningSeconds = 8
	if got := (Tuning{"TurnSeconds": 30, "TurnWarningSeconds": 8}).Apply(base); got != want {
		t.Fatalf("tuned %+v", got)
	}
}

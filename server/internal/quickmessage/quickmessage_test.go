package quickmessage_test

import (
	"errors"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/quickmessage"
)

func TestParseChecksTheKeys(t *testing.T) {
	cases := map[string]string{
		"not quick":     `{"keys":["match.result.win"]}`,
		"upper case":    `{"keys":["quick.Hello"]}`,
		"listed twice":  `{"keys":["quick.hello","quick.hello"]}`,
		"unknown field": `{"keys":[],"cooldown":3}`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := quickmessage.Parse([]byte(raw)); !errors.Is(err, quickmessage.ErrInvalid) {
				t.Fatalf("err %v", err)
			}
		})
	}
	catalog, err := quickmessage.Parse([]byte(`{"keys":["quick.hello","quick.well_played"]}`))
	if err != nil || !catalog.Contains("quick.well_played") || catalog.Contains("quick.bye") {
		t.Fatalf("catalog %+v %v", catalog, err)
	}
}

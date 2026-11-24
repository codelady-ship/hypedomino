package profile

import (
	"errors"
	"testing"
)

const sampleModeration = `{
	"words": {
		"any": [{"entry": "badword", "match": "substring"}],
		"az": [{"entry": "Pis Söz", "match": "substring"}],
		"en": [{"entry": "scum", "match": "word"}],
		"tr": [{"entry": "çirkin", "match": "word"}],
		"ru": [{"entry": "плохо", "match": "substring"}]
	},
	"reserved": ["admin", "moderator", "HypeDomino"]
}`

func sample(t *testing.T) Moderation {
	t.Helper()
	m, err := ParseModeration([]byte(sampleModeration))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestTheFilterCatchesAtLeastNinetyFivePercentOfTheSampleEvasions(t *testing.T) {
	m := sample(t)
	evasions := []string{
		"badword", "BadWord", "BADWORD", "xXbadwordXx", "b4dw0rd", "B.a.d.W.o.r.d", "bad_word", "bad word", "bad-w0rd",
		"bаdword", "bаdwоrd", "B4DW0RD_99", "pis söz", "PisSoz", "p1s_s0z", "pıs söz", "scum", "SCUM", "the scum",
		"5cum", "s.c.u.m", "çirkin", "Cirkin", "ÇİRKİN", "c1rk1n", "плохо", "ПЛОХО", "плoxo", "admin", "Admin_Ayla",
		"4dm1n", "adm1n", "a.d.m.i.n", "ADMlN", "moderator", "M0der4tor", "hypedomino", "Hype_Domino", "HYPE.D0M1N0", "the_admin",
	}
	caught := 0
	var missed []string
	for _, name := range evasions {
		if !m.Allows(name) {
			caught++
		} else {
			missed = append(missed, name)
		}
	}
	if caught*100 < len(evasions)*95 {
		t.Fatalf("caught %d of %d, missed %v", caught, len(evasions), missed)
	}
}

func TestTheFilterLetsCleanNamesThroughAndWholeWordsMatchOnlyWholeWords(t *testing.T) {
	m := sample(t)
	for _, name := range []string{"Ayla", "Kamal_07", "Scumbag", "Günəş-1", "Çirkinlik", "Domino Fan", "Hype Man", "Mod 7"} {
		if !m.Allows(name) {
			t.Errorf("%q refused", name)
		}
	}
	if !(Moderation{}).Allows("badword") {
		t.Fatal("an empty config refused a name")
	}
}

func TestAnInvalidModerationConfigIsRefused(t *testing.T) {
	for _, raw := range []string{
		`{"words": {"de": [{"entry": "x", "match": "word"}]}}`,
		`{"words": {"en": [{"entry": "x", "match": "prefix"}]}}`,
		`{"words": {"en": [{"entry": " - ", "match": "word"}]}}`,
		`{"reserved": [""]}`,
		`{"lists": {}}`,
		`not json`,
	} {
		if _, err := ParseModeration([]byte(raw)); !errors.Is(err, ErrInvalidModeration) {
			t.Errorf("%s: %v", raw, err)
		}
	}
	if _, err := ParseModeration([]byte(`{"words": {}, "reserved": []}`)); err != nil {
		t.Fatalf("empty config: %v", err)
	}
}

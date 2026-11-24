package profile

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

const ModerationDomain = "moderation"

const (
	MatchSubstring = "substring"
	MatchWord      = "word"
)

var (
	ErrNotAllowed        = errors.New("nickname.not_allowed")
	ErrInvalidModeration = errors.New("profile: invalid moderation config")
)

var WordLists = []string{"any", "az", "en", "tr", "ru"}

var leet = strings.NewReplacer("0", "o", "1", "i", "3", "e", "4", "a", "5", "s", "7", "t", "@", "a", "$", "s")

type WordEntry struct {
	Entry string `json:"entry"`
	Match string `json:"match"`
}

type Moderation struct {
	Words      map[string][]WordEntry `json:"words"`
	Reserved   []string               `json:"reserved"`
	substrings []string
	words      []string
}

func ParseModeration(raw []byte) (Moderation, error) {
	var m Moderation
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&m); err != nil {
		return Moderation{}, fmt.Errorf("%w: %v", ErrInvalidModeration, err)
	}
	for list, entries := range m.Words {
		if !slices.Contains(WordLists, list) {
			return Moderation{}, fmt.Errorf("%w: unknown word list %q", ErrInvalidModeration, list)
		}
		for _, e := range entries {
			if err := m.add(e); err != nil {
				return Moderation{}, fmt.Errorf("%w: list %s: %v", ErrInvalidModeration, list, err)
			}
		}
	}
	for _, name := range m.Reserved {
		if err := m.add(WordEntry{Entry: name, Match: MatchSubstring}); err != nil {
			return Moderation{}, fmt.Errorf("%w: reserved: %v", ErrInvalidModeration, err)
		}
	}
	return m, nil
}

func (m *Moderation) add(e WordEntry) error {
	joined := joinedForm(e.Entry)
	if joined == "" {
		return fmt.Errorf("entry %q is empty after folding", e.Entry)
	}
	switch e.Match {
	case MatchSubstring:
		m.substrings = append(m.substrings, joined)
	case MatchWord:
		m.words = append(m.words, joined)
	default:
		return fmt.Errorf("entry %q has match %q, want %s or %s", e.Entry, e.Match, MatchSubstring, MatchWord)
	}
	return nil
}

func (m Moderation) Allows(display string) bool {
	spaced := filterForm(display)
	joined := withoutSeparators(spaced)
	for _, s := range m.substrings {
		if strings.Contains(joined, s) {
			return false
		}
	}
	words := strings.FieldsFunc(spaced, isSeparator)
	for _, w := range m.words {
		if joined == w || slices.Contains(words, w) {
			return false
		}
	}
	return true
}

func filterForm(display string) string {
	return leet.Replace(Key(Display(display)))
}

func joinedForm(entry string) string {
	return withoutSeparators(filterForm(entry))
}

func withoutSeparators(s string) string {
	return strings.Join(strings.FieldsFunc(s, isSeparator), "")
}

func isSeparator(r rune) bool {
	return strings.ContainsRune(separators, r)
}

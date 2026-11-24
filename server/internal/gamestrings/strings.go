package gamestrings

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var (
	ErrInvalidEntry = errors.New("gamestrings: invalid entry")
	ErrInvalidValue = errors.New("gamestrings: invalid value")
	ErrBlocked      = errors.New("gamestrings: bundle blocked")
)

type Language string

const (
	Azerbaijani Language = "az"
	English     Language = "en"
	Turkish     Language = "tr"
	Russian     Language = "ru"
)

var Languages = []Language{Azerbaijani, English, Turkish, Russian}

var pluralCategories = map[Language][]string{
	Azerbaijani: {"one", "other"},
	English:     {"one", "other"},
	Turkish:     {"one", "other"},
	Russian:     {"one", "few", "many", "other"},
}

type Status string

const (
	Missing  Status = "missing"
	Draft    Status = "draft"
	Approved Status = "approved"
)

type PlaceholderType string

const (
	Integer   PlaceholderType = "integer"
	Decimal   PlaceholderType = "decimal"
	String    PlaceholderType = "string"
	Timestamp PlaceholderType = "timestamp"
	Duration  PlaceholderType = "duration"
	ItemID    PlaceholderType = "item"
)

var placeholderTypes = []PlaceholderType{Integer, Decimal, String, Timestamp, Duration, ItemID}

type Placeholder struct {
	Name string          `json:"name"`
	Type PlaceholderType `json:"type"`
}

type Value struct {
	Text   string
	Status Status
}

type Entry struct {
	Key          string
	Note         string
	MaxLength    int
	Placeholders []Placeholder
	Values       map[Language]Value
	Deprecated   bool
}

var keyPattern = regexp.MustCompile(`^[a-z0-9_]+(\.[a-z0-9_]+)+$`)

func ValidKey(key string) bool {
	return keyPattern.MatchString(key)
}

func (e Entry) Validate() error {
	switch {
	case !ValidKey(e.Key):
		return fmt.Errorf("%w: key %q", ErrInvalidEntry, e.Key)
	case strings.TrimSpace(e.Note) == "":
		return fmt.Errorf("%w: %s has no translator note", ErrInvalidEntry, e.Key)
	case e.MaxLength <= 0:
		return fmt.Errorf("%w: %s has no max length", ErrInvalidEntry, e.Key)
	}
	seen := map[string]bool{}
	for _, placeholder := range e.Placeholders {
		if !validName(placeholder.Name) || seen[placeholder.Name] || !slices.Contains(placeholderTypes, placeholder.Type) {
			return fmt.Errorf("%w: %s placeholder %+v", ErrInvalidEntry, e.Key, placeholder)
		}
		seen[placeholder.Name] = true
	}
	return nil
}

func (e Entry) CheckValue(language Language, text string) error {
	categories, known := pluralCategories[language]
	if !known {
		return fmt.Errorf("%w: unknown language %q", ErrInvalidValue, language)
	}
	message, err := parseMessage(text)
	if err != nil {
		return fmt.Errorf("%w: %s %s: %w", ErrInvalidValue, e.Key, language, err)
	}
	if message.length > e.MaxLength {
		return fmt.Errorf("%w: %s %s is %d long, max %d", ErrInvalidValue, e.Key, language, message.length, e.MaxLength)
	}
	if err := e.checkPlaceholders(message); err != nil {
		return fmt.Errorf("%w: %s %s: %w", ErrInvalidValue, e.Key, language, err)
	}
	if err := checkSelectors(message, categories); err != nil {
		return fmt.Errorf("%w: %s %s: %w", ErrInvalidValue, e.Key, language, err)
	}
	return nil
}

func (e Entry) checkPlaceholders(message parsed) error {
	declared := map[string]PlaceholderType{}
	for _, placeholder := range e.Placeholders {
		declared[placeholder.Name] = placeholder.Type
	}
	for name, uses := range message.arguments {
		kind, isDeclared := declared[name]
		if !isDeclared {
			return fmt.Errorf("undeclared placeholder {%s}", name)
		}
		for _, use := range uses {
			if use.kind == pluralArgument && kind != Integer && kind != Decimal {
				return fmt.Errorf("plural on {%s} of type %s", name, kind)
			}
			if use.kind == selectArgument && kind != String {
				return fmt.Errorf("select on {%s} of type %s", name, kind)
			}
		}
	}
	for name := range declared {
		if _, used := message.arguments[name]; !used {
			return fmt.Errorf("placeholder {%s} is not used", name)
		}
	}
	return nil
}

func checkSelectors(message parsed, categories []string) error {
	for name, uses := range message.arguments {
		for _, use := range uses {
			if err := checkArgumentSelectors(name, use, categories); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkArgumentSelectors(name string, use argument, categories []string) error {
	if use.kind == simpleArgument {
		return nil
	}
	if len(slices.Compact(slices.Sorted(slices.Values(use.selectors)))) != len(use.selectors) {
		return fmt.Errorf("{%s} repeats a selector", name)
	}
	if !slices.Contains(use.selectors, "other") {
		return fmt.Errorf("{%s} has no other branch", name)
	}
	if use.kind == selectArgument {
		return nil
	}
	for _, selector := range use.selectors {
		if !slices.Contains(categories, selector) && !exactNumber(selector) {
			return fmt.Errorf("{%s} has category %q, not one of %v", name, selector, categories)
		}
	}
	for _, category := range categories {
		if !slices.Contains(use.selectors, category) {
			return fmt.Errorf("{%s} lacks category %q", name, category)
		}
	}
	return nil
}

func exactNumber(selector string) bool {
	digits, found := strings.CutPrefix(selector, "=")
	_, err := strconv.ParseUint(digits, 10, 32)
	return found && err == nil
}

type Problem struct {
	Key string
	Err error
}

type Bundle struct {
	Language Language          `json:"language"`
	Strings  map[string]string `json:"strings"`
}

func BuildBundle(entries []Entry, language Language) (Bundle, []Problem) {
	bundle := Bundle{Language: language, Strings: map[string]string{}}
	var problems []Problem
	for _, entry := range entries {
		value, has := entry.Values[language]
		if entry.Deprecated || !has || value.Status != Approved {
			continue
		}
		if err := entry.Validate(); err != nil {
			problems = append(problems, Problem{Key: entry.Key, Err: err})
			continue
		}
		if err := entry.CheckValue(language, value.Text); err != nil {
			problems = append(problems, Problem{Key: entry.Key, Err: err})
			continue
		}
		bundle.Strings[entry.Key] = value.Text
	}
	if len(problems) > 0 {
		return Bundle{}, problems
	}
	return bundle, nil
}

func (b Bundle) Encode() ([]byte, string, error) {
	data, err := json.Marshal(b)
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(data)
	return data, hex.EncodeToString(sum[:]), nil
}

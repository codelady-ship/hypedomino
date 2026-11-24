package quickmessage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
)

const Domain = "quick_messages"

var (
	ErrInvalid = errors.New("quickmessage: invalid catalog")
	keyFormat  = regexp.MustCompile(`^quick\.[a-z0-9_]+$`)
)

type Catalog struct {
	Keys []string `json:"keys"`
}

func Parse(raw []byte) (Catalog, error) {
	var catalog Catalog
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&catalog); err != nil {
		return Catalog{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	return catalog, catalog.validate()
}

func (c Catalog) validate() error {
	for i, key := range c.Keys {
		if !keyFormat.MatchString(key) {
			return fmt.Errorf("%w: key %q is not quick.<slug>", ErrInvalid, key)
		}
		if slices.Contains(c.Keys[:i], key) {
			return fmt.Errorf("%w: key %q is listed twice", ErrInvalid, key)
		}
	}
	return nil
}

func (c Catalog) Contains(key string) bool {
	return slices.Contains(c.Keys, key)
}

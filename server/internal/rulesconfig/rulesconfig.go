package rulesconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

const Domain = "rules"

var ErrInvalid = errors.New("rulesconfig: invalid rules snapshot")

type Key struct {
	Name  string
	Unit  string
	Min   int
	Max   int
	field func(*rules.Config) *int
}

var keys = []Key{
	{Name: "TurnSeconds", Unit: "seconds", Min: 10, Max: 60, field: func(c *rules.Config) *int { return &c.TurnSeconds }},
	{Name: "TurnWarningSeconds", Unit: "seconds", Min: 1, Max: 10, field: func(c *rules.Config) *int { return &c.TurnWarningSeconds }},
	{Name: "AfkConsecutiveTimeouts", Unit: "timeouts", Min: 1, Max: 10, field: func(c *rules.Config) *int { return &c.AfkConsecutiveTimeouts }},
	{Name: "MatchLoadingSeconds", Unit: "seconds", Min: 5, Max: 60, field: func(c *rules.Config) *int { return &c.MatchLoadingSeconds }},
}

func Keys() []Key {
	return keys
}

func (k Key) Default() int {
	config := rules.DefaultConfig()
	return *k.field(&config)
}

type Tuning map[string]int

func (t Tuning) Apply(config rules.Config) rules.Config {
	for _, key := range keys {
		if value, set := t[key.Name]; set {
			*key.field(&config) = value
		}
	}
	return config
}

func Parse(raw []byte) (Tuning, error) {
	var values map[string]json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil || values == nil {
		return nil, fmt.Errorf("%w: not a JSON object: %v", ErrInvalid, err)
	}
	tuning := Tuning{}
	for name, value := range values {
		key, known := lookup(name)
		if !known {
			return nil, fmt.Errorf("%w: unknown key %s", ErrInvalid, name)
		}
		number, err := strconv.Atoi(string(value))
		if err != nil {
			return nil, fmt.Errorf("%w: %s is not an integer: %s", ErrInvalid, name, value)
		}
		if number < key.Min || number > key.Max {
			return nil, fmt.Errorf("%w: %s %d outside %d to %d %s", ErrInvalid, name, number, key.Min, key.Max, key.Unit)
		}
		tuning[name] = number
	}
	return tuning, nil
}

func lookup(name string) (Key, bool) {
	for _, key := range keys {
		if key.Name == name {
			return key, true
		}
	}
	return Key{}, false
}

package promo_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/promo"
)

func TestCodesAreNormalizedAndGeneratedFromTheSafeAlphabet(t *testing.T) {
	for raw, want := range map[string]string{"abcd-efgh": "ABCDEFGH", " wxyz 2345 ": "WXYZ2345", "AB_CD": "ABCD"} {
		got, err := promo.Normalize(raw)
		if err != nil || got != want {
			t.Errorf("Normalize(%q) = %q, %v", raw, got, err)
		}
	}
	for _, raw := range []string{"abc", "ABC0DEFG", "ABCODEFG", "ABC1DEFG", "ABCIDEFG", "ABCLDEFG", strings.Repeat("A", 33)} {
		if _, err := promo.Normalize(raw); !errors.Is(err, promo.ErrCodeFormat) {
			t.Errorf("Normalize(%q) = %v", raw, err)
		}
	}
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		code, err := promo.Generate(12)
		if err != nil {
			t.Fatal(err)
		}
		if len(code) != 12 || strings.Trim(code, promo.Alphabet) != "" || seen[code] {
			t.Fatalf("generated %q", code)
		}
		seen[code] = true
	}
	if _, err := promo.Generate(3); !errors.Is(err, promo.ErrCodeFormat) {
		t.Errorf("short length accepted: %v", err)
	}
	if _, err := promo.NewHasher("short"); !errors.Is(err, promo.ErrKey) {
		t.Errorf("short key accepted: %v", err)
	}
	hasher, err := promo.NewHasher("0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	if string(hasher.Hash("ABCD")) == string(hasher.Hash("ABCE")) || len(hasher.Hash("ABCD")) != 32 {
		t.Error("hash is not a keyed sha256")
	}
}

func TestConfigRejectsOutOfRangeValues(t *testing.T) {
	if err := promo.DefaultConfig().Validate(); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*promo.Config){
		"code length": func(c *promo.Config) { c.CodeLength = 2 },
		"validity":    func(c *promo.Config) { c.MaxValidityDays = 0 },
		"attempts":    func(c *promo.Config) { c.AttemptsPerHour = 0 },
		"failures":    func(c *promo.Config) { c.FailedAttemptsBeforeLock = 0 },
		"lock":        func(c *promo.Config) { c.LockMinutes = 0 },
		"threshold":   func(c *promo.Config) { c.CoinsApprovalThreshold = -1 },
	} {
		config := promo.DefaultConfig()
		mutate(&config)
		if err := config.Validate(); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

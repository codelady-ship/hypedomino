package friends

import (
	"strings"
	"testing"
)

func TestFriendCodesReadForgivingly(t *testing.T) {
	for _, tc := range []struct {
		input, want string
		ok          bool
	}{
		{"7K3M-9QPZ", "7K3M9QPZ", true},
		{" 7k3m 9qpz ", "7K3M9QPZ", true},
		{"O1IL-ABCD", "0111ABCD", true},
		{"7K3M-9QP", "", false},
		{"7K3M-9QPU", "", false},
		{"", "", false},
	} {
		if got, ok := normalizeCode(tc.input); got != tc.want || ok != tc.ok {
			t.Errorf("normalizeCode(%q) = %q %v, want %q %v", tc.input, got, ok, tc.want, tc.ok)
		}
	}
}

func TestNewCodesUseTheAlphabetAndDisplayInTwoHalves(t *testing.T) {
	seen := map[string]bool{}
	for range 200 {
		code, err := newCode()
		if err != nil {
			t.Fatal(err)
		}
		if normalized, ok := normalizeCode(code); !ok || normalized != code {
			t.Fatalf("code %q does not read back", code)
		}
		seen[code] = true
	}
	if len(seen) < 199 {
		t.Fatalf("only %d distinct codes in 200", len(seen))
	}
	if got := DisplayCode("7K3M9QPZ"); got != "7K3M-9QPZ" || !strings.Contains(got, "-") {
		t.Fatalf("display %q", got)
	}
}

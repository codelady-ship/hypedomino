package analytics_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/analytics"
)

const id = "6f1c2a7e-3b4d-4e5f-8a9b-0c1d2e3f4a5b"

func TestEventValidity(t *testing.T) {
	cases := []struct {
		event analytics.Event
		valid bool
	}{
		{analytics.Event{Name: "app_open", ClientTimeMs: 1}, true},
		{analytics.Event{Name: "nickname_set", ClientTimeMs: 1}, true},
		{analytics.Event{Name: "sign_in", Label: "google", ClientTimeMs: 1}, true},
		{analytics.Event{Name: "sign_in", Label: "apple", ClientTimeMs: 1}, true},
		{analytics.Event{Name: "screen_view", Label: "shop", ClientTimeMs: 1}, true},
		{analytics.Event{Name: "queue_join", Label: "ranked_1v1", ClientTimeMs: 1}, true},
		{analytics.Event{Name: "purchase_start", Label: "crystals_tier1", ClientTimeMs: 1}, true},
		{analytics.Event{Name: "purchase_cancel", Label: "com.game.crystals-2", ClientTimeMs: 1}, true},
		{analytics.Event{Name: "screen_view", Label: strings.Repeat("a", 64), ClientTimeMs: 1}, true},
		{analytics.Event{Name: "screen_view", Label: strings.Repeat("a", 65), ClientTimeMs: 1}, false},
		{analytics.Event{Name: "app_open", Label: "x", ClientTimeMs: 1}, false},
		{analytics.Event{Name: "app_open", Value: 5, ClientTimeMs: 1}, false},
		{analytics.Event{Name: "app_open"}, false},
		{analytics.Event{Name: "app_open", ClientTimeMs: -1}, false},
		{analytics.Event{Name: "sign_in", Label: "email", ClientTimeMs: 1}, false},
		{analytics.Event{Name: "sign_in", ClientTimeMs: 1}, false},
		{analytics.Event{Name: "screen_view", ClientTimeMs: 1}, false},
		{analytics.Event{Name: "screen_view", Label: "ayla@example.com", ClientTimeMs: 1}, false},
		{analytics.Event{Name: "screen_view", Label: "free text", ClientTimeMs: 1}, false},
		{analytics.Event{Name: "screen_view", Label: "_shop", ClientTimeMs: 1}, false},
		{analytics.Event{Name: "match_end", ClientTimeMs: 1}, false},
		{analytics.Event{Name: "purchase_complete", Label: "crystals_tier1", ClientTimeMs: 1}, false},
		{analytics.Event{Name: "reward_grant", ClientTimeMs: 1}, false},
		{analytics.Event{Name: "", ClientTimeMs: 1}, false},
	}
	for _, c := range cases {
		if got := c.event.Valid(); got != c.valid {
			t.Errorf("%+v valid %v, want %v", c.event, got, c.valid)
		}
	}
}

func TestBatchNeedsACanonicalAnalyticsID(t *testing.T) {
	for _, bad := range []string{"", "not-an-id", "00000000-0000-0000-0000-000000000000", "urn:uuid:" + id, "{" + id + "}", strings.ReplaceAll(id, "-", "")} {
		if _, err := (analytics.Batch{AnalyticsID: bad}).Accepted(); !errors.Is(err, analytics.ErrMalformed) {
			t.Errorf("analytics id %q: %v", bad, err)
		}
	}
	accepted, err := analytics.Batch{AnalyticsID: strings.ToUpper(id)}.Accepted()
	if err != nil || accepted.AnalyticsID != id {
		t.Fatalf("upper case id: %+v %v", accepted, err)
	}
}

func TestBatchKeepsOnlyValidEventsInOrder(t *testing.T) {
	batch := analytics.Batch{AnalyticsID: id, Events: []analytics.Event{
		{Name: "app_open", ClientTimeMs: 1},
		{Name: "match_end", ClientTimeMs: 2},
		{Name: "sign_in", Label: "google", ClientTimeMs: 3},
		{Name: "screen_view", Label: "ayla@example.com", ClientTimeMs: 4},
		{Name: "screen_view", Label: "lobby", ClientTimeMs: 5},
	}}
	accepted, err := batch.Accepted()
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range accepted.Events {
		names = append(names, e.Name)
	}
	if got := strings.Join(names, ","); got != "app_open,sign_in,screen_view" {
		t.Fatalf("accepted %s", got)
	}
}

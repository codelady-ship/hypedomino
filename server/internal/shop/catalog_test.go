package shop

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func example(t *testing.T) *Catalog {
	t.Helper()
	raw, err := os.ReadFile("../../../docs/economy/examples/shop.json")
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := Parse(raw, DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func TestTheExampleCatalogParses(t *testing.T) {
	catalog := example(t)
	if len(catalog.Offers) != 6 || len(catalog.Rotations) != 2 {
		t.Fatalf("parsed %d offers and %d rotations", len(catalog.Offers), len(catalog.Rotations))
	}
	fox, ok := catalog.Offer("offer.avatar_comet_fox")
	if !ok || fox.PriceCoins != 400 || fox.Bundle.Cosmetics[0] != "avatar.comet_fox" || fox.LimitPerAccount != 1 {
		t.Fatalf("fox offer %+v", fox)
	}
	starter, _ := catalog.Offer("offer.starter")
	if starter.ProductID != "offer_starter" || !starter.Eligibility.FirstPurchaseOnly || starter.Eligibility.MaxAccountAgeHours != 168 || starter.Bundle.Coins != 1500 {
		t.Fatalf("starter offer %+v", starter)
	}
	if _, ok := catalog.Offer("offer.missing"); ok {
		t.Fatal("unknown offer found")
	}
}

func TestPeriodsStartAtTheResetHour(t *testing.T) {
	cases := []struct {
		period string
		reset  int
		now    string
		start  string
		end    string
	}{
		{PeriodDaily, 0, "2026-10-30T15:00:00Z", "2026-10-30T00:00:00Z", "2026-10-31T00:00:00Z"},
		{PeriodDaily, 6, "2026-10-30T05:59:59Z", "2026-10-29T06:00:00Z", "2026-10-30T06:00:00Z"},
		{PeriodDaily, 6, "2026-10-30T06:00:00Z", "2026-10-30T06:00:00Z", "2026-10-31T06:00:00Z"},
		{PeriodDaily, 0, "2026-10-30T03:00:00+04:00", "2026-10-29T00:00:00Z", "2026-10-30T00:00:00Z"},
		{PeriodWeekly, 0, "2026-10-30T15:00:00Z", "2026-10-26T00:00:00Z", "2026-11-02T00:00:00Z"},
		{PeriodWeekly, 0, "2026-10-26T00:00:00Z", "2026-10-26T00:00:00Z", "2026-11-02T00:00:00Z"},
		{PeriodWeekly, 3, "2026-10-26T02:00:00Z", "2026-10-19T03:00:00Z", "2026-10-26T03:00:00Z"},
		{PeriodWeekly, 0, "2026-11-01T23:59:59Z", "2026-10-26T00:00:00Z", "2026-11-02T00:00:00Z"},
	}
	for _, c := range cases {
		start, end := Config{DailySlots: 4, WeeklySlots: 2, ResetHourUTC: c.reset}.Period(c.period, mustTime(t, c.now))
		if !start.Equal(mustTime(t, c.start)) || !end.Equal(mustTime(t, c.end)) {
			t.Fatalf("%s reset %d at %s: %s to %s", c.period, c.reset, c.now, start.Format(time.RFC3339), end.Format(time.RFC3339))
		}
	}
}

const twoDaily = `{"schema":1,"offers":[
{"id":"offer.a","kind":"cosmetic","section":"daily_rotation","priceCoins":100,"bundle":{"id":"bundle.a","revision":1,"cosmetics":["avatar.a"]},"window":{"start":"2026-01-01T00:00:00Z"},"limitPerAccount":1,"nameKey":"a"},
{"id":"offer.b","kind":"cosmetic","section":"daily_rotation","priceCoins":100,"bundle":{"id":"bundle.b","revision":1,"cosmetics":["avatar.b"]},"window":{"start":"2026-01-01T00:00:00Z","end":"2026-11-03T00:00:00Z"},"limitPerAccount":1,"nameKey":"b"},
{"id":"offer.c","kind":"cosmetic","section":"featured","priceCoins":100,"bundle":{"id":"bundle.c","revision":1,"cosmetics":["avatar.c"]},"window":{"start":"2026-11-01T00:00:00Z","end":"2026-11-02T00:00:00Z"},"limitPerAccount":1,"nameKey":"c"}],
"rotations":[
{"period":"daily","start":"2026-11-01T00:00:00Z","offers":["offer.b"]},
{"period":"daily","start":"2026-10-01T00:00:00Z","offers":["offer.a"]}]}`

func TestAnUnscheduledPeriodRepeatsThePreviousRotation(t *testing.T) {
	catalog, err := Parse([]byte(twoDaily), DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	config := DefaultConfig()
	cases := []struct {
		now       string
		daily     []string
		featured  []string
		available map[string]bool
	}{
		{"2026-09-30T23:00:00Z", nil, nil, map[string]bool{"offer.a": false, "offer.b": false, "offer.c": false}},
		{"2026-10-15T12:00:00Z", []string{"offer.a"}, nil, map[string]bool{"offer.a": true, "offer.b": false, "offer.c": false}},
		{"2026-10-31T23:59:59Z", []string{"offer.a"}, nil, map[string]bool{"offer.a": true, "offer.b": false}},
		{"2026-11-01T00:00:00Z", []string{"offer.b"}, []string{"offer.c"}, map[string]bool{"offer.a": false, "offer.b": true, "offer.c": true}},
		{"2026-11-02T12:00:00Z", []string{"offer.b"}, nil, map[string]bool{"offer.b": true, "offer.c": false}},
		{"2026-11-03T12:00:00Z", nil, nil, map[string]bool{"offer.b": false}},
	}
	for _, c := range cases {
		now := mustTime(t, c.now)
		sections := catalog.Sections(config, now)
		if got := ids(sections[1].Offers); !same(got, c.daily) || sections[1].Name != SectionDaily {
			t.Fatalf("daily at %s: %v", c.now, got)
		}
		if got := ids(sections[0].Offers); !same(got, c.featured) {
			t.Fatalf("featured at %s: %v", c.now, got)
		}
		if _, end := config.Period(PeriodDaily, now); !sections[1].EndsAt.Equal(end) || !sections[0].EndsAt.IsZero() {
			t.Fatalf("section ends at %s: %v %v", c.now, sections[1].EndsAt, sections[0].EndsAt)
		}
		for id, want := range c.available {
			offer, _ := catalog.Offer(id)
			if got := catalog.Available(config, offer, now); got != want {
				t.Fatalf("%s available at %s = %v", id, c.now, got)
			}
		}
	}
	if len(catalog.Sections(config, mustTime(t, "2026-10-15T12:00:00Z"))) != len(Sections) {
		t.Fatal("a section is missing from the view")
	}
}

func ids(offers []Offer) []string {
	out := make([]string, 0, len(offers))
	for _, o := range offers {
		out = append(out, o.ID)
	}
	return out
}

func same(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func offerJSON(fields string) string {
	return `{"schema":1,"offers":[{` + fields + `}],"rotations":[]}`
}

const okWindow = `"window":{"start":"2026-01-01T00:00:00Z"}`

func TestInvalidCatalogsAreRefused(t *testing.T) {
	cases := map[string]struct {
		raw    string
		config Config
		reason string
	}{
		"not json":        {`{`, DefaultConfig(), "invalid catalog"},
		"schema":          {`{"schema":2,"offers":[]}`, DefaultConfig(), "schema 2"},
		"id":              {offerJSON(`"id":"avatar_x","kind":"cosmetic","section":"featured","priceCoins":1,"bundle":{"id":"bundle.x","revision":1,"cosmetics":["avatar.x"]},` + okWindow + `,"limitPerAccount":1,"nameKey":"k"`), DefaultConfig(), "offer id"},
		"kind":            {offerJSON(`"id":"offer.x","kind":"hat","section":"featured","priceCoins":1,"bundle":{"id":"bundle.x","revision":1,"cosmetics":["avatar.x"]},` + okWindow + `,"limitPerAccount":1,"nameKey":"k"`), DefaultConfig(), "kind"},
		"section":         {offerJSON(`"id":"offer.x","kind":"cosmetic","section":"hidden","priceCoins":1,"bundle":{"id":"bundle.x","revision":1,"cosmetics":["avatar.x"]},` + okWindow + `,"limitPerAccount":1,"nameKey":"k"`), DefaultConfig(), "section"},
		"name key":        {offerJSON(`"id":"offer.x","kind":"cosmetic","section":"featured","priceCoins":1,"bundle":{"id":"bundle.x","revision":1,"cosmetics":["avatar.x"]},` + okWindow + `,"limitPerAccount":1`), DefaultConfig(), "nameKey"},
		"both prices":     {offerJSON(`"id":"offer.x","kind":"cosmetic","section":"featured","priceCoins":1,"productId":"p","bundle":{"id":"bundle.x","revision":1,"cosmetics":["avatar.x"]},` + okWindow + `,"limitPerAccount":1,"nameKey":"k"`), DefaultConfig(), "never both"},
		"no price":        {offerJSON(`"id":"offer.x","kind":"cosmetic","section":"featured","bundle":{"id":"bundle.x","revision":1,"cosmetics":["avatar.x"]},` + okWindow + `,"limitPerAccount":1,"nameKey":"k"`), DefaultConfig(), "never both"},
		"product id":      {offerJSON(`"id":"offer.x","kind":"coins","section":"coins","productId":"Coins-1","bundle":{"id":"bundle.x","revision":1,"coins":5},` + okWindow + `,"nameKey":"k"`), DefaultConfig(), "product id"},
		"no window":       {offerJSON(`"id":"offer.x","kind":"cosmetic","section":"featured","priceCoins":1,"bundle":{"id":"bundle.x","revision":1,"cosmetics":["avatar.x"]},"limitPerAccount":1,"nameKey":"k"`), DefaultConfig(), "window start"},
		"window order":    {offerJSON(`"id":"offer.x","kind":"cosmetic","section":"featured","priceCoins":1,"bundle":{"id":"bundle.x","revision":1,"cosmetics":["avatar.x"]},"window":{"start":"2026-01-02T00:00:00Z","end":"2026-01-01T00:00:00Z"},"limitPerAccount":1,"nameKey":"k"`), DefaultConfig(), "ends before"},
		"reference":       {offerJSON(`"id":"offer.x","kind":"cosmetic","section":"featured","priceCoins":100,"referencePriceCoins":100,"bundle":{"id":"bundle.x","revision":1,"cosmetics":["avatar.x"]},` + okWindow + `,"limitPerAccount":1,"nameKey":"k"`), DefaultConfig(), "reference price"},
		"bundle":          {offerJSON(`"id":"offer.x","kind":"cosmetic","section":"featured","priceCoins":1,"bundle":{"id":"bundle.x","revision":0,"cosmetics":["avatar.x"]},` + okWindow + `,"limitPerAccount":1,"nameKey":"k"`), DefaultConfig(), "revision"},
		"medal":           {offerJSON(`"id":"offer.x","kind":"bundle","section":"featured","priceCoins":1,"bundle":{"id":"bundle.x","revision":1,"cosmetics":["avatar.x"],"medals":["medal.x"]},` + okWindow + `,"nameKey":"k"`), DefaultConfig(), "medal"},
		"coins for coins": {offerJSON(`"id":"offer.x","kind":"bundle","section":"featured","priceCoins":1,"bundle":{"id":"bundle.x","revision":1,"coins":5,"cosmetics":["avatar.x"]},` + okWindow + `,"nameKey":"k"`), DefaultConfig(), "grants no coins"},
		"two cosmetics":   {offerJSON(`"id":"offer.x","kind":"cosmetic","section":"featured","priceCoins":1,"bundle":{"id":"bundle.x","revision":1,"cosmetics":["avatar.x","avatar.y"]},` + okWindow + `,"limitPerAccount":1,"nameKey":"k"`), DefaultConfig(), "exactly one"},
		"cosmetic limit":  {offerJSON(`"id":"offer.x","kind":"cosmetic","section":"featured","priceCoins":1,"bundle":{"id":"bundle.x","revision":1,"cosmetics":["avatar.x"]},` + okWindow + `,"nameKey":"k"`), DefaultConfig(), "one per account"},
		"empty bundle":    {offerJSON(`"id":"offer.x","kind":"bundle","section":"featured","productId":"p","bundle":{"id":"bundle.x","revision":1,"coins":5},` + okWindow + `,"nameKey":"k"`), DefaultConfig(), "at least one"},
		"coins kind":      {offerJSON(`"id":"offer.x","kind":"coins","section":"coins","priceCoins":1,"bundle":{"id":"bundle.x","revision":1,"cosmetics":["avatar.x"]},` + okWindow + `,"nameKey":"k"`), DefaultConfig(), "coin pack"},
		"rotation price":  {offerJSON(`"id":"offer.x","kind":"bundle","section":"daily_rotation","productId":"p","bundle":{"id":"bundle.x","revision":1,"cosmetics":["avatar.x"]},` + okWindow + `,"nameKey":"k"`), DefaultConfig(), "paid with coins"},
		"coins section":   {offerJSON(`"id":"offer.x","kind":"cosmetic","section":"coins","priceCoins":1,"bundle":{"id":"bundle.x","revision":1,"cosmetics":["avatar.x"]},` + okWindow + `,"limitPerAccount":1,"nameKey":"k"`), DefaultConfig(), "coin packs"},
		"pass section":    {offerJSON(`"id":"offer.x","kind":"coins","section":"pass","productId":"p","bundle":{"id":"bundle.x","revision":1,"coins":5},` + okWindow + `,"nameKey":"k"`), DefaultConfig(), "pass products"},
		"duplicate":       {strings.Replace(twoDaily, `"id":"offer.b"`, `"id":"offer.a"`, 1), DefaultConfig(), "listed twice"},
		"rotation period": {strings.Replace(twoDaily, `"period":"daily","start":"2026-11-01`, `"period":"hourly","start":"2026-11-01`, 1), DefaultConfig(), "rotation period"},
		"rotation start":  {strings.Replace(twoDaily, `"start":"2026-11-01T00:00:00Z","offers"`, `"start":"2026-11-01T00:30:00Z","offers"`, 1), DefaultConfig(), "not a period start"},
		"rotation slots":  {strings.Replace(twoDaily, `"offers":["offer.b"]`, `"offers":["offer.b","offer.a"]`, 1), Config{DailySlots: 1, WeeklySlots: 1}, "slots 1 to 1"},
		"rotation twice":  {strings.Replace(twoDaily, `"start":"2026-10-01T00:00:00Z","offers":["offer.a"]`, `"start":"2026-11-01T00:00:00Z","offers":["offer.a"]`, 1), DefaultConfig(), "listed twice"},
		"rotation offer":  {strings.Replace(twoDaily, `"offers":["offer.b"]`, `"offers":["offer.z"]`, 1), DefaultConfig(), "unknown offer"},
		"rotation sect":   {strings.Replace(twoDaily, `"offers":["offer.b"]`, `"offers":["offer.c"]`, 1), DefaultConfig(), "of section featured"},
		"rotation dup":    {strings.Replace(twoDaily, `"offers":["offer.b"]`, `"offers":["offer.b","offer.b"]`, 1), DefaultConfig(), "twice"},
		"config":          {twoDaily, Config{DailySlots: 0, WeeklySlots: 2}, "slots"},
	}
	for name, c := range cases {
		_, err := Parse([]byte(c.raw), c.config)
		if err == nil || !strings.Contains(err.Error(), c.reason) {
			t.Fatalf("%s: %v", name, err)
		}
		if name != "config" && !errors.Is(err, ErrInvalid) {
			t.Fatalf("%s: not ErrInvalid: %v", name, err)
		}
	}
	if err := (Config{DailySlots: 4, WeeklySlots: 2, ResetHourUTC: 24}).Validate(); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("reset hour 24: %v", err)
	}
	if _, err := Parse([]byte(offerJSON(`"id":"offer.x","kind":"pass","section":"pass","productId":"pass_premium",`+okWindow+`,"nameKey":"k"`)), DefaultConfig()); err != nil {
		t.Fatalf("pass offer without a bundle: %v", err)
	}
}

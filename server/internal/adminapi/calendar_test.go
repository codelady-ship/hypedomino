package adminapi_test

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/adminapi"
	"github.com/Alievs-corp/hypedomino/server/internal/pass"
	"github.com/Alievs-corp/hypedomino/server/internal/shop"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
)

func (a *api) calendar(token string) []string {
	a.t.Helper()
	status, body := a.call("GET", "/admin/calendar", token, nil)
	if status != http.StatusOK {
		a.t.Fatalf("calendar %d %v", status, body)
	}
	var lines []string
	for _, raw := range body["entries"].([]any) {
		entry := raw.(map[string]any)
		end, _ := entry["end"].(string)
		lines = append(lines, entry["kind"].(string)+" "+entry["id"].(string)+" "+entry["start"].(string)+" "+end)
	}
	return lines
}

func TestTheCalendarListsRunningAndPlannedWindowsInStartOrder(t *testing.T) {
	a := newAPI(t)
	a.now = time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	ctx := context.Background()
	day := func(d int) time.Time { return time.Date(2026, 11, d, 0, 0, 0, 0, time.UTC) }
	if _, err := a.pool.Exec(ctx, `insert into ranked_seasons (id, name_key, starts_at, ends_at, rewards, created_at) values
		('season.old', 'n', $1, $2, true, $1), ('season.now', 'n', $2, $3, true, $1)`, day(1), day(5), day(20)); err != nil {
		t.Fatal(err)
	}
	if _, err := a.pool.Exec(ctx, `insert into promo_codes (promo_id, campaign_id, kind, bundle, code_hash, starts_at, ends_at, cap, state, created_by) values
		('p1', 'camp.winter', 'multi_use', '{}', '\x01', $1, $2, 5, 'active', 'root'),
		('p2', 'camp.winter', 'multi_use', '{}', '\x02', $2, $3, 5, 'pending_approval', 'root'),
		('p3', 'camp.off', 'multi_use', '{}', '\x03', $1, $3, 5, 'disabled', 'root')`, day(9), day(12), day(15)); err != nil {
		t.Fatal(err)
	}
	if _, err := snapshot.Publish(ctx, a.pool, pass.Domain, []byte(`{"seasons":[{"id":"season_1","start":"2026-11-10T00:00:00Z","end":"2026-12-10T00:00:00Z"}]}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := snapshot.Publish(ctx, a.pool, shop.Domain, []byte(`{"offers":[
		{"id":"offer.feature","section":"featured","window":{"start":"2026-11-06T00:00:00Z"}},
		{"id":"offer.daily","section":"daily","window":{"start":"2026-11-07T00:00:00Z","end":"2026-11-30T00:00:00Z"}},
		{"id":"offer.event","section":"event","window":{"start":"2026-11-01T00:00:00Z","end":"2026-11-08T00:00:00Z"}}],
		"rotations":[
		{"period":"daily","start":"2026-11-07T00:00:00Z","offers":["offer.daily"]},
		{"period":"daily","start":"2026-11-08T00:00:00Z","offers":["offer.daily"]},
		{"period":"weekly","start":"2026-11-09T00:00:00Z","offers":["offer.weekly"]}]}`)); err != nil {
		t.Fatal(err)
	}

	want := []string{
		adminapi.CalendarRankedSeason + " season.now 2026-11-05T00:00:00Z 2026-11-20T00:00:00Z",
		adminapi.CalendarShopOffer + " offer.feature 2026-11-06T00:00:00Z ",
		adminapi.CalendarRotation + " daily 2026-11-08T00:00:00Z 2026-11-09T00:00:00Z",
		adminapi.CalendarPromo + " camp.winter 2026-11-09T00:00:00Z 2026-11-15T00:00:00Z",
		adminapi.CalendarRotation + " weekly 2026-11-09T00:00:00Z 2026-11-16T00:00:00Z",
		adminapi.CalendarPassSeason + " season_1 2026-11-10T00:00:00Z 2026-12-10T00:00:00Z",
	}
	if got := a.calendar(a.signIn("root")); !slices.Equal(got, want) {
		t.Fatalf("root calendar\n%v\nwant\n%v", got, want)
	}
	withoutPromo := []string{want[0], want[1], want[2], want[4], want[5]}
	if got := a.calendar(a.signIn("ayla")); !slices.Equal(got, withoutPromo) {
		t.Fatalf("content calendar\n%v\nwant\n%v", got, withoutPromo)
	}
}

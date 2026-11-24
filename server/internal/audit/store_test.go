package audit_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

var superAdmin = []rbac.Role{rbac.SuperAdmin}

var start = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

func entry(actor, action string, objectType rbac.Object, objectID string) audit.Entry {
	return audit.Entry{
		Actor:       audit.Actor{ID: actor, Roles: []rbac.Role{rbac.LiveOps}},
		Environment: audit.Production,
		Action:      action,
		Object:      audit.Object{Type: objectType, ID: objectID},
		Source:      audit.Source{SessionID: "s-" + actor, IP: netip.MustParseAddr("203.0.113.7"), UserAgent: "Browser/1"},
	}
}

func appendAll(t *testing.T, store *audit.Store, entries ...audit.Entry) []audit.Recorded {
	t.Helper()
	recorded := make([]audit.Recorded, len(entries))
	for i, e := range entries {
		r, err := store.Append(context.Background(), e, start.Add(time.Duration(i)*time.Minute))
		if err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
		recorded[i] = r
	}
	return recorded
}

func ids(records []audit.Recorded) []int64 {
	out := make([]int64, len(records))
	for i, r := range records {
		out[i] = r.ID
	}
	return out
}

func TestAnEntryReadsBackWithEveryField(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := audit.NewStore(pool)
	ctx := context.Background()
	proposal := appendAll(t, store, entry("ayla", "submit", rbac.PromoCodes, "SPRING"))[0]
	full := entry("ben", "approve", rbac.PromoCodes, "SPRING")
	full.Actor.Roles = []rbac.Role{rbac.LiveOps, rbac.Support}
	full.Before = json.RawMessage(`{"state": "pending"}`)
	full.After = json.RawMessage(`{"state":"approved","coins":2500}`)
	full.Reason = "Spring campaign"
	full.ApprovalID = proposal.ID
	full.Ref = "INC-12"
	full.ChangeSet = "cs-7"
	full.ConfigVersion = 41
	written, err := store.Append(ctx, full, start.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	found, err := store.Search(ctx, superAdmin, audit.Filter{ActorID: "ben"})
	if err != nil || len(found) != 1 {
		t.Fatalf("search %+v %v", found, err)
	}
	got := found[0]
	if got.ID != written.ID || !got.At.Equal(start.Add(time.Hour)) || got.ApprovalID != proposal.ID || got.ConfigVersion != 41 {
		t.Fatalf("read back %+v", got)
	}
	if !reflect.DeepEqual(got.Actor, full.Actor) || got.Object != full.Object || got.Source != full.Source {
		t.Fatalf("actor, object or source %+v", got)
	}
	if got.Reason != full.Reason || got.Ref != full.Ref || got.ChangeSet != full.ChangeSet || got.Environment != audit.Production {
		t.Fatalf("text fields %+v", got)
	}
	assertSameJSON(t, got.Before, full.Before)
	assertSameJSON(t, got.After, full.After)
}

func TestOptionalFieldsStayEmpty(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := audit.NewStore(pool)
	bare := entry("ayla", "view", rbac.Players, "p-1")
	bare.Source.IP = netip.Addr{}
	appendAll(t, store, bare)
	found, err := store.Search(context.Background(), superAdmin, audit.Filter{})
	if err != nil || len(found) != 1 {
		t.Fatalf("search %+v %v", found, err)
	}
	got := found[0]
	if got.Before != nil || got.After != nil || got.ApprovalID != 0 || got.ConfigVersion != 0 || got.Source.IP.IsValid() {
		t.Fatalf("optional fields %+v", got)
	}
}

func TestEntriesCannotBeChangedOrRemoved(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := audit.NewStore(pool)
	appendAll(t, store, entry("ayla", "publish", rbac.ShopOffers, "o-1"))
	ctx := context.Background()
	for _, statement := range []string{
		`update admin_audit_entries set reason = 'edited'`,
		`delete from admin_audit_entries`,
		`truncate admin_audit_entries`,
	} {
		_, err := pool.Exec(ctx, statement)
		if err == nil || !strings.Contains(err.Error(), "append-only") {
			t.Errorf("%s: %v", statement, err)
		}
	}
	if count := rowCount(t, pool); count != 1 {
		t.Fatalf("rows after the attempts %d", count)
	}
}

func TestInvalidEntriesAreRejected(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := audit.NewStore(pool)
	valid := entry("ayla", "edit", rbac.CosmeticItems, "frame-1")
	tests := map[string]func(*audit.Entry){
		"no actor":       func(e *audit.Entry) { e.Actor.ID = "" },
		"no roles":       func(e *audit.Entry) { e.Actor.Roles = nil },
		"unknown role":   func(e *audit.Entry) { e.Actor.Roles = []rbac.Role{rbac.LiveOps, "owner"} },
		"unknown object": func(e *audit.Entry) { e.Object.Type = "shop_offer" },
		"environment":    func(e *audit.Entry) { e.Environment = "dev" },
		"no action":      func(e *audit.Entry) { e.Action = "" },
		"no object type": func(e *audit.Entry) { e.Object.Type = "" },
		"no object id":   func(e *audit.Entry) { e.Object.ID = "" },
		"no session":     func(e *audit.Entry) { e.Source.SessionID = "" },
		"before json":    func(e *audit.Entry) { e.Before = json.RawMessage(`{`) },
		"after json":     func(e *audit.Entry) { e.After = json.RawMessage(`nope`) },
	}
	for name, change := range tests {
		e := valid
		change(&e)
		if _, err := store.Append(context.Background(), e, start); !errors.Is(err, audit.ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if count := rowCount(t, pool); count != 0 {
		t.Fatalf("rows after invalid entries %d", count)
	}
}

func TestAnApprovalMustPointAtAnExistingEntry(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := audit.NewStore(pool)
	approval := entry("ben", "approve", rbac.PromoCodes, "SPRING")
	approval.ApprovalID = 999
	if _, err := store.Append(context.Background(), approval, start); err == nil {
		t.Fatal("approval of a missing entry was stored")
	}
}

func TestSearchFiltersNewestFirst(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := audit.NewStore(pool)
	rollback := entry("ben", "rollback", rbac.Snapshots, "shop")
	rollback.Reason = "Broken PRICE in rotation"
	rollback.ConfigVersion = 7
	r := appendAll(t, store,
		entry("ayla", "edit", rbac.ShopOffers, "o-1"),
		entry("ben", "edit", rbac.ShopOffers, "o-1"),
		entry("ayla", "publish", rbac.ShopOffers, "o-2"),
		rollback,
		entry("ayla", "edit", rbac.PromoCodes, "o-1"),
	)
	tests := []struct {
		name   string
		filter audit.Filter
		want   []audit.Recorded
	}{
		{"all", audit.Filter{}, []audit.Recorded{r[4], r[3], r[2], r[1], r[0]}},
		{"actor", audit.Filter{ActorID: "ayla"}, []audit.Recorded{r[4], r[2], r[0]}},
		{"object", audit.Filter{Object: audit.Object{Type: rbac.ShopOffers, ID: "o-1"}}, []audit.Recorded{r[1], r[0]}},
		{"object type", audit.Filter{Object: audit.Object{Type: rbac.ShopOffers}}, []audit.Recorded{r[2], r[1], r[0]}},
		{"action", audit.Filter{Action: "edit"}, []audit.Recorded{r[4], r[1], r[0]}},
		{"time range", audit.Filter{From: start.Add(time.Minute), To: start.Add(3 * time.Minute)}, []audit.Recorded{r[2], r[1]}},
		{"reason text ignores case", audit.Filter{ReasonContains: "price"}, []audit.Recorded{r[3]}},
		{"config version", audit.Filter{ConfigVersion: 7}, []audit.Recorded{r[3]}},
		{"combined", audit.Filter{ActorID: "ayla", Action: "edit", Object: audit.Object{ID: "o-1"}}, []audit.Recorded{r[4], r[0]}},
		{"nothing", audit.Filter{ActorID: "cem"}, []audit.Recorded{}},
	}
	for _, test := range tests {
		got, err := store.Search(context.Background(), superAdmin, test.filter)
		if err != nil {
			t.Fatalf("%s: %v", test.name, err)
		}
		if !reflect.DeepEqual(ids(got), ids(test.want)) {
			t.Errorf("%s: ids %v, want %v", test.name, ids(got), ids(test.want))
		}
	}
}

func TestSearchPagesByIDWithABoundedLimit(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := audit.NewStore(pool)
	entries := make([]audit.Entry, audit.MaxLimit+5)
	for i := range entries {
		entries[i] = entry("ayla", "view", rbac.Players, "p-1")
	}
	r := appendAll(t, store, entries...)
	ctx := context.Background()
	first, err := store.Search(ctx, superAdmin, audit.Filter{Limit: 2})
	if err != nil || !reflect.DeepEqual(ids(first), []int64{r[len(r)-1].ID, r[len(r)-2].ID}) {
		t.Fatalf("first page %v %v", ids(first), err)
	}
	next, err := store.Search(ctx, superAdmin, audit.Filter{Limit: 2, BeforeID: first[1].ID})
	if err != nil || !reflect.DeepEqual(ids(next), []int64{r[len(r)-3].ID, r[len(r)-4].ID}) {
		t.Fatalf("second page %v %v", ids(next), err)
	}
	if page, err := store.Search(ctx, superAdmin, audit.Filter{}); err != nil || len(page) != audit.DefaultLimit {
		t.Fatalf("default limit %d %v", len(page), err)
	}
	if page, err := store.Search(ctx, superAdmin, audit.Filter{Limit: audit.MaxLimit * 10}); err != nil || len(page) != audit.MaxLimit {
		t.Fatalf("capped limit %d %v", len(page), err)
	}
}

func TestEachRoleSeesOnlyEntriesOnObjectsItCanView(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := audit.NewStore(pool)
	r := appendAll(t, store,
		entry("ayla", "edit", rbac.CosmeticItems, "frame-1"),
		entry("ayla", "view", rbac.Players, "p-1"),
		entry("ayla", "edit", rbac.AntiCheatConfig, "ReviewScore"),
		entry("ayla", "create", rbac.Assets, "a-1"),
		entry("ayla", "act_on_queue", rbac.Queues, "report-1"),
		entry("ayla", "edit", rbac.RightsRequests, "rr-1"),
		entry("ayla", "edit", rbac.StoreProducts, "coins-500"),
	)
	tests := []struct {
		name  string
		roles []rbac.Role
		want  []audit.Recorded
	}{
		{"super admin sees everything", []rbac.Role{rbac.SuperAdmin}, []audit.Recorded{r[6], r[5], r[4], r[3], r[2], r[1], r[0]}},
		{"content", []rbac.Role{rbac.Content}, []audit.Recorded{r[3], r[0]}},
		{"live ops", []rbac.Role{rbac.LiveOps}, []audit.Recorded{r[6], r[3], r[0]}},
		{"support sees no player entries", []rbac.Role{rbac.Support}, []audit.Recorded{r[6], r[0]}},
		{"roles add up", []rbac.Role{rbac.Content, rbac.Support}, []audit.Recorded{r[6], r[3], r[0]}},
	}
	for _, test := range tests {
		got, err := store.Search(context.Background(), test.roles, audit.Filter{})
		if err != nil {
			t.Fatalf("%s: %v", test.name, err)
		}
		if !reflect.DeepEqual(ids(got), ids(test.want)) {
			t.Errorf("%s: ids %v, want %v", test.name, ids(got), ids(test.want))
		}
	}
	hidden, err := store.Search(context.Background(), []rbac.Role{rbac.Content}, audit.Filter{Object: audit.Object{Type: rbac.Players}})
	if err != nil || len(hidden) != 0 {
		t.Fatalf("content filtering for player entries %v %v", ids(hidden), err)
	}
}

func TestRolesWithoutTheAuditLogCannotSearch(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := audit.NewStore(pool)
	appendAll(t, store, entry("ayla", "view", rbac.Dashboards, "home"))
	for _, roles := range [][]rbac.Role{nil, {rbac.Analyst}, {"owner"}} {
		if got, err := store.Search(context.Background(), roles, audit.Filter{}); !errors.Is(err, audit.ErrForbidden) || got != nil {
			t.Errorf("%v: %v %v", roles, ids(got), err)
		}
	}
}

func rowCount(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), `select count(*) from admin_audit_entries`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func assertSameJSON(t *testing.T, got, want json.RawMessage) {
	t.Helper()
	var g, w any
	if err := json.Unmarshal(got, &g); err != nil {
		t.Fatalf("stored json %s: %v", got, err)
	}
	if err := json.Unmarshal(want, &w); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g, w) {
		t.Fatalf("json %s, want %s", got, want)
	}
}

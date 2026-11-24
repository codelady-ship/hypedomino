package adminapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/account"
	"github.com/Alievs-corp/hypedomino/server/internal/adminapi"
	"github.com/Alievs-corp/hypedomino/server/internal/appeal"
	"github.com/Alievs-corp/hypedomino/server/internal/penalties"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/sanction"
)

func (a *api) sanctioned(accountID string, kind sanction.Kind, at time.Time) int64 {
	a.t.Helper()
	ctx := context.Background()
	var ends *time.Time
	if kind != sanction.PermanentBan && kind != sanction.ForcedRename {
		until := at.Add(72 * time.Hour)
		ends = &until
	}
	imposed, err := sanction.Impose(ctx, a.pool, accountID, kind, sanction.Cheating, at, ends)
	if err != nil {
		a.t.Fatal(err)
	}
	var effect string
	var args []any
	switch kind {
	case sanction.TemporaryBan:
		effect, args = `update accounts set banned_until = $2 where id = $1`, []any{accountID, *ends}
	case sanction.PermanentBan:
		effect, args = `update accounts set state = 'banned' where id = $1`, []any{accountID}
	case sanction.ForcedRename:
		effect, args = `update profiles set rename_required = true where account_id = $1`, []any{accountID}
	case sanction.PenaltyBan:
		record, _ := json.Marshal(penalties.Record{Ranked: penalties.Ladder{BanUntil: *ends, BanSanction: imposed.ID}})
		effect, args = `insert into penalty_records (account_id, record, updated_at) values ($1, $2, $3)`, []any{accountID, record, at}
	default:
		return imposed.ID
	}
	if _, err := a.pool.Exec(ctx, effect, args...); err != nil {
		a.t.Fatal(err)
	}
	return imposed.ID
}

func (a *api) appealed(accountID string, sanctionID int64, at time.Time) int64 {
	a.t.Helper()
	filed, err := appeal.Submit(context.Background(), a.pool, accountID, sanctionID, "it was not me", at)
	if err != nil {
		a.t.Fatal(err)
	}
	return filed.ID
}

func listedAppeals(a *api, token, query string) []map[string]any {
	a.t.Helper()
	status, body := a.call("GET", "/admin/appeals"+query, token, nil)
	if status != http.StatusOK {
		a.t.Fatalf("appeals %d %v", status, body)
	}
	var listed []map[string]any
	for _, entry := range body["appeals"].([]any) {
		listed = append(listed, entry.(map[string]any))
	}
	return listed
}

func appealIDs(listed []map[string]any) string {
	ids := make([]string, len(listed))
	for index, entry := range listed {
		ids[index] = strconv.FormatInt(int64(entry["id"].(float64)), 10)
	}
	return strings.Join(ids, ",")
}

func TestTheAppealQueueListsOpenAppealsOldestFirstWithTheirDueTime(t *testing.T) {
	a := newAPI(t)
	a.now = time.Now().UTC().Truncate(time.Second)
	a.create("orxan@example.com", rbac.Support)
	a.create("leyla@example.com", rbac.LiveOps)
	support, content, liveOps := a.signIn("orxan"), a.signIn("ayla"), a.signIn("leyla")
	late := a.player("g-1", "late@mail.example", "Late")
	fresh := a.player("g-2", "fresh@mail.example", "Fresh")
	decided := a.player("g-3", "decided@mail.example", "Decided")
	lateID := a.appealed(late, a.sanctioned(late, sanction.RankedBan, a.now.Add(-80*time.Hour)), a.now.Add(-73*time.Hour))
	freshID := a.appealed(fresh, a.sanctioned(fresh, sanction.TemporaryBan, a.now.Add(-2*time.Hour)), a.now.Add(-time.Hour))
	decidedID := a.appealed(decided, a.sanctioned(decided, sanction.RankedBan, a.now.Add(-2*time.Hour)), a.now.Add(-time.Hour))
	status, body := a.call("POST", "/admin/appeals/"+strconv.FormatInt(decidedID, 10)+"/uphold", support, map[string]any{"reason": "evidence holds"})
	expect(t, status, body, http.StatusOK, "")

	status, body = a.call("GET", "/admin/appeals", content, nil)
	expect(t, status, body, http.StatusForbidden, "forbidden")
	listed := listedAppeals(a, support, "")
	if appealIDs(listed) != strconv.FormatInt(lateID, 10)+","+strconv.FormatInt(freshID, 10) {
		t.Fatalf("queue %v", listed)
	}
	first := listed[0]
	sanctionView := first["sanction"].(map[string]any)
	if first["account_id"] != late || first["nickname"] != "Late" || first["text"] != "it was not me" || first["state"] != appeal.StateOpen ||
		sanctionView["kind"] != string(sanction.RankedBan) || sanctionView["category"] != string(sanction.Cheating) || first["overdue"] != true {
		t.Fatalf("entry %v", first)
	}
	if due, _ := time.Parse(time.RFC3339Nano, listed[1]["due_at"].(string)); !due.Equal(a.now.Add(71*time.Hour)) || listed[1]["overdue"] != false {
		t.Fatalf("default appeal sla %v", listed[1])
	}
	if page := listedAppeals(a, support, "?limit=1&after="+strconv.FormatInt(lateID, 10)); appealIDs(page) != strconv.FormatInt(freshID, 10) {
		t.Fatalf("second page %v", page)
	}
	if found := listedAppeals(a, support, "?account="+fresh[:8]); appealIDs(found) != strconv.FormatInt(freshID, 10) {
		t.Fatalf("by account %v", found)
	}
	if ranked := listedAppeals(a, support, "?kind=ranked_ban"); appealIDs(ranked) != strconv.FormatInt(lateID, 10) {
		t.Fatalf("by kind %v", ranked)
	}
	status, body = a.call("GET", "/admin/appeals?after=nope", support, nil)
	expect(t, status, body, http.StatusBadRequest, "invalid")

	publishLiveops(a, liveOps, map[string]int{"AppealSlaHours": 1})
	if listed := listedAppeals(a, support, "?account="+fresh); listed[0]["overdue"] != false {
		t.Fatalf("at the due time %v", listed)
	}
	a.now = a.now.Add(time.Second)
	if listed := listedAppeals(a, support, "?account="+fresh); listed[0]["overdue"] != true {
		t.Fatalf("after the published sla %v", listed)
	}
}

func TestTheAppealQueueShowsTheEarlierDecisionsOnTheAccount(t *testing.T) {
	a := newAPI(t)
	a.now = time.Now().UTC()
	a.create("orxan@example.com", rbac.Support)
	support := a.signIn("orxan")
	player := a.player("g-1", "player@mail.example", "Player")
	first := a.appealed(player, a.sanctioned(player, sanction.RankedBan, a.now.Add(-3*time.Hour)), a.now.Add(-2*time.Hour))
	status, body := a.call("POST", "/admin/appeals/"+strconv.FormatInt(first, 10)+"/uphold", support, map[string]any{"reason": "evidence holds"})
	expect(t, status, body, http.StatusOK, "")
	a.appealed(player, a.sanctioned(player, sanction.TemporaryBan, a.now.Add(-time.Hour)), a.now.Add(-30*time.Minute))

	listed := listedAppeals(a, support, "")
	decisions := listed[0]["decisions"].([]any)
	if len(listed) != 1 || len(decisions) != 1 {
		t.Fatalf("queue %v", listed)
	}
	decision := decisions[0].(map[string]any)
	if decision["queue"] != "appeals" || decision["outcome"] != appeal.StateUpheld || decision["reasons"].([]any)[0] != string(sanction.RankedBan) {
		t.Fatalf("decision %v", decision)
	}
}

func TestUpholdingKeepsTheSanctionAndLiftingReversesIt(t *testing.T) {
	cases := []struct {
		kind     sanction.Kind
		reversed func(a *api, accountID string) bool
	}{
		{sanction.TemporaryBan, func(a *api, accountID string) bool {
			var until *time.Time
			err := a.pool.QueryRow(context.Background(), `select banned_until from accounts where id = $1`, accountID).Scan(&until)
			return err == nil && until == nil
		}},
		{sanction.PermanentBan, func(a *api, accountID string) bool {
			var state string
			err := a.pool.QueryRow(context.Background(), `select state from accounts where id = $1`, accountID).Scan(&state)
			return err == nil && state == account.StateActive
		}},
		{sanction.RankedBan, func(a *api, accountID string) bool {
			return sanction.RankedBanCheck(a.pool)(context.Background(), accountID, a.now) == nil
		}},
		{sanction.PenaltyBan, func(a *api, accountID string) bool {
			record, err := penalties.NewStore(a.pool, nil).Record(context.Background(), accountID)
			_, banned := record.Banned(a.now)
			return err == nil && !banned && record.Ranked.BanSanction == 0
		}},
		{sanction.ForcedRename, func(a *api, accountID string) bool {
			var required bool
			err := a.pool.QueryRow(context.Background(), `select rename_required from profiles where account_id = $1`, accountID).Scan(&required)
			return err == nil && !required
		}},
	}
	for _, c := range cases {
		t.Run(string(c.kind), func(t *testing.T) {
			a := newAPI(t)
			a.now = time.Now().UTC()
			a.create("orxan@example.com", rbac.Support)
			support, root := a.signIn("orxan"), a.signIn("root")
			upheld := a.player("g-upheld", "upheld@mail.example", "Upheld")
			lifted := a.player("g-"+string(c.kind), "lifted@mail.example", "Lifted")
			upheldSanction := a.sanctioned(upheld, c.kind, a.now.Add(-time.Hour))
			liftedSanction := a.sanctioned(lifted, c.kind, a.now.Add(-time.Hour))
			upheldID := strconv.FormatInt(a.appealed(upheld, upheldSanction, a.now.Add(-time.Minute)), 10)
			liftedID := strconv.FormatInt(a.appealed(lifted, liftedSanction, a.now.Add(-time.Minute)), 10)

			status, body := a.call("POST", "/admin/appeals/"+liftedID+"/lift", support, map[string]any{})
			expect(t, status, body, http.StatusBadRequest, "reason_required")
			status, body = a.call("POST", "/admin/appeals/"+upheldID+"/uphold", support, map[string]any{"reason": "evidence holds", "ref": "APL-1"})
			if status != http.StatusOK || body["state"] != appeal.StateUpheld || body["reversed"] != false || body["kind"] != string(c.kind) {
				t.Fatalf("uphold %d %v", status, body)
			}
			if c.reversed(a, upheld) {
				t.Fatal("upholding reversed the sanction")
			}
			status, body = a.call("POST", "/admin/appeals/"+liftedID+"/lift", support, map[string]any{"reason": "false positive"})
			if status != http.StatusOK || body["state"] != appeal.StateLifted || body["reversed"] != true || body["sanction_id"] != float64(liftedSanction) {
				t.Fatalf("lift %d %v", status, body)
			}
			if !c.reversed(a, lifted) {
				t.Fatal("lifting kept the sanction")
			}
			if current, running, err := sanction.Current(context.Background(), a.pool, lifted, a.now, c.kind); err != nil || running {
				t.Fatalf("lifted sanction still runs %+v %v", current, err)
			}
			status, body = a.call("POST", "/admin/appeals/"+liftedID+"/uphold", support, map[string]any{"reason": "again"})
			expect(t, status, body, http.StatusNotFound, "no_open_appeal")

			upholds := a.entries(root, "object_type=queues&action="+adminapi.UpholdAppealAction)
			lifts := a.entries(root, "object_type=queues&action="+adminapi.LiftAppealAction)
			if len(upholds) != 1 || len(lifts) != 1 || upholds[0]["ref"] != "APL-1" || lifts[0]["reason"] != "false positive" ||
				lifts[0]["before"].(map[string]any)["state"] != appeal.StateOpen || lifts[0]["after"].(map[string]any)["reversed"] != true {
				t.Fatalf("audit %v %v", upholds, lifts)
			}
		})
	}
}

func TestLiftingAnAppealOfAnEndedSanctionChangesNothingElse(t *testing.T) {
	a := newAPI(t)
	a.now = time.Now().UTC()
	a.create("orxan@example.com", rbac.Support)
	support := a.signIn("orxan")
	player := a.player("g-1", "player@mail.example", "Player")
	first := a.sanctioned(player, sanction.TemporaryBan, a.now.Add(-2*time.Hour))
	id := a.appealed(player, first, a.now.Add(-time.Hour))
	renewed := a.sanctioned(player, sanction.TemporaryBan, a.now.Add(-time.Minute))
	status, body := a.call("POST", "/admin/appeals/"+strconv.FormatInt(id, 10)+"/lift", support, map[string]any{"reason": "first one was wrong"})
	if status != http.StatusOK || body["reversed"] != false {
		t.Fatalf("lift %d %v", status, body)
	}
	if current, running, err := sanction.Current(context.Background(), a.pool, player, a.now, sanction.TemporaryBan); err != nil || !running || current.ID != renewed {
		t.Fatalf("the renewed ban %+v %v %v", current, running, err)
	}
	if err := a.playerSignIn("g-1"); err == nil || !strings.Contains(err.Error(), account.ErrBanned.Error()) {
		t.Fatalf("sign-in during the renewed ban %v", err)
	}
}

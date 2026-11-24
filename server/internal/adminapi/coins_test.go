package adminapi_test

import (
	"context"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

func (a *api) balance(accountID string) int64 {
	a.t.Helper()
	balance, err := wallet.NewStore(a.pool).Balance(context.Background(), accountID, wallet.Coin)
	if err != nil {
		a.t.Fatal(err)
	}
	return balance
}

func adjustment(amount int64, requestID, reason string) map[string]any {
	return map[string]any{"amount": amount, "request_id": requestID, "reason": reason}
}

func TestSupportAdjustsCoinsOnceWithAReasonAndNeverBelowZero(t *testing.T) {
	a := newAPI(t)
	a.create("orxan@example.com", rbac.Support)
	support, content, root := a.signIn("orxan"), a.signIn("ayla"), a.signIn("root")
	id := a.player("g-1", "kamal@mail.example", "Kamal")
	path := "/admin/players/" + id + "/coins"

	for name, request := range map[string]map[string]any{
		"no amount":     adjustment(0, "r-0", "ticket"),
		"no request id": adjustment(100, "", "ticket"),
		"bad id":        adjustment(100, "r 1", "ticket"),
	} {
		status, body := a.call("POST", path, support, request)
		if status != http.StatusBadRequest || body["error"] != "invalid" {
			t.Fatalf("%s: %d %v", name, status, body)
		}
	}
	status, body := a.call("POST", path, support, adjustment(500, "r-1", " "))
	expect(t, status, body, http.StatusBadRequest, "reason_required")
	status, body = a.call("POST", path, content, adjustment(500, "r-1", "ticket"))
	expect(t, status, body, http.StatusForbidden, "forbidden")
	status, body = a.call("POST", "/admin/players/00000000-0000-0000-0000-000000000000/coins", support, adjustment(500, "r-1", "ticket"))
	expect(t, status, body, http.StatusNotFound, "not_found")

	status, body = a.call("POST", path, support, map[string]any{"amount": 500, "request_id": "r-1", "reason": "lost match reward", "ref": "SUP-7"})
	if status != http.StatusOK || body["balance"] != float64(500) || body["repeated"] != false {
		t.Fatalf("adjust %d %v", status, body)
	}
	status, body = a.call("POST", path, support, adjustment(500, "r-1", "lost match reward"))
	if status != http.StatusOK || body["balance"] != float64(500) || body["repeated"] != true {
		t.Fatalf("repeat %d %v", status, body)
	}
	status, body = a.call("POST", path, support, adjustment(400, "r-1", "lost match reward"))
	expect(t, status, body, http.StatusBadRequest, "invalid")
	status, body = a.call("POST", path, support, adjustment(-600, "r-2", "abuse"))
	expect(t, status, body, http.StatusConflict, "insufficient")
	status, body = a.call("POST", path, support, adjustment(-200, "r-3", "abuse"))
	if status != http.StatusOK || body["balance"] != float64(300) {
		t.Fatalf("debit %d %v", status, body)
	}
	if got := a.balance(id); got != 300 {
		t.Fatalf("balance %d", got)
	}

	a.now = start.Add(11 * time.Minute)
	status, body = a.call("POST", path, support, adjustment(100, "r-4", "late"))
	expect(t, status, body, http.StatusForbidden, "step_up_required")

	entries := a.entries(root, "object_type=players&object_id="+id+"&action=adjust_coins")
	if len(entries) != 2 {
		t.Fatalf("entries %v", entries)
	}
	debit, credit := entries[0], entries[1]
	if credit["reason"] != "lost match reward" || credit["ref"] != "SUP-7" || credit["before"].(map[string]any)["balance"] != float64(0) ||
		credit["after"].(map[string]any)["balance"] != float64(500) || debit["after"].(map[string]any)["amount"] != float64(-200) {
		t.Fatalf("entries %v %v", credit, debit)
	}
	var reason, sourceType string
	if err := a.pool.QueryRow(context.Background(), `select reason, source_type from ledger_entries where account_id = $1 order by id limit 1`, id).Scan(&reason, &sourceType); err != nil ||
		reason != "admin_adjust" || sourceType != "admin" {
		t.Fatalf("ledger %s %s %v", reason, sourceType, err)
	}
}

func TestALargeAdjustmentWaitsForASecondAdmin(t *testing.T) {
	a := newAPI(t)
	a.create("orxan@example.com", rbac.Support)
	a.create("leyla@example.com", rbac.Support)
	proposer, approver, root := a.signIn("orxan"), a.signIn("leyla"), a.signIn("root")
	id := a.player("g-1", "kamal@mail.example", "Kamal")
	path := "/admin/players/" + id + "/coins"

	status, body := a.call("POST", path, root, adjustment(2500, "r-1", "compensation"))
	expect(t, status, body, http.StatusForbidden, "forbidden")
	status, body = a.call("POST", path, proposer, adjustment(2000, "r-0", "compensation"))
	if status != http.StatusOK || body["balance"] != float64(2000) {
		t.Fatalf("at the threshold %d %v", status, body)
	}
	status, body = a.call("POST", path, proposer, adjustment(2500, "r-1", "outage compensation"))
	if status != http.StatusAccepted || body["kind"] != "player_grant" || body["state"] != "pending" || body["object_id"] != id {
		t.Fatalf("propose %d %v", status, body)
	}
	granted := strconv.FormatFloat(body["id"].(float64), 'f', 0, 64)
	status, body = a.call("POST", path, proposer, adjustment(-4600, "r-2", "fraud"))
	if status != http.StatusAccepted {
		t.Fatalf("propose debit %d %v", status, body)
	}
	debit := strconv.FormatFloat(body["id"].(float64), 'f', 0, 64)
	if got := a.balance(id); got != 2000 {
		t.Fatalf("balance before approval %d", got)
	}

	status, body = a.call("POST", "/admin/approvals/"+granted+"/approve", proposer, map[string]any{"reason": "checked"})
	expect(t, status, body, http.StatusForbidden, "self_decision")
	status, body = a.call("POST", "/admin/approvals/"+debit+"/approve", approver, map[string]any{"reason": "checked"})
	expect(t, status, body, http.StatusConflict, "insufficient")
	status, body = a.call("POST", "/admin/approvals/"+granted+"/approve", approver, map[string]any{"reason": "checked", "ref": "INC-4"})
	if status != http.StatusOK || body["state"] != "approved" {
		t.Fatalf("approve %d %v", status, body)
	}
	if got := a.balance(id); got != 4500 {
		t.Fatalf("balance after approval %d", got)
	}
	status, body = a.call("GET", "/admin/approvals", approver, nil)
	if pending := body["proposals"].([]any); status != http.StatusOK || len(pending) != 1 || pending[0].(map[string]any)["state"] != "pending" {
		t.Fatalf("pending %d %v", status, body)
	}

	entries := a.entries(root, "object_type=players&object_id="+id)
	if len(entries) != 5 {
		t.Fatalf("entries %v", entries)
	}
	applied, decision := entries[0], entries[1]
	if applied["action"] != "adjust_coins" || applied["approval_id"] != decision["id"] || applied["reason"] != "outage compensation" ||
		applied["after"].(map[string]any)["balance"] != float64(4500) || decision["action"] != "approve" || decision["ref"] != "INC-4" ||
		entries[2]["action"] != "submit" || entries[3]["action"] != "submit" || entries[4]["action"] != "adjust_coins" {
		t.Fatalf("entries %v", entries)
	}
}

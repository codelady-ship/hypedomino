package adminapi_test

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
)

func (a *api) publishExampleCatalog() {
	a.t.Helper()
	raw, err := os.ReadFile("../../../docs/cosmetics/examples/catalog.json")
	if err != nil {
		a.t.Fatal(err)
	}
	if _, err := snapshot.Publish(context.Background(), a.pool, cosmetics.Domain, raw); err != nil {
		a.t.Fatal(err)
	}
}

func (a *api) owned(accountID string) map[string]string {
	a.t.Helper()
	rows, err := a.pool.Query(context.Background(), `select item_id, source_type || ':' || source_ref from inventory_items where account_id = $1 and revoked_at is null`, accountID)
	if err != nil {
		a.t.Fatal(err)
	}
	defer rows.Close()
	owned := map[string]string{}
	for rows.Next() {
		var item, source string
		if err := rows.Scan(&item, &source); err != nil {
			a.t.Fatal(err)
		}
		owned[item] = source
	}
	return owned
}

func grant(bundle map[string]any, sourceType, sourceRef, requestID, reason string) map[string]any {
	return map[string]any{"bundle": bundle, "source": map[string]any{"type": sourceType, "ref": sourceRef}, "request_id": requestID, "reason": reason}
}

func springCup(coins int, items ...string) map[string]any {
	return map[string]any{"id": "bundle.spring_cup", "revision": 1, "coins": coins, "cosmetics": items}
}

func TestSupportGrantsAMissedRewardUnderASourceItsItemsList(t *testing.T) {
	a := newAPI(t)
	a.publishExampleCatalog()
	a.create("orxan@example.com", rbac.Support)
	support, content, root := a.signIn("orxan"), a.signIn("ayla"), a.signIn("root")
	id := a.player("g-1", "kamal@mail.example", "Kamal")
	path := "/admin/players/" + id + "/grant"

	for name, request := range map[string]map[string]any{
		"empty bundle":   grant(map[string]any{"id": "bundle.spring_cup", "revision": 1}, "event", "event.spring_cup", "r-0", "ticket"),
		"no request id":  grant(springCup(300), "event", "event.spring_cup", "", "ticket"),
		"default source": grant(springCup(300), "default", "event.spring_cup", "r-0", "ticket"),
		"unknown type":   grant(springCup(300), "admin", "event.spring_cup", "r-0", "ticket"),
		"no ref":         grant(springCup(300), "event", "", "r-0", "ticket"),
		"not listed":     grant(springCup(300, "avatar.comet_fox", "emoji.laugh_spin"), "event", "event.spring_cup", "r-0", "ticket"),
		"unknown item":   grant(springCup(300, "avatar.missing"), "event", "event.spring_cup", "r-0", "ticket"),
		"unknown medal":  grant(map[string]any{"id": "bundle.spring_cup", "revision": 1, "medals": []string{"medal.first_win"}}, "event", "event.spring_cup", "r-0", "ticket"),
	} {
		status, body := a.call("POST", path, support, request)
		if status != http.StatusBadRequest || body["error"] != "invalid" {
			t.Fatalf("%s: %d %v", name, status, body)
		}
	}
	if got := a.owned(id); len(got) != 0 || a.balance(id) != 0 {
		t.Fatalf("refused grants stored %v %d", got, a.balance(id))
	}
	status, body := a.call("POST", path, support, grant(springCup(300, "avatar.comet_fox"), "event", "event.spring_cup", "r-1", " "))
	expect(t, status, body, http.StatusBadRequest, "reason_required")
	status, body = a.call("POST", path, content, grant(springCup(300, "avatar.comet_fox"), "event", "event.spring_cup", "r-1", "ticket"))
	expect(t, status, body, http.StatusForbidden, "forbidden")

	request := grant(springCup(300, "avatar.comet_fox", "emoji.victory_dance"), "event", "event.spring_cup", "r-1", "missed spring cup reward")
	request["ref"] = "SUP-9"
	status, body = a.call("POST", path, support, request)
	report, _ := body["report"].(map[string]any)
	if status != http.StatusOK || body["repeated"] != false || report["coins"] != float64(300) || len(report["cosmetics"].([]any)) != 2 {
		t.Fatalf("grant %d %v", status, body)
	}
	status, body = a.call("POST", path, support, request)
	if status != http.StatusOK || body["repeated"] != true {
		t.Fatalf("repeat %d %v", status, body)
	}
	status, body = a.call("POST", path, support, grant(map[string]any{"id": "bundle.other", "revision": 1, "coins": 300}, "event", "event.spring_cup", "r-1", "ticket"))
	expect(t, status, body, http.StatusBadRequest, "invalid")
	status, body = a.call("POST", path, support, grant(springCup(0, "avatar.comet_fox"), "shop", "offer.starter", "r-2", "second copy"))
	report, _ = body["report"].(map[string]any)
	if status != http.StatusOK || report["duplicates"].(map[string]any)["avatar.comet_fox"] != float64(300) {
		t.Fatalf("duplicate %d %v", status, body)
	}

	owned := a.owned(id)
	if len(owned) != 2 || owned["avatar.comet_fox"] != "event:event.spring_cup" || owned["emoji.victory_dance"] != "event:event.spring_cup" {
		t.Fatalf("owned %v", owned)
	}
	if got := a.balance(id); got != 600 {
		t.Fatalf("balance %d", got)
	}
	var reason, sourceType string
	if err := a.pool.QueryRow(context.Background(), `select reason, source_type from ledger_entries where account_id = $1 order by id limit 1`, id).Scan(&reason, &sourceType); err != nil ||
		reason != "admin_adjust" || sourceType != "admin" {
		t.Fatalf("ledger %s %s %v", reason, sourceType, err)
	}
	entries := a.entries(root, "object_type=players&object_id="+id+"&action=grant_reward")
	if len(entries) != 2 {
		t.Fatalf("entries %v", entries)
	}
	first := entries[1]
	if first["reason"] != "missed spring cup reward" || first["ref"] != "SUP-9" || first["after"].(map[string]any)["bundle_id"] != "bundle.spring_cup" {
		t.Fatalf("entry %v", first)
	}
}

func TestAMythicOrLargeGrantWaitsForASecondAdmin(t *testing.T) {
	a := newAPI(t)
	a.publishExampleCatalog()
	a.create("orxan@example.com", rbac.Support)
	a.create("leyla@example.com", rbac.Support)
	proposer, approver, root := a.signIn("orxan"), a.signIn("leyla"), a.signIn("root")
	id := a.player("g-1", "kamal@mail.example", "Kamal")
	path := "/admin/players/" + id + "/grant"
	launch := map[string]any{"id": "bundle.launch_week", "revision": 1, "cosmetics": []string{"table.night_market"}}

	status, body := a.call("POST", path, root, grant(launch, "promo", "campaign.launch_week", "r-1", "lost code"))
	expect(t, status, body, http.StatusForbidden, "forbidden")
	status, body = a.call("POST", path, proposer, grant(launch, "promo", "campaign.launch_week", "r-1", "lost code"))
	if status != http.StatusAccepted || body["kind"] != "player_grant" || body["state"] != "pending" {
		t.Fatalf("mythic %d %v", status, body)
	}
	mythic := strconv.FormatFloat(body["id"].(float64), 'f', 0, 64)
	status, body = a.call("POST", path, proposer, grant(springCup(2500), "event", "event.spring_cup", "r-2", "outage"))
	if status != http.StatusAccepted {
		t.Fatalf("large %d %v", status, body)
	}
	status, body = a.call("POST", path, proposer, grant(springCup(2000), "event", "event.spring_cup", "r-3", "outage"))
	if status != http.StatusOK {
		t.Fatalf("at the threshold %d %v", status, body)
	}
	if got := a.owned(id); len(got) != 0 {
		t.Fatalf("owned before approval %v", got)
	}

	status, body = a.call("POST", "/admin/approvals/"+mythic+"/approve", proposer, map[string]any{"reason": "checked"})
	expect(t, status, body, http.StatusForbidden, "self_decision")
	status, body = a.call("POST", "/admin/approvals/"+mythic+"/approve", approver, map[string]any{"reason": "checked", "ref": "SUP-12"})
	if status != http.StatusOK || body["state"] != "approved" {
		t.Fatalf("approve %d %v", status, body)
	}
	if got := a.owned(id); got["table.night_market"] != "promo:campaign.launch_week" {
		t.Fatalf("owned after approval %v", got)
	}

	entries := a.entries(root, "object_type=players&object_id="+id+"&action=grant_reward")
	if len(entries) != 2 {
		t.Fatalf("entries %v", entries)
	}
	approved := entries[0]
	if approved["approval_id"] == nil || approved["reason"] != "lost code" || approved["actor_id"] == entries[1]["actor_id"] {
		t.Fatalf("approved entry %v", approved)
	}
}

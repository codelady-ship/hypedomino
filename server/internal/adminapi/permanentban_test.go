package adminapi_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/account"
	"github.com/Alievs-corp/hypedomino/server/internal/adminapi"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/risk"
)

func TestAPermanentBanFromTheQueueWaitsForASecondAdmin(t *testing.T) {
	a := newAPI(t)
	a.now = time.Now().UTC()
	a.create("orxan@example.com", rbac.Support)
	support, content, root := a.signIn("orxan"), a.signIn("ayla"), a.signIn("root")
	id := a.player("g-1", "kamal@mail.example", "Kamal")
	if _, err := a.pool.Exec(context.Background(), `insert into auth_sessions (account_id, device_id, refresh_hash, expires_at) values ($1, 'd-1', '\x01', $2)`,
		id, a.now.Add(24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	a.signal(id, risk.KindFakeReceipt, 20, a.now)
	_, body := a.call("GET", "/admin/reviews", support, nil)
	review := reviewID(t, body, 0)
	path := "/admin/reviews/" + review + "/permanent-ban"

	status, body := a.call("POST", path, support, map[string]any{})
	expect(t, status, body, http.StatusBadRequest, "reason_required")
	status, body = a.call("POST", path, content, map[string]any{"reason": "fake receipts", "category": "payment_fraud"})
	expect(t, status, body, http.StatusForbidden, "forbidden")
	status, body = a.call("POST", "/admin/reviews/999999/permanent-ban", support, map[string]any{"reason": "none", "category": "payment_fraud"})
	expect(t, status, body, http.StatusNotFound, "not_found")
	for _, category := range []any{nil, "", "speed_hack"} {
		status, body = a.call("POST", path, support, map[string]any{"reason": "fake receipts", "category": category})
		expect(t, status, body, http.StatusBadRequest, "invalid_category")
	}
	status, body = a.call("POST", path, support, map[string]any{"reason": "fake receipts", "ref": "REV-7", "category": "payment_fraud"})
	if status != http.StatusAccepted || body["kind"] != "permanent_ban" || body["state"] != "pending" || body["object_id"] != id {
		t.Fatalf("propose %d %v", status, body)
	}
	proposal := "/admin/approvals/" + strconv.FormatFloat(body["id"].(float64), 'f', 0, 64) + "/approve"
	if err := a.playerSignIn("g-1"); err != nil {
		t.Fatalf("sign-in before approval: %v", err)
	}
	if _, body := a.call("GET", "/admin/reviews", support, nil); len(body["reviews"].([]any)) != 1 {
		t.Fatalf("the proposal closed the review: %v", body)
	}

	status, body = a.call("POST", proposal, support, map[string]any{"reason": "checked"})
	expect(t, status, body, http.StatusForbidden, "self_decision")
	status, body = a.call("POST", proposal, root, map[string]any{"reason": "checked", "ref": "REV-7"})
	if status != http.StatusOK || body["state"] != "approved" {
		t.Fatalf("approve %d %v", status, body)
	}
	var banned *account.BannedError
	if err := a.playerSignIn("g-1"); !errors.As(err, &banned) || banned.Category != "payment_fraud" {
		t.Fatalf("sign-in after the ban: %v", err)
	}
	if got := a.liveSessions(id); got != 0 {
		t.Fatalf("live sessions %d", got)
	}
	if sanctions := a.sanctions(support, id); len(sanctions) != 1 || sanctions[0]["kind"] != "permanent_ban" || sanctions[0]["category"] != "payment_fraud" || sanctions[0]["ends_at"] != nil {
		t.Fatalf("sanctions %v", sanctions)
	}
	if _, body := a.call("GET", "/admin/reviews", support, nil); len(body["reviews"].([]any)) != 0 {
		t.Fatalf("the ban left the review open: %v", body)
	}
	status, body = a.call("POST", path, support, map[string]any{"reason": "again", "category": "cheating"})
	expect(t, status, body, http.StatusConflict, "decided")

	entries := a.entries(root, "object_type=players&action="+adminapi.PermanentBanAction)
	if len(entries) != 1 || entries[0]["reason"] != "fake receipts" || entries[0]["after"].(map[string]any)["sessions_revoked"] != float64(1) || entries[0]["after"].(map[string]any)["category"] != "payment_fraud" {
		t.Fatalf("ban entries %v", entries)
	}
	if confirmed := a.entries(root, "object_type=queues&action="+adminapi.ConfirmReviewAction); len(confirmed) != 1 {
		t.Fatalf("review entries %v", confirmed)
	}
}

func TestAnApprovedBanIsRefusedWhenTheReviewWasClearedMeanwhile(t *testing.T) {
	a := newAPI(t)
	a.now = time.Now().UTC()
	a.create("orxan@example.com", rbac.Support)
	support, root := a.signIn("orxan"), a.signIn("root")
	id := a.player("g-1", "kamal@mail.example", "Kamal")
	config := risk.DefaultConfig()
	a.signal(id, risk.KindReplay, config.ShadowbanScore, a.now)
	if _, err := risk.NewWave(a.pool, config, slog.New(slog.NewTextHandler(io.Discard, nil))).RunDue(context.Background(), a.now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	_, body := a.call("GET", "/admin/reviews", support, nil)
	review := "/admin/reviews/" + reviewID(t, body, 0)
	status, body := a.call("POST", review+"/permanent-ban", support, map[string]any{"reason": "replays", "category": "cheating"})
	if status != http.StatusAccepted {
		t.Fatalf("propose %d %v", status, body)
	}
	proposal := "/admin/approvals/" + strconv.FormatFloat(body["id"].(float64), 'f', 0, 64)
	if status, body := a.call("POST", review+"/clear", root, map[string]any{"reason": "false positive"}); status != http.StatusOK {
		t.Fatalf("clear %d %v", status, body)
	}

	status, body = a.call("POST", proposal+"/approve", root, map[string]any{"reason": "checked"})
	expect(t, status, body, http.StatusConflict, "decided")
	if err := a.playerSignIn("g-1"); err != nil {
		t.Fatalf("a refused approval banned the player: %v", err)
	}
	if status, body := a.call("POST", proposal+"/reject", root, map[string]any{"reason": "review cleared"}); status != http.StatusOK || body["state"] != "rejected" {
		t.Fatalf("reject %d %v", status, body)
	}
}

package approval_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/approval"
	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

var start = time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

func newStore(t *testing.T) *approval.Store {
	t.Helper()
	_, pool := dbtest.Fresh(t)
	return approval.NewStore(pool, approval.DefaultConfig())
}

func admin(id string, roles ...rbac.Role) audit.Actor {
	return audit.Actor{ID: id, Roles: roles}
}

var change = json.RawMessage(`{"coins": 5000}`)

func propose(t *testing.T, store *approval.Store, kind approval.Kind, proposer audit.Actor) approval.Proposal {
	t.Helper()
	p, err := store.Propose(context.Background(), kind, "acct-7", change, "Compensation for incident INC-12", proposer, start)
	if err != nil {
		t.Fatalf("propose %s: %v", kind, err)
	}
	return p
}

func TestEachHighRiskActionHasItsProposersAndApprovers(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()
	tests := []struct {
		kind      approval.Kind
		proposer  audit.Actor
		proposeOK bool
		approver  audit.Actor
		approveOK bool
	}{
		{approval.PlayerGrant, admin("s1", rbac.Support), true, admin("s2", rbac.Support), true},
		{approval.PlayerGrant, admin("s1", rbac.Support), true, admin("root", rbac.SuperAdmin), true},
		{approval.PlayerGrant, admin("s1", rbac.Support), true, admin("l1", rbac.LiveOps), false},
		{approval.PlayerGrant, admin("l1", rbac.LiveOps), false, audit.Actor{}, false},
		{approval.PromoCode, admin("l1", rbac.LiveOps), true, admin("l2", rbac.LiveOps), true},
		{approval.PromoCode, admin("l1", rbac.LiveOps), true, admin("s1", rbac.Support), false},
		{approval.PromoCode, admin("s1", rbac.Support), false, audit.Actor{}, false},
		{approval.PermanentBan, admin("s1", rbac.Support), true, admin("root", rbac.SuperAdmin), true},
		{approval.PermanentBan, admin("s1", rbac.Support), true, admin("c1", rbac.Content), false},
		{approval.LiveControl, admin("root", rbac.SuperAdmin), true, admin("l1", rbac.LiveOps), true},
		{approval.LiveControl, admin("l1", rbac.LiveOps), true, admin("root", rbac.SuperAdmin), true},
		{approval.LiveControl, admin("a1", rbac.Analyst), false, audit.Actor{}, false},
		{approval.SuperAdminGrant, admin("root", rbac.SuperAdmin), true, admin("root2", rbac.SuperAdmin), true},
		{approval.SuperAdminGrant, admin("root", rbac.SuperAdmin), true, admin("l1", rbac.LiveOps), false},
		{approval.SuperAdminGrant, admin("l1", rbac.LiveOps), false, audit.Actor{}, false},
		{approval.BreakGlass, admin("l1", rbac.LiveOps), true, admin("root", rbac.SuperAdmin), true},
		{approval.BreakGlass, admin("root", rbac.SuperAdmin), true, admin("l1", rbac.LiveOps), true},
		{approval.BreakGlass, admin("l1", rbac.LiveOps), true, admin("s1", rbac.Support), false},
		{approval.BreakGlass, admin("s1", rbac.Support), false, audit.Actor{}, false},
		{approval.BreakGlass, admin("c1", rbac.Content), false, audit.Actor{}, false},
	}
	for _, test := range tests {
		p, err := store.Propose(ctx, test.kind, "object-1", change, "reason", test.proposer, start)
		if !test.proposeOK {
			if !errors.Is(err, approval.ErrNotAllowed) {
				t.Errorf("%s proposed by %v: %v", test.kind, test.proposer.Roles, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s proposed by %v: %v", test.kind, test.proposer.Roles, err)
		}
		if approval.Decides(test.approver.Roles, test.kind) != test.approveOK {
			t.Errorf("%s decided by %v: Decides disagrees", test.kind, test.approver.Roles)
		}
		_, err = store.Approve(ctx, p.ID, test.approver, start.Add(time.Hour))
		if test.approveOK && err != nil || !test.approveOK && !errors.Is(err, approval.ErrNotAllowed) {
			t.Errorf("%s approved by %v: %v", test.kind, test.approver.Roles, err)
		}
	}
}

func TestAnApprovalRecordsTheSecondAdminAndLeavesThePendingList(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()
	p := propose(t, store, approval.PlayerGrant, admin("s1", rbac.Support))
	if p.Object != (audit.Object{Type: rbac.Players, ID: "acct-7"}) || !p.ExpiresAt.Equal(start.Add(24*time.Hour)) || p.State != approval.Pending {
		t.Fatalf("proposal %+v", p)
	}
	pending, err := store.Pending(ctx, start)
	if err != nil || len(pending) != 1 || pending[0].ID != p.ID || pending[0].Reason != p.Reason || pending[0].Proposer.ID != "s1" {
		t.Fatalf("pending %+v %v", pending, err)
	}
	var decoded map[string]int
	if err := json.Unmarshal(pending[0].Change, &decoded); err != nil || decoded["coins"] != 5000 {
		t.Fatalf("change %s %v", pending[0].Change, err)
	}
	approved, err := store.Approve(ctx, p.ID, admin("s2", rbac.Support), start.Add(time.Hour))
	if err != nil || approved.State != approval.Approved || approved.Decider.ID != "s2" || !approved.DecidedAt.Equal(start.Add(time.Hour)) {
		t.Fatalf("approved %+v %v", approved, err)
	}
	if pending, err := store.Pending(ctx, start.Add(time.Hour)); err != nil || len(pending) != 0 {
		t.Fatalf("still pending %+v %v", pending, err)
	}
	if got, err := store.Get(ctx, p.ID); err != nil || got.State != approval.Approved || got.Decider.ID != "s2" || got.Reason != p.Reason {
		t.Fatalf("get %+v %v", got, err)
	}
	if _, err := store.Get(ctx, p.ID+100); !errors.Is(err, approval.ErrNotFound) {
		t.Fatalf("get unknown: %v", err)
	}
}

func TestTheProposerCannotDecideTheirOwnProposal(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()
	root := admin("root", rbac.SuperAdmin, rbac.LiveOps)
	p := propose(t, store, approval.LiveControl, root)
	if _, err := store.Approve(ctx, p.ID, root, start); !errors.Is(err, approval.ErrSelfDecision) {
		t.Fatalf("self approval: %v", err)
	}
	if _, err := store.Reject(ctx, p.ID, root, start); !errors.Is(err, approval.ErrSelfDecision) {
		t.Fatalf("self rejection: %v", err)
	}
}

func TestAProposalExpiresAfterTheTtl(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()
	late := propose(t, store, approval.PromoCode, admin("l1", rbac.LiveOps))
	inTime := propose(t, store, approval.PromoCode, admin("l1", rbac.LiveOps))
	deadline := start.Add(24 * time.Hour)
	if pending, err := store.Pending(ctx, deadline); err != nil || len(pending) != 0 {
		t.Fatalf("expired proposals listed %+v %v", pending, err)
	}
	if _, err := store.Approve(ctx, late.ID, admin("l2", rbac.LiveOps), deadline); !errors.Is(err, approval.ErrExpired) {
		t.Fatalf("approved at expiry: %v", err)
	}
	if _, err := store.Approve(ctx, inTime.ID, admin("l2", rbac.LiveOps), deadline.Add(-time.Microsecond)); err != nil {
		t.Fatalf("approved just before expiry: %v", err)
	}
}

func TestAnUnconfirmedBreakGlassStaysListedAfterTheTtl(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()
	glass := propose(t, store, approval.BreakGlass, admin("l1", rbac.LiveOps))
	propose(t, store, approval.LiveControl, admin("l1", rbac.LiveOps))
	later := start.Add(72 * time.Hour)
	if pending, err := store.Pending(ctx, later); err != nil || len(pending) != 1 || pending[0].ID != glass.ID {
		t.Fatalf("pending after the ttl %+v %v", pending, err)
	}
	if unconfirmed, err := store.Unconfirmed(ctx); err != nil || len(unconfirmed) != 1 || unconfirmed[0].ID != glass.ID {
		t.Fatalf("unconfirmed %+v %v", unconfirmed, err)
	}
	if _, err := store.Approve(ctx, glass.ID, admin("l1", rbac.LiveOps), later); !errors.Is(err, approval.ErrSelfDecision) {
		t.Fatalf("self confirmation: %v", err)
	}
	if _, err := store.Approve(ctx, glass.ID, admin("root", rbac.SuperAdmin), later); err != nil {
		t.Fatalf("confirm after the ttl: %v", err)
	}
	if unconfirmed, err := store.Unconfirmed(ctx); err != nil || len(unconfirmed) != 0 {
		t.Fatalf("confirmed still listed %+v %v", unconfirmed, err)
	}
}

func TestADecisionIsFinal(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()
	p := propose(t, store, approval.PermanentBan, admin("s1", rbac.Support))
	rejected, err := store.Reject(ctx, p.ID, admin("s2", rbac.Support), start)
	if err != nil || rejected.State != approval.Rejected {
		t.Fatalf("rejected %+v %v", rejected, err)
	}
	if _, err := store.Approve(ctx, p.ID, admin("root", rbac.SuperAdmin), start); !errors.Is(err, approval.ErrDecided) {
		t.Fatalf("approved after a rejection: %v", err)
	}
	if _, err := store.Approve(ctx, 999, admin("root", rbac.SuperAdmin), start); !errors.Is(err, approval.ErrNotFound) {
		t.Fatalf("unknown proposal: %v", err)
	}
}

func TestParallelApprovalsDecideOnce(t *testing.T) {
	store := newStore(t)
	p := propose(t, store, approval.SuperAdminGrant, admin("root", rbac.SuperAdmin))
	var wg sync.WaitGroup
	results := make([]error, 32)
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, results[i] = store.Approve(context.Background(), p.ID, admin("root-"+string(rune('a'+i)), rbac.SuperAdmin), start)
		}()
	}
	wg.Wait()
	approvals := 0
	for _, err := range results {
		switch {
		case err == nil:
			approvals++
		case !errors.Is(err, approval.ErrDecided):
			t.Errorf("unexpected error %v", err)
		}
	}
	if approvals != 1 {
		t.Fatalf("%d approvals", approvals)
	}
}

func TestAProposalNeedsAKnownKindAReasonAndJsonChange(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()
	support := admin("s1", rbac.Support)
	tests := map[string]func() error{
		"unknown kind": func() error {
			_, err := store.Propose(ctx, "wipe_everything", "acct-7", change, "reason", support, start)
			return err
		},
		"blank reason": func() error {
			_, err := store.Propose(ctx, approval.PlayerGrant, "acct-7", change, "  ", support, start)
			return err
		},
		"no object": func() error {
			_, err := store.Propose(ctx, approval.PlayerGrant, "", change, "reason", support, start)
			return err
		},
		"change not JSON": func() error {
			_, err := store.Propose(ctx, approval.PlayerGrant, "acct-7", json.RawMessage(`coins=5000`), "reason", support, start)
			return err
		},
	}
	for name, run := range tests {
		if err := run(); !errors.Is(err, approval.ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestADecisionCommittedDuringAnApprovalWins(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := approval.NewStore(pool, approval.DefaultConfig())
	ctx := context.Background()
	p := propose(t, store, approval.PlayerGrant, admin("s1", rbac.Support))
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `select id from admin_approvals where id = $1 for update`, p.ID); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() {
		_, err := store.Approve(ctx, p.ID, admin("s2", rbac.Support), start)
		result <- err
	}()
	time.Sleep(200 * time.Millisecond)
	if _, err := tx.Exec(ctx, `update admin_approvals set state = 'rejected', decider_id = 'root', decider_roles = '{super_admin}', decided_at = $2 where id = $1`, p.ID, start); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-result; !errors.Is(err, approval.ErrDecided) {
		t.Fatalf("approval overwrote a concurrent decision: %v", err)
	}
}

package auth_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/account"
	"github.com/Alievs-corp/hypedomino/server/internal/auth"
	"github.com/Alievs-corp/hypedomino/server/internal/idtoken"
)

type fakeDeletions struct {
	requested map[string]time.Time
	banned    map[string]bool
}

func (d *fakeDeletions) Lookup(_ context.Context, identity idtoken.Identity) (account.Account, error) {
	id := identity.Provider.String() + "-" + identity.Subject
	switch at, pending := d.requested[id]; {
	case identity.Subject == "nobody":
		return account.Account{}, account.ErrUnknown
	case d.banned[id]:
		return account.Account{ID: id, State: account.StateBanned}, account.ErrBanned
	case pending:
		return account.Account{ID: id, State: account.StateDeleting, DeletionAt: &at}, account.ErrDeletionPending
	}
	return account.Account{ID: id, State: account.StateActive}, nil
}

func (d *fakeDeletions) RequestDeletion(_ context.Context, id string, now time.Time) (time.Time, error) {
	d.requested[id] = now
	return now, nil
}

func (d *fakeDeletions) CancelDeletion(_ context.Context, id string) error {
	delete(d.requested, id)
	return nil
}

func TestDeletionNeedsAFreshSignInRevokesSessionsAndCanBeCancelled(t *testing.T) {
	f := newFixture(t)
	f.verifier.identities["nobody"] = idtoken.Identity{Subject: "nobody"}
	f.verifier.identities["banned"] = idtoken.Identity{Subject: "banned"}
	f.verifier.identities["playing"] = idtoken.Identity{Subject: "playing"}
	f.verifier.identities["unreachable"] = idtoken.Identity{Subject: "unreachable"}
	deletions := &fakeDeletions{requested: map[string]time.Time{}, banned: map[string]bool{"google-banned": true}}
	now := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	var hidden []string
	handler := auth.New(f.verifier, f.accounts, f.tokens, f.connections, slog.New(slog.NewTextHandler(io.Discard, nil)))
	handler.EnableDeletion(auth.Deletion{
		Accounts: deletions,
		Grace:    30 * 24 * time.Hour,
		InMatch: func(_ context.Context, id string) (bool, error) {
			if id == "google-unreachable" {
				return false, errors.New("database down")
			}
			return id == "google-playing", nil
		},
		Hide: func(_ context.Context, id string) error {
			hidden = append(hidden, id)
			return nil
		},
		Now: func() time.Time { return now },
	})
	mux := http.NewServeMux()
	mux.Handle("POST /auth/", handler)
	f.server = httptest.NewServer(mux)
	t.Cleanup(f.server.Close)

	refusals := []struct {
		path, body string
		status     int
		reason     string
	}{
		{"/auth/deletion", `{"provider":"google","id_token":"forged","nonce":"n"}`, http.StatusUnauthorized, "token_rejected"},
		{"/auth/deletion", `{"provider":"mail","id_token":"good","nonce":"n"}`, http.StatusBadRequest, "unknown_provider"},
		{"/auth/deletion", `{"provider":"google","id_token":"nobody","nonce":"n"}`, http.StatusNotFound, "account_unknown"},
		{"/auth/deletion", `{"provider":"google","id_token":"banned","nonce":"n"}`, http.StatusForbidden, "banned"},
		{"/auth/deletion", `{"provider":"google","id_token":"playing","nonce":"n"}`, http.StatusConflict, "in_match"},
		{"/auth/deletion", `{"provider":"google","id_token":"unreachable","nonce":"n"}`, http.StatusInternalServerError, "internal"},
		{"/auth/deletion/cancel", `{"provider":"google","id_token":"good","nonce":"n"}`, http.StatusConflict, "no_deletion_pending"},
	}
	for _, c := range refusals {
		if status, body := f.post(c.path, c.body); status != c.status || body["error"] != c.reason {
			t.Fatalf("%s %s answered %d %v, want %d %s", c.path, c.body, status, body, c.status, c.reason)
		}
	}
	if len(deletions.requested) != 0 || len(f.tokens.loggedOut) != 0 {
		t.Fatalf("a refusal changed state: %v %v", deletions.requested, f.tokens.loggedOut)
	}

	deleteAt := "2026-12-10T09:00:00Z"
	if status, body := f.post("/auth/deletion", `{"provider":"google","id_token":"good","nonce":"n"}`); status != http.StatusAccepted || body["delete_at"] != deleteAt {
		t.Fatalf("request answered %d %v", status, body)
	}
	if strings.Join(f.tokens.loggedOut, " ") != "google-sub-1" || strings.Join(f.connections.revoked, " ") != "google-sub-1" || strings.Join(hidden, " ") != "google-sub-1" {
		t.Fatalf("logged out %v, revoked %v, hidden %v", f.tokens.loggedOut, f.connections.revoked, hidden)
	}
	now = now.Add(time.Hour)
	if status, body := f.post("/auth/deletion", `{"provider":"google","id_token":"good","nonce":"n"}`); status != http.StatusAccepted || body["delete_at"] != deleteAt {
		t.Fatalf("repeated request answered %d %v", status, body)
	}
	if status, body := f.post("/auth/deletion/cancel", `{"provider":"google","id_token":"good","nonce":"n"}`); status != http.StatusNoContent || len(deletions.requested) != 0 {
		t.Fatalf("cancel answered %d %v, pending %v", status, body, deletions.requested)
	}
}

func TestSignInDuringTheGraceShowsOnlyThePendingState(t *testing.T) {
	f := newFixture(t)
	requested := time.Date(2026, 11, 1, 8, 30, 0, 0, time.UTC)
	f.accounts.pending = map[string]time.Time{"google-sub-1": requested}
	handler := auth.New(f.verifier, f.accounts, f.tokens, f.connections, slog.New(slog.NewTextHandler(io.Discard, nil)))
	handler.EnableDeletion(auth.Deletion{Grace: 30 * 24 * time.Hour, Now: time.Now})
	mux := http.NewServeMux()
	mux.Handle("POST /auth/", handler)
	f.server = httptest.NewServer(mux)
	t.Cleanup(f.server.Close)
	status, body := f.post("/auth/signin", `{"provider":"google","id_token":"good","nonce":"n","device_id":"d"}`)
	if status != http.StatusConflict || body["error"] != "deletion_pending" || body["delete_at"] != "2026-12-01T08:30:00Z" || len(f.tokens.issued) != 0 {
		t.Fatalf("sign-in while pending answered %d %v, issued %v", status, body, f.tokens.issued)
	}
}

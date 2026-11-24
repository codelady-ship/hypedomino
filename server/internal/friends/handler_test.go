package friends_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/friends"
	"github.com/Alievs-corp/hypedomino/server/internal/profile"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

var now = time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)

type verifier struct{}

func (verifier) Verify(access string) (token.Claims, error) {
	if account, ok := strings.CutPrefix(access, "access:"); ok && account != "" {
		return token.Claims{Account: account}, nil
	}
	return token.Claims{}, token.ErrSignature
}

type world struct {
	t       *testing.T
	pool    *pgxpool.Pool
	handler http.Handler
}

func newWorld(t *testing.T, config friends.Config) *world {
	t.Helper()
	_, pool := dbtest.Fresh(t)
	handler := friends.NewHandler(verifier{}, friends.NewStore(pool, config), slog.New(slog.NewTextHandler(io.Discard, nil)), func() time.Time { return now })
	return &world{t: t, pool: pool, handler: handler}
}

func (w *world) player(nickname string) string {
	w.t.Helper()
	var id string
	if err := w.pool.QueryRow(context.Background(), `insert into accounts default values returning id::text`).Scan(&id); err != nil {
		w.t.Fatal(err)
	}
	if _, err := w.pool.Exec(context.Background(), `insert into profiles (account_id, nickname, nickname_key) values ($1, $2, $3)`, id, nickname, profile.Key(nickname)); err != nil {
		w.t.Fatal(err)
	}
	return id
}

func (w *world) call(method, path, account string, body any) (int, map[string]any) {
	w.t.Helper()
	var reader io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = strings.NewReader(string(raw))
	}
	request := httptest.NewRequest(method, path, reader)
	request.Header.Set("Authorization", "Bearer access:"+account)
	recorder := httptest.NewRecorder()
	w.handler.ServeHTTP(recorder, request)
	decoded := map[string]any{}
	json.Unmarshal(recorder.Body.Bytes(), &decoded)
	return recorder.Code, decoded
}

func (w *world) expect(method, path, account string, body any, status int, code string) {
	w.t.Helper()
	got, decoded := w.call(method, path, account, body)
	if got != status || (code != "" && decoded["error"] != code) {
		w.t.Fatalf("%s %s: %d %v, want %d %s", method, path, got, decoded, status, code)
	}
}

func names(list map[string]any, part string) string {
	players, _ := list[part].([]any)
	out := make([]string, 0, len(players))
	for _, p := range players {
		out = append(out, p.(map[string]any)["nickname"].(string))
	}
	return strings.Join(out, ",")
}

func TestARequestBecomesAFriendshipWhenAccepted(t *testing.T) {
	w := newWorld(t, friends.DefaultConfig())
	ayla, kamal := w.player("Ayla"), w.player("Kamal")

	w.expect("POST", "/friends/requests", ayla, map[string]any{"nickname": "Kamal", "code": "7K3M-9QPZ"}, http.StatusBadRequest, "friend.target_required")
	w.expect("POST", "/friends/requests", ayla, map[string]any{}, http.StatusBadRequest, "friend.target_required")
	w.expect("POST", "/friends/requests", ayla, map[string]any{"nickname": "ayla"}, http.StatusBadRequest, "friend.self")
	w.expect("POST", "/friends/requests", ayla, map[string]any{"nickname": "nobody"}, http.StatusNotFound, "friend.unknown_player")
	if status, body := w.call("POST", "/friends/requests", ayla, map[string]any{"nickname": "KAMAL"}); status != http.StatusOK || body["status"] != "requested" {
		t.Fatalf("request %d %v", status, body)
	}
	w.expect("POST", "/friends/requests", ayla, map[string]any{"nickname": "Kamal"}, http.StatusOK, "")
	if _, list := w.call("GET", "/friends", kamal, nil); names(list, "incoming") != "Ayla" || names(list, "friends") != "" {
		t.Fatalf("kamal %v", list)
	}
	if _, list := w.call("GET", "/friends", ayla, nil); names(list, "outgoing") != "Kamal" {
		t.Fatalf("ayla %v", list)
	}

	w.expect("POST", "/friends/requests/"+kamal+"/accept", ayla, nil, http.StatusNotFound, "friend.no_request")
	w.expect("POST", "/friends/requests/"+ayla+"/accept", kamal, nil, http.StatusNoContent, "")
	for _, account := range []string{ayla, kamal} {
		if _, list := w.call("GET", "/friends", account, nil); names(list, "friends") == "" || names(list, "incoming")+names(list, "outgoing") != "" {
			t.Fatalf("after accept %v", list)
		}
	}
	w.expect("POST", "/friends/requests", kamal, map[string]any{"nickname": "Ayla"}, http.StatusConflict, "friend.already_friends")
	store := friends.NewStore(w.pool, friends.DefaultConfig())
	if ids, err := store.FriendIDs(context.Background(), ayla); err != nil || len(ids) != 1 || ids[0] != kamal {
		t.Fatalf("friend ids %v %v", ids, err)
	}
	if ids, err := store.FriendIDs(context.Background(), "not-an-account"); err != nil || len(ids) != 0 {
		t.Fatalf("friend ids of a malformed account %v %v", ids, err)
	}
	w.expect("DELETE", "/friends/"+ayla, kamal, nil, http.StatusNoContent, "")
	if ids, err := store.FriendIDs(context.Background(), ayla); err != nil || len(ids) != 0 {
		t.Fatalf("friend ids after remove %v %v", ids, err)
	}
	w.expect("DELETE", "/friends/"+ayla, kamal, nil, http.StatusNotFound, "friend.not_friends")
	if _, list := w.call("GET", "/friends", ayla, nil); names(list, "friends") != "" {
		t.Fatalf("after remove %v", list)
	}
}

func TestCrossedRequestsMakeFriendsAndDeclinesLeaveNoTrace(t *testing.T) {
	w := newWorld(t, friends.DefaultConfig())
	ayla, kamal, nigar := w.player("Ayla"), w.player("Kamal"), w.player("Nigar")
	w.expect("POST", "/friends/requests", ayla, map[string]any{"nickname": "Kamal"}, http.StatusOK, "")
	if status, body := w.call("POST", "/friends/requests", kamal, map[string]any{"nickname": "Ayla"}); status != http.StatusOK || body["status"] != "friends" {
		t.Fatalf("crossed %d %v", status, body)
	}
	w.expect("POST", "/friends/requests", nigar, map[string]any{"nickname": "Ayla"}, http.StatusOK, "")
	w.expect("POST", "/friends/requests/"+nigar+"/decline", ayla, nil, http.StatusNoContent, "")
	w.expect("POST", "/friends/requests/"+nigar+"/decline", ayla, nil, http.StatusNotFound, "friend.no_request")
	w.expect("POST", "/friends/requests", nigar, map[string]any{"nickname": "Kamal"}, http.StatusOK, "")
	w.expect("DELETE", "/friends/requests/"+kamal, nigar, nil, http.StatusNoContent, "")
	if _, list := w.call("GET", "/friends", nigar, nil); names(list, "outgoing") != "" {
		t.Fatalf("cancelled request still listed %v", list)
	}
}

func TestABlockEndsTheFriendshipAndHidesTheBlocker(t *testing.T) {
	w := newWorld(t, friends.DefaultConfig())
	ayla, kamal := w.player("Ayla"), w.player("Kamal")
	w.expect("POST", "/friends/requests", ayla, map[string]any{"nickname": "Kamal"}, http.StatusOK, "")
	w.expect("POST", "/friends/requests/"+ayla+"/accept", kamal, nil, http.StatusNoContent, "")
	w.expect("PUT", "/friends/blocks/"+ayla, ayla, nil, http.StatusBadRequest, "friend.self")
	w.expect("PUT", "/friends/blocks/00000000-0000-0000-0000-000000000000", ayla, nil, http.StatusNotFound, "friend.unknown_player")
	w.expect("PUT", "/friends/blocks/"+kamal, ayla, nil, http.StatusNoContent, "")
	w.expect("PUT", "/friends/blocks/"+kamal, ayla, nil, http.StatusNoContent, "")

	if _, list := w.call("GET", "/friends", ayla, nil); names(list, "friends") != "" || names(list, "blocked") != "Kamal" {
		t.Fatalf("blocker %v", list)
	}
	if _, list := w.call("GET", "/friends", kamal, nil); names(list, "friends") != "" || names(list, "blocked") != "" {
		t.Fatalf("blocked %v", list)
	}
	w.expect("POST", "/friends/requests", kamal, map[string]any{"nickname": "Ayla"}, http.StatusNotFound, "friend.unknown_player")
	w.expect("POST", "/friends/requests", ayla, map[string]any{"nickname": "Kamal"}, http.StatusNotFound, "friend.unknown_player")
	blocked, err := friends.NewStore(w.pool, friends.DefaultConfig()).Blocked(context.Background(), kamal, ayla)
	if err != nil || !blocked {
		t.Fatalf("blocked %v %v", blocked, err)
	}

	w.expect("DELETE", "/friends/blocks/"+kamal, ayla, nil, http.StatusNoContent, "")
	if _, list := w.call("GET", "/friends", ayla, nil); names(list, "friends")+names(list, "blocked") != "" {
		t.Fatalf("unblock restored something %v", list)
	}
	w.expect("POST", "/friends/requests", kamal, map[string]any{"nickname": "Ayla"}, http.StatusOK, "")
}

func TestLimitsRefuseRequestsAndFriendships(t *testing.T) {
	w := newWorld(t, friends.Config{MaxFriends: 1, MaxRequests: 1})
	ayla, kamal, nigar, orxan := w.player("Ayla"), w.player("Kamal"), w.player("Nigar"), w.player("Orxan")
	w.expect("POST", "/friends/requests", ayla, map[string]any{"nickname": "Kamal"}, http.StatusOK, "")
	w.expect("POST", "/friends/requests", ayla, map[string]any{"nickname": "Nigar"}, http.StatusConflict, "friend.requests_full")
	w.expect("POST", "/friends/requests", nigar, map[string]any{"nickname": "Kamal"}, http.StatusConflict, "friend.requests_full")
	w.expect("POST", "/friends/requests/"+ayla+"/accept", kamal, nil, http.StatusNoContent, "")
	w.expect("POST", "/friends/requests", nigar, map[string]any{"nickname": "Ayla"}, http.StatusOK, "")
	w.expect("POST", "/friends/requests/"+nigar+"/accept", ayla, nil, http.StatusConflict, "friend.friends_full")
	w.expect("POST", "/friends/requests", orxan, map[string]any{"nickname": "Nigar"}, http.StatusOK, "")
	if _, err := w.pool.Exec(context.Background(), `update accounts set state = 'banned' where id = $1`, orxan); err != nil {
		t.Fatal(err)
	}
	if _, list := w.call("GET", "/friends", nigar, nil); names(list, "incoming") != "" {
		t.Fatalf("a banned account still asks %v", list)
	}
	w.expect("POST", "/friends/requests/"+orxan+"/accept", nigar, nil, http.StatusNotFound, "friend.no_request")
	w.expect("GET", "/friends", "", nil, http.StatusUnauthorized, "access_rejected")
}

func TestAFriendCodeIsStableAndFindsItsPlayer(t *testing.T) {
	w := newWorld(t, friends.DefaultConfig())
	ayla, kamal := w.player("Ayla"), w.player("Kamal")
	_, first := w.call("GET", "/friends/code", kamal, nil)
	_, again := w.call("GET", "/friends/code", kamal, nil)
	code, _ := first["code"].(string)
	if len(code) != 9 || code[4] != '-' || again["code"] != code {
		t.Fatalf("codes %v %v", first, again)
	}
	w.expect("POST", "/friends/requests", kamal, map[string]any{"code": code}, http.StatusBadRequest, "friend.self")
	w.expect("POST", "/friends/requests", ayla, map[string]any{"code": "0000-000"}, http.StatusNotFound, "friend.unknown_player")
	if status, body := w.call("POST", "/friends/requests", ayla, map[string]any{"code": strings.ToLower(strings.ReplaceAll(code, "-", " "))}); status != http.StatusOK || body["status"] != "requested" {
		t.Fatalf("request by code %d %v", status, body)
	}
	if _, list := w.call("GET", "/friends", kamal, nil); names(list, "incoming") != "Ayla" {
		t.Fatalf("kamal %v", list)
	}
}

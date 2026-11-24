package push_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/push"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

var now = time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)

type verifier struct{}

func (verifier) Verify(access string) (token.Claims, error) {
	if access == "expired" {
		return token.Claims{}, token.ErrExpired
	}
	if account, ok := strings.CutPrefix(access, "access:"); ok && account != "" {
		return token.Claims{Account: account}, nil
	}
	return token.Claims{}, token.ErrSignature
}

type world struct {
	t       *testing.T
	pool    *pgxpool.Pool
	store   *push.Store
	handler http.Handler
}

func newWorld(t *testing.T) *world {
	t.Helper()
	_, pool := dbtest.Fresh(t)
	store := push.NewStore(pool)
	handler := push.NewHandler(verifier{}, store, slog.New(slog.NewTextHandler(io.Discard, nil)), func() time.Time { return now })
	return &world{t: t, pool: pool, store: store, handler: handler}
}

func (w *world) player() string {
	w.t.Helper()
	var id string
	if err := w.pool.QueryRow(context.Background(), `insert into accounts default values returning id::text`).Scan(&id); err != nil {
		w.t.Fatal(err)
	}
	return id
}

func (w *world) call(method, path, bearer, body string) (int, map[string]any) {
	w.t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if bearer != "" {
		request.Header.Set("Authorization", "Bearer "+bearer)
	}
	recorder := httptest.NewRecorder()
	w.handler.ServeHTTP(recorder, request)
	decoded := map[string]any{}
	json.Unmarshal(recorder.Body.Bytes(), &decoded)
	return recorder.Code, decoded
}

func (w *world) expect(method, path, account, body string, status int, code string) map[string]any {
	w.t.Helper()
	got, decoded := w.call(method, path, "access:"+account, body)
	if got != status || (code != "" && decoded["error"] != code) {
		w.t.Fatalf("%s %s %s: %d %v, want %d %s", method, path, body, got, decoded, status, code)
	}
	return decoded
}

func (w *world) devices(account string) string {
	w.t.Helper()
	devices, err := w.store.Devices(context.Background(), account)
	if err != nil {
		w.t.Fatal(err)
	}
	tokens := make([]string, 0, len(devices))
	for _, d := range devices {
		tokens = append(tokens, d.Platform+"/"+d.Token)
	}
	return strings.Join(tokens, ",")
}

func TestRoutesNeedAValidAccessToken(t *testing.T) {
	w := newWorld(t)
	if status, body := w.call("GET", "/push/settings", "", ""); status != http.StatusUnauthorized || body["error"] != "access_rejected" {
		t.Fatalf("without a token %d %v", status, body)
	}
	if status, body := w.call("GET", "/push/settings", "expired", ""); status != http.StatusUnauthorized || body["error"] != "access_expired" {
		t.Fatalf("expired %d %v", status, body)
	}
}

func TestDevicesRegisterMoveAndUnregister(t *testing.T) {
	w := newWorld(t)
	ayla, kamal := w.player(), w.player()

	w.expect("PUT", "/push/devices", ayla, `{"platform":"android"}`, http.StatusBadRequest, "push.token_required")
	w.expect("PUT", "/push/devices", ayla, `{"token":"t1","platform":"windows"}`, http.StatusBadRequest, "push.unknown_platform")
	w.expect("PUT", "/push/devices", ayla, `{`, http.StatusBadRequest, "push.malformed")
	w.expect("PUT", "/push/devices", ayla, `{"token":"t1","platform":"android"}`, http.StatusNoContent, "")
	w.expect("PUT", "/push/devices", ayla, `{"token":"t1","platform":"android"}`, http.StatusNoContent, "")
	w.expect("PUT", "/push/devices", ayla, `{"token":"t2","platform":"ios"}`, http.StatusNoContent, "")
	if got := w.devices(ayla); got != "android/t1,ios/t2" {
		t.Fatalf("ayla devices %s", got)
	}

	w.expect("PUT", "/push/devices", kamal, `{"token":"t2","platform":"ios"}`, http.StatusNoContent, "")
	if got := w.devices(ayla) + "|" + w.devices(kamal); got != "android/t1|ios/t2" {
		t.Fatalf("after the move %s", got)
	}

	w.expect("DELETE", "/push/devices", ayla, `{}`, http.StatusBadRequest, "push.token_required")
	w.expect("DELETE", "/push/devices", ayla, `{"token":"t2"}`, http.StatusNoContent, "")
	if got := w.devices(kamal); got != "ios/t2" {
		t.Fatalf("another player's token was removed: %s", got)
	}
	w.expect("DELETE", "/push/devices", ayla, `{"token":"t1"}`, http.StatusNoContent, "")
	if got := w.devices(ayla); got != "" {
		t.Fatalf("ayla devices after unregister %s", got)
	}

	if err := w.store.Forget(context.Background(), "t2"); err != nil || w.devices(kamal) != "" {
		t.Fatalf("forget %v %s", err, w.devices(kamal))
	}
}

func TestSettingsDefaultAndChangeOneCategoryAtATime(t *testing.T) {
	w := newWorld(t)
	ayla := w.player()

	if body := w.expect("GET", "/push/settings", ayla, "", http.StatusOK, ""); body["gameplay"] != true || body["promotional"] != false {
		t.Fatalf("defaults %v", body)
	}
	if body := w.expect("PUT", "/push/settings", ayla, `{"promotional":true,"policyVersion":"1","appVersion":"1.0.0"}`, http.StatusOK, ""); body["gameplay"] != true || body["promotional"] != true {
		t.Fatalf("promotional on %v", body)
	}
	if body := w.expect("PUT", "/push/settings", ayla, `{"gameplay":false}`, http.StatusOK, ""); body["gameplay"] != false || body["promotional"] != true {
		t.Fatalf("gameplay off %v", body)
	}
	w.expect("PUT", "/push/settings", ayla, `[`, http.StatusBadRequest, "push.malformed")
	if body := w.expect("GET", "/push/settings", ayla, "", http.StatusOK, ""); body["gameplay"] != false || body["promotional"] != true {
		t.Fatalf("stored %v", body)
	}
}

func TestPromotionalChangesWriteAppendOnlyConsentRecords(t *testing.T) {
	w := newWorld(t)
	ayla := w.player()

	w.expect("PUT", "/push/settings", ayla, `{"promotional":true}`, http.StatusBadRequest, "push.consent_context_required")
	w.expect("PUT", "/push/settings", ayla, `{"promotional":true,"policyVersion":"3"}`, http.StatusBadRequest, "push.consent_context_required")
	if body := w.expect("GET", "/push/settings", ayla, "", http.StatusOK, ""); body["promotional"] != false {
		t.Fatalf("a refused consent change was saved %v", body)
	}
	w.expect("PUT", "/push/settings", ayla, `{"promotional":false,"gameplay":false}`, http.StatusOK, "")
	w.expect("PUT", "/push/settings", ayla, `{"promotional":true,"policyVersion":"3","appVersion":"1.2.0"}`, http.StatusOK, "")
	w.expect("PUT", "/push/settings", ayla, `{"promotional":true,"gameplay":true}`, http.StatusOK, "")
	w.expect("PUT", "/push/settings", ayla, `{"promotional":false,"policyVersion":"4","appVersion":"1.3.0"}`, http.StatusOK, "")

	rows, err := w.pool.Query(context.Background(), `select purpose, granted, policy_version, source, app_version, recorded_at from consent_records where account_id = $1 order by id`, ayla)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var purpose, version, source, app string
		var granted bool
		var at time.Time
		if err := rows.Scan(&purpose, &granted, &version, &source, &app, &at); err != nil {
			t.Fatal(err)
		}
		if !at.Equal(now) {
			t.Fatalf("recorded at %v, want server time %v", at, now)
		}
		got = append(got, fmt.Sprintf("%s:%t:%s:%s:%s", purpose, granted, version, source, app))
	}
	want := "promotional_push:true:3:settings:1.2.0,promotional_push:false:4:settings:1.3.0"
	if strings.Join(got, ",") != want {
		t.Fatalf("consent records %v, want %s", got, want)
	}
}

func TestATurnedOffCategoryReachesNoDevice(t *testing.T) {
	w := newWorld(t)
	ayla := w.player()
	w.expect("PUT", "/push/devices", ayla, `{"token":"t1","platform":"android"}`, http.StatusNoContent, "")
	w.expect("PUT", "/push/settings", ayla, `{"gameplay":false}`, http.StatusOK, "")
	sender := &fakeSender{}
	notifier := push.NewNotifier(w.store, sender)

	if sent, err := notifier.Notify(context.Background(), ayla, push.Message{Category: push.Gameplay, TitleKey: "g"}); err != nil || sent != 0 {
		t.Fatalf("gameplay off sent %d %v", sent, err)
	}
	if sent, err := notifier.Notify(context.Background(), ayla, push.Message{Category: push.Promotional, TitleKey: "p"}); err != nil || sent != 0 {
		t.Fatalf("promotional without consent sent %d %v", sent, err)
	}
	w.expect("PUT", "/push/settings", ayla, `{"gameplay":true}`, http.StatusOK, "")
	if sent, err := notifier.Notify(context.Background(), ayla, push.Message{Category: push.Gameplay, TitleKey: "g"}); err != nil || sent != 1 || sender.sent[0] != "t1:g" {
		t.Fatalf("gameplay on sent %d %v %v", sent, sender.sent, err)
	}
}

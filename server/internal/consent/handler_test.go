package consent_test

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

	"github.com/Alievs-corp/hypedomino/server/internal/consent"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

var now = time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)

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
	store   *consent.Store
	handler http.Handler
}

func newWorld(t *testing.T) *world {
	t.Helper()
	_, pool := dbtest.Fresh(t)
	store := consent.NewStore(pool)
	handler := consent.NewHandler(verifier{}, store, slog.New(slog.NewTextHandler(io.Discard, nil)), func() time.Time { return now })
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

func (w *world) call(method, bearer, body string) (int, map[string]any) {
	w.t.Helper()
	return w.callAt(method, "/consents", bearer, body)
}

func (w *world) callAt(method, path, bearer, body string) (int, map[string]any) {
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

func (w *world) expect(method, account, body string, status int, want string) {
	w.t.Helper()
	w.expectAt(method, "/consents", account, body, status, want)
}

func (w *world) expectFirstLaunch(account, body string, status int, want string) {
	w.t.Helper()
	w.expectAt("POST", consent.FirstLaunchPath, account, body, status, want)
}

func (w *world) expectAt(method, path, account, body string, status int, want string) {
	w.t.Helper()
	got, decoded := w.callAt(method, path, "access:"+account, body)
	if got != status || fmt.Sprint(decoded) != want {
		w.t.Fatalf("%s %s %s: %d %v, want %d %s", method, path, body, got, decoded, status, want)
	}
}

func (w *world) records(account string) string {
	w.t.Helper()
	rows, err := w.pool.Query(context.Background(), `select purpose, granted, policy_version, source, app_version, recorded_at from consent_records where account_id = $1 order by id`, account)
	if err != nil {
		w.t.Fatal(err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var purpose, policy, source, app string
		var granted bool
		var recorded time.Time
		if err := rows.Scan(&purpose, &granted, &policy, &source, &app, &recorded); err != nil {
			w.t.Fatal(err)
		}
		if !recorded.Equal(now) {
			w.t.Fatalf("recorded at %v, want server time %v", recorded, now)
		}
		got = append(got, fmt.Sprintf("%s/%t/%s/%s/%s", purpose, granted, policy, source, app))
	}
	return strings.Join(got, ",")
}

const (
	neither   = "map[analytics:false diagnostics:false]"
	analysed  = "map[analytics:true diagnostics:false]"
	diagnosed = "map[analytics:false diagnostics:true]"
	both      = "map[analytics:true diagnostics:true]"
)

func TestConsentsNeedAValidAccessToken(t *testing.T) {
	w := newWorld(t)
	for _, method := range []string{"GET", "PUT"} {
		if status, decoded := w.call(method, "", `{"analytics":true}`); status != http.StatusUnauthorized || decoded["error"] != "access_rejected" {
			t.Fatalf("%s without a token %d %v", method, status, decoded)
		}
		if status, decoded := w.call(method, "expired", `{"analytics":true}`); status != http.StatusUnauthorized || decoded["error"] != "access_expired" {
			t.Fatalf("%s expired %d %v", method, status, decoded)
		}
	}
	body := `{"choices":[{"purpose":"analytics","granted":true}],"policyVersion":"2026-11","appVersion":"1.4.0"}`
	for _, bearer := range []string{"", "expired"} {
		if status, _ := w.callAt("POST", consent.FirstLaunchPath, bearer, body); status != http.StatusUnauthorized {
			t.Fatalf("first launch with %q answered %d", bearer, status)
		}
	}
}

func TestConsentsStartOffAndRecordEveryChange(t *testing.T) {
	w := newWorld(t)
	ayla := w.player()

	w.expect("GET", ayla, "", http.StatusOK, neither)
	w.expect("PUT", ayla, `{`, http.StatusBadRequest, "map[error:consent.malformed]")
	w.expect("PUT", ayla, `{"analytics":true}`, http.StatusBadRequest, "map[error:consent.context_required]")
	w.expect("PUT", ayla, `{"analytics":true,"policyVersion":"2026-11"}`, http.StatusBadRequest, "map[error:consent.context_required]")
	if got := w.records(ayla); got != "" {
		t.Fatalf("recorded without a context: %s", got)
	}

	w.expect("PUT", ayla, `{"analytics":true,"policyVersion":"2026-11","appVersion":"1.4.0"}`, http.StatusOK, analysed)
	w.expect("PUT", ayla, `{"diagnostics":true,"policyVersion":"2026-11","appVersion":"1.4.1"}`, http.StatusOK, both)
	w.expect("PUT", ayla, `{"analytics":true}`, http.StatusOK, both)
	w.expect("GET", ayla, "", http.StatusOK, both)
	w.expect("PUT", ayla, `{"analytics":false,"diagnostics":false,"policyVersion":"2026-12","appVersion":"1.5.0"}`, http.StatusOK, neither)
	w.expect("GET", ayla, "", http.StatusOK, neither)

	want := "analytics/true/2026-11/settings/1.4.0,diagnostics/true/2026-11/settings/1.4.1,analytics/false/2026-12/settings/1.5.0,diagnostics/false/2026-12/settings/1.5.0"
	if got := w.records(ayla); got != want {
		t.Fatalf("records %s, want %s", got, want)
	}
}

func TestConsentsBelongToTheirAccountAndIgnoreOtherPurposes(t *testing.T) {
	w := newWorld(t)
	ayla, kamal := w.player(), w.player()
	if _, err := w.pool.Exec(context.Background(), `insert into consent_records (account_id, purpose, granted, policy_version, source, app_version, recorded_at)
		values ($1, 'promotional_push', true, '2026-11', 'settings', '1.4.0', $2)`, kamal, now); err != nil {
		t.Fatal(err)
	}

	w.expect("PUT", ayla, `{"analytics":true,"policyVersion":"2026-11","appVersion":"1.4.0"}`, http.StatusOK, analysed)
	w.expect("GET", kamal, "", http.StatusOK, neither)

	for _, check := range []struct {
		account string
		purpose consent.Purpose
		want    bool
	}{
		{ayla, consent.Analytics, true},
		{ayla, consent.Diagnostics, false},
		{kamal, consent.Analytics, false},
	} {
		granted, err := w.store.Granted(context.Background(), check.account, check.purpose)
		if err != nil || granted != check.want {
			t.Fatalf("%s granted %t %v, want %t", check.purpose, granted, err, check.want)
		}
	}
}

func TestFirstLaunchChoicesAreWrittenInTheOrderTheyWereMade(t *testing.T) {
	w := newWorld(t)
	ayla := w.player()

	w.expectFirstLaunch(ayla, `{`, http.StatusBadRequest, "map[error:consent.malformed]")
	w.expectFirstLaunch(ayla, `{"choices":[{"purpose":"promotional_push","granted":true}],"policyVersion":"2026-11","appVersion":"1.4.0"}`, http.StatusBadRequest, "map[error:consent.malformed]")
	w.expectFirstLaunch(ayla, `{"choices":[{"purpose":"analytics","granted":true}]}`, http.StatusBadRequest, "map[error:consent.context_required]")
	w.expectFirstLaunch(ayla, `{"choices":[{"purpose":"analytics","granted":true}],"policyVersion":"2026-11"}`, http.StatusBadRequest, "map[error:consent.context_required]")
	w.expectFirstLaunch(ayla, `{"choices":[]}`, http.StatusOK, neither)
	if got := w.records(ayla); got != "" {
		t.Fatalf("recorded a refused or empty first launch: %s", got)
	}

	w.expectFirstLaunch(ayla, `{"choices":[{"purpose":"analytics","granted":true},{"purpose":"diagnostics","granted":true},{"purpose":"analytics","granted":false}],"policyVersion":"2026-11","appVersion":"1.4.0"}`, http.StatusOK, diagnosed)
	w.expect("GET", ayla, "", http.StatusOK, diagnosed)

	want := "analytics/true/2026-11/first_launch/1.4.0,diagnostics/true/2026-11/first_launch/1.4.0,analytics/false/2026-11/first_launch/1.4.0"
	if got := w.records(ayla); got != want {
		t.Fatalf("records %s, want %s", got, want)
	}
}

func TestFirstLaunchKeepsTheRecordTheAccountHasForThatPolicy(t *testing.T) {
	w := newWorld(t)
	ayla, kamal := w.player(), w.player()

	w.expectFirstLaunch(ayla, `{"choices":[{"purpose":"analytics","granted":false}],"policyVersion":"2026-11","appVersion":"1.4.0"}`, http.StatusOK, neither)
	w.expectFirstLaunch(ayla, `{"choices":[{"purpose":"analytics","granted":true}],"policyVersion":"2026-11","appVersion":"1.4.1"}`, http.StatusOK, neither)
	w.expectFirstLaunch(ayla, `{"choices":[{"purpose":"analytics","granted":true}],"policyVersion":"2026-12","appVersion":"1.5.0"}`, http.StatusOK, analysed)

	want := "analytics/false/2026-11/first_launch/1.4.0,analytics/true/2026-12/first_launch/1.5.0"
	if got := w.records(ayla); got != want {
		t.Fatalf("records %s, want %s", got, want)
	}

	if _, err := w.pool.Exec(context.Background(), `insert into consent_records (account_id, purpose, granted, policy_version, source, app_version, recorded_at)
		values ($1, 'promotional_push', true, '2026-11', 'settings', '1.4.0', $2)`, kamal, now); err != nil {
		t.Fatal(err)
	}
	w.expectFirstLaunch(kamal, `{"choices":[{"purpose":"diagnostics","granted":true}],"policyVersion":"2026-11","appVersion":"1.4.0"}`, http.StatusOK, neither)
	if got := w.records(kamal); got != "promotional_push/true/2026-11/settings/1.4.0" {
		t.Fatalf("first launch wrote over the account's record: %s", got)
	}
}

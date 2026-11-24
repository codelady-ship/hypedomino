package analytics_test

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

	"github.com/Alievs-corp/hypedomino/server/internal/analytics"
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
	t        *testing.T
	pool     *pgxpool.Pool
	consents *consent.Store
	handler  http.Handler
}

func newWorld(t *testing.T) *world {
	t.Helper()
	_, pool := dbtest.Fresh(t)
	consents := consent.NewStore(pool)
	handler := analytics.NewHandler(verifier{}, analytics.NewStore(pool), consents, slog.New(slog.NewTextHandler(io.Discard, nil)), func() time.Time { return now })
	return &world{t: t, pool: pool, consents: consents, handler: handler}
}

func (w *world) stranger() string {
	w.t.Helper()
	var id string
	if err := w.pool.QueryRow(context.Background(), `insert into accounts default values returning id::text`).Scan(&id); err != nil {
		w.t.Fatal(err)
	}
	return id
}

func (w *world) player() string {
	w.t.Helper()
	account := w.stranger()
	w.chooseAnalytics(account, true)
	return account
}

func (w *world) chooseAnalytics(account string, granted bool) {
	w.t.Helper()
	change := consent.Change{Analytics: &granted, PolicyVersion: "2026-11", AppVersion: "1.4.0"}
	if _, err := w.consents.Save(context.Background(), account, change, now); err != nil {
		w.t.Fatal(err)
	}
}

func (w *world) post(bearer, body string) (int, map[string]any) {
	w.t.Helper()
	request := httptest.NewRequest("POST", "/analytics/events", strings.NewReader(body))
	if bearer != "" {
		request.Header.Set("Authorization", "Bearer "+bearer)
	}
	recorder := httptest.NewRecorder()
	w.handler.ServeHTTP(recorder, request)
	decoded := map[string]any{}
	json.Unmarshal(recorder.Body.Bytes(), &decoded)
	return recorder.Code, decoded
}

func (w *world) expect(account, body string, status int, field string, want any) {
	w.t.Helper()
	got, decoded := w.post("access:"+account, body)
	if got != status || decoded[field] != want {
		w.t.Fatalf("%s: %d %v, want %d %s=%v", body, got, decoded, status, field, want)
	}
}

func (w *world) events() string {
	w.t.Helper()
	rows, err := w.pool.Query(context.Background(), `select e.analytics_id::text, e.name, e.label, e.value, e.client_time_ms, e.received_at from analytics_events e order by e.id`)
	if err != nil {
		w.t.Fatal(err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var analyticsID, name, label string
		var value, clientTime int64
		var received time.Time
		if err := rows.Scan(&analyticsID, &name, &label, &value, &clientTime, &received); err != nil {
			w.t.Fatal(err)
		}
		if !received.Equal(now) {
			w.t.Fatalf("received at %v, want server time %v", received, now)
		}
		got = append(got, fmt.Sprintf("%s/%s/%s/%d/%d", analyticsID[:8], name, label, value, clientTime))
	}
	return strings.Join(got, ",")
}

func batch(analyticsID string, events ...string) string {
	return `{"analyticsId":"` + analyticsID + `","events":[` + strings.Join(events, ",") + `]}`
}

func TestIngestNeedsAValidAccessToken(t *testing.T) {
	w := newWorld(t)
	body := batch(id, `{"name":"app_open","clientTimeMs":1}`)
	if status, decoded := w.post("", body); status != http.StatusUnauthorized || decoded["error"] != "access_rejected" {
		t.Fatalf("without a token %d %v", status, decoded)
	}
	if status, decoded := w.post("expired", body); status != http.StatusUnauthorized || decoded["error"] != "access_expired" {
		t.Fatalf("expired %d %v", status, decoded)
	}
	if got := w.events(); got != "" {
		t.Fatalf("stored without a token: %s", got)
	}
}

func TestIngestNeedsTheAnalyticsConsent(t *testing.T) {
	w := newWorld(t)
	ayla := w.stranger()
	body := batch(id, `{"name":"app_open","clientTimeMs":1}`)

	w.expect(ayla, body, http.StatusForbidden, "error", "analytics.consent_required")
	w.chooseAnalytics(ayla, true)
	w.expect(ayla, body, http.StatusOK, "accepted", float64(1))
	w.chooseAnalytics(ayla, false)
	w.expect(ayla, batch(id, `{"name":"nickname_set","clientTimeMs":2}`), http.StatusForbidden, "error", "analytics.consent_required")

	if got := w.events(); got != "6f1c2a7e/app_open//0/1" {
		t.Fatalf("stored %s", got)
	}
}

func TestIngestStoresValidClientEventsWithServerTime(t *testing.T) {
	w := newWorld(t)
	ayla := w.player()

	w.expect(ayla, `{`, http.StatusBadRequest, "error", "analytics.malformed")
	w.expect(ayla, batch("not-an-id", `{"name":"app_open","clientTimeMs":1}`), http.StatusBadRequest, "error", "analytics.malformed")
	w.expect(ayla, batch(id), http.StatusOK, "accepted", float64(0))
	w.expect(ayla, batch(id,
		`{"name":"app_open","clientTimeMs":1700000000000}`,
		`{"name":"match_end","value":40,"clientTimeMs":1700000000001}`,
		`{"name":"sign_in","label":"google","clientTimeMs":1700000000002}`,
		`{"name":"screen_view","label":"ayla@example.com","clientTimeMs":1700000000003}`,
		`{"name":"purchase_start","label":"crystals_tier1","clientTimeMs":1700000000004}`,
	), http.StatusOK, "accepted", float64(3))

	want := "6f1c2a7e/app_open//0/1700000000000,6f1c2a7e/sign_in/google/0/1700000000002,6f1c2a7e/purchase_start/crystals_tier1/0/1700000000004"
	if got := w.events(); got != want {
		t.Fatalf("stored %s, want %s", got, want)
	}
	var owner string
	var firstSeen time.Time
	if err := w.pool.QueryRow(context.Background(), `select account_id::text, first_seen_at from analytics_ids where analytics_id = $1`, id).Scan(&owner, &firstSeen); err != nil {
		t.Fatal(err)
	}
	if owner != ayla || !firstSeen.Equal(now) {
		t.Fatalf("analytics id owner %s first seen %v", owner, firstSeen)
	}
}

func TestAnalyticsIDBelongsToTheFirstAccountThatSendsIt(t *testing.T) {
	w := newWorld(t)
	ayla, kamal := w.player(), w.player()
	w.expect(ayla, batch(id, `{"name":"app_open","clientTimeMs":1}`), http.StatusOK, "accepted", float64(1))
	w.expect(kamal, batch(id, `{"name":"app_open","clientTimeMs":2}`), http.StatusConflict, "error", "analytics.id_taken")
	w.expect(ayla, batch(strings.ToUpper(id), `{"name":"nickname_set","clientTimeMs":3}`), http.StatusOK, "accepted", float64(1))
	if got := w.events(); got != "6f1c2a7e/app_open//0/1,6f1c2a7e/nickname_set//0/3" {
		t.Fatalf("stored %s", got)
	}
}

func TestIngestRefusesAnOversizedBody(t *testing.T) {
	w := newWorld(t)
	ayla := w.player()
	event := `{"name":"app_open","clientTimeMs":1}`
	events := make([]string, analytics.MaxBodyBytes/len(event)+1)
	for i := range events {
		events[i] = event
	}
	w.expect(ayla, batch(id, events...), http.StatusBadRequest, "error", "analytics.malformed")
	if got := w.events(); got != "" {
		t.Fatalf("stored from an oversized body: %s", got)
	}
}

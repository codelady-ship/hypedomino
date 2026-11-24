package report_test

import (
	"context"
	"encoding/json"
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
	"github.com/Alievs-corp/hypedomino/server/internal/profile"
	"github.com/Alievs-corp/hypedomino/server/internal/report"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

func TestMain(m *testing.M) {
	code := m.Run()
	dbtest.Stop()
	os.Exit(code)
}

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
	now     time.Time
}

func newWorld(t *testing.T) *world {
	t.Helper()
	_, pool := dbtest.Fresh(t)
	w := &world{t: t, pool: pool, now: time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)}
	w.handler = report.NewHandler(verifier{}, report.NewStore(pool, report.DefaultConfig()), slog.New(slog.NewTextHandler(io.Discard, nil)), func() time.Time { return w.now })
	return w
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

func (w *world) report(reporter, target, reason string) (int, string) {
	w.t.Helper()
	raw, _ := json.Marshal(report.Report{Target: target, Reason: reason})
	request := httptest.NewRequest("POST", "/reports", strings.NewReader(string(raw)))
	request.Header.Set("Authorization", "Bearer access:"+reporter)
	recorder := httptest.NewRecorder()
	w.handler.ServeHTTP(recorder, request)
	var body map[string]string
	json.Unmarshal(recorder.Body.Bytes(), &body)
	return recorder.Code, body["error"]
}

func (w *world) count(query string, args ...any) int {
	w.t.Helper()
	var n int
	if err := w.pool.QueryRow(context.Background(), query, args...).Scan(&n); err != nil {
		w.t.Fatal(err)
	}
	return n
}

func TestAReportIsCheckedAndRepeatsOnOneTargetMerge(t *testing.T) {
	w := newWorld(t)
	kamal, rude := w.player("Kamal"), w.player("Rude")
	for _, c := range []struct {
		target, reason string
		status         int
		code           string
	}{
		{rude, "spam", http.StatusBadRequest, "report.reason"},
		{kamal, report.ReasonOther, http.StatusBadRequest, "report.self"},
		{"00000000-0000-0000-0000-000000000000", report.ReasonOther, http.StatusNotFound, "report.unknown_player"},
		{rude, report.ReasonOther, http.StatusAccepted, ""},
		{rude, report.ReasonNickname, http.StatusAccepted, ""},
	} {
		if status, code := w.report(kamal, c.target, c.reason); status != c.status || code != c.code {
			t.Fatalf("%s %s: %d %s", c.target, c.reason, status, code)
		}
	}
	if n := w.count(`select count(*) from player_reports where reporter = $1 and target = $2 and reason = $3 and name_text = 'Rude'`, kamal, rude, report.ReasonNickname); n != 1 {
		t.Fatalf("merged rows %d", n)
	}
	if n := w.count(`select count(*) from player_reports`); n != 1 {
		t.Fatalf("rows %d", n)
	}
}

func TestFiveReportersOfANicknameRequireARename(t *testing.T) {
	w := newWorld(t)
	rude := w.player("Rude")
	for i := range 5 {
		w.report(w.player("Reporter"+string(rune('a'+i))), rude, report.ReasonNickname)
		want := 0
		if i == 4 {
			want = 1
		}
		if n := w.count(`select count(*) from profiles where account_id = $1 and rename_required`, rude); n != want {
			t.Fatalf("after %d reporters rename_required=%d", i+1, n)
		}
	}
}

func TestAReporterOverTheDailyLimitOrWithRejectedReportsIsIgnoredQuietly(t *testing.T) {
	w := newWorld(t)
	kamal := w.player("Kamal")
	for i := range 11 {
		if status, _ := w.report(kamal, w.player("Target"+string(rune('a'+i))), report.ReasonOther); status != http.StatusAccepted {
			t.Fatalf("report %d answered %d", i, status)
		}
	}
	if n := w.count(`select count(*) from player_reports where reporter = $1`, kamal); n != 10 {
		t.Fatalf("stored %d, want the daily limit", n)
	}

	w.now = w.now.Add(25 * time.Hour)
	if _, err := w.pool.Exec(context.Background(), `update player_reports set state = 'rejected', decided_at = $2 where reporter = $1 and id in (select id from player_reports where reporter = $1 limit 5)`, kamal, w.now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	w.report(kamal, w.player("Late"), report.ReasonOther)
	if n := w.count(`select count(*) from player_reports where reporter = $1`, kamal); n != 10 {
		t.Fatalf("a silenced reporter stored a report: %d", n)
	}
	w.now = w.now.Add(7 * 24 * time.Hour)
	w.report(kamal, w.player("Later"), report.ReasonOther)
	if n := w.count(`select count(*) from player_reports where reporter = $1`, kamal); n != 11 {
		t.Fatalf("the silence did not end: %d", n)
	}
}

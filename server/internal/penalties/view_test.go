package penalties

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/matchmaking"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

type viewVerifier struct{}

func (viewVerifier) Verify(access string) (token.Claims, error) {
	return token.Claims{Account: access}, nil
}

type fixedRecords struct {
	record Record
}

func (f fixedRecords) Record(context.Context, string) (Record, error) {
	return f.record, nil
}

func (f fixedRecords) Config() (Config, int) {
	return DefaultConfig(), 1
}

func TestThePlayerSeesTheActiveOffensesTheirExpiryAndTheNextStep(t *testing.T) {
	config := DefaultConfig()
	var record Record
	record.MatchEnd(config, matchmaking.QueueRanked, "m1", rules.SeatX, left(false), 1, start)
	record.MatchEnd(config, matchmaking.QueueCasual, "m2", rules.SeatX, afk(true), 1, start)
	now := start.Add(time.Hour)
	handler := NewHandler(viewVerifier{}, fixedRecords{record: record}, slog.New(slog.NewTextHandler(io.Discard, nil)), func() time.Time { return now })
	request := httptest.NewRequest(http.MethodGet, "/penalties", nil)
	request.Header.Set("Authorization", "Bearer kamal")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	var view View
	if err := json.NewDecoder(recorder.Body).Decode(&view); recorder.Code != http.StatusOK || err != nil {
		t.Fatalf("status %d, %v", recorder.Code, err)
	}
	ranked := view.Ranked
	if ranked.Level != 2 || ranked.Step != "warning" || ranked.NextStepLevel != config.QueueDelayLevel || ranked.BanUntil != nil || len(ranked.Offenses) != 1 {
		t.Fatalf("ranked %+v", ranked)
	}
	if o := ranked.Offenses[0]; o.MatchID != "m1" || o.Kind != "leave_returned" || !o.ExpiresAt.Equal(start.Add(time.Duration(config.OffenseWindowDays)*24*time.Hour)) {
		t.Fatalf("offense %+v", o)
	}
	if casual := view.Casual; casual.Level != config.AfkWeight || len(casual.Offenses) != 1 || casual.CleanMatchesToForgive != config.CasualCleanMatchesToForgive {
		t.Fatalf("casual %+v", casual)
	}
	if expired := record.View(config, start.Add(time.Duration(config.OffenseWindowDays)*24*time.Hour)); expired.Ranked.Level != 0 || len(record.Ranked.Offenses) != 1 {
		t.Fatalf("expired view %+v, record %+v", expired.Ranked, record.Ranked)
	}
}

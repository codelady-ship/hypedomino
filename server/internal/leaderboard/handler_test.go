package leaderboard_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/leaderboard"
	"github.com/Alievs-corp/hypedomino/server/internal/rating"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

type fakeVerifier struct{}

func (fakeVerifier) Verify(access string) (token.Claims, error) {
	switch {
	case access == "expired":
		return token.Claims{}, token.ErrExpired
	case strings.HasPrefix(access, "access:"):
		return token.Claims{Account: strings.TrimPrefix(access, "access:")}, nil
	}
	return token.Claims{}, token.ErrSignature
}

type fakeBoards struct {
	calls []string
}

func (f *fakeBoards) Top(_ context.Context, season string, mode rules.Mode, board string, size int) ([]leaderboard.Row, error) {
	f.calls = append(f.calls, "top "+season+" "+mode.String()+" "+board)
	switch {
	case board != leaderboard.BoardGlobal:
		return nil, leaderboard.ErrUnknownBoard
	case season == "season.down":
		return nil, leaderboard.ErrUnavailable
	case season == "season.broken":
		return nil, errors.New("boom")
	}
	rows := []leaderboard.Row{{Position: 1, Account: "top", Rank: rating.Rank{Index: 28, LP: 700}, Tier: rating.TierChallenger, Wins: 90}, {Position: 2, Account: "second", Rank: rating.Rank{Index: 14, LP: 50}, Tier: "Gold", Wins: 10}}
	return rows[:min(size, len(rows))], nil
}

func (f *fakeBoards) Around(_ context.Context, season string, mode rules.Mode, board, account string, radius int) ([]leaderboard.Row, bool, error) {
	f.calls = append(f.calls, "around "+account)
	if account != "second" {
		return nil, false, nil
	}
	return []leaderboard.Row{{Position: 1, Account: "top"}, {Position: 2, Account: "second", Rank: rating.Rank{Index: 14, LP: 50}, Tier: "Gold"}, {Position: 3, Account: "third"}}, true, nil
}

type fakeSeasons struct{}

func (fakeSeasons) SeasonAt(_ context.Context, at time.Time) (rating.Season, error) {
	if at.Year() < 2026 {
		return rating.Season{}, rating.ErrNoSeason
	}
	return rating.Season{ID: "season.s1"}, nil
}

func (fakeSeasons) TopSize() int { return 1 }

func TestTheHandlerServesTopAndTheViewersPlace(t *testing.T) {
	boards := &fakeBoards{}
	clock := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	handler := leaderboard.NewHandler(fakeVerifier{}, boards, fakeSeasons{}, slog.New(slog.NewTextHandler(io.Discard, nil)), func() time.Time { return clock })
	call := func(access, path string) (int, map[string]any) {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		if access != "" {
			request.Header.Set("Authorization", "Bearer "+access)
		}
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		var body map[string]any
		json.Unmarshal(recorder.Body.Bytes(), &body)
		return recorder.Code, body
	}
	status, body := call("access:second", "/leaderboard/1v1")
	if status != http.StatusOK || body["season"] != "season.s1" || body["board"] != "global" || len(body["top"].([]any)) != 1 || body["me"].(map[string]any)["position"] != float64(2) || len(body["around"].([]any)) != 3 {
		t.Fatalf("view: %d %v", status, body)
	}
	if boards.calls[0] != "top season.s1 1v1 global" || boards.calls[1] != "around second" {
		t.Fatalf("calls %v", boards.calls)
	}
	status, body = call("access:nobody", "/leaderboard/2v2?season=season.s2")
	if status != http.StatusOK || body["season"] != "season.s2" || body["me"] != nil || len(body["around"].([]any)) != 0 {
		t.Fatalf("unlisted viewer: %d %v", status, body)
	}
	for _, c := range []struct {
		access, path string
		status       int
		err          string
	}{
		{"", "/leaderboard/1v1", http.StatusUnauthorized, "access_rejected"},
		{"expired", "/leaderboard/1v1", http.StatusUnauthorized, "access_expired"},
		{"access:second", "/leaderboard/3v3", http.StatusBadRequest, "mode_invalid"},
		{"access:second", "/leaderboard/1v1?board=moon", http.StatusNotFound, "leaderboard.unknown_board"},
		{"access:second", "/leaderboard/1v1?season=season.down", http.StatusServiceUnavailable, "leaderboard.unavailable"},
		{"access:second", "/leaderboard/1v1?season=season.broken", http.StatusInternalServerError, "internal"},
	} {
		if status, body := call(c.access, c.path); status != c.status || body["error"] != c.err {
			t.Errorf("%s %s: %d %v, want %d %s", c.access, c.path, status, body, c.status, c.err)
		}
	}
	clock = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	if status, body := call("access:second", "/leaderboard/1v1"); status != http.StatusNotFound || body["error"] != "rating.season_unknown" {
		t.Errorf("no season: %d %v", status, body)
	}
	disabled := leaderboard.NewHandler(fakeVerifier{}, nil, fakeSeasons{}, slog.New(slog.NewTextHandler(io.Discard, nil)), func() time.Time { return clock })
	request := httptest.NewRequest(http.MethodGet, "/leaderboard/1v1", nil)
	request.Header.Set("Authorization", "Bearer access:second")
	recorder := httptest.NewRecorder()
	disabled.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Errorf("without redis: %d", recorder.Code)
	}
}

package profile_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/account"
	"github.com/Alievs-corp/hypedomino/server/internal/profile"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

type fakeVerifier struct{}

func (fakeVerifier) Verify(access string) (token.Claims, error) {
	switch {
	case access == "expired":
		return token.Claims{}, token.ErrExpired
	case strings.HasPrefix(access, "access:"):
		return token.Claims{Account: strings.TrimPrefix(access, "access:"), Session: "sid"}, nil
	}
	return token.Claims{}, token.ErrSignature
}

type fakeProfiles struct {
	config   profile.Config
	profiles map[string]profile.Profile
	taken    map[string]bool
	now      time.Time
}

func (f *fakeProfiles) Get(_ context.Context, account string) (profile.Profile, bool, error) {
	if account == "broken" {
		return profile.Profile{}, false, errors.New("database down")
	}
	p, ok := f.profiles[account]
	return p, ok, nil
}

func (f *fakeProfiles) Exists(_ context.Context, account string) (bool, error) {
	return account != "nobody", nil
}

func (f *fakeProfiles) SetNickname(_ context.Context, account, raw string, now time.Time) (profile.Profile, error) {
	display := profile.Display(raw)
	if err := f.config.Validate(display); err != nil {
		return profile.Profile{}, err
	}
	if f.taken[profile.Key(display)] {
		return profile.Profile{}, profile.ErrTaken
	}
	if display == "Banned" {
		return profile.Profile{}, profile.ErrNotAllowed
	}
	p, found := f.profiles[account]
	if found && !p.RenamedAt.IsZero() && now.Before(p.RenamedAt.Add(f.config.NicknameRenameCooldown)) {
		return profile.Profile{}, fmt.Errorf("%w until %s", profile.ErrCooldown, p.RenamedAt.Add(f.config.NicknameRenameCooldown).UTC().Format(time.RFC3339))
	}
	if found {
		p.RenamedAt = now
	} else {
		p = profile.Profile{Account: account}
	}
	p.Nickname = display
	f.profiles[account] = p
	return p, nil
}

func (f *fakeProfiles) Suggest(context.Context, time.Time) (string, error) {
	return "Player123456", nil
}

func (f *fakeProfiles) RenameAvailableAt(p profile.Profile) (time.Time, bool) {
	if p.RenamedAt.IsZero() {
		return time.Time{}, false
	}
	return p.RenamedAt.Add(f.config.NicknameRenameCooldown), true
}

type fixture struct {
	t        *testing.T
	server   *httptest.Server
	profiles *fakeProfiles
	handler  *profile.Handler
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	profiles := &fakeProfiles{config: profile.DefaultConfig(), profiles: map[string]profile.Profile{}, taken: map[string]bool{}, now: start}
	profiles.profiles["ben"] = profile.Profile{
		Account: "ben", Nickname: "Ben",
		Loadout: profile.Loadout{Avatar: "avatar.comet_fox", Frame: "frame.gold", Table: "table.classic", Emojis: []string{"emoji.wave"}},
		Stats:   profile.Stats{Games: 10, Wins: 7, Losses: 3, Streak: 2, BestStreak: 4, Modes: map[string]profile.ModeStats{"1v1": {Games: 10, Wins: 7, Losses: 3}}},
	}
	handler := profile.NewHandler(fakeVerifier{}, profiles, slog.New(slog.NewTextHandler(io.Discard, nil)), func() time.Time { return profiles.now })
	handler.SetStanding(fakeStanding)
	mux := http.NewServeMux()
	mux.Handle("/profile/", handler)
	f := &fixture{t: t, server: httptest.NewServer(mux), profiles: profiles, handler: handler}
	t.Cleanup(f.server.Close)
	return f
}

func (f *fixture) call(method, path, bearer, body string) (int, map[string]any) {
	f.t.Helper()
	request, err := http.NewRequest(method, f.server.URL+path, strings.NewReader(body))
	if err != nil {
		f.t.Fatal(err)
	}
	if bearer != "" {
		request.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		f.t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.Header.Get("Cache-Control") != "no-store" {
		f.t.Fatalf("headers %v", resp.Header)
	}
	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		f.t.Fatal(err)
	}
	return resp.StatusCode, decoded
}

func TestEveryRouteNeedsAValidAccessToken(t *testing.T) {
	f := newFixture(t)
	for _, route := range []struct{ method, path string }{{"GET", "/profile/me"}, {"PUT", "/profile/nickname"}, {"GET", "/profile/ben"}} {
		if status, body := f.call(route.method, route.path, "", `{"nickname":"Ayla"}`); status != http.StatusUnauthorized || body["error"] != "access_rejected" {
			t.Fatalf("%s %s without a token answered %d %v", route.method, route.path, status, body)
		}
		if status, body := f.call(route.method, route.path, "expired", `{"nickname":"Ayla"}`); status != http.StatusUnauthorized || body["error"] != "access_expired" {
			t.Fatalf("%s %s with an expired token answered %d %v", route.method, route.path, status, body)
		}
	}
}

func TestANewAccountIsAskedForANicknameWithASuggestion(t *testing.T) {
	f := newFixture(t)
	status, body := f.call("GET", "/profile/me", "access:ayla", "")
	if status != http.StatusOK || body["setup_required"] != true || body["suggested_nickname"] != "Player123456" || body["nickname"] != "" || body["account_id"] != "ayla" || body["level"] != float64(1) {
		t.Fatalf("me before setup answered %d %v", status, body)
	}
	status, body = f.call("PUT", "/profile/nickname", "access:ayla", `{"nickname":" Ayla "}`)
	if status != http.StatusOK || body["nickname"] != "Ayla" || body["setup_required"] != false || body["rename_available_at"] != nil {
		t.Fatalf("setup answered %d %v", status, body)
	}
	status, body = f.call("GET", "/profile/me", "access:ayla", "")
	if status != http.StatusOK || body["nickname"] != "Ayla" || body["setup_required"] != false || body["suggested_nickname"] != nil {
		t.Fatalf("me after setup answered %d %v", status, body)
	}
}

func TestAForcedRenameCarriesItsAppealGrantOnlyWhileRequired(t *testing.T) {
	f := newFixture(t)
	f.profiles.profiles["rude"] = profile.Profile{Account: "rude", Nickname: "Rude", RenameRequired: true}
	f.handler.SetRenameAppeals(func(_ context.Context, account string) (any, error) {
		return map[string]any{"sanction_id": 9, "token": "appeal:" + account}, nil
	})
	status, body := f.call("GET", "/profile/me", "access:rude", "")
	if status != http.StatusOK || !reflect.DeepEqual(body["rename_appeal"], map[string]any{"sanction_id": float64(9), "token": "appeal:rude"}) {
		t.Fatalf("me during a forced rename answered %d %v", status, body)
	}
	status, body = f.call("GET", "/profile/me", "access:ben", "")
	if _, carried := body["rename_appeal"]; status != http.StatusOK || carried {
		t.Fatalf("me without a forced rename answered %d %v", status, body)
	}
	f.handler.SetRenameAppeals(func(context.Context, string) (any, error) { return nil, errors.New("down") })
	if status, _ := f.call("GET", "/profile/me", "access:rude", ""); status != http.StatusInternalServerError {
		t.Fatalf("a failed grant answered %d", status)
	}
}

func TestNicknameRefusalsCarryTheirReason(t *testing.T) {
	f := newFixture(t)
	f.profiles.taken[profile.Key("Taken")] = true
	cases := []struct {
		body   string
		status int
		reason string
	}{
		{`{"nickname":`, http.StatusBadRequest, "malformed"},
		{`{"nickname":"ab"}`, http.StatusBadRequest, "nickname.invalid_length"},
		{`{"nickname":"Ayla!"}`, http.StatusBadRequest, "nickname.invalid_characters"},
		{`{"nickname":"Café"}`, http.StatusBadRequest, "nickname.invalid_characters"},
		{`{"nickname":"Ae\u0301lla"}`, http.StatusBadRequest, "nickname.invalid_characters"},
		{`{"nickname":"Straße"}`, http.StatusBadRequest, "nickname.invalid_characters"},
		{`{"nickname":"Іван"}`, http.StatusBadRequest, "nickname.invalid_characters"},
		{`{"nickname":"_Ayla"}`, http.StatusBadRequest, "nickname.invalid_separators"},
		{`{"nickname":"Иvan"}`, http.StatusBadRequest, "nickname.mixed_scripts"},
		{`{"nickname":"12345"}`, http.StatusBadRequest, "nickname.digits_only"},
		{`{"nickname":"TAKEN"}`, http.StatusConflict, "nickname.taken"},
		{`{"nickname":"Banned"}`, http.StatusBadRequest, "nickname.not_allowed"},
	}
	for _, c := range cases {
		if status, body := f.call("PUT", "/profile/nickname", "access:ayla", c.body); status != c.status || body["error"] != c.reason {
			t.Fatalf("%s answered %d %v, want %d %s", c.body, status, body, c.status, c.reason)
		}
	}
	if status, _ := f.call("PUT", "/profile/nickname", "access:ayla", `{"nickname":"Ayla"}`); status != http.StatusOK {
		t.Fatalf("setup answered %d", status)
	}
	if status, _ := f.call("PUT", "/profile/nickname", "access:ayla", `{"nickname":"Ayla_1"}`); status != http.StatusOK {
		t.Fatalf("the free rename answered %d", status)
	}
	f.profiles.now = start.Add(time.Hour)
	status, body := f.call("PUT", "/profile/nickname", "access:ayla", `{"nickname":"Ayla_2"}`)
	if status != http.StatusTooManyRequests || body["error"] != "nickname.cooldown" || body["retry_at"] != "2026-11-29T12:00:00Z" {
		t.Fatalf("rename inside the cooldown answered %d %v", status, body)
	}
	if status, body := f.call("GET", "/profile/me", "access:ayla", ""); status != http.StatusOK || body["rename_available_at"] != "2026-11-29T12:00:00Z" {
		t.Fatalf("me inside the cooldown answered %d %v", status, body)
	}
}

func TestThePublicProfileShowsOnlyAllowedFields(t *testing.T) {
	f := newFixture(t)
	status, body := f.call("GET", "/profile/ben", "access:ayla", "")
	if status != http.StatusOK || body["nickname"] != "Ben" || body["level"] != float64(7) || body["xp"] != nil || body["setup_required"] != nil {
		t.Fatalf("public profile answered %d %v", status, body)
	}
	stats := body["stats"].(map[string]any)
	if stats["games"] != float64(10) || stats["win_rate_percent"] != float64(70) || stats["best_streak"] != float64(4) || stats["wins"] != nil || stats["streak"] != nil || stats["modes"] != nil {
		t.Fatalf("public stats %v", stats)
	}
	loadout := body["loadout"].(map[string]any)
	if loadout["avatar"] != "avatar.comet_fox" || loadout["table"] != "table.classic" {
		t.Fatalf("public loadout %v", loadout)
	}
	if status, body := f.call("GET", "/profile/nobody", "access:ayla", ""); status != http.StatusNotFound || body["error"] != "profile.unknown" {
		t.Fatalf("unknown profile answered %d %v", status, body)
	}
	if status, body := f.call("GET", "/profile/broken", "access:ayla", ""); status != http.StatusInternalServerError || body["error"] != "internal" {
		t.Fatalf("failing store answered %d %v", status, body)
	}
	status, body = f.call("GET", "/profile/me", "access:ben", "")
	own := body["stats"].(map[string]any)
	if status != http.StatusOK || body["xp"] != float64(1200) || own["wins"] != float64(7) || own["streak"] != float64(2) || own["modes"].(map[string]any)["1v1"] == nil {
		t.Fatalf("own profile answered %d %v", status, body)
	}
}

type fakeCountries struct {
	codes   map[string]account.Country
	changed time.Time
}

func (f *fakeCountries) Country(_ context.Context, id string) (account.Country, error) {
	c, ok := f.codes[id]
	if !ok {
		return account.Country{}, account.ErrUnknown
	}
	return c, nil
}

func (f *fakeCountries) SetCountry(_ context.Context, id, code string, now time.Time, cooldown time.Duration) (account.Country, error) {
	switch {
	case !account.ValidCountry(code):
		return account.Country{}, account.ErrCountryInvalid
	case id == "cooling":
		return account.Country{}, account.ErrCountryCooldown
	}
	c := account.Country{Code: code, ChangedAt: now}
	f.codes[id] = c
	return c, nil
}

func TestTheCountryRoutesReadAndChangeTheAccountCountry(t *testing.T) {
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	countries := &fakeCountries{codes: map[string]account.Country{"acc-1": {Code: "AZ"}, "cooling": {Code: "TR", ChangedAt: now.Add(-time.Hour)}}}
	handler := profile.NewHandler(fakeVerifier{}, &fakeProfiles{profiles: map[string]profile.Profile{}, taken: map[string]bool{}, now: now}, slog.New(slog.NewTextHandler(io.Discard, nil)), func() time.Time { return now })
	handler.SetCountries(countries, 24*time.Hour)
	call := func(method, access, body string) (int, map[string]any) {
		t.Helper()
		request := httptest.NewRequest(method, "/profile/country", strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer "+access)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		var decoded map[string]any
		json.Unmarshal(recorder.Body.Bytes(), &decoded)
		return recorder.Code, decoded
	}
	if status, body := call(http.MethodGet, "access:acc-1", ""); status != http.StatusOK || body["country"] != "AZ" || body["change_available_at"] != nil {
		t.Fatalf("get: %d %v", status, body)
	}
	if status, body := call(http.MethodGet, "access:cooling", ""); status != http.StatusOK || body["country"] != "TR" || body["change_available_at"] != now.Add(23*time.Hour).Format(time.RFC3339) {
		t.Fatalf("get cooling: %d %v", status, body)
	}
	if status, body := call(http.MethodPut, "access:acc-1", `{"country":" tr "}`); status != http.StatusOK || body["country"] != "TR" || body["change_available_at"] != now.Add(24*time.Hour).Format(time.RFC3339) {
		t.Fatalf("put: %d %v", status, body)
	}
	for _, c := range []struct {
		access, body string
		status       int
		err          string
	}{
		{"access:acc-1", `{"country":"Azerbaijan"}`, http.StatusBadRequest, "country.invalid"},
		{"access:acc-1", `{"country":`, http.StatusBadRequest, "malformed"},
		{"access:cooling", `{"country":"DE"}`, http.StatusConflict, "country.cooldown"},
		{"access:ghost", `{"country":"DE"}`, http.StatusOK, ""},
	} {
		if status, body := call(http.MethodPut, c.access, c.body); status != c.status || (c.err != "" && body["error"] != c.err) {
			t.Errorf("%s %s: %d %v, want %d %s", c.access, c.body, status, body, c.status, c.err)
		}
	}
	if status, body := call(http.MethodGet, "access:nobody", ""); status != http.StatusNotFound || body["error"] != "not_found" {
		t.Fatalf("unknown account: %d %v", status, body)
	}
	bare := profile.NewHandler(fakeVerifier{}, &fakeProfiles{profiles: map[string]profile.Profile{}, taken: map[string]bool{}, now: now}, slog.New(slog.NewTextHandler(io.Discard, nil)), func() time.Time { return now })
	request := httptest.NewRequest(http.MethodGet, "/profile/country", nil)
	request.Header.Set("Authorization", "Bearer access:acc-1")
	recorder := httptest.NewRecorder()
	bare.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("without countries: %d", recorder.Code)
	}
}

func TestThePublicProfileCarriesTheRankedAndMedalViewsWhenWired(t *testing.T) {
	f := newFixture(t)
	if _, body := f.call("GET", "/profile/ben", "access:ayla", ""); body["ranked"] != nil {
		t.Fatalf("unwired ranked view %v", body["ranked"])
	}
	f.handler.SetRanked(func(_ context.Context, account string) (any, error) {
		if account == "ben" {
			return map[string]any{"ranks": map[string]string{"1v1": "Silver I"}}, nil
		}
		return nil, errors.New("down")
	})
	status, body := f.call("GET", "/profile/ben", "access:ayla", "")
	ranked, _ := body["ranked"].(map[string]any)
	if status != http.StatusOK || ranked["ranks"].(map[string]any)["1v1"] != "Silver I" {
		t.Fatalf("ranked view answered %d %v", status, body)
	}
	f.handler.SetMedals(func(context.Context, string) (any, error) { return []string{"medal.wins_50"}, nil })
	if _, body := f.call("GET", "/profile/ben", "access:ayla", ""); body["medals"].([]any)[0] != "medal.wins_50" {
		t.Fatalf("medal showcase %v", body)
	}
	f.handler.SetMedals(func(context.Context, string) (any, error) { return nil, errors.New("down") })
	if status, _ := f.call("GET", "/profile/ben", "access:ayla", ""); status != http.StatusInternalServerError {
		t.Fatalf("failing medal showcase answered %d", status)
	}
}

func fakeStanding(_ context.Context, account string, now time.Time) (int, int64, error) {
	switch {
	case now.IsZero():
		return 0, 0, errors.New("no server time")
	case account == "ben":
		return 7, 1200, nil
	case account == "stalled":
		return 0, 0, errors.New("pass store down")
	}
	return 1, 0, nil
}

func TestLevelAndXpAreThePassStandingOfTheActiveSeason(t *testing.T) {
	f := newFixture(t)
	if status, body := f.call("GET", "/profile/me", "access:ben", ""); status != http.StatusOK || body["level"] != float64(7) || body["xp"] != float64(1200) {
		t.Fatalf("own profile answered %d %v", status, body)
	}
	if status, body := f.call("PUT", "/profile/nickname", "access:ben", `{"nickname":"Benny"}`); status != http.StatusOK || body["level"] != float64(7) || body["xp"] != float64(1200) {
		t.Fatalf("rename answered %d %v", status, body)
	}
	if status, body := f.call("GET", "/profile/ben", "access:ayla", ""); status != http.StatusOK || body["level"] != float64(7) || body["xp"] != nil {
		t.Fatalf("public profile answered %d %v", status, body)
	}
	f.profiles.profiles["stalled"] = profile.Profile{Account: "stalled", Nickname: "Stalled"}
	for _, path := range []string{"/profile/stalled", "/profile/me"} {
		if status, body := f.call("GET", path, "access:stalled", ""); status != http.StatusInternalServerError || body["error"] != "internal" {
			t.Fatalf("%s with a failing pass store answered %d %v", path, status, body)
		}
	}
}

func TestWithoutAPassStoreLevelIsOneWithZeroXp(t *testing.T) {
	f := newFixture(t)
	f.handler.SetStanding(nil)
	if status, body := f.call("GET", "/profile/me", "access:ben", ""); status != http.StatusOK || body["level"] != float64(1) || body["xp"] != float64(0) {
		t.Fatalf("own profile answered %d %v", status, body)
	}
}

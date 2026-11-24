package profile_test

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

	"github.com/Alievs-corp/hypedomino/server/internal/account"
	"github.com/Alievs-corp/hypedomino/server/internal/profile"
)

type fakeLanguages struct {
	codes map[string]string
}

func (f *fakeLanguages) Language(_ context.Context, id string) (string, error) {
	code, ok := f.codes[id]
	if !ok {
		return "", account.ErrUnknown
	}
	return code, nil
}

func (f *fakeLanguages) SetLanguage(_ context.Context, id, code string) (string, error) {
	if id == "broken" {
		return "", errors.New("down")
	}
	if !account.ValidLanguage(code) {
		return "", account.ErrLanguageInvalid
	}
	if _, ok := f.codes[id]; !ok {
		return "", account.ErrUnknown
	}
	f.codes[id] = code
	return code, nil
}

func TestTheLanguageRoutesReadAndChangeTheAccountLanguage(t *testing.T) {
	now := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	languages := &fakeLanguages{codes: map[string]string{"acc-1": "", "acc-2": "ru", "broken": "az"}}
	handler := profile.NewHandler(fakeVerifier{}, &fakeProfiles{profiles: map[string]profile.Profile{}, taken: map[string]bool{}, now: now}, slog.New(slog.NewTextHandler(io.Discard, nil)), func() time.Time { return now })
	handler.SetLanguages(languages)
	call := func(method, access, body string) (int, map[string]any) {
		t.Helper()
		request := httptest.NewRequest(method, "/profile/language", strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer "+access)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		var decoded map[string]any
		json.Unmarshal(recorder.Body.Bytes(), &decoded)
		return recorder.Code, decoded
	}
	if status, body := call(http.MethodGet, "access:acc-1", ""); status != http.StatusOK || body["language"] != "" {
		t.Fatalf("get without a language: %d %v", status, body)
	}
	if status, body := call(http.MethodGet, "access:acc-2", ""); status != http.StatusOK || body["language"] != "ru" {
		t.Fatalf("get: %d %v", status, body)
	}
	if status, body := call(http.MethodPut, "access:acc-1", `{"language":" TR "}`); status != http.StatusOK || body["language"] != "tr" || languages.codes["acc-1"] != "tr" {
		t.Fatalf("put: %d %v", status, body)
	}
	for _, c := range []struct {
		access, body string
		status       int
		err          string
	}{
		{"access:acc-1", `{"language":"de"}`, http.StatusBadRequest, "language.invalid"},
		{"access:acc-1", `{"language":""}`, http.StatusBadRequest, "language.invalid"},
		{"access:acc-1", `{"language":`, http.StatusBadRequest, "malformed"},
		{"access:ghost", `{"language":"az"}`, http.StatusNotFound, "not_found"},
		{"access:broken", `{"language":"az"}`, http.StatusInternalServerError, "internal"},
		{"expired", `{"language":"az"}`, http.StatusUnauthorized, "access_expired"},
	} {
		if status, body := call(http.MethodPut, c.access, c.body); status != c.status || body["error"] != c.err {
			t.Errorf("%s %s: %d %v, want %d %s", c.access, c.body, status, body, c.status, c.err)
		}
	}
	if languages.codes["acc-1"] != "tr" {
		t.Fatalf("a refused change was stored: %q", languages.codes["acc-1"])
	}
	if status, body := call(http.MethodGet, "access:nobody", ""); status != http.StatusNotFound || body["error"] != "not_found" {
		t.Fatalf("unknown account: %d %v", status, body)
	}
	bare := profile.NewHandler(fakeVerifier{}, &fakeProfiles{profiles: map[string]profile.Profile{}, taken: map[string]bool{}, now: now}, slog.New(slog.NewTextHandler(io.Discard, nil)), func() time.Time { return now })
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		request := httptest.NewRequest(method, "/profile/language", strings.NewReader(`{"language":"az"}`))
		request.Header.Set("Authorization", "Bearer access:acc-1")
		recorder := httptest.NewRecorder()
		bare.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("%s without languages: %d", method, recorder.Code)
		}
	}
}

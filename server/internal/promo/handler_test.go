package promo_test

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

	"github.com/Alievs-corp/hypedomino/server/internal/inventory"
	"github.com/Alievs-corp/hypedomino/server/internal/promo"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
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

type fakePromos struct {
	calls []string
}

var outcomes = map[string]error{
	"UNKNWN22": promo.ErrUnknown, "PENDNG22": promo.ErrNotActive, "DSABED22": promo.ErrDisabled, "EARY2345": promo.ErrNotStarted,
	"ATE23456": promo.ErrExpired, "GNE23456": promo.ErrExhausted, "AGAN2345": promo.ErrAlreadyRedeemed, "VP234567": promo.ErrNotEligible,
	"FAST2345": promo.ErrRateLimited, "CKED2345": promo.ErrLocked, "UNSTED22": inventory.ErrSourceNotListed, "BRKEN234": errors.New("database down"),
	"NSEASN22": promo.ErrNoPassSeason, "WNED2345": promo.ErrPassOwned,
}

func (f *fakePromos) Redeem(_ context.Context, account, code string, at time.Time) (promo.Redemption, error) {
	f.calls = append(f.calls, account+" "+code+" "+at.Format(time.RFC3339))
	normalized, err := promo.Normalize(code)
	if err != nil {
		return promo.Redemption{}, err
	}
	if err := outcomes[normalized]; err != nil {
		return promo.Redemption{}, err
	}
	if normalized == "SPKE2345" {
		return promo.Redemption{Promo: "promo.leaky", Campaign: "campaign.launch_week", Spiked: true}, nil
	}
	if normalized == "PREM2345" {
		return promo.Redemption{Promo: "promo.backer_sky", Campaign: "campaign.ideyanest_2026", Grant: reward.Report{Coins: 300}, PassPremium: true}, nil
	}
	return promo.Redemption{Promo: "promo.welcome", Campaign: "campaign.launch_week", Grant: reward.Report{Coins: 300}}, nil
}

func TestTheReplySaysWhetherThePremiumPassWasUnlocked(t *testing.T) {
	handler := promo.NewHandler(fakeVerifier{}, &fakePromos{}, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Now)
	for _, c := range []struct {
		code, promo string
		premium     bool
	}{
		{"PREM2345", "promo.backer_sky", true},
		{"prem-2345", "promo.backer_sky", true},
		{"HYPE2345", "promo.welcome", false},
		{"SPKE2345", "promo.leaky", false},
	} {
		request := httptest.NewRequest(http.MethodPost, "/promo/redeem", strings.NewReader(`{"code":"`+c.code+`"}`))
		request.Header.Set("Authorization", "Bearer access:acc-1")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		var body map[string]any
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s: %v", c.code, err)
		}
		if premium, present := body["pass_premium"].(bool); recorder.Code != http.StatusOK || body["promo"] != c.promo || !present || premium != c.premium {
			t.Errorf("%s: %d %s, want %s with pass_premium %t", c.code, recorder.Code, recorder.Body.String(), c.promo, c.premium)
		}
	}
}

func TestTheHandlerRedeemsAndMapsEveryRefusal(t *testing.T) {
	promos := &fakePromos{}
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	handler := promo.NewHandler(fakeVerifier{}, promos, slog.New(slog.NewTextHandler(io.Discard, nil)), func() time.Time { return now })
	call := func(access, body string) (int, map[string]any) {
		t.Helper()
		request := httptest.NewRequest(http.MethodPost, "/promo/redeem", strings.NewReader(body))
		if access != "" {
			request.Header.Set("Authorization", "Bearer "+access)
		}
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		var decoded map[string]any
		json.Unmarshal(recorder.Body.Bytes(), &decoded)
		return recorder.Code, decoded
	}
	if status, body := call("access:acc-1", `{"code":"hype-2345"}`); status != http.StatusOK || body["promo"] != "promo.welcome" || body["grant"].(map[string]any)["coins"] != float64(300) {
		t.Fatalf("redeem: %d %v", status, body)
	}
	if len(promos.calls) != 1 || promos.calls[0] != "acc-1 hype-2345 2026-10-30T12:00:00Z" {
		t.Fatalf("calls %v", promos.calls)
	}
	for _, c := range []struct {
		access, body string
		status       int
		err          string
	}{
		{"", `{"code":"HYPE2345"}`, http.StatusUnauthorized, "access_rejected"},
		{"expired", `{"code":"HYPE2345"}`, http.StatusUnauthorized, "access_expired"},
		{"access:acc-1", `{"code":`, http.StatusBadRequest, "malformed"},
		{"access:acc-1", `{"code":"bad!"}`, http.StatusBadRequest, "promo.code_invalid"},
		{"access:acc-1", `{"code":"UNKNWN22"}`, http.StatusNotFound, "promo.unknown"},
		{"access:acc-1", `{"code":"PENDNG22"}`, http.StatusConflict, "promo.not_active"},
		{"access:acc-1", `{"code":"DSABED22"}`, http.StatusConflict, "promo.disabled"},
		{"access:acc-1", `{"code":"EARY2345"}`, http.StatusConflict, "promo.not_started"},
		{"access:acc-1", `{"code":"ATE23456"}`, http.StatusConflict, "promo.expired"},
		{"access:acc-1", `{"code":"GNE23456"}`, http.StatusConflict, "promo.exhausted"},
		{"access:acc-1", `{"code":"AGAN2345"}`, http.StatusConflict, "promo.already_redeemed"},
		{"access:acc-1", `{"code":"UNSTED22"}`, http.StatusConflict, "item.source_not_listed"},
		{"access:acc-1", `{"code":"NSEASN22"}`, http.StatusConflict, "promo.no_pass_season"},
		{"access:acc-1", `{"code":"WNED2345"}`, http.StatusConflict, "promo.pass_premium_owned"},
		{"access:acc-1", `{"code":"VP234567"}`, http.StatusForbidden, "promo.not_eligible"},
		{"access:acc-1", `{"code":"FAST2345"}`, http.StatusTooManyRequests, "promo.rate_limited"},
		{"access:acc-1", `{"code":"CKED2345"}`, http.StatusTooManyRequests, "promo.locked"},
		{"access:acc-1", `{"code":"BRKEN234"}`, http.StatusInternalServerError, "internal"},
	} {
		if status, body := call(c.access, c.body); status != c.status || body["error"] != c.err {
			t.Errorf("%s %s: %d %v, want %d %s", c.access, c.body, status, body, c.status, c.err)
		}
	}
	disabled := promo.NewHandler(fakeVerifier{}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), func() time.Time { return now })
	request := httptest.NewRequest(http.MethodPost, "/promo/redeem", strings.NewReader(`{"code":"HYPE2345"}`))
	request.Header.Set("Authorization", "Bearer access:acc-1")
	recorder := httptest.NewRecorder()
	disabled.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable || !strings.Contains(recorder.Body.String(), "promo.unavailable") {
		t.Fatalf("without a key: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestASpikeIsLoggedOnceAndNeverShownToThePlayer(t *testing.T) {
	logs := &strings.Builder{}
	handler := promo.NewHandler(fakeVerifier{}, &fakePromos{}, slog.New(slog.NewTextHandler(logs, nil)), time.Now)
	for _, code := range []string{"SPKE2345", "HYPE2345"} {
		request := httptest.NewRequest(http.MethodPost, "/promo/redeem", strings.NewReader(`{"code":"`+code+`"}`))
		request.Header.Set("Authorization", "Bearer access:acc-1")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK || strings.Contains(recorder.Body.String(), "pike") {
			t.Fatalf("%s answered %d %s", code, recorder.Code, recorder.Body.String())
		}
	}
	if strings.Count(logs.String(), "promo redemption spike") != 1 || !strings.Contains(logs.String(), "promo=promo.leaky") {
		t.Fatalf("logs %q", logs.String())
	}
}

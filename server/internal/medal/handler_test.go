package medal_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/medal"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

type acceptingVerifier struct{}

func (acceptingVerifier) Verify(access string) (token.Claims, error) {
	return token.Claims{Account: strings.TrimPrefix(access, "access:")}, nil
}

type showcaseStore struct{}

func (showcaseStore) Collection(context.Context, string) (medal.Collection, error) {
	return medal.Collection{}, nil
}

func (showcaseStore) SetShowcase(_ context.Context, _ string, medals []string, _ time.Time) ([]string, error) {
	if len(medals) == 1 && medals[0] == "medal.locked" {
		return nil, medal.ErrNotOwned
	}
	if len(medals) > 3 {
		return nil, medal.ErrShowcase
	}
	return medals, nil
}

type observedAccounts struct {
	accounts []string
}

func (o *observedAccounts) Observe(_ context.Context, account string) {
	o.accounts = append(o.accounts, account)
}

func TestShowcaseOfAnUnownedMedalIsObserved(t *testing.T) {
	handler := medal.NewHandler(acceptingVerifier{}, showcaseStore{}, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Now)
	observed := &observedAccounts{}
	handler.FlagUnowned(observed)
	for _, c := range []struct {
		body   string
		status int
	}{
		{`{"medals":["medal.locked"]}`, http.StatusConflict},
		{`{"medals":["a","b","c","d"]}`, http.StatusBadRequest},
		{`{"medals":["medal.domino_100"]}`, http.StatusOK},
	} {
		request := httptest.NewRequest(http.MethodPut, "/medals/showcase", strings.NewReader(c.body))
		request.Header.Set("Authorization", "Bearer access:ayla")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != c.status {
			t.Fatalf("%s answered %d %s", c.body, recorder.Code, recorder.Body.String())
		}
	}
	if len(observed.accounts) != 1 || observed.accounts[0] != "ayla" {
		t.Fatalf("observed %v, want only the not_owned refusal", observed.accounts)
	}
}

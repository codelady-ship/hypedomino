package adminapi

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/mission"
)

func discardingHandler() *Handler {
	return &Handler{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

func TestARefusalForUnnamedKeysListsThemInKeys(t *testing.T) {
	refusal := unapprovedKeysError{refusal: fmt.Errorf("%w: no approved en name for mission.a.name, mission.b.name", mission.ErrInvalidCatalog), keys: []string{"mission.a.name", "mission.b.name"}}
	for name, err := range map[string]error{"direct": refusal, "wrapped": fmt.Errorf("publish: %w", refusal)} {
		recorder := httptest.NewRecorder()
		discardingHandler().refuse(recorder, err)
		var body errorResponse
		if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if recorder.Code != http.StatusBadRequest || body.Error != "invalid" || body.Detail != err.Error() || !slices.Equal(body.Keys, []string{"mission.a.name", "mission.b.name"}) {
			t.Errorf("%s: %d %+v", name, recorder.Code, body)
		}
	}
}

func TestAnotherRefusalHasNoKeys(t *testing.T) {
	recorder := httptest.NewRecorder()
	discardingHandler().refuse(recorder, fmt.Errorf("%w: start must be next_start", mission.ErrInvalidCatalog))
	var body map[string]any
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if _, listed := body["keys"]; listed || body["error"] != "invalid" {
		t.Errorf("%d %v", recorder.Code, body)
	}
}

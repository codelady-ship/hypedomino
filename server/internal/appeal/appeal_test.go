package appeal_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/appeal"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/liveops"
	"github.com/Alievs-corp/hypedomino/server/internal/sanction"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

var now = time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)

func TestOnlyBansAndForcedRenamesAreAppealable(t *testing.T) {
	cases := []struct {
		kind       sanction.Kind
		appealable bool
	}{
		{sanction.TemporaryBan, true},
		{sanction.PermanentBan, true},
		{sanction.RankedBan, true},
		{sanction.PenaltyBan, true},
		{sanction.ForcedRename, true},
		{sanction.Kind("shadowban"), false},
		{sanction.Kind("mute"), false},
		{sanction.Kind(""), false},
	}
	for _, c := range cases {
		if got := appeal.Appealable(c.kind); got != c.appealable {
			t.Errorf("%q appealable %v", c.kind, got)
		}
	}
}

func TestAnAppealTextIsTrimmedAndBoundedInCharacters(t *testing.T) {
	cases := []struct {
		raw  string
		want string
		err  error
	}{
		{"  it was not me  ", "it was not me", nil},
		{strings.Repeat("ə", 50), strings.Repeat("ə", 50), nil},
		{strings.Repeat("ə", 51), "", appeal.ErrTextTooLong},
		{"   ", "", appeal.ErrTextRequired},
		{"", "", appeal.ErrTextRequired},
		{"bad \xff byte", "", appeal.ErrTextTooLong},
	}
	for _, c := range cases {
		got, err := appeal.CleanText(c.raw, 50)
		if got != c.want || !errors.Is(err, c.err) {
			t.Errorf("%q: %q %v", c.raw, got, err)
		}
	}
}

func TestOneAppealPerSanctionWhileTheSanctionRuns(t *testing.T) {
	ends := now.Add(time.Hour)
	ended := now.Add(-time.Minute)
	running := sanction.Sanction{ID: 1, Kind: sanction.RankedBan, EndsAt: &ends}
	cases := []struct {
		name     string
		appealed sanction.Sanction
		existing string
		want     error
	}{
		{"first appeal", running, "", nil},
		{"permanent ban", sanction.Sanction{ID: 2, Kind: sanction.PermanentBan}, "", nil},
		{"an open appeal", running, appeal.StateOpen, appeal.ErrOpen},
		{"a rejected appeal", running, appeal.StateUpheld, appeal.ErrRejected},
		{"a lifted sanction", sanction.Sanction{ID: 3, Kind: sanction.RankedBan, EndsAt: &ends, LiftedAt: &ended}, appeal.StateLifted, appeal.ErrEnded},
		{"a lifted appeal of a sanction still running", running, appeal.StateLifted, appeal.ErrEnded},
		{"an ended sanction", sanction.Sanction{ID: 4, Kind: sanction.TemporaryBan, EndsAt: &ended}, "", appeal.ErrEnded},
		{"ending now", sanction.Sanction{ID: 5, Kind: sanction.TemporaryBan, EndsAt: &now}, "", appeal.ErrEnded},
		{"a shadowban", sanction.Sanction{ID: 6, Kind: sanction.Kind("shadowban")}, "", appeal.ErrNotAppealable},
	}
	for _, c := range cases {
		if err := appeal.Admissible(c.appealed, c.existing, now); !errors.Is(err, c.want) {
			t.Errorf("%s: %v, want %v", c.name, err, c.want)
		}
	}
}

type fakeSigner struct{}

func (fakeSigner) IssueAppeal(account string, sanctionID int64, ttl time.Duration) (string, time.Time, error) {
	return account + ":" + ttl.String(), now.Add(ttl), nil
}

func TestAGrantCarriesTheSanctionAndATokenOfTheConfiguredLifetime(t *testing.T) {
	granter := appeal.NewGranter(fakeSigner{}, func() liveops.Appeals { return liveops.Values{liveops.AppealTokenMinutes: 10}.Appeals() })
	grant, err := granter.Grant("acc-1", 41)
	if err != nil || grant.SanctionID != 41 || grant.Token != "acc-1:10m0s" || !grant.ExpiresAt.Equal(now.Add(10*time.Minute)) {
		t.Fatalf("grant %+v %v", grant, err)
	}
	if proto := grant.Proto(); proto.GetSanctionId() != 41 || proto.GetToken() != grant.Token || proto.GetExpiresUnixMs() != uint64(now.Add(10*time.Minute).UnixMilli()) {
		t.Fatalf("proto %v", proto)
	}
	for name, g := range map[string]*appeal.Granter{"no sanction": granter, "no granter": nil} {
		id := int64(0)
		if g == nil {
			id = 41
		}
		if grant, err := g.Grant("acc-1", id); grant != nil || err != nil || grant.Proto() != nil {
			t.Errorf("%s: %+v %v", name, grant, err)
		}
	}
}

type fakeVerifier struct{}

func (fakeVerifier) VerifyAppeal(raw string) (token.AppealClaims, error) {
	switch {
	case raw == "expired":
		return token.AppealClaims{}, token.ErrExpired
	case strings.HasPrefix(raw, "appeal:"):
		return token.AppealClaims{Account: "acc-1", Sanction: 41}, nil
	}
	return token.AppealClaims{}, token.ErrUse
}

func TestPostingAnAppealNeedsTheAppealTokenOfThatSanction(t *testing.T) {
	var filed []string
	submit := func(_ context.Context, account string, sanctionID int64, text string, at time.Time) (appeal.Appeal, error) {
		switch text {
		case "again":
			return appeal.Appeal{}, appeal.ErrOpen
		case "rejected":
			return appeal.Appeal{}, appeal.ErrRejected
		case "over":
			return appeal.Appeal{}, appeal.ErrEnded
		case "shadow":
			return appeal.Appeal{}, appeal.ErrNotAppealable
		case "gone":
			return appeal.Appeal{}, sanction.ErrUnknown
		case "down":
			return appeal.Appeal{}, errors.New("database down")
		}
		filed = append(filed, account+":"+text)
		return appeal.Appeal{ID: 7, AccountID: account, SanctionID: sanctionID, Text: text, State: appeal.StateOpen, CreatedAt: at}, nil
	}
	handler := appeal.NewHandler(fakeVerifier{}, submit, func() liveops.Appeals { return liveops.Values{liveops.AppealTextMaxLength: 50}.Appeals() },
		slog.New(slog.NewTextHandler(io.Discard, nil)), func() time.Time { return now })
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	cases := []struct {
		name   string
		bearer string
		body   string
		status int
		want   string
	}{
		{"filed", "appeal:x", `{"sanction_id":41,"text":" not me "}`, http.StatusCreated, `{"id":7,"sanction_id":41,"state":"open","created_at":"2026-11-22T12:00:00Z"}`},
		{"no token", "", `{"sanction_id":41,"text":"not me"}`, http.StatusUnauthorized, `{"error":"appeal_token_rejected"}`},
		{"an access token", "access:acc-1", `{"sanction_id":41,"text":"not me"}`, http.StatusUnauthorized, `{"error":"appeal_token_rejected"}`},
		{"expired", "expired", `{"sanction_id":41,"text":"not me"}`, http.StatusUnauthorized, `{"error":"appeal_token_expired"}`},
		{"another sanction", "appeal:x", `{"sanction_id":42,"text":"not me"}`, http.StatusForbidden, `{"error":"sanction_mismatch"}`},
		{"malformed", "appeal:x", `{"sanction_id":`, http.StatusBadRequest, `{"error":"malformed"}`},
		{"empty", "appeal:x", `{"sanction_id":41,"text":"  "}`, http.StatusBadRequest, `{"error":"text_required"}`},
		{"too long", "appeal:x", `{"sanction_id":41,"text":"` + strings.Repeat("a", 51) + `"}`, http.StatusBadRequest, `{"error":"text_too_long"}`},
		{"open", "appeal:x", `{"sanction_id":41,"text":"again"}`, http.StatusConflict, `{"error":"appeal_open"}`},
		{"rejected", "appeal:x", `{"sanction_id":41,"text":"rejected"}`, http.StatusConflict, `{"error":"appeal_rejected"}`},
		{"ended", "appeal:x", `{"sanction_id":41,"text":"over"}`, http.StatusConflict, `{"error":"sanction_ended"}`},
		{"not appealable", "appeal:x", `{"sanction_id":41,"text":"shadow"}`, http.StatusConflict, `{"error":"not_appealable"}`},
		{"unknown", "appeal:x", `{"sanction_id":41,"text":"gone"}`, http.StatusNotFound, `{"error":"unknown_sanction"}`},
		{"failure", "appeal:x", `{"sanction_id":41,"text":"down"}`, http.StatusInternalServerError, `{"error":"internal"}`},
	}
	for _, c := range cases {
		request, err := http.NewRequest(http.MethodPost, server.URL+"/appeals", strings.NewReader(c.body))
		if err != nil {
			t.Fatal(err)
		}
		if c.bearer != "" {
			request.Header.Set("Authorization", "Bearer "+c.bearer)
		}
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(response.Body)
		response.Body.Close()
		if response.StatusCode != c.status || strings.TrimSpace(string(body)) != c.want || response.Header.Get("Cache-Control") != "no-store" {
			t.Errorf("%s: %d %s", c.name, response.StatusCode, body)
		}
	}
	if strings.Join(filed, ",") != "acc-1:not me" {
		t.Fatalf("filed %v", filed)
	}
}

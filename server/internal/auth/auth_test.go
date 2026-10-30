package auth_test

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

	"github.com/Alievs-corp/hypedomino/server/internal/account"
	"github.com/Alievs-corp/hypedomino/server/internal/auth"
	"github.com/Alievs-corp/hypedomino/server/internal/idtoken"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

type fakeVerifier struct {
	nonce      string
	nonceErr   error
	identities map[string]idtoken.Identity
	errs       map[string]error
	seen       []string
}

func (v *fakeVerifier) IssueNonce(context.Context) (string, error) {
	return v.nonce, v.nonceErr
}

func (v *fakeVerifier) Verify(_ context.Context, provider idtoken.Provider, token, nonce string) (idtoken.Identity, error) {
	v.seen = append(v.seen, provider.String()+":"+token+":"+nonce)
	if err := v.errs[token]; err != nil {
		return idtoken.Identity{}, err
	}
	identity, ok := v.identities[token]
	if !ok {
		return idtoken.Identity{}, idtoken.ErrSignature
	}
	identity.Provider = provider
	return identity, nil
}

type fakeAccounts struct {
	adopted []string
	states  map[string]error
	known   map[string]bool
	calls   int
}

func (a *fakeAccounts) SignIn(_ context.Context, identity idtoken.Identity) (account.Account, bool, error) {
	a.calls++
	id := identity.Provider.String() + "-" + identity.Subject
	if err := a.states[id]; err != nil {
		return account.Account{}, false, err
	}
	created := !a.known[id]
	a.known[id] = true
	return account.Account{ID: id, State: account.StateActive, Email: identity.Email}, created, nil
}

type fakeTokens struct {
	issued    []string
	refreshes map[string]error
	logouts   []string
	loggedOut []string
}

func (f *fakeTokens) Issue(_ context.Context, account, device string) (token.Pair, error) {
	f.issued = append(f.issued, account+"@"+device)
	return token.Pair{AccessToken: "access:" + account + ":" + device, RefreshToken: "refresh:" + account + ":1", ExpiresIn: 900, Session: "sid-" + account}, nil
}

func (f *fakeTokens) Refresh(_ context.Context, refresh, device string) (token.Pair, error) {
	if err := f.refreshes[refresh]; err != nil {
		return token.Pair{}, err
	}
	account, _, _ := strings.Cut(strings.TrimPrefix(refresh, "refresh:"), ":")
	return token.Pair{AccessToken: "access:" + account + ":" + device, RefreshToken: "refresh:" + account + ":2", ExpiresIn: 900, Session: "sid-" + account}, nil
}

func (f *fakeTokens) Logout(_ context.Context, refresh string) error {
	f.logouts = append(f.logouts, refresh)
	return nil
}

func (f *fakeTokens) LogoutAll(_ context.Context, account string) (int, error) {
	f.loggedOut = append(f.loggedOut, account)
	return 2, nil
}

func (f *fakeTokens) Verify(access string) (token.Claims, error) {
	parts := strings.Split(access, ":")
	if len(parts) != 3 || parts[0] != "access" {
		return token.Claims{}, token.ErrSignature
	}
	return token.Claims{Issuer: token.Issuer, Account: parts[1], Session: "sid-" + parts[1], Device: parts[2]}, nil
}

type fakeConnections struct {
	revoked []string
}

func (c *fakeConnections) RevokeAccount(account string) int {
	c.revoked = append(c.revoked, account)
	return 1
}

type fixture struct {
	t           *testing.T
	server      *httptest.Server
	verifier    *fakeVerifier
	accounts    *fakeAccounts
	tokens      *fakeTokens
	connections *fakeConnections
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{
		t:           t,
		verifier:    &fakeVerifier{nonce: "nonce-1", identities: map[string]idtoken.Identity{"good": {Subject: "sub-1", Email: "a@example.com"}}, errs: map[string]error{}},
		accounts:    &fakeAccounts{states: map[string]error{}, known: map[string]bool{}},
		tokens:      &fakeTokens{refreshes: map[string]error{}},
		connections: &fakeConnections{},
	}
	handler := auth.New(f.verifier, f.accounts, f.tokens, f.connections, slog.New(slog.NewTextHandler(io.Discard, nil)))
	mux := http.NewServeMux()
	mux.Handle("POST /auth/", handler)
	f.server = httptest.NewServer(mux)
	t.Cleanup(f.server.Close)
	return f
}

func (f *fixture) post(path, body string) (int, map[string]any) {
	f.t.Helper()
	return f.send(path, body, "")
}

func (f *fixture) send(path, body, bearer string) (int, map[string]any) {
	f.t.Helper()
	request, err := http.NewRequest(http.MethodPost, f.server.URL+path, strings.NewReader(body))
	if err != nil {
		f.t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		request.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		f.t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return resp.StatusCode, nil
	}
	if resp.Header.Get("Cache-Control") != "no-store" || !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		f.t.Fatalf("headers %v", resp.Header)
	}
	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		f.t.Fatal(err)
	}
	return resp.StatusCode, decoded
}

func TestNonceIsIssuedByTheVerifier(t *testing.T) {
	f := newFixture(t)
	if status, body := f.post("/auth/nonce", ""); status != http.StatusOK || body["nonce"] != "nonce-1" {
		t.Fatalf("nonce answered %d %v", status, body)
	}
	f.verifier.nonceErr = errors.New("redis down")
	if status, body := f.post("/auth/nonce", ""); status != http.StatusServiceUnavailable || body["error"] != "unavailable" {
		t.Fatalf("nonce without a store answered %d %v", status, body)
	}
}

func TestSignInVerifiesTheTokenFindsTheAccountAndIssuesTokens(t *testing.T) {
	f := newFixture(t)
	status, body := f.post("/auth/signin", `{"provider":"google","id_token":"good","nonce":"nonce-1","device_id":"install-1"}`)
	if status != http.StatusOK || body["access_token"] != "access:google-sub-1:install-1" || body["refresh_token"] != "refresh:google-sub-1:1" || body["expires_in"] != float64(900) || body["account_id"] != "google-sub-1" || body["created"] != true {
		t.Fatalf("first sign-in answered %d %v", status, body)
	}
	status, body = f.post("/auth/signin", `{"provider":"apple","id_token":"good","nonce":"nonce-2","device_id":"install-1"}`)
	if status != http.StatusOK || body["account_id"] != "apple-sub-1" || body["created"] != true {
		t.Fatalf("apple sign-in answered %d %v", status, body)
	}
	status, body = f.post("/auth/signin", `{"provider":"google","id_token":"good","nonce":"nonce-3","device_id":"install-2"}`)
	if status != http.StatusOK || body["created"] != nil {
		t.Fatalf("returning sign-in answered %d %v", status, body)
	}
	if want := []string{"google:good:nonce-1", "apple:good:nonce-2", "google:good:nonce-3"}; strings.Join(f.verifier.seen, " ") != strings.Join(want, " ") {
		t.Fatalf("verifier saw %v", f.verifier.seen)
	}
	if want := []string{"google-sub-1@install-1", "apple-sub-1@install-1", "google-sub-1@install-2"}; strings.Join(f.tokens.issued, " ") != strings.Join(want, " ") {
		t.Fatalf("tokens issued %v", f.tokens.issued)
	}
}

func TestSignInRefusalsCarryAStatusAndAReason(t *testing.T) {
	f := newFixture(t)
	f.verifier.errs["expired"] = idtoken.ErrExpired
	f.verifier.errs["replayed"] = idtoken.ErrNonceUsed
	f.verifier.errs["offline"] = idtoken.ErrKeyFetch
	f.verifier.identities["banned"] = idtoken.Identity{Subject: "banned"}
	f.accounts.states["google-banned"] = account.ErrBanned
	f.verifier.identities["deleted"] = idtoken.Identity{Subject: "deleted"}
	f.accounts.states["google-deleted"] = account.ErrDeleted
	f.verifier.identities["broken"] = idtoken.Identity{Subject: "broken"}
	f.accounts.states["google-broken"] = errors.New("database down")
	cases := []struct {
		body   string
		status int
		reason string
	}{
		{`{"provider":"google","id_token":"good","nonce":"n","device_id":"d"`, http.StatusBadRequest, "malformed"},
		{`{"provider":"facebook","id_token":"good","nonce":"n","device_id":"d"}`, http.StatusBadRequest, "unknown_provider"},
		{`{"provider":"google","id_token":"good","nonce":"n"}`, http.StatusBadRequest, "device_required"},
		{`{"provider":"google","id_token":"forged","nonce":"n","device_id":"d"}`, http.StatusUnauthorized, "token_rejected"},
		{`{"provider":"google","id_token":"expired","nonce":"n","device_id":"d"}`, http.StatusUnauthorized, "token_expired"},
		{`{"provider":"google","id_token":"replayed","nonce":"n","device_id":"d"}`, http.StatusUnauthorized, "nonce_rejected"},
		{`{"provider":"google","id_token":"offline","nonce":"n","device_id":"d"}`, http.StatusServiceUnavailable, "provider_unavailable"},
		{`{"provider":"google","id_token":"banned","nonce":"n","device_id":"d"}`, http.StatusForbidden, "banned"},
		{`{"provider":"google","id_token":"deleted","nonce":"n","device_id":"d"}`, http.StatusGone, "deleted"},
		{`{"provider":"google","id_token":"broken","nonce":"n","device_id":"d"}`, http.StatusInternalServerError, "internal"},
	}
	for _, c := range cases {
		if status, body := f.post("/auth/signin", c.body); status != c.status || body["error"] != c.reason {
			t.Fatalf("%s answered %d %v, want %d %s", c.body, status, body, c.status, c.reason)
		}
	}
	if len(f.tokens.issued) != 0 {
		t.Fatalf("a refused sign-in issued tokens: %v", f.tokens.issued)
	}
}

func TestRefreshRotatesThePairAndReportsTheft(t *testing.T) {
	f := newFixture(t)
	status, body := f.post("/auth/refresh", `{"refresh_token":"refresh:google-sub-1:1","device_id":"install-1"}`)
	if status != http.StatusOK || body["access_token"] != "access:google-sub-1:install-1" || body["refresh_token"] != "refresh:google-sub-1:2" || body["account_id"] != "google-sub-1" || body["created"] != nil {
		t.Fatalf("refresh answered %d %v", status, body)
	}
	f.tokens.refreshes["stolen"] = token.ErrReused
	f.tokens.refreshes["moved"] = token.ErrDeviceMismatch
	f.tokens.refreshes["old"] = token.ErrRefreshExpired
	f.tokens.refreshes["gone"] = token.ErrRevoked
	f.tokens.refreshes["garbage"] = token.ErrUnknownRefresh
	cases := []struct {
		body   string
		status int
		reason string
	}{
		{`{"refresh_token":"stolen"}`, http.StatusBadRequest, "device_required"},
		{`{"refresh_token":"stolen","device_id":"d"}`, http.StatusUnauthorized, "refresh_reused"},
		{`{"refresh_token":"moved","device_id":"d"}`, http.StatusUnauthorized, "refresh_reused"},
		{`{"refresh_token":"old","device_id":"d"}`, http.StatusUnauthorized, "refresh_expired"},
		{`{"refresh_token":"gone","device_id":"d"}`, http.StatusUnauthorized, "refresh_rejected"},
		{`{"refresh_token":"garbage","device_id":"d"}`, http.StatusUnauthorized, "refresh_rejected"},
	}
	for _, c := range cases {
		if status, body := f.post("/auth/refresh", c.body); status != c.status || body["error"] != c.reason {
			t.Fatalf("%s answered %d %v, want %d %s", c.body, status, body, c.status, c.reason)
		}
	}
}

func TestLogoutEndsOneDeviceAndLogoutAllEndsEveryDevice(t *testing.T) {
	f := newFixture(t)
	if status, _ := f.post("/auth/logout", `{"refresh_token":"refresh:google-sub-1:1"}`); status != http.StatusNoContent {
		t.Fatalf("logout answered %d", status)
	}
	if strings.Join(f.tokens.logouts, " ") != "refresh:google-sub-1:1" {
		t.Fatalf("logouts %v", f.tokens.logouts)
	}
	if status, body := f.send("/auth/logout-all", "", ""); status != http.StatusUnauthorized || body["error"] != "access_rejected" {
		t.Fatalf("logout-all without a bearer answered %d %v", status, body)
	}
	if status, body := f.send("/auth/logout-all", "", "not-an-access-token"); status != http.StatusUnauthorized || body["error"] != "access_rejected" {
		t.Fatalf("logout-all with a bad bearer answered %d %v", status, body)
	}
	if status, _ := f.send("/auth/logout-all", "", "access:google-sub-1:install-1"); status != http.StatusNoContent {
		t.Fatalf("logout-all answered %d", status)
	}
	if strings.Join(f.tokens.loggedOut, " ") != "google-sub-1" || strings.Join(f.connections.revoked, " ") != "google-sub-1" {
		t.Fatalf("logged out %v, connections revoked %v", f.tokens.loggedOut, f.connections.revoked)
	}
}

func TestOversizedBodiesAndOtherMethodsAreRefused(t *testing.T) {
	f := newFixture(t)
	big := `{"provider":"google","id_token":"` + strings.Repeat("x", auth.MaxBodyBytes) + `","nonce":"n","device_id":"d"}`
	if status, body := f.post("/auth/signin", big); status != http.StatusBadRequest || body["error"] != "malformed" {
		t.Fatalf("oversized body answered %d %v", status, body)
	}
	resp, err := http.Get(f.server.URL + "/auth/nonce")
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("GET nonce answered %d", resp.StatusCode)
	}
	if f.accounts.calls != 0 {
		t.Fatalf("accounts were consulted %d times", f.accounts.calls)
	}
}

func (a *fakeAccounts) AdoptCountry(_ context.Context, id, code string) error {
	a.adopted = append(a.adopted, id+"="+code)
	return nil
}

func TestSignInAdoptsTheCountryHeaderOnce(t *testing.T) {
	f := newFixture(t)
	handler := auth.New(f.verifier, f.accounts, f.tokens, f.connections, slog.New(slog.NewTextHandler(io.Discard, nil)))
	handler.SetCountryHeader("CF-IPCountry")
	server := httptest.NewServer(handler)
	defer server.Close()
	post := func(nonce, country string) int {
		t.Helper()
		f.verifier.nonce = nonce
		request, _ := http.NewRequest(http.MethodPost, server.URL+"/auth/signin", strings.NewReader(`{"provider":"google","id_token":"good","nonce":"`+nonce+`","device_id":"install-1"}`))
		request.Header.Set("Content-Type", "application/json")
		if country != "" {
			request.Header.Set("CF-IPCountry", country)
		}
		resp, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	if status := post("n1", " az "); status != http.StatusOK {
		t.Fatalf("sign-in %d", status)
	}
	if status := post("n2", ""); status != http.StatusOK {
		t.Fatalf("sign-in without header %d", status)
	}
	if want := []string{"google-sub-1=AZ", "google-sub-1="}; strings.Join(f.accounts.adopted, " ") != strings.Join(want, " ") {
		t.Fatalf("adopted %v", f.accounts.adopted)
	}
	f.accounts.adopted = nil
	if status, _ := f.post("/auth/signin", `{"provider":"google","id_token":"good","nonce":"n3","device_id":"install-1"}`); status != http.StatusOK || len(f.accounts.adopted) != 0 {
		t.Fatalf("without a configured header: %d %v", status, f.accounts.adopted)
	}
}

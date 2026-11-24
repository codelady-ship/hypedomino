package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"

	"github.com/Alievs-corp/hypedomino/server/internal/app"
	"github.com/Alievs-corp/hypedomino/server/internal/botclient"
	"github.com/Alievs-corp/hypedomino/server/internal/cluster/redistest"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/gateway"
	"github.com/Alievs-corp/hypedomino/server/internal/pass"
	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
	"github.com/Alievs-corp/hypedomino/server/internal/profile"
	"github.com/Alievs-corp/hypedomino/server/internal/secrand"
	"github.com/Alievs-corp/hypedomino/server/internal/session"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
	"github.com/Alievs-corp/hypedomino/server/internal/storeprice"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

func TestMain(m *testing.M) {
	code := m.Run()
	dbtest.Stop()
	redistest.Stop()
	os.Exit(code)
}

const (
	redTeamPackage  = "az.hypedomino.redteam"
	redTeamBundle   = "az.hypedomino.redteam"
	redTeamDevice   = "device-redteam"
	redTeamIPHeader = "X-Red-Team-Client"
	startingCoins   = 1000
)

var seededConfigs = map[string]string{
	"cosmetics":     "cosmetics/examples/catalog.json",
	"daily_login":   "economy/examples/daily_login.json",
	"match_rewards": "economy/examples/match_rewards.json",
	"shop":          "economy/examples/shop.json",
	"store_prices":  "economy/examples/store_prices.json",
	"rating":        "ranking/examples/rating.json",
	"medals":        "pass/examples/medals.json",
	"missions":      "pass/examples/missions.json",
}

type lockedBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

type redTeam struct {
	ws         string
	base       string
	pool       *pgxpool.Pool
	tokens     *token.Service
	sessions   *session.Manager
	play       *fakePlayStore
	promoKey   string
	pushSecret string
	tokenKeys  string
	logs       *lockedBuffer
	accounts   atomic.Int64
}

type attacker struct {
	account string
	pair    token.Pair
	address string
}

func randomSecret(t *testing.T) string {
	t.Helper()
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(raw)
}

func startRedTeam(t *testing.T) *redTeam {
	t.Helper()
	dsn, pool := dbtest.Fresh(t)
	seedConfigs(t, pool)
	h := &redTeam{pool: pool, play: newFakePlayStore(t), promoKey: randomSecret(t), pushSecret: randomSecret(t), tokenKeys: "redteam:" + randomSecret(t), logs: &lockedBuffer{}}
	env := map[string]string{
		"HD_LISTEN_ADDR":             "127.0.0.1:0",
		"HD_DATABASE_URL":            dsn,
		"HD_REDIS_URL":               redistest.URL(t),
		"HD_TOKEN_KEYS":              h.tokenKeys,
		"HD_GOOGLE_AUDIENCES":        "redteam.apps.googleusercontent.com",
		"HD_CONFIG_POLL":             "1s",
		"HD_DRAIN_TIMEOUT":           "1s",
		"HD_SHUTDOWN_TIMEOUT":        "2s",
		"HD_CLIENT_IP_HEADER":        redTeamIPHeader,
		"HD_PLAY_PACKAGE":            redTeamPackage,
		"HD_PLAY_SERVICE_ACCOUNT":    h.play.serviceAccount,
		"HD_PLAY_NOTIFICATION_TOKEN": h.pushSecret,
		"HD_APPLE_BUNDLE_ID":         redTeamBundle,
		"HD_PROMO_KEY":               h.promoKey,
	}
	config, err := app.ConfigFromEnv(func(name string) (string, bool) {
		value, ok := env[name]
		return value, ok
	})
	if err != nil {
		t.Fatal(err)
	}
	config.Play.Endpoint, config.Play.Client = h.play.server.URL, h.play.server.Client()
	config.Cluster.KeyPrefix = redistest.Prefix(t)
	h.sessions = session.New(config.Session, systemClock{}, secrand.Source{})
	ctx, cancel := context.WithCancel(context.Background())
	a, release, err := build(ctx, config, app.NewLogger(h.logs, "info"), h.sessions)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- a.Run(ctx) }()
	select {
	case addr := <-a.Addr():
		h.ws, h.base = "ws://"+addr+"/ws", "http://"+addr
	case err := <-done:
		cancel()
		release()
		t.Fatalf("server did not start: %v", err)
	}
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("server stopped with %v", err)
		}
		release()
		if logs := h.logs.String(); strings.Contains(logs, "panic recovered") {
			t.Errorf("the server recovered a panic under attack:\n%s", logs)
		}
	})
	h.awaitConfigs(t)
	if h.tokens, err = token.NewService(config.Token, token.NewStore(pool), time.Now); err != nil {
		t.Fatal(err)
	}
	return h
}

func (h *redTeam) awaitConfigs(t *testing.T) {
	t.Helper()
	domains := []string{pass.Domain}
	for domain := range seededConfigs {
		if domain != storeprice.Domain {
			domains = append(domains, domain)
		}
	}
	deadline := time.Now().Add(10 * time.Second)
	for _, domain := range domains {
		applied := fmt.Sprintf(`"msg":"config snapshot applied","domain":%q`, domain)
		for !strings.Contains(h.logs.String(), applied) {
			if time.Now().After(deadline) {
				t.Fatalf("the server did not load the seeded %s config:\n%s", domain, h.logs.String())
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func seedConfigs(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for domain, file := range seededConfigs {
		raw, err := os.ReadFile("../../../docs/" + file)
		if err != nil {
			t.Fatal(err)
		}
		publish(t, pool, domain, raw)
	}
	example, err := os.ReadFile("../../../docs/pass/examples/pass.json")
	if err != nil {
		t.Fatal(err)
	}
	running, err := pass.Parse(example)
	if err != nil {
		t.Fatal(err)
	}
	season := &running.Seasons[0]
	length := season.End.Sub(season.Start)
	season.Start = time.Now().UTC().Add(-24 * time.Hour).Truncate(time.Hour)
	season.End = season.Start.Add(length)
	raw, err := json.Marshal(running)
	if err != nil {
		t.Fatal(err)
	}
	publish(t, pool, pass.Domain, raw)
}

func publish(t *testing.T, pool *pgxpool.Pool, domain string, raw []byte) {
	t.Helper()
	if _, err := snapshot.Publish(context.Background(), pool, domain, raw); err != nil {
		t.Fatalf("publish %s: %v", domain, err)
	}
}

func (h *redTeam) attacker(t *testing.T) *attacker {
	t.Helper()
	ctx := context.Background()
	n := h.accounts.Add(1)
	var account string
	if err := h.pool.QueryRow(ctx, `insert into accounts default values returning id`).Scan(&account); err != nil {
		t.Fatal(err)
	}
	if _, err := profile.NewStore(h.pool, profile.DefaultConfig()).SetNickname(ctx, account, fmt.Sprintf("Raider%d", n), time.Now()); err != nil {
		t.Fatal(err)
	}
	pair, err := h.tokens.Issue(ctx, account, redTeamDevice)
	if err != nil {
		t.Fatal(err)
	}
	return &attacker{account: account, pair: pair, address: fmt.Sprintf("198.51.100.%d", n%250+1)}
}

func (h *redTeam) fund(t *testing.T, a *attacker, currency wallet.Currency, amount int64) {
	t.Helper()
	entry := wallet.Entry{Account: a.account, Currency: currency, Amount: amount, Reason: wallet.AdminAdjust, Source: wallet.Source{Type: "redteam", Ref: "seed"}, Key: fmt.Sprintf("redteam:seed:%s", currency)}
	if _, err := wallet.NewStore(h.pool).Apply(context.Background(), entry, time.Now()); err != nil {
		t.Fatal(err)
	}
}

type reply struct {
	status int
	body   map[string]any
}

func (r reply) code() string {
	code, _ := r.body["error"].(string)
	return code
}

func (h *redTeam) call(t *testing.T, a *attacker, method, path string, body []byte, mac string) reply {
	t.Helper()
	request, err := http.NewRequest(method, h.base+path, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	if a != nil {
		request.Header.Set("Authorization", "Bearer "+a.pair.AccessToken)
		request.Header.Set(redTeamIPHeader, a.address)
	}
	if mac != "" {
		request.Header.Set(token.MacHeader, mac)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	decoded := map[string]any{}
	json.Unmarshal(raw, &decoded)
	return reply{status: response.StatusCode, body: decoded}
}

func (h *redTeam) signed(t *testing.T, a *attacker, method, path string, body []byte) reply {
	t.Helper()
	return h.call(t, a, method, path, body, token.SignRequest(a.pair.RequestKey, method, path, a.pair.AccessToken, body))
}

func (h *redTeam) post(t *testing.T, a *attacker, path string, body any) reply {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return h.signed(t, a, http.MethodPost, path, raw)
}

var economyQueries = []string{
	`select currency, balance from wallets where account_id = $1 order by currency`,
	`select reason, currency, amount from ledger_entries where account_id = $1 order by id`,
	`select item_id from inventory_items where account_id = $1 and revoked_at is null order by item_id`,
	`select season_id, xp_total, premium_ref is not null from pass_progress where account_id = $1 order by season_id`,
	`select platform, offer_id, state from purchases where account_id = $1 order by id`,
	`select promo_id from promo_redemptions where account_id = $1 order by promo_id`,
	`select claims from daily_logins where account_id = $1`,
	`select loadout from profiles where account_id = $1`,
}

func (h *redTeam) economy(t *testing.T, a *attacker) string {
	t.Helper()
	var state strings.Builder
	for _, query := range economyQueries {
		rows, err := h.pool.Query(context.Background(), query, a.account)
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			values, err := rows.Values()
			if err != nil {
				rows.Close()
				t.Fatal(err)
			}
			fmt.Fprintln(&state, values...)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		state.WriteString("--\n")
	}
	return state.String()
}

func (h *redTeam) unchanged(t *testing.T, a *attacker, before, attack string) {
	t.Helper()
	if after := h.economy(t, a); after != before {
		t.Errorf("%s changed the economy of the account:\nbefore\n%s\nafter\n%s", attack, before, after)
	}
}

func (h *redTeam) signals(t *testing.T, a *attacker, kind string) int {
	t.Helper()
	var count int
	if err := h.pool.QueryRow(context.Background(), `select count(*) from risk_signals where account_id = $1 and kind = $2`, a.account, kind).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func (h *redTeam) awaitSignal(t *testing.T, a *attacker, kind string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for h.signals(t, a, kind) == 0 {
		if time.Now().After(deadline) {
			t.Fatalf("no %s risk signal was recorded", kind)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func (h *redTeam) connect(t *testing.T, a *attacker) *botclient.Client {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	c, err := botclient.Dial(ctx, h.ws, botclient.Options{Token: a.pair.AccessToken, AppVersion: "1.0.0", Platform: pb.Platform_PLATFORM_ANDROID})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Drop() })
	return c
}

type rawConn struct {
	t      *testing.T
	ctx    context.Context
	socket *websocket.Conn
}

func (h *redTeam) dialRaw(t *testing.T) *rawConn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	socket, _, err := websocket.Dial(ctx, h.ws, nil)
	if err != nil {
		t.Fatal(err)
	}
	socket.SetReadLimit(1 << 20)
	t.Cleanup(func() { socket.CloseNow() })
	return &rawConn{t: t, ctx: ctx, socket: socket}
}

func (c *rawConn) write(kind websocket.MessageType, data []byte) {
	c.t.Helper()
	if err := c.socket.Write(c.ctx, kind, data); err != nil {
		c.t.Fatal(err)
	}
}

func (c *rawConn) writeFrame(frame *pb.Envelope) {
	c.t.Helper()
	data, err := proto.Marshal(frame)
	if err != nil {
		c.t.Fatal(err)
	}
	c.write(websocket.MessageBinary, data)
}

func (c *rawConn) read() (*pb.Envelope, error) {
	_, data, err := c.socket.Read(c.ctx)
	if err != nil {
		return nil, err
	}
	frame := &pb.Envelope{}
	return frame, proto.Unmarshal(data, frame)
}

func (c *rawConn) closed() websocket.StatusCode {
	c.t.Helper()
	for {
		if _, err := c.read(); err != nil {
			return websocket.CloseStatus(err)
		}
	}
}

func hello(version uint32, client *pb.ClientHello) *pb.Envelope {
	return &pb.Envelope{ProtocolVersion: version, Payload: &pb.Envelope_ClientHello{ClientHello: client}}
}

func clientHello(sessionToken string) *pb.ClientHello {
	return &pb.ClientHello{AppVersion: "1.0.0", Platform: pb.Platform_PLATFORM_ANDROID, SessionToken: sessionToken}
}

func protocolHello(sessionToken string) *pb.Envelope {
	return hello(gateway.ProtocolVersion, clientHello(sessionToken))
}

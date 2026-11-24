package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/botclient"
	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
	"github.com/Alievs-corp/hypedomino/server/internal/ratelimit"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func limited(config Config) func(*App) {
	return func(a *App) {
		a.Limit(ratelimit.New(config.RateLimit, ratelimit.NewMemory(), NewLogger(io.Discard, "info"), time.Now))
		a.Handle("POST /echo", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := io.ReadAll(r.Body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
			}
		}))
	}
}

func closedWithin(t *testing.T, conn net.Conn, limit time.Duration) {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(limit))
	if _, err := io.ReadAll(conn); err != nil {
		t.Fatalf("the server kept the connection open: %v", err)
	}
}

func TestSlowAndIdleClientsAreCutOffButMatchesOutliveTheReadTimeout(t *testing.T) {
	config := DefaultConfig()
	config.RateLimit.ReadTimeout = 300 * time.Millisecond
	config.RateLimit.IdleTimeout = 300 * time.Millisecond
	r := start(t, config, limited(config))
	addr := strings.TrimPrefix(r.url, "http://")
	slow, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer slow.Close()
	fmt.Fprintf(slow, "POST /echo HTTP/1.1\r\nHost: x\r\nContent-Length: 100\r\n\r\na")
	closedWithin(t, slow, 3*time.Second)
	idle, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer idle.Close()
	fmt.Fprintf(idle, "GET /healthz HTTP/1.1\r\nHost: x\r\n\r\n")
	closedWithin(t, idle, 3*time.Second)
	r.openRoom(rules.Mode1v1)
	x, y := r.dial(rules.SeatX), r.dial(rules.SeatY)
	for _, c := range []*botclient.Client{x, y} {
		if _, err := c.PlayUntil(func(v *pb.MatchView) bool { return v.GetState() != pb.MatchState_MATCH_STATE_WAITING }); err != nil {
			t.Fatal(err)
		}
	}
	time.Sleep(3 * config.RateLimit.ReadTimeout)
	var wg sync.WaitGroup
	for _, c := range []*botclient.Client{x, y} {
		wg.Go(func() {
			if _, err := c.PlayUntil(botclient.Finished); err != nil {
				t.Errorf("the match connection was cut by the read timeout: %v", err)
			}
		})
	}
	wg.Wait()
}

func TestAFloodFromOneAddressIsRefusedWhileOthersAndProbesAreServed(t *testing.T) {
	config := DefaultConfig()
	config.RateLimit.IPHeader = "CF-Connecting-IP"
	r := start(t, config, limited(config))
	status := func(path, ip string) int {
		request, _ := http.NewRequest(http.MethodPost, r.url+path, nil)
		if strings.HasPrefix(path, "/healthz") || strings.HasPrefix(path, "/loadz") {
			request.Method = http.MethodGet
		}
		request.Header.Set("CF-Connecting-IP", ip)
		resp, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Error(err)
			return 0
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	const flood = 1000
	counts := map[int]int{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			for range flood / 50 {
				code := status("/echo", "203.0.113.9")
				mu.Lock()
				counts[code]++
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	burst := config.RateLimit.IP.Burst
	if counts[http.StatusOK] < burst || counts[http.StatusOK] > burst+5 || counts[http.StatusOK]+counts[http.StatusTooManyRequests] != flood {
		t.Fatalf("flood answers %v", counts)
	}
	if code := status("/healthz", "203.0.113.9"); code != http.StatusOK {
		t.Fatalf("healthz from the flooding address: %d", code)
	}
	if code := status("/echo", "198.51.100.4"); code != http.StatusOK {
		t.Fatalf("another address: %d", code)
	}
	resp, err := http.Get(r.url + "/loadz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var report struct {
		RateLimited int `json:"rateLimited"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil || report.RateLimited != counts[http.StatusTooManyRequests] {
		t.Fatalf("loadz rateLimited %d, refused %d: %v", report.RateLimited, counts[http.StatusTooManyRequests], err)
	}
}

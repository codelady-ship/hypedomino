package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/cluster"
	"github.com/Alievs-corp/hypedomino/server/internal/gateway"
	"github.com/Alievs-corp/hypedomino/server/internal/metrics"
	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
	"github.com/Alievs-corp/hypedomino/server/internal/ratelimit"
)

func parseLevel(text string) (slog.Level, bool) {
	switch strings.ToLower(text) {
	case "debug":
		return slog.LevelDebug, true
	case "info":
		return slog.LevelInfo, true
	case "warn":
		return slog.LevelWarn, true
	case "error":
		return slog.LevelError, true
	}
	return slog.LevelInfo, false
}

func NewLogger(out io.Writer, level string) *slog.Logger {
	parsed, _ := parseLevel(level)
	return slog.New(slog.NewJSONHandler(out, &slog.HandlerOptions{Level: parsed}))
}

type App struct {
	config       Config
	logger       *slog.Logger
	gateway      *gateway.Gateway
	rooms        *Rooms
	cluster      *cluster.Cluster
	mux          *http.ServeMux
	limiter      *ratelimit.Limiter
	ready        atomic.Bool
	listener     net.Listener
	addr         chan string
	localization atomic.Pointer[func() gateway.Strings]
	cosmetics    atomic.Pointer[func() *pb.CosmeticsManifest]
	analytics    atomic.Pointer[func() *pb.AnalyticsConfig]
	admission    atomic.Pointer[func() gateway.Admission]
	moves        *metrics.Histogram
	responses    responseCounts
	exposed      []io.WriterTo
	awaited      []func() bool
}

func New(config Config, logger *slog.Logger, handler gateway.Handler) *App {
	a := &App{
		config: config,
		logger: logger,
		rooms:  NewRooms(),
		moves:  newMoveRoundTrip(),
		mux:    http.NewServeMux(),
		addr:   make(chan string, 1),
	}
	a.rooms.ObserveMoves(a.moves)
	a.rooms.LimitEmoji(config.Emoji)
	a.gateway = gateway.New(gateway.Options{Config: config.Gateway, Handler: handler, Origins: config.Origins, Strings: a.localized, Cosmetics: a.announcedCosmetics, Analytics: a.announcedAnalytics, Admission: a.admitted})
	a.mux.HandleFunc("GET /healthz", a.healthz)
	a.mux.HandleFunc("GET /readyz", a.readyz)
	a.mux.HandleFunc("GET /ws", a.ws)
	a.mux.HandleFunc("GET /loadz", a.loadz)
	return a
}

func (a *App) Handle(pattern string, handler http.Handler) {
	a.mux.Handle(pattern, handler)
}

func (a *App) Limit(limiter *ratelimit.Limiter) {
	a.limiter = limiter
}

func (a *App) Expose(source io.WriterTo) {
	a.exposed = append(a.exposed, source)
}

func (a *App) AwaitBeforeReady(done func() bool) {
	a.awaited = append(a.awaited, done)
}

func (a *App) loading() bool {
	for _, done := range a.awaited {
		if !done() {
			return true
		}
	}
	return false
}

func (a *App) Localize(current func() gateway.Strings) {
	a.localization.Store(&current)
}

func (a *App) localized() gateway.Strings {
	current := a.localization.Load()
	if current == nil {
		return gateway.Strings{}
	}
	return (*current)()
}

func (a *App) AnnounceCosmetics(current func() *pb.CosmeticsManifest) {
	a.cosmetics.Store(&current)
}

func (a *App) announcedCosmetics() *pb.CosmeticsManifest {
	current := a.cosmetics.Load()
	if current == nil {
		return nil
	}
	return (*current)()
}

func (a *App) AnnounceAnalytics(current func() *pb.AnalyticsConfig) {
	a.analytics.Store(&current)
}

func (a *App) announcedAnalytics() *pb.AnalyticsConfig {
	current := a.analytics.Load()
	if current == nil {
		return nil
	}
	return (*current)()
}

func (a *App) Admit(current func() gateway.Admission) {
	a.admission.Store(&current)
}

func (a *App) admitted() gateway.Admission {
	current := a.admission.Load()
	if current == nil {
		return gateway.Admission{MinimumAppVersion: a.config.Gateway.MinimumAppVersion}
	}
	return (*current)()
}

func (a *App) EnforceAdmission() {
	a.gateway.Enforce()
}

func (a *App) Join(c *cluster.Cluster) {
	a.cluster = c
	c.Observe(a.load)
}

func (a *App) load() cluster.Load {
	return cluster.Load{Rooms: a.rooms.Len(), Connections: a.gateway.Connections(), Draining: !a.ready.Load()}
}

func (a *App) Rooms() *Rooms {
	return a.rooms
}

func (a *App) Handler() http.Handler {
	if a.limiter == nil {
		return a.observed(a.mux)
	}
	limited := a.limiter.Wrap(a.mux)
	return a.observed(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.relayed(r) {
			a.mux.ServeHTTP(w, r)
			return
		}
		limited.ServeHTTP(w, r)
	}))
}

func (a *App) relayed(r *http.Request) bool {
	return a.cluster != nil && r.URL.Path == "/ws" && r.Header.Get(cluster.RelayHeader) != ""
}

func (a *App) Addr() <-chan string {
	return a.addr
}

func (a *App) healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.WriteString(w, "ok\n")
}

func (a *App) readyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if !a.ready.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		io.WriteString(w, "draining\n")
		return
	}
	if a.loading() {
		w.WriteHeader(http.StatusServiceUnavailable)
		io.WriteString(w, "loading config\n")
		return
	}
	io.WriteString(w, "ready\n")
}

func (a *App) ws(w http.ResponseWriter, r *http.Request) {
	relayed := a.relayed(r)
	if !a.ready.Load() && !relayed {
		http.Error(w, "draining", http.StatusServiceUnavailable)
		return
	}
	if relayed {
		r = r.WithContext(cluster.MarkRelayed(r.Context()))
	}
	a.gateway.ServeHTTP(w, r)
}

func (a *App) loadz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	report := map[string]any{"load": a.load()}
	if a.limiter != nil {
		report["rateLimited"] = a.limiter.Refused()
	}
	if a.cluster != nil {
		report["node"] = a.cluster.Self()
		summary, err := a.cluster.Summarize(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		report["cluster"] = summary
	}
	json.NewEncoder(w).Encode(report)
}

func (a *App) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", a.config.ListenAddr)
	if err != nil {
		return err
	}
	stopDebug, err := a.serveDebug()
	if err != nil {
		listener.Close()
		return err
	}
	defer stopDebug()
	a.listener = listener
	a.ready.Store(true)
	server := &http.Server{Handler: a.Handler(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: a.config.RateLimit.ReadTimeout, IdleTimeout: a.config.RateLimit.IdleTimeout}
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(listener) }()
	stopCluster := a.announce()
	defer stopCluster()
	a.logger.Info("listening", "addr", listener.Addr().String())
	a.addr <- listener.Addr().String()
	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}
	return a.shutdown(server)
}

func (a *App) shutdown(server *http.Server) error {
	a.rooms.Close()
	a.ready.Store(false)
	a.logger.Info("draining", "rooms", a.rooms.Len(), "connections", a.gateway.Connections())
	drainCtx, cancelDrain := context.WithTimeout(context.Background(), a.config.DrainTimeout)
	defer cancelDrain()
	if err := a.rooms.Drain(drainCtx); err != nil {
		a.logger.Warn("drain timeout, cancelling rooms", "rooms", a.rooms.Len())
	}
	a.rooms.Stop()
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), a.config.ShutdownTimeout)
	defer cancelShutdown()
	if err := a.gateway.Shutdown(shutdownCtx); err != nil {
		a.logger.Warn("connections did not close in time", "connections", a.gateway.Connections())
	}
	err := server.Shutdown(shutdownCtx)
	a.logger.Info("stopped")
	if errors.Is(err, context.DeadlineExceeded) {
		return nil
	}
	return err
}

func (a *App) announce() func() {
	if a.cluster == nil {
		return func() {}
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := a.cluster.Run(ctx); err != nil && ctx.Err() == nil {
			a.logger.Error("cluster membership failed", "node", a.cluster.NodeID(), "error", err)
		}
	}()
	return func() {
		cancel()
		<-done
	}
}

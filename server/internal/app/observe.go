package app

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sync/atomic"
	"time"
)

const RequestIDHeader = "X-Request-Id"

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,64}$`)

type responseCounts struct {
	classes [6]atomic.Uint64
}

func (c *responseCounts) count(status int) {
	if class := status / 100; class >= 1 && class <= 5 {
		c.classes[class].Add(1)
	}
}

func (c *responseCounts) WriteTo(w io.Writer) (int64, error) {
	n, err := fmt.Fprint(w, "# HELP hd_http_responses_total HTTP responses by status class, WebSocket upgrades as 1xx.\n# TYPE hd_http_responses_total counter\n")
	written := int64(n)
	for class := 1; class <= 5 && err == nil; class++ {
		n, err = fmt.Fprintf(w, "hd_http_responses_total{class=\"%dxx\"} %d\n", class, c.classes[class].Load())
		written += int64(n)
	}
	return written, err
}

type statusRecorder struct {
	http.ResponseWriter
	counts *responseCounts
	status int
}

func (r *statusRecorder) record(status int) {
	if r.status == 0 {
		r.status = status
		r.counts.count(status)
	}
}

func (r *statusRecorder) WriteHeader(status int) {
	r.record(status)
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(body []byte) (int, error) {
	r.record(http.StatusOK)
	return r.ResponseWriter.Write(body)
}

func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func requestID(r *http.Request) string {
	if id := r.Header.Get(RequestIDHeader); requestIDPattern.MatchString(id) {
		return id
	}
	random := make([]byte, 8)
	rand.Read(random)
	return hex.EncodeToString(random)
}

func (a *App) observed(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := requestID(r)
		w.Header().Set(RequestIDHeader, id)
		recorder := &statusRecorder{ResponseWriter: w, counts: &a.responses}
		started := time.Now()
		next.ServeHTTP(recorder, r)
		recorder.record(http.StatusOK)
		a.logger.Info("http request", "request_id", id, "method", r.Method, "path", r.URL.Path, "status", recorder.status, "duration_ms", time.Since(started).Milliseconds())
	})
}

package metrics

import (
	"fmt"
	"io"
	"strconv"
	"sync/atomic"
	"time"
)

type Histogram struct {
	name    string
	help    string
	bounds  []time.Duration
	buckets []atomic.Uint64
	count   atomic.Uint64
	sum     atomic.Int64
}

func NewHistogram(name, help string, bounds []time.Duration) *Histogram {
	return &Histogram{name: name, help: help, bounds: bounds, buckets: make([]atomic.Uint64, len(bounds))}
}

func (h *Histogram) Observe(elapsed time.Duration) {
	for i, bound := range h.bounds {
		if elapsed <= bound {
			h.buckets[i].Add(1)
			break
		}
	}
	h.count.Add(1)
	h.sum.Add(int64(elapsed))
}

func (h *Histogram) WriteTo(w io.Writer) (int64, error) {
	out := &countingWriter{w: w}
	out.printf("# HELP %s %s\n# TYPE %s histogram\n", h.name, h.help, h.name)
	h.writeSeries(out, "")
	return out.written, out.err
}

func (h *Histogram) writeSeries(out *countingWriter, labels string) {
	var cumulative uint64
	for i, bound := range h.bounds {
		cumulative += h.buckets[i].Load()
		out.printf("%s_bucket{%sle=\"%s\"} %d\n", h.name, labels, seconds(bound), cumulative)
	}
	count := h.count.Load()
	selector := ""
	if labels != "" {
		selector = "{" + labels[:len(labels)-1] + "}"
	}
	out.printf("%s_bucket{%sle=\"+Inf\"} %d\n%s_sum%s %s\n%s_count%s %d\n", h.name, labels, count, h.name, selector, seconds(time.Duration(h.sum.Load())), h.name, selector, count)
}

type countingWriter struct {
	w       io.Writer
	written int64
	err     error
}

func (c *countingWriter) printf(format string, args ...any) {
	if c.err != nil {
		return
	}
	n, err := fmt.Fprintf(c.w, format, args...)
	c.written += int64(n)
	c.err = err
}

func seconds(d time.Duration) string {
	return strconv.FormatFloat(d.Seconds(), 'g', -1, 64)
}

func WriteGauge(w io.Writer, name, help string, value float64) error {
	_, err := fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s gauge\n%s %s\n", name, help, name, name, strconv.FormatFloat(value, 'f', -1, 64))
	return err
}

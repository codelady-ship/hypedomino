package metrics

import (
	"cmp"
	"io"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Labels [2]string

func (names Labels) pairs(values Labels) string {
	var out strings.Builder
	for i, name := range names {
		if name != "" {
			out.WriteString(name + "=" + strconv.Quote(values[i]) + ",")
		}
	}
	return out.String()
}

func sortedKeys[V any](series map[Labels]V) []Labels {
	keys := make([]Labels, 0, len(series))
	for key := range series {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(a, b Labels) int {
		return cmp.Or(cmp.Compare(a[0], b[0]), cmp.Compare(a[1], b[1]))
	})
	return keys
}

type Counter struct {
	name   string
	help   string
	labels Labels
	mu     sync.Mutex
	counts map[Labels]uint64
}

func NewCounter(name, help string, labels Labels) *Counter {
	return &Counter{name: name, help: help, labels: labels, counts: map[Labels]uint64{}}
}

func (c *Counter) Inc(values Labels) {
	c.mu.Lock()
	c.counts[values]++
	c.mu.Unlock()
}

func (c *Counter) WriteTo(w io.Writer) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := &countingWriter{w: w}
	out.printf("# HELP %s %s\n# TYPE %s counter\n", c.name, c.help, c.name)
	for _, values := range sortedKeys(c.counts) {
		selector := ""
		if pairs := c.labels.pairs(values); pairs != "" {
			selector = "{" + pairs[:len(pairs)-1] + "}"
		}
		out.printf("%s%s %d\n", c.name, selector, c.counts[values])
	}
	return out.written, out.err
}

type Histograms struct {
	name   string
	help   string
	labels Labels
	bounds []time.Duration
	mu     sync.Mutex
	series map[Labels]*Histogram
}

func NewHistograms(name, help string, labels Labels, bounds []time.Duration) *Histograms {
	return &Histograms{name: name, help: help, labels: labels, bounds: bounds, series: map[Labels]*Histogram{}}
}

func (h *Histograms) Observe(values Labels, elapsed time.Duration) {
	h.mu.Lock()
	series, ok := h.series[values]
	if !ok {
		series = NewHistogram(h.name, h.help, h.bounds)
		h.series[values] = series
	}
	h.mu.Unlock()
	series.Observe(elapsed)
}

func (h *Histograms) WriteTo(w io.Writer) (int64, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := &countingWriter{w: w}
	out.printf("# HELP %s %s\n# TYPE %s histogram\n", h.name, h.help, h.name)
	for _, values := range sortedKeys(h.series) {
		h.series[values].writeSeries(out, h.labels.pairs(values))
	}
	return out.written, out.err
}

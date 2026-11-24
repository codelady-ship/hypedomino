package loadbot

import (
	"bufio"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func ScrapeNode(url string) func() (NodeSample, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	return func() (NodeSample, error) {
		response, err := client.Get(url)
		if err != nil {
			return NodeSample{}, err
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return NodeSample{}, fmt.Errorf("loadbot: %s answered %d", url, response.StatusCode)
		}
		return parseNode(bufio.NewScanner(response.Body))
	}
}

func parseNode(lines *bufio.Scanner) (NodeSample, error) {
	gauges := map[string]uint64{}
	for lines.Scan() {
		name, value, ok := strings.Cut(lines.Text(), " ")
		if !ok || (name != "go_heap_live_bytes" && name != "go_goroutines") {
			continue
		}
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return NodeSample{}, fmt.Errorf("loadbot: %s: %w", name, err)
		}
		gauges[name] = uint64(parsed)
	}
	if err := lines.Err(); err != nil {
		return NodeSample{}, err
	}
	heap, okHeap := gauges["go_heap_live_bytes"]
	goroutines, okGoroutines := gauges["go_goroutines"]
	if !okHeap || !okGoroutines {
		return NodeSample{}, fmt.Errorf("loadbot: metrics lack go_heap_live_bytes or go_goroutines")
	}
	return NodeSample{HeapBytes: heap, Goroutines: goroutines}, nil
}

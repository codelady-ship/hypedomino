package storesync

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/storeprice"
)

const (
	GooglePlay = "google_play"
	AppStore   = "app_store"

	StatusChange     = "change"
	StatusUnchanged  = "unchanged"
	StatusNotInStore = "not_in_store"
	StatusError      = "error"
	StatusWritten    = "written"
	StatusFailed     = "failed"

	DefaultPause = 500 * time.Millisecond
	attempts     = 4
	maxBody      = 4 << 20
)

var ErrStore = errors.New("storesync: store request failed")

type Price struct {
	Currency string `json:"currency"`
	Amount   string `json:"amount"`
}

type Row struct {
	Product     string `json:"product"`
	Country     string `json:"country"`
	TargetCents int64  `json:"target_cents"`
	Current     *Price `json:"current,omitempty"`
	Next        *Price `json:"next,omitempty"`
	Status      string `json:"status"`
	Detail      string `json:"detail,omitempty"`
}

type Report struct {
	Store      string         `json:"store"`
	Configured bool           `json:"configured"`
	Rows       []Row          `json:"rows"`
	Counts     map[string]int `json:"counts"`
}

type Syncer interface {
	Sync(ctx context.Context, targets []storeprice.Target, write bool) []Row
}

type Stores struct {
	GooglePlay Syncer
	AppStore   Syncer
}

func (s Stores) Run(ctx context.Context, targets []storeprice.Target, write bool) []Report {
	reports := []Report{}
	for _, store := range []struct {
		name   string
		syncer Syncer
	}{{GooglePlay, s.GooglePlay}, {AppStore, s.AppStore}} {
		report := Report{Store: store.name, Rows: []Row{}, Counts: map[string]int{}}
		if store.syncer != nil {
			report.Configured, report.Rows = true, store.syncer.Sync(ctx, withoutRussia(targets), write)
		}
		for _, row := range report.Rows {
			report.Counts[row.Status]++
		}
		reports = append(reports, report)
	}
	return reports
}

func withoutRussia(targets []storeprice.Target) []storeprice.Target {
	kept := make([]storeprice.Target, 0, len(targets))
	for _, target := range targets {
		if target.Country != "RU" {
			kept = append(kept, target)
		}
	}
	return kept
}

func byProduct(targets []storeprice.Target) ([]string, map[string][]storeprice.Target) {
	order, groups := []string{}, map[string][]storeprice.Target{}
	for _, target := range targets {
		if _, seen := groups[target.Product]; !seen {
			order = append(order, target.Product)
		}
		groups[target.Product] = append(groups[target.Product], target)
	}
	return order, groups
}

func rowsFor(targets []storeprice.Target, status, detail string) []Row {
	rows := make([]Row, 0, len(targets))
	for _, target := range targets {
		rows = append(rows, Row{Product: target.Product, Country: target.Country, TargetCents: target.Cents, Status: status, Detail: detail})
	}
	return rows
}

func settle(rows []Row, err error) {
	for i := range rows {
		if rows[i].Status != StatusChange {
			continue
		}
		rows[i].Status = StatusWritten
		if err != nil {
			rows[i].Status, rows[i].Detail = StatusFailed, err.Error()
		}
	}
}

func changes(rows []Row) int {
	count := 0
	for _, row := range rows {
		if row.Status == StatusChange {
			count++
		}
	}
	return count
}

type caller struct {
	client    *http.Client
	authorize func(ctx context.Context) (string, error)
	pause     time.Duration
}

func (c caller) do(ctx context.Context, method, target string, body []byte) (int, []byte, error) {
	wait := c.pause
	for attempt := 1; ; attempt++ {
		status, payload, err := c.once(ctx, method, target, body)
		if err == nil && status != http.StatusTooManyRequests && status < http.StatusInternalServerError {
			return status, payload, nil
		}
		if attempt == attempts {
			if err == nil {
				err = fmt.Errorf("%w: %s %s answered %d", ErrStore, method, target, status)
			}
			return 0, nil, err
		}
		select {
		case <-ctx.Done():
			return 0, nil, ctx.Err()
		case <-time.After(wait):
		}
		wait *= 2
	}
}

func (c caller) once(ctx context.Context, method, target string, body []byte) (int, []byte, error) {
	authorization, err := c.authorize(ctx)
	if err != nil {
		return 0, nil, fmt.Errorf("%w: %v", ErrStore, err)
	}
	request, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	request.Header.Set("Authorization", authorization)
	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.client.Do(request)
	if err != nil {
		return 0, nil, fmt.Errorf("%w: %v", ErrStore, err)
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(response.Body, maxBody))
	if err != nil {
		return 0, nil, fmt.Errorf("%w: %v", ErrStore, err)
	}
	return response.StatusCode, payload, nil
}

func unexpected(method, target string, status int) error {
	return fmt.Errorf("%w: %s %s answered %d", ErrStore, method, target, status)
}

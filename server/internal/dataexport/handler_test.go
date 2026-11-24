package dataexport_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/account"
	"github.com/Alievs-corp/hypedomino/server/internal/dataexport"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

type verifier struct{}

func (verifier) Verify(access string) (token.Claims, error) {
	if account, ok := strings.CutPrefix(access, "access:"); ok && account != "" {
		return token.Claims{Account: account}, nil
	}
	return token.Claims{}, token.ErrSignature
}

type files struct {
	stored  map[string][]byte
	deleted []string
	broken  bool
}

func (f *files) Put(_ context.Context, key, contentType string, body []byte, _ time.Time) error {
	if f.broken || contentType != "application/json" {
		return errors.New("store down")
	}
	f.stored[key] = body
	return nil
}

func (f *files) SignedGet(key string, _ time.Time) string {
	return "https://files.example/" + key + "?signed"
}

func (f *files) Delete(_ context.Context, key string, _ time.Time) error {
	f.deleted = append(f.deleted, key)
	delete(f.stored, key)
	return nil
}

type reply struct {
	Error     string    `json:"error"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
	RetryAt   time.Time `json:"retry_at"`
}

func TestExportUploadsOneFileAndHonoursTheCooldownAndLinkLifetime(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	ctx := context.Background()
	var player string
	if err := pool.QueryRow(ctx, `insert into accounts default values returning id::text`).Scan(&player); err != nil {
		t.Fatal(err)
	}
	settings := dataexport.Settings{LinkTTL: 72 * time.Hour, Cooldown: 7 * 24 * time.Hour}
	now := time.Date(2026, 11, 11, 9, 0, 0, 0, time.UTC)
	store := dataexport.NewStore(pool)
	bucket := &files{stored: map[string][]byte{}}
	handler := dataexport.NewHandler(verifier{}, store, account.New(pool), bucket, settings, slog.New(slog.NewTextHandler(io.Discard, nil)), func() time.Time { return now })
	call := func(bearer string) (int, reply) {
		t.Helper()
		request := httptest.NewRequest(http.MethodPost, "/account/export", nil)
		request.Header.Set("Authorization", "Bearer "+bearer)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		var body reply
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("%d %s", recorder.Code, recorder.Body)
		}
		return recorder.Code, body
	}

	if status, body := call("forged"); status != http.StatusUnauthorized || body.Error != "access_rejected" {
		t.Fatalf("no token: %d %+v", status, body)
	}
	bucket.broken = true
	if status, body := call("access:" + player); status != http.StatusServiceUnavailable || body.Error != "export_unavailable" {
		t.Fatalf("store down: %d %+v", status, body)
	}
	bucket.broken = false
	status, first := call("access:" + player)
	if status != http.StatusCreated || !strings.HasPrefix(first.URL, "https://files.example/exports/") || !first.ExpiresAt.Equal(now.Add(72*time.Hour)) {
		t.Fatalf("first export: %d %+v", status, first)
	}
	if len(bucket.stored) != 1 {
		t.Fatalf("stored %d files", len(bucket.stored))
	}
	for _, document := range bucket.stored {
		if !strings.Contains(string(document), `"schema_version":1`) || !strings.Contains(string(document), player) {
			t.Fatalf("document %s", document)
		}
	}
	now = now.Add(24 * time.Hour)
	if status, body := call("access:" + player); status != http.StatusTooManyRequests || body.Error != "export_cooldown" || !body.RetryAt.Equal(now.Add(6*24*time.Hour)) {
		t.Fatalf("inside the cooldown: %d %+v", status, body)
	}
	now = now.Add(48 * time.Hour)
	if purged, err := store.Purge(ctx, bucket, now); err != nil || purged != 1 || len(bucket.stored) != 0 {
		t.Fatalf("purge at expiry: %d %v %d", purged, err, len(bucket.stored))
	}
	if purged, err := store.Purge(ctx, bucket, now); err != nil || purged != 0 || len(bucket.deleted) != 1 {
		t.Fatalf("second purge: %d %v %v", purged, err, bucket.deleted)
	}
	now = now.Add(4 * 24 * time.Hour)
	if status, body := call("access:" + player); status != http.StatusCreated || body.URL == first.URL {
		t.Fatalf("after the cooldown: %d %+v", status, body)
	}
}

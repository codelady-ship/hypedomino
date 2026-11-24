package analytics_test

import (
	"context"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/analytics"
	"github.com/Alievs-corp/hypedomino/server/internal/consent"
)

const (
	rotatedID = "6f1c2a7e-3b4d-4e5f-8a9b-0c1d2e3f4a5c"
	newestID  = "6f1c2a7e-3b4d-4e5f-8a9b-0c1d2e3f4a5d"
	othersID  = "6f1c2a7e-3b4d-4e5f-8a9b-0c1d2e3f4a5e"
)

func (w *world) chooseAnalyticsAt(account string, granted bool, at time.Time) {
	w.t.Helper()
	change := consent.Change{Analytics: &granted, PolicyVersion: "2026-11", AppVersion: "1.4.0"}
	if _, err := w.consents.Save(context.Background(), account, change, at); err != nil {
		w.t.Fatal(err)
	}
}

func (w *world) sendWith(account, analyticsID string, at time.Time) {
	w.t.Helper()
	batch := analytics.Batch{AnalyticsID: analyticsID, Events: []analytics.Event{{Name: "app_open", ClientTimeMs: 1}}}
	if err := analytics.NewStore(w.pool).Record(context.Background(), account, batch, at); err != nil {
		w.t.Fatal(err)
	}
}

func (w *world) expectServerEvents(name, account, want string) {
	w.t.Helper()
	ctx := context.Background()
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		w.t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if err := analytics.RecordServerIn(ctx, tx, name, account, 2, now); err != nil {
		w.t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		w.t.Fatal(err)
	}
	var events, bound int
	var analyticsID string
	err = w.pool.QueryRow(ctx, `select count(*), count(analytics_id), coalesce(max(analytics_id::text), '') from analytics_events
		where name = $1 and label = '' and value = 0 and client_time_ms is null and received_at = $2`, name, now).Scan(&events, &bound, &analyticsID)
	if err != nil {
		w.t.Fatal(err)
	}
	wantBound := 0
	if want != "" {
		wantBound = 2
	}
	if events != 2 || bound != wantBound || analyticsID != want {
		w.t.Fatalf("%s: %d events, %d bound to %q, want 2 events, %d bound to %q", name, events, bound, analyticsID, wantBound, want)
	}
}

func TestServerEventsCarryTheNewestAnalyticsIDSeenSinceTheLastWithdrawal(t *testing.T) {
	w := newWorld(t)
	ayla, bob := w.stranger(), w.stranger()
	start := now.Add(-time.Hour)
	w.expectServerEvents("before_any_choice", ayla, "")
	w.chooseAnalyticsAt(ayla, true, start)
	w.expectServerEvents("granted_without_an_id", ayla, "")
	w.sendWith(ayla, id, start.Add(time.Minute))
	w.sendWith(bob, othersID, start.Add(time.Minute))
	w.expectServerEvents("first_id", ayla, id)
	w.expectServerEvents("other_account_without_consent", bob, "")
	w.sendWith(ayla, rotatedID, start.Add(2*time.Minute))
	w.expectServerEvents("second_id", ayla, rotatedID)
	w.chooseAnalyticsAt(ayla, false, start.Add(3*time.Minute))
	w.expectServerEvents("withdrawn", ayla, "")
	w.chooseAnalyticsAt(ayla, true, start.Add(4*time.Minute))
	w.expectServerEvents("granted_again_before_a_new_id", ayla, "")
	w.sendWith(ayla, newestID, start.Add(5*time.Minute))
	w.expectServerEvents("granted_again_with_a_new_id", ayla, newestID)
}

func TestALabeledServerEventKeepsItsLabelAndTheConsentedAnalyticsID(t *testing.T) {
	w := newWorld(t)
	ayla := w.stranger()
	w.chooseAnalyticsAt(ayla, true, now.Add(-time.Hour))
	w.sendWith(ayla, id, now.Add(-time.Minute))
	ctx := context.Background()
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if err := analytics.RecordServerLabeledIn(ctx, tx, analytics.RewardGrant, ayla, "shop", 1, now); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	var label, analyticsID string
	var value int64
	if err := w.pool.QueryRow(ctx, `select label, value, analytics_id::text from analytics_events where name = $1`, analytics.RewardGrant).Scan(&label, &value, &analyticsID); err != nil {
		t.Fatal(err)
	}
	if label != "shop" || value != 0 || analyticsID != id {
		t.Fatalf("labeled event %q %d %q, want shop 0 %q", label, value, analyticsID, id)
	}
}

package analytics_test

import (
	"context"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/analytics"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
)

func TestPurgeRawDeletesOnlyEventsReceivedBeforeTheCutoff(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := analytics.NewStore(pool)
	ctx := context.Background()
	var ayla string
	if err := pool.QueryRow(ctx, `insert into accounts default values returning id::text`).Scan(&ayla); err != nil {
		t.Fatal(err)
	}
	cutoff := now.Add(-395 * 24 * time.Hour)
	for i, received := range []time.Time{cutoff.Add(-time.Second), cutoff.Add(-24 * time.Hour), cutoff, now} {
		event := analytics.Event{Name: "app_open", ClientTimeMs: int64(i + 1)}
		if err := store.Record(ctx, ayla, analytics.Batch{AnalyticsID: id, Events: []analytics.Event{event}}, received); err != nil {
			t.Fatal(err)
		}
	}
	for _, want := range []int64{2, 0} {
		deleted, err := store.PurgeRaw(ctx, cutoff)
		if err != nil || deleted != want {
			t.Fatalf("purge deleted %d %v, want %d", deleted, err, want)
		}
	}
	var kept []int64
	rows, err := pool.Query(ctx, `select client_time_ms from analytics_events order by client_time_ms`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var clientTime int64
		if err := rows.Scan(&clientTime); err != nil {
			t.Fatal(err)
		}
		kept = append(kept, clientTime)
	}
	var owners int
	if err := pool.QueryRow(ctx, `select count(*) from analytics_ids where account_id = $1`, ayla).Scan(&owners); err != nil {
		t.Fatal(err)
	}
	if len(kept) != 2 || kept[0] != 3 || kept[1] != 4 || owners != 1 {
		t.Fatalf("kept events %v and %d analytics ids", kept, owners)
	}
}

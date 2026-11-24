package adminapi

import (
	"testing"
	"time"
)

func TestAQueueEntryIsOverdueOnlyAfterItsDueTime(t *testing.T) {
	opened := time.Date(2026, 11, 22, 10, 0, 0, 0, time.FixedZone("AZT", 4*60*60))
	due := time.Date(2026, 11, 23, 6, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		now     time.Time
		overdue bool
	}{
		{"just opened", opened, false},
		{"one second before", due.Add(-time.Second), false},
		{"at the due time", due, false},
		{"one microsecond after", due.Add(time.Microsecond), true},
		{"a day after", due.Add(24 * time.Hour), true},
	}
	for _, test := range tests {
		got := deadlineOf(opened, 24*time.Hour, test.now)
		if !got.DueAt.Equal(due) || got.DueAt.Location() != time.UTC || got.Overdue != test.overdue {
			t.Errorf("%s: %+v, want due %v overdue %v", test.name, got, due, test.overdue)
		}
	}
}

package penalties_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/matchmaking"
	"github.com/Alievs-corp/hypedomino/server/internal/penalties"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

func TestTheStoreKeepsTheRecordAndAppliesEachMatchEndOnce(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	var account string
	if err := pool.QueryRow(ctx, `insert into accounts (created_at) values ($1) returning id`, now).Scan(&account); err != nil {
		t.Fatal(err)
	}
	store := penalties.NewStore(pool, penalties.DefaultConfig)
	if record, err := store.Record(ctx, account); err != nil || record.Level(penalties.DefaultConfig(), matchmaking.QueueRanked, now) != 0 {
		t.Fatalf("empty record %+v, %v", record, err)
	}
	leave := rules.PresenceReport{Left: true, ByBot: true}
	for index, match := range []string{"m1", "m1", "m2"} {
		notice, err := store.MatchEnd(ctx, account, matchmaking.QueueRanked, match, rules.SeatX, leave, 0, now.Add(time.Duration(index)*time.Minute))
		if err != nil || (index == 1 && notice.Applied) {
			t.Fatalf("match end %d: %+v, %v", index, notice, err)
		}
	}
	record, err := store.Record(ctx, account)
	if err != nil {
		t.Fatal(err)
	}
	if level := record.Level(penalties.DefaultConfig(), matchmaking.QueueRanked, now); level != 6 || len(record.Ranked.Offenses) != 2 {
		t.Fatalf("level %d, offenses %+v", level, record.Ranked.Offenses)
	}
	if _, banned := record.Banned(now.Add(time.Hour)); !banned {
		t.Fatal("level 6 is not a matchmaking ban")
	}
}

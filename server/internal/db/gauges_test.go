package db_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/db"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
)

func TestPoolGaugesReportTheConnections(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	out := &strings.Builder{}
	if _, err := db.Gauges(pool).WriteTo(out); err != nil {
		t.Fatal(err)
	}
	for _, gauge := range []string{`(?m)^hd_db_connections_max [1-9]`, `(?m)^hd_db_connections_open \d+$`, `(?m)^hd_db_connections_acquired \d+$`, `(?m)^hd_db_acquire_waits \d+$`, `(?m)^hd_db_acquire_canceled \d+$`} {
		if !regexp.MustCompile(gauge).MatchString(out.String()) {
			t.Fatalf("missing %s in\n%s", gauge, out.String())
		}
	}
}

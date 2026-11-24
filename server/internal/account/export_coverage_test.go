package account

import (
	"context"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
)

func TestEveryTableThatReferencesAnAccountIsExportedOrWithheld(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	references, err := accountReferences(ctx, tx)
	if err != nil {
		t.Fatal(err)
	}
	exported := map[string]bool{}
	for _, section := range exportSections {
		exported[section.table] = true
	}
	for _, r := range references {
		if exported[r.table] == withheldFromExport[r.table] {
			t.Errorf("table %s must be either exported or withheld, exported %v withheld %v", r.table, exported[r.table], withheldFromExport[r.table])
		}
	}
}

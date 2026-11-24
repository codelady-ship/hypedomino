package purchase_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/purchase"
)

func TestMeasuredValidatorCountsResultsAndTimesEachStoreCall(t *testing.T) {
	clock := now
	tick := func() time.Time {
		clock = clock.Add(200 * time.Millisecond)
		return clock
	}
	receipts := purchase.NewReceiptMetrics(tick)
	inner := &fakeValidator{
		receipts: map[string]purchase.Receipt{"paid": {State: purchase.StatePurchased}},
		errs:     map[string]error{"down": fmt.Errorf("play: %w", purchase.ErrStore), "odd": errors.New("boom")},
	}
	google := receipts.Measure(purchase.PlatformGoogle, inner)
	for _, token := range []string{"paid", "paid", "down", "missing", "odd"} {
		google.Verify(context.Background(), "coins_100", token)
	}
	if receipt, err := google.Verify(context.Background(), "coins_100", "paid"); err != nil || receipt.TransactionID != "paid" {
		t.Fatalf("measured verify returned %+v %v", receipt, err)
	}
	if err := google.Finish(context.Background(), purchase.Receipt{TransactionID: "paid"}); err != nil || len(inner.finished) != 1 {
		t.Fatalf("finish was not passed through: %v %v", err, inner.finished)
	}
	var results strings.Builder
	receipts.Results().WriteTo(&results)
	want := `hd_receipt_validations_total{store="google",result="error"} 1
hd_receipt_validations_total{store="google",result="store_unavailable"} 1
hd_receipt_validations_total{store="google",result="unknown"} 1
hd_receipt_validations_total{store="google",result="valid"} 3
`
	if !strings.HasSuffix(results.String(), want) {
		t.Fatalf("results\n%s", results.String())
	}
	var durations strings.Builder
	receipts.Durations().WriteTo(&durations)
	for _, line := range []string{
		`hd_receipt_validation_seconds_bucket{store="google",le="0.1"} 0`,
		`hd_receipt_validation_seconds_bucket{store="google",le="0.25"} 6`,
		`hd_receipt_validation_seconds_count{store="google"} 6`,
	} {
		if !strings.Contains(durations.String(), line+"\n") {
			t.Fatalf("durations miss %q\n%s", line, durations.String())
		}
	}
}

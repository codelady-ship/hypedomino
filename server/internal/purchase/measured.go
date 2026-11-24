package purchase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/metrics"
)

var receiptBounds = []time.Duration{
	50 * time.Millisecond, 100 * time.Millisecond, 250 * time.Millisecond, 500 * time.Millisecond,
	time.Second, 2500 * time.Millisecond, 5 * time.Second, 10 * time.Second,
}

var receiptErrors = []error{ErrMalformed, ErrSignature, ErrApp, ErrEnvironment, ErrNotFound, ErrStore}

type ReceiptMetrics struct {
	results   *metrics.Counter
	durations *metrics.Histograms
	now       func() time.Time
}

func NewReceiptMetrics(now func() time.Time) *ReceiptMetrics {
	return &ReceiptMetrics{
		results:   metrics.NewCounter("hd_receipt_validations_total", "Store receipt validations by store and result: valid or the receipt error.", metrics.Labels{"store", "result"}),
		durations: metrics.NewHistograms("hd_receipt_validation_seconds", "Time to validate one store receipt, store call included.", metrics.Labels{"store"}, receiptBounds),
		now:       now,
	}
}

func (m *ReceiptMetrics) Measure(platform Platform, next Validator) Validator {
	return measured{Validator: next, platform: string(platform), metrics: m}
}

func (m *ReceiptMetrics) Results() *metrics.Counter {
	return m.results
}

func (m *ReceiptMetrics) Durations() *metrics.Histograms {
	return m.durations
}

type measured struct {
	Validator
	platform string
	metrics  *ReceiptMetrics
}

func (v measured) Verify(ctx context.Context, productID, token string) (Receipt, error) {
	started := v.metrics.now()
	receipt, err := v.Validator.Verify(ctx, productID, token)
	v.metrics.durations.Observe(metrics.Labels{v.platform}, v.metrics.now().Sub(started))
	v.metrics.results.Inc(metrics.Labels{v.platform, receiptResult(err)})
	return receipt, err
}

func receiptResult(err error) string {
	if err == nil {
		return "valid"
	}
	for _, known := range receiptErrors {
		if errors.Is(err, known) {
			return strings.TrimPrefix(known.Error(), "receipt.")
		}
	}
	return "error"
}

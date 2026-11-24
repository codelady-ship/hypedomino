package purchase_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/purchase"
	"github.com/Alievs-corp/hypedomino/server/internal/risk"
)

func TestEveryFakeReceiptRaisesOneHardSignalAndOtherRefusalsNone(t *testing.T) {
	f := newFixture(t, purchase.DefaultConfig())
	ctx := context.Background()
	ayla, bora := f.account(t), f.account(t)
	f.receipt("tok-1", "offer_starter", ayla, purchase.StatePurchased)
	if _, err := f.store.Complete(ctx, ayla, "offer.starter", purchase.PlatformGoogle, "tok-1", now); err != nil {
		t.Fatal(err)
	}
	f.receipt("tok-1", "offer_starter", bora, purchase.StatePurchased)
	f.receipt("tok-other", "crystals_tier1", ayla, purchase.StatePurchased)
	f.receipt("tok-wrong-product", "crystals_tier2", bora, purchase.StatePurchased)
	f.receipt("tok-cancelled", "crystals_tier1", bora, purchase.StateCancelled)
	f.google.errs["tok-forged"] = purchase.ErrSignature
	f.google.errs["tok-sandbox"] = purchase.ErrEnvironment
	f.google.errs["tok-down"] = purchase.ErrStore
	f.google.errs["tok-malformed"] = purchase.ErrMalformed
	hashed := func(token string) string {
		sum := sha256.Sum256([]byte(token))
		return "google:token:" + hex.EncodeToString(sum[:])
	}
	cases := []struct {
		offer, token string
		want         error
		signals      int
		dedupe, code string
	}{
		{"offer.starter", "tok-1", purchase.ErrReused, 1, "google:tok-1", "receipt.reused"},
		{"offer.crystals_tier1", "tok-other", purchase.ErrAccount, 2, "google:tok-other", "receipt.account"},
		{"offer.crystals_tier1", "tok-forged", purchase.ErrSignature, 3, hashed("tok-forged"), "receipt.signature"},
		{"offer.crystals_tier1", "tok-nope", purchase.ErrNotFound, 4, hashed("tok-nope"), "receipt.unknown"},
		{"offer.crystals_tier1", "tok-sandbox", purchase.ErrEnvironment, 5, hashed("tok-sandbox"), "receipt.environment"},
		{"offer.crystals_tier2", "tok-nope", purchase.ErrNotFound, 5, "", ""},
		{"offer.crystals_tier1", "tok-other", purchase.ErrAccount, 5, "", ""},
		{"offer.crystals_tier1", "tok-wrong-product", purchase.ErrProduct, 5, "", ""},
		{"offer.crystals_tier1", "tok-cancelled", purchase.ErrCancelled, 5, "", ""},
		{"offer.crystals_tier1", "tok-down", purchase.ErrStore, 5, "", ""},
		{"offer.crystals_tier1", "tok-malformed", purchase.ErrMalformed, 5, "", ""},
	}
	for _, c := range cases {
		if _, err := f.store.Complete(ctx, bora, c.offer, purchase.PlatformGoogle, c.token, now); !errors.Is(err, c.want) {
			t.Fatalf("%s: %v", c.token, err)
		}
		signals, err := f.signals.Signals(ctx, bora)
		if err != nil || len(signals) != c.signals {
			t.Fatalf("%s: %d signals %v", c.token, len(signals), err)
		}
		if c.code == "" {
			continue
		}
		last := signals[len(signals)-1]
		if last.Kind != risk.KindFakeReceipt || last.Weight != 20 || last.Dedupe != c.dedupe || last.Evidence["code"] != c.code || last.Evidence["offer"] != c.offer || last.Evidence["platform"] != "google" {
			t.Fatalf("%s: signal %+v", c.token, last)
		}
	}
	if signals, _ := f.signals.Signals(ctx, ayla); len(signals) != 0 {
		t.Fatalf("the honest buyer was flagged: %+v", signals)
	}
}

func TestARefundRaisesOneSoftSignalPerTransaction(t *testing.T) {
	f := newFixture(t, purchase.DefaultConfig())
	ctx := context.Background()
	ayla := f.account(t)
	f.receipt("tok-1", "crystals_tier1", ayla, purchase.StatePurchased)
	if _, err := f.store.Complete(ctx, ayla, "offer.crystals_tier1", purchase.PlatformGoogle, "tok-1", now); err != nil {
		t.Fatal(err)
	}
	for i, notice := range []purchase.Notice{
		{ID: "r-1", Kind: purchase.KindRefund, TransactionID: "tok-1"},
		{ID: "r-1", Kind: purchase.KindRefund, TransactionID: "tok-1"},
		{ID: "v-1", Kind: purchase.KindReversal, TransactionID: "tok-1"},
		{ID: "r-2", Kind: purchase.KindRefund, TransactionID: "tok-1"},
	} {
		if _, err := f.store.Notice(ctx, purchase.PlatformGoogle, notice, now.Add(time.Duration(i+1)*time.Hour)); err != nil {
			t.Fatalf("notice %d: %v", i, err)
		}
	}
	if f.state(t, "tok-1") != purchase.RecordRefunded {
		t.Fatal("the second refund was not applied")
	}
	signals, err := f.signals.Signals(ctx, ayla)
	if err != nil || len(signals) != 1 {
		t.Fatalf("%d signals %v", len(signals), err)
	}
	if s := signals[0]; s.Kind != risk.KindRefund || s.Weight != 3 || s.Dedupe != "google:tok-1" || s.Evidence["notice"] != "r-1" || s.Evidence["offer"] != "offer.crystals_tier1" || !s.At.Equal(now.Add(time.Hour)) {
		t.Fatalf("refund signal %+v", s)
	}
}

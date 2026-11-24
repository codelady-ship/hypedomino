package purchase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/risk"
)

var fakeRefusals = []error{ErrSignature, ErrNotFound, ErrEnvironment, ErrAccount, ErrReused}

func fakeRefusal(err error) (string, bool) {
	for _, refusal := range fakeRefusals {
		if errors.Is(err, refusal) {
			return refusal.Error(), true
		}
	}
	return "", false
}

func (s *Store) flagFake(ctx context.Context, account string, platform Platform, offer string, refusal error, transaction, token string, now time.Time) {
	code, ok := fakeRefusal(refusal)
	if !ok {
		return
	}
	s.logger.Warn("purchase refused as a fake receipt", "account", account, "platform", platform, "offer", offer, "code", code)
	if s.signals == nil {
		return
	}
	dedupe := string(platform) + ":" + transaction
	if transaction == "" {
		sum := sha256.Sum256([]byte(token))
		dedupe = string(platform) + ":token:" + hex.EncodeToString(sum[:])
	}
	signal := risk.Signal{
		Account: account, Kind: risk.KindFakeReceipt, Weight: s.risk.FakeReceiptWeight, Dedupe: dedupe, At: now,
		Evidence: map[string]any{"platform": platform, "offer": offer, "code": code, "transaction_id": transaction},
	}
	if _, err := s.signals.Record(ctx, signal); err != nil {
		s.logger.Error("fake receipt signal not recorded", "account", account, "error", err)
	}
}

func (s *Store) flagRefund(ctx context.Context, tx pgx.Tx, r record, notice Notice, now time.Time) error {
	if s.signals == nil {
		return nil
	}
	_, err := risk.RecordIn(ctx, tx, risk.Signal{
		Account: r.Account, Kind: risk.KindRefund, Weight: s.risk.RefundWeight, Dedupe: string(r.Platform) + ":" + r.TransactionID, At: now,
		Evidence: map[string]any{"platform": r.Platform, "offer": r.Offer, "transaction_id": r.TransactionID, "notice": notice.ID},
	})
	return err
}

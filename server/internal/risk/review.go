package risk

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	DecisionClear   = "clear"
	DecisionConfirm = "confirm"
)

var (
	ErrNoReview     = errors.New("risk: no such review")
	ErrReviewClosed = errors.New("risk: review already decided")
)

type ReviewSignal struct {
	Kind     string          `json:"kind"`
	Weight   int             `json:"weight"`
	Evidence json.RawMessage `json:"evidence"`
	At       time.Time       `json:"at"`
}

type Review struct {
	ID             int64          `json:"id"`
	Account        string         `json:"account_id"`
	Nickname       string         `json:"nickname"`
	Cause          string         `json:"cause"`
	OpenedAt       time.Time      `json:"opened_at"`
	ShadowbannedAt *time.Time     `json:"shadowbanned_at"`
	Signals        []ReviewSignal `json:"signals"`
}

type Decided struct {
	Account           string
	WasShadowbanned   bool
	StillShadowbanned bool
}

type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type ReviewQuery struct {
	Cause   string
	Account string
	After   int64
	Limit   int
}

func OpenReviews(ctx context.Context, db Querier, q ReviewQuery, signalLimit int) ([]Review, error) {
	rows, err := db.Query(ctx, `select r.id, r.account_id::text, coalesce(p.nickname, ''), r.cause, r.opened_at, a.shadowbanned_at
		from risk_reviews r join accounts a on a.id = r.account_id left join profiles p on p.account_id = r.account_id
		where r.closed_at is null and ($1 = '' or r.cause = $1) and starts_with(r.account_id::text, $2)
			and ($3::bigint = 0 or (r.opened_at, r.id) > (select opened_at, id from risk_reviews where id = $3))
		order by r.opened_at, r.id limit $4`, q.Cause, q.Account, q.After, q.Limit)
	if err != nil {
		return nil, err
	}
	reviews, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Review, error) {
		var review Review
		err := row.Scan(&review.ID, &review.Account, &review.Nickname, &review.Cause, &review.OpenedAt, &review.ShadowbannedAt)
		return review, err
	})
	if err != nil {
		return nil, err
	}
	for index := range reviews {
		if reviews[index].Signals, err = reviewSignals(ctx, db, reviews[index].Account, signalLimit); err != nil {
			return nil, err
		}
	}
	return reviews, nil
}

func reviewSignals(ctx context.Context, db Querier, account string, limit int) ([]ReviewSignal, error) {
	rows, err := db.Query(ctx, `select s.kind, s.weight, s.evidence, s.created_at from risk_signals s join accounts a on a.id = s.account_id
		where s.account_id = $1 and (a.risk_reviewed_at is null or s.created_at > a.risk_reviewed_at)
		order by s.created_at desc, s.id desc limit $2`, account, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (ReviewSignal, error) {
		var signal ReviewSignal
		err := row.Scan(&signal.Kind, &signal.Weight, &signal.Evidence, &signal.At)
		return signal, err
	})
}

func OpenReviewAccount(ctx context.Context, tx pgx.Tx, id int64) (string, error) {
	var account string
	var closedAt *time.Time
	err := tx.QueryRow(ctx, `select account_id::text, closed_at from risk_reviews where id = $1 for update`, id).Scan(&account, &closedAt)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return "", ErrNoReview
	case err != nil:
		return "", err
	case closedAt != nil:
		return "", ErrReviewClosed
	}
	return account, nil
}

func DecideReview(ctx context.Context, tx pgx.Tx, id int64, decision, decider string, now time.Time) (Decided, error) {
	var decided Decided
	var closedAt *time.Time
	err := tx.QueryRow(ctx, `select r.account_id::text, r.closed_at, a.shadowbanned_at is not null from risk_reviews r
		join accounts a on a.id = r.account_id where r.id = $1 for update of r, a`, id).Scan(&decided.Account, &closedAt, &decided.WasShadowbanned)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Decided{}, ErrNoReview
	case err != nil:
		return Decided{}, err
	case closedAt != nil:
		return Decided{}, ErrReviewClosed
	}
	decided.StillShadowbanned = decided.WasShadowbanned && decision != DecisionClear
	if _, err := tx.Exec(ctx, `update risk_reviews set closed_at = $2, decision = $3, decided_by = $4 where id = $1`, id, now, decision, decider); err != nil {
		return Decided{}, err
	}
	_, err = tx.Exec(ctx, `update accounts set risk_reviewed_at = $2,
		shadowbanned_at = case when $3::boolean then shadowbanned_at end where id = $1`, decided.Account, now, decided.StillShadowbanned)
	return decided, err
}

package appeal

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/sanction"
)

type Entry struct {
	Appeal
	Nickname string            `json:"nickname"`
	Sanction sanction.Sanction `json:"sanction"`
}

type Query struct {
	Kind    string
	Account string
	After   int64
	Limit   int
}

func Open(ctx context.Context, db sanction.Querier, q Query) ([]Entry, error) {
	rows, err := db.Query(ctx, `select a.id, a.account_id::text, a.sanction_id, a.text, a.state, a.created_at, a.decided_at, coalesce(p.nickname, ''),
			s.id, s.kind, s.category, s.starts_at, s.ends_at, s.lifted_at
		from appeals a join account_sanctions s on s.id = a.sanction_id left join profiles p on p.account_id = a.account_id
		where a.state = 'open' and starts_with(a.account_id::text, $1) and a.id > $2 and ($4 = '' or s.kind = $4)
		order by a.id limit $3`, q.Account, q.After, q.Limit, q.Kind)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Entry, error) {
		var e Entry
		s := &e.Sanction
		err := row.Scan(&e.ID, &e.AccountID, &e.SanctionID, &e.Text, &e.State, &e.CreatedAt, &e.DecidedAt, &e.Nickname,
			&s.ID, &s.Kind, &s.Category, &s.StartsAt, &s.EndsAt, &s.LiftedAt)
		e.CreatedAt = e.CreatedAt.UTC()
		return e, err
	})
}

func AccountOf(ctx context.Context, db sanction.Querier, id int64) (string, error) {
	var account string
	err := db.QueryRow(ctx, `select account_id::text from appeals where id = $1 and state = 'open'`, id).Scan(&account)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNoAppeal
	}
	return account, err
}

func Decide(ctx context.Context, db sanction.Querier, id int64, decision, decider string, now time.Time) (Appeal, sanction.Sanction, error) {
	if decision != StateUpheld && decision != StateLifted {
		return Appeal{}, sanction.Sanction{}, ErrDecision
	}
	decided, err := scan(db.QueryRow(ctx, `update appeals set state = $2, decided_at = $3, decided_by = $4 where id = $1 and state = 'open' returning `+columns,
		id, decision, now, decider))
	if errors.Is(err, pgx.ErrNoRows) {
		return Appeal{}, sanction.Sanction{}, ErrNoAppeal
	}
	if err != nil {
		return Appeal{}, sanction.Sanction{}, err
	}
	appealed, err := sanction.Lock(ctx, db, decided.AccountID, decided.SanctionID)
	return decided, appealed, err
}

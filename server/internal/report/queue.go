package report

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	StateUpheld   = "upheld"
	StateRejected = "rejected"
)

var ErrNothingOpen = errors.New("report: no open report on the account")

type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type Entry struct {
	ID        int64     `json:"id"`
	Reporter  string    `json:"reporter"`
	Reason    string    `json:"reason"`
	NameText  string    `json:"name_text"`
	MatchID   *string   `json:"match_id"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Target struct {
	AccountID      string  `json:"account_id"`
	Nickname       string  `json:"nickname"`
	RenameRequired bool    `json:"rename_required"`
	Reports        []Entry `json:"reports"`
}

type Query struct {
	Reason  string
	Account string
	After   string
	Limit   int
}

func Open(ctx context.Context, db Querier, q Query) ([]Target, error) {
	rows, err := db.Query(ctx, `select r.target::text, coalesce(p.nickname, ''), coalesce(p.rename_required, false),
		r.id, r.reporter::text, r.reason, r.name_text, r.match_id::text, r.updated_at
		from player_reports r left join profiles p on p.account_id = r.target
		where r.state = 'open' and r.target in (
			select target from player_reports where state = 'open' and starts_with(target::text, $2) group by target
			having ($1 = '' or bool_or(reason = $1))
				and ($3 = '' or (min(updated_at), target) > (select coalesce(min(updated_at) filter (where state = 'open'), min(updated_at)), nullif($3, '')::uuid
					from player_reports where target = nullif($3, '')::uuid))
			order by min(updated_at), target limit $4)
		order by min(r.updated_at) over (partition by r.target), r.target, r.updated_at`, q.Reason, q.Account, q.After, q.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var targets []Target
	for rows.Next() {
		var target Target
		var entry Entry
		if err := rows.Scan(&target.AccountID, &target.Nickname, &target.RenameRequired, &entry.ID, &entry.Reporter, &entry.Reason, &entry.NameText, &entry.MatchID, &entry.UpdatedAt); err != nil {
			return nil, err
		}
		entry.UpdatedAt = entry.UpdatedAt.UTC()
		if n := len(targets); n > 0 && targets[n-1].AccountID == target.AccountID {
			targets[n-1].Reports = append(targets[n-1].Reports, entry)
			continue
		}
		target.Reports = []Entry{entry}
		targets = append(targets, target)
	}
	return targets, rows.Err()
}

func Decide(ctx context.Context, tx pgx.Tx, target, state, by string, now time.Time) (int64, error) {
	tag, err := tx.Exec(ctx, `update player_reports set state = $2, decided_at = $3, decided_by = $4 where target = $1 and state = 'open'`, target, state, now, by)
	if err != nil {
		return 0, err
	}
	if tag.RowsAffected() == 0 {
		return 0, ErrNothingOpen
	}
	return tag.RowsAffected(), nil
}

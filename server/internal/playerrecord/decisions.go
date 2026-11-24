package playerrecord

import (
	"context"
	"time"
)

type Decision struct {
	Queue     string    `json:"queue"`
	Outcome   string    `json:"outcome"`
	Reasons   []string  `json:"reasons"`
	Reports   int       `json:"reports"`
	DecidedAt time.Time `json:"decided_at"`
}

func Decisions(ctx context.Context, db Querier, accounts []string, limit int) (map[string][]Decision, error) {
	decisions := make(map[string][]Decision, len(accounts))
	if len(accounts) == 0 {
		return decisions, nil
	}
	rows, err := db.Query(ctx, `select account, queue, outcome, reasons, reports, decided_at from (
			select d.*, row_number() over (partition by account order by decided_at desc, queue) as n from (
				select target::text as account, 'reports' as queue, state as outcome, array_agg(distinct reason order by reason) as reasons,
					count(*)::int as reports, decided_at
				from player_reports where target = any($1::uuid[]) and state <> 'open' and decided_at is not null
				group by target, state, decided_at
				union all
				select account_id::text, 'reviews', decision, array[cause], 0, closed_at
				from risk_reviews where account_id = any($1::uuid[]) and closed_at is not null and decision is not null
				union all
				select a.account_id::text, 'appeals', a.state, array[s.kind], 0, a.decided_at
				from appeals a join account_sanctions s on s.id = a.sanction_id
				where a.account_id = any($1::uuid[]) and a.decided_at is not null
			) d
		) ranked where n <= $2 order by account, decided_at desc, queue`, accounts, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var account string
		var decision Decision
		if err := rows.Scan(&account, &decision.Queue, &decision.Outcome, &decision.Reasons, &decision.Reports, &decision.DecidedAt); err != nil {
			return nil, err
		}
		decision.DecidedAt = decision.DecidedAt.UTC()
		decisions[account] = append(decisions[account], decision)
	}
	return decisions, rows.Err()
}

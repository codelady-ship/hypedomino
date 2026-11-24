package playerrecord

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/rating"
)

type SuspiciousMatch struct {
	MatchID   string    `json:"match_id"`
	StartedAt time.Time `json:"started_at"`
	Mode      string    `json:"mode"`
	Kind      string    `json:"kind"`
	Signals   []string  `json:"signals"`
	Accounts  []string  `json:"accounts"`
}

type SuspiciousQuery struct {
	Signal  string
	Account string
	After   string
	Limit   int
}

func LoadSuspiciousMatches(ctx context.Context, db Querier, q SuspiciousQuery) ([]SuspiciousMatch, error) {
	var matches []SuspiciousMatch
	err := collect(ctx, db, &matches, `select m.id::text, m.started_at, m.mode, m.kind, array_agg(distinct s.kind order by s.kind),
		array(select a from unnest(array_agg(mp.account_id::text) || array_agg(o.account_id::text)) a group by a order by a collate "C")
		from rating_signals s
		join match_participants mp on mp.account_id = s.account_id
		join match_participants o on o.match_id = mp.match_id and o.started_at = mp.started_at and o.account_id = s.counterpart
		join matches m on m.id = mp.match_id and m.started_at = mp.started_at and m.mode = s.mode
		where s.kind = any($1) and m.started_at > s.updated_at - make_interval(hours => s.window_hours)
			and ($4 = '' or (m.started_at, m.id) < (select started_at, id from matches where id = nullif($4, '')::uuid order by started_at desc limit 1))
		group by m.id, m.started_at, m.mode, m.kind
		having ($2 = '' or bool_or(s.kind = $2)) and bool_or(starts_with(s.account_id::text, $3) or starts_with(s.counterpart::text, $3))
		order by m.started_at desc, m.id desc limit $5`,
		func(row pgx.CollectableRow, m *SuspiciousMatch) error {
			return row.Scan(&m.MatchID, &m.StartedAt, &m.Mode, &m.Kind, &m.Signals, &m.Accounts)
		}, rating.CollusionSignals, q.Signal, q.Account, q.After, q.Limit)
	if matches == nil {
		matches = []SuspiciousMatch{}
	}
	return matches, err
}

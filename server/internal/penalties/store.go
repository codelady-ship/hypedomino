package penalties

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/matchmaking"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
	"github.com/Alievs-corp/hypedomino/server/internal/sanction"
)

type Store struct {
	pool    *pgxpool.Pool
	current func() (Config, int)
}

func NewStore(pool *pgxpool.Pool, current func() (Config, int)) *Store {
	return &Store{pool: pool, current: current}
}

func (s *Store) Config() (Config, int) {
	return s.current()
}

func (s *Store) Record(ctx context.Context, account string) (Record, error) {
	var raw []byte
	err := s.pool.QueryRow(ctx, `select record from penalty_records where account_id = $1`, account).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return Record{}, nil
	}
	if err != nil {
		return Record{}, err
	}
	var record Record
	return record, json.Unmarshal(raw, &record)
}

func (s *Store) MatchEnd(ctx context.Context, account string, config Config, queue matchmaking.Queue, matchID string, seat rules.Seat, report rules.PresenceReport, configVersion int, now time.Time) (Notice, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Notice{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `insert into penalty_records (account_id, record, updated_at) values ($1, '{}', $2) on conflict do nothing`, account, now); err != nil {
		return Notice{}, err
	}
	var raw []byte
	if err := tx.QueryRow(ctx, `select record from penalty_records where account_id = $1 for update`, account).Scan(&raw); err != nil {
		return Notice{}, err
	}
	var record Record
	if err := json.Unmarshal(raw, &record); err != nil {
		return Notice{}, err
	}
	bans := len(record.Ranked.Bans)
	notice := record.MatchEnd(config, queue, matchID, seat, report, configVersion, now)
	if len(record.Ranked.Bans) > bans {
		if record.Ranked.BanSanction, err = imposeBan(ctx, tx, account, record.Ranked, now); err != nil {
			return Notice{}, err
		}
	}
	if raw, err = json.Marshal(record); err != nil {
		return Notice{}, err
	}
	if _, err := tx.Exec(ctx, `update penalty_records set record = $2, updated_at = $3 where account_id = $1`, account, raw, now); err != nil {
		return Notice{}, err
	}
	return notice, tx.Commit(ctx)
}

func imposeBan(ctx context.Context, tx pgx.Tx, account string, ladder Ladder, now time.Time) (int64, error) {
	var ends *time.Time
	if !ladder.UnderReview {
		until := ladder.BanUntil.UTC().Truncate(time.Microsecond)
		ends = &until
	}
	imposed, err := sanction.Impose(ctx, tx, account, sanction.PenaltyBan, sanction.AfkLeaving, now, ends)
	return imposed.ID, err
}

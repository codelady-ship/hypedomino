package report

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/sanction"
)

const (
	ReasonNickname      = "offensive_nickname"
	ReasonRoomName      = "offensive_room_name"
	ReasonImpersonation = "impersonation"
	ReasonOther         = "other"
)

var (
	Reasons = []string{ReasonNickname, ReasonRoomName, ReasonImpersonation, ReasonOther}

	ErrUnknownPlayer = errors.New("report.unknown_player")
	ErrSelf          = errors.New("report.self")
	ErrReason        = errors.New("report.reason")
	ErrMatch         = errors.New("report.match")
	ErrInvalidConfig = errors.New("report: invalid config")
)

type Config struct {
	AutoRenameThreshold int
	AutoRenameWindow    time.Duration
	DailyLimit          int
	RejectedLimit       int
	RejectedWindow      time.Duration
	RejectedSilence     time.Duration
}

func DefaultConfig() Config {
	return Config{AutoRenameThreshold: 5, AutoRenameWindow: 7 * 24 * time.Hour, DailyLimit: 10, RejectedLimit: 5, RejectedWindow: 30 * 24 * time.Hour, RejectedSilence: 7 * 24 * time.Hour}
}

func (c Config) Validate() error {
	if c.AutoRenameThreshold < 0 || c.DailyLimit < 1 || c.RejectedLimit < 1 {
		return fmt.Errorf("%w: the auto-rename threshold must not be negative and the limits must be at least 1", ErrInvalidConfig)
	}
	if c.AutoRenameWindow <= 0 || c.RejectedWindow <= 0 || c.RejectedSilence <= 0 {
		return fmt.Errorf("%w: windows must be positive", ErrInvalidConfig)
	}
	return nil
}

type Report struct {
	Target  string `json:"target"`
	Reason  string `json:"reason"`
	MatchID string `json:"match_id,omitempty"`
}

type Store struct {
	pool   *pgxpool.Pool
	config Config
}

func NewStore(pool *pgxpool.Pool, config Config) *Store {
	return &Store{pool: pool, config: config}
}

func (r Report) check(reporter string) (string, *uuid.UUID, error) {
	if !slices.Contains(Reasons, r.Reason) {
		return "", nil, ErrReason
	}
	target, err := uuid.Parse(r.Target)
	if err != nil {
		return "", nil, ErrUnknownPlayer
	}
	if target.String() == reporter {
		return "", nil, ErrSelf
	}
	if r.MatchID == "" {
		return target.String(), nil, nil
	}
	match, err := uuid.Parse(r.MatchID)
	if err != nil {
		return "", nil, ErrMatch
	}
	return target.String(), &match, nil
}

func (s *Store) File(ctx context.Context, reporter string, r Report, now time.Time) error {
	target, match, err := r.check(reporter)
	if err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var name string
		err := tx.QueryRow(ctx, `select coalesce(p.nickname, '') from accounts a left join profiles p on p.account_id = a.id where a.id = $1`, target).Scan(&name)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUnknownPlayer
		}
		if err != nil {
			return err
		}
		silenced, err := s.silenced(ctx, tx, reporter, now)
		if err != nil || silenced {
			return err
		}
		if _, err := tx.Exec(ctx, `insert into player_reports (reporter, target, reason, name_text, match_id, created_at, updated_at)
			values ($1, $2, $3, $4, $5, $6, $6)
			on conflict (reporter, target) where state = 'open'
			do update set reason = excluded.reason, name_text = excluded.name_text, match_id = excluded.match_id, updated_at = excluded.updated_at`,
			reporter, target, r.Reason, name, match, now); err != nil {
			return err
		}
		if r.Reason != ReasonNickname {
			return nil
		}
		return s.renameOnThreshold(ctx, tx, target, now)
	})
}

func (s *Store) silenced(ctx context.Context, tx pgx.Tx, reporter string, now time.Time) (bool, error) {
	var recent, rejected int
	var lastRejected *time.Time
	err := tx.QueryRow(ctx, `select
		count(*) filter (where updated_at > $2),
		count(*) filter (where state = 'rejected' and decided_at > $3),
		max(decided_at) filter (where state = 'rejected')
		from player_reports where reporter = $1`,
		reporter, now.Add(-24*time.Hour), now.Add(-s.config.RejectedWindow)).Scan(&recent, &rejected, &lastRejected)
	if err != nil {
		return false, err
	}
	if recent >= s.config.DailyLimit {
		return true, nil
	}
	return rejected >= s.config.RejectedLimit && lastRejected != nil && now.Sub(*lastRejected) < s.config.RejectedSilence, nil
}

func (s *Store) renameOnThreshold(ctx context.Context, tx pgx.Tx, target string, now time.Time) error {
	if s.config.AutoRenameThreshold == 0 {
		return nil
	}
	var reporters int
	if err := tx.QueryRow(ctx, `select count(distinct reporter) from player_reports
		where target = $1 and reason = $2 and state = 'open' and updated_at > $3`,
		target, ReasonNickname, now.Add(-s.config.AutoRenameWindow)).Scan(&reporters); err != nil {
		return err
	}
	if reporters < s.config.AutoRenameThreshold {
		return nil
	}
	tag, err := tx.Exec(ctx, `update profiles set rename_required = true, updated_at = $2 where account_id = $1 and not rename_required`, target, now)
	if err != nil || tag.RowsAffected() == 0 {
		return err
	}
	_, err = sanction.Impose(ctx, tx, target, sanction.ForcedRename, sanction.OffensiveName, now, nil)
	return err
}

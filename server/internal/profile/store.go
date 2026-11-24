package profile

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/sanction"
)

var ErrUnknownAccount = errors.New("profile: unknown account")

type Loadout struct {
	Avatar string   `json:"avatar,omitempty"`
	Frame  string   `json:"frame,omitempty"`
	Table  string   `json:"table,omitempty"`
	Tile   string   `json:"tile,omitempty"`
	Emojis []string `json:"emojis,omitempty"`
}

type ModeStats struct {
	Games  int `json:"games"`
	Wins   int `json:"wins"`
	Losses int `json:"losses"`
}

type Stats struct {
	Games      int                  `json:"games"`
	Wins       int                  `json:"wins"`
	Losses     int                  `json:"losses"`
	Streak     int                  `json:"streak"`
	BestStreak int                  `json:"best_streak"`
	Modes      map[string]ModeStats `json:"modes,omitempty"`
}

func (s Stats) WinRatePercent() int {
	if s.Games == 0 {
		return 0
	}
	return s.Wins * 100 / s.Games
}

type Profile struct {
	Account        string
	Nickname       string
	RenamedAt      time.Time
	RenameRequired bool
	Loadout        Loadout
	Stats          Stats
}

type Store struct {
	pool       *pgxpool.Pool
	config     Config
	moderation func() Moderation
}

func NewStore(pool *pgxpool.Pool, config Config) *Store {
	return &Store{pool: pool, config: config}
}

func (s *Store) Moderate(current func() Moderation) {
	s.moderation = current
}

const profileColumns = `account_id, nickname, nickname_changed_at, rename_required, loadout, stats`

const noAccount = "00000000-0000-0000-0000-000000000000"

func scan(row pgx.Row) (Profile, error) {
	var p Profile
	var loadout, stats []byte
	var renamed *time.Time
	if err := row.Scan(&p.Account, &p.Nickname, &renamed, &p.RenameRequired, &loadout, &stats); err != nil {
		return Profile{}, err
	}
	if renamed != nil {
		p.RenamedAt = renamed.UTC()
	}
	if err := json.Unmarshal(loadout, &p.Loadout); err != nil {
		return Profile{}, err
	}
	if err := json.Unmarshal(stats, &p.Stats); err != nil {
		return Profile{}, err
	}
	return p, nil
}

func (s *Store) Get(ctx context.Context, account string) (Profile, bool, error) {
	p, err := scan(s.pool.QueryRow(ctx, `select `+profileColumns+` from profiles where account_id = $1`, account))
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, false, nil
	}
	return p, err == nil, err
}

func (s *Store) Nicknames(ctx context.Context, accounts []string) (map[string]string, error) {
	rows, err := s.pool.Query(ctx, `select account_id::text, nickname from profiles where account_id = any($1::uuid[])`, accounts)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	names := make(map[string]string, len(accounts))
	for rows.Next() {
		var account, nickname string
		if err := rows.Scan(&account, &nickname); err != nil {
			return nil, err
		}
		names[account] = nickname
	}
	return names, rows.Err()
}

func (s *Store) Exists(ctx context.Context, account string) (bool, error) {
	var state string
	err := s.pool.QueryRow(ctx, `select state from accounts where id = $1`, account).Scan(&state)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (s *Store) RenameAvailableAt(p Profile) (time.Time, bool) {
	if p.RenamedAt.IsZero() || p.RenameRequired {
		return time.Time{}, false
	}
	return p.RenamedAt.Add(s.config.NicknameRenameCooldown), true
}

func (s *Store) SetNickname(ctx context.Context, account, raw string, now time.Time) (Profile, error) {
	display := Display(raw)
	if err := s.config.Validate(display); err != nil {
		return Profile{}, err
	}
	if s.moderation != nil && !s.moderation().Allows(display) {
		return Profile{}, ErrNotAllowed
	}
	key := Key(display)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Profile{}, err
	}
	defer tx.Rollback(ctx)
	current, found, err := lock(ctx, tx, account)
	if err != nil {
		return Profile{}, err
	}
	if found && current.RenameRequired && Key(current.Nickname) == key {
		return Profile{}, ErrUnchanged
	}
	if found && current.Nickname == display {
		return current, nil
	}
	if found {
		if available, cooling := s.RenameAvailableAt(current); cooling && now.Before(available) {
			return Profile{}, fmt.Errorf("%w until %s", ErrCooldown, available.UTC().Format(time.RFC3339))
		}
	}
	if err := s.free(ctx, tx, key, account, now); err != nil {
		return Profile{}, err
	}
	if found {
		if _, err := tx.Exec(ctx, `insert into nickname_history (account_id, nickname, nickname_key, replaced_at) values ($1, $2, $3, $4)`,
			account, current.Nickname, Key(current.Nickname), now); err != nil {
			return Profile{}, err
		}
		_, err = tx.Exec(ctx, `update profiles set nickname = $2, nickname_key = $3, updated_at = $4, rename_required = false,
			nickname_changed_at = case when rename_required then nickname_changed_at else $4 end where account_id = $1`, account, display, key, now)
		if err == nil && current.RenameRequired {
			err = sanction.End(ctx, tx, account, sanction.ForcedRename, now)
		}
	} else {
		_, err = tx.Exec(ctx, `insert into profiles (account_id, nickname, nickname_key, updated_at) values ($1, $2, $3, $4)`, account, display, key, now)
	}
	if isUniqueViolation(err) {
		return Profile{}, ErrTaken
	}
	if err != nil {
		return Profile{}, err
	}
	p, err := scan(tx.QueryRow(ctx, `select `+profileColumns+` from profiles where account_id = $1`, account))
	if err != nil {
		return Profile{}, err
	}
	return p, tx.Commit(ctx)
}

func lock(ctx context.Context, tx pgx.Tx, account string) (Profile, bool, error) {
	p, err := scan(tx.QueryRow(ctx, `select `+profileColumns+` from profiles where account_id = $1 for update`, account))
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, false, nil
	}
	return p, err == nil, err
}

func (s *Store) free(ctx context.Context, tx pgx.Tx, key, account string, now time.Time) error {
	var taken bool
	err := tx.QueryRow(ctx, `select exists (select 1 from profiles where nickname_key = $1 and account_id <> $2)
		or exists (select 1 from nickname_history where nickname_key = $1 and account_id <> $2 and replaced_at > $3)`,
		key, account, now.Add(-s.config.NicknameRelease)).Scan(&taken)
	if err != nil {
		return err
	}
	if taken {
		return ErrTaken
	}
	return nil
}

func (s *Store) Suggest(ctx context.Context, now time.Time) (string, error) {
	for attempt := 0; attempt < 8; attempt++ {
		n, err := rand.Int(rand.Reader, big.NewInt(900000))
		if err != nil {
			return "", err
		}
		candidate := fmt.Sprintf("Player%06d", n.Int64()+100000)
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return "", err
		}
		err = s.free(ctx, tx, Key(candidate), noAccount, now)
		tx.Rollback(ctx)
		if err == nil {
			return candidate, nil
		}
		if !errors.Is(err, ErrTaken) {
			return "", err
		}
	}
	return "", ErrTaken
}

func (s *Store) RecordMatch(ctx context.Context, account, mode string, won bool, now time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	p, found, err := lock(ctx, tx, account)
	if err != nil {
		return err
	}
	if !found {
		return ErrUnknownAccount
	}
	stats := p.Stats
	if stats.Modes == nil {
		stats.Modes = map[string]ModeStats{}
	}
	m := stats.Modes[mode]
	stats.Games++
	m.Games++
	if won {
		stats.Wins++
		m.Wins++
		stats.Streak++
		stats.BestStreak = max(stats.BestStreak, stats.Streak)
	} else {
		stats.Losses++
		m.Losses++
		stats.Streak = 0
	}
	stats.Modes[mode] = m
	encoded, err := json.Marshal(stats)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `update profiles set stats = $2, updated_at = $3 where account_id = $1`, account, encoded, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

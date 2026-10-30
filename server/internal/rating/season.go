package rating

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

const (
	StateScheduled = "scheduled"
	StateLive      = "live"
	StateEnding    = "ending"
	StateClosed    = "closed"
)

var (
	ErrSeason           = errors.New("rating: invalid season")
	ErrSeasonUnknown    = errors.New("rating.season_unknown")
	ErrSeasonOverlap    = errors.New("rating.season_overlap")
	ErrSeasonGap        = errors.New("rating.season_gap")
	ErrSeasonEarlier    = errors.New("rating.season_earlier")
	ErrNoSeason         = errors.New("rating.no_season")
	ErrSeasonTransition = errors.New("rating.season_transition")
	ErrSeasonOpen       = errors.New("rating.season_open")

	seasonID = regexp.MustCompile(`^season\.[a-z0-9_]+$`)
	modes    = []rules.Mode{rules.Mode1v1, rules.Mode2v2}
)

type Season struct {
	ID        string     `json:"id"`
	NameKey   string     `json:"name_key"`
	Start     time.Time  `json:"start"`
	End       time.Time  `json:"end"`
	Closed1v1 *time.Time `json:"closed_1v1,omitempty"`
	Closed2v2 *time.Time `json:"closed_2v2,omitempty"`
}

func (s Season) ClosedFor(mode rules.Mode) bool {
	if mode == rules.Mode2v2 {
		return s.Closed2v2 != nil
	}
	return s.Closed1v1 != nil
}

func (s Season) State(now time.Time) string {
	switch {
	case s.Closed1v1 != nil && s.Closed2v2 != nil:
		return StateClosed
	case !now.Before(s.End):
		return StateEnding
	case !now.Before(s.Start):
		return StateLive
	}
	return StateScheduled
}

type Closed struct {
	Skipped bool
	Placed  int
	Next    string
	Apex    Recalculated
}

const seasonColumns = `id, name_key, starts_at, ends_at, closed_1v1_at, closed_2v2_at`

func scanSeason(row pgx.Row) (Season, error) {
	var s Season
	if err := row.Scan(&s.ID, &s.NameKey, &s.Start, &s.End, &s.Closed1v1, &s.Closed2v2); err != nil {
		return Season{}, err
	}
	s.Start, s.End = s.Start.UTC(), s.End.UTC()
	return s, nil
}

func (s *Store) CreateSeason(ctx context.Context, season Season, now time.Time) (Season, error) {
	switch {
	case !seasonID.MatchString(season.ID):
		return Season{}, fmt.Errorf("%w: id %q", ErrSeason, season.ID)
	case season.Start.IsZero() || !season.End.After(season.Start):
		return Season{}, fmt.Errorf("%w: the end must follow the start", ErrSeason)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Season{}, err
	}
	defer tx.Rollback(ctx)
	var latestEnd *time.Time
	if err := tx.QueryRow(ctx, `select max(ends_at) from ranked_seasons`).Scan(&latestEnd); err != nil {
		return Season{}, err
	}
	if latestEnd != nil && !latestEnd.Equal(season.Start) {
		return Season{}, fmt.Errorf("%w: the next season starts at %s", ErrSeasonGap, latestEnd.UTC().Format(time.RFC3339))
	}
	created, err := insertSeason(ctx, tx, season, now)
	if err != nil {
		return Season{}, err
	}
	return created, tx.Commit(ctx)
}

func insertSeason(ctx context.Context, tx pgx.Tx, season Season, now time.Time) (Season, error) {
	season.Start, season.End = season.Start.UTC(), season.End.UTC()
	_, err := tx.Exec(ctx, `insert into ranked_seasons (id, name_key, starts_at, ends_at, created_at) values ($1, $2, $3, $4, $5)`, season.ID, season.NameKey, season.Start, season.End, now)
	switch {
	case err != nil && strings.Contains(err.Error(), "23P01"):
		return Season{}, fmt.Errorf("%w: %s", ErrSeasonOverlap, season.ID)
	case err != nil && strings.Contains(err.Error(), "23505"):
		return Season{}, fmt.Errorf("%w: id %s exists", ErrSeason, season.ID)
	case err != nil:
		return Season{}, err
	}
	return Season{ID: season.ID, NameKey: season.NameKey, Start: season.Start, End: season.End}, nil
}

func (s *Store) ExtendSeason(ctx context.Context, id string, end time.Time) (Season, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Season{}, err
	}
	defer tx.Rollback(ctx)
	season, err := scanSeason(tx.QueryRow(ctx, `select `+seasonColumns+` from ranked_seasons where id = $1 for update`, id))
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Season{}, ErrSeasonUnknown
	case err != nil:
		return Season{}, err
	case end.Before(season.End):
		return Season{}, fmt.Errorf("%w: a season end only moves later", ErrSeasonEarlier)
	}
	shift := end.Sub(season.End)
	if _, err := tx.Exec(ctx, `update ranked_seasons set starts_at = starts_at + $2, ends_at = ends_at + $2 where starts_at >= $1`, season.End, shift); err != nil {
		return Season{}, err
	}
	if _, err := tx.Exec(ctx, `update ranked_seasons set ends_at = $2 where id = $1`, id, end.UTC()); err != nil {
		return Season{}, err
	}
	season.End = end.UTC()
	return season, tx.Commit(ctx)
}

func (s *Store) Seasons(ctx context.Context) ([]Season, error) {
	rows, err := s.pool.Query(ctx, `select `+seasonColumns+` from ranked_seasons order by starts_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var seasons []Season
	for rows.Next() {
		season, err := scanSeason(rows)
		if err != nil {
			return nil, err
		}
		seasons = append(seasons, season)
	}
	return seasons, rows.Err()
}

func (s *Store) SeasonAt(ctx context.Context, at time.Time) (Season, error) {
	return seasonAt(ctx, s.pool, at)
}

type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func seasonAt(ctx context.Context, q querier, at time.Time) (Season, error) {
	season, err := scanSeason(q.QueryRow(ctx, `select `+seasonColumns+` from ranked_seasons where starts_at <= $1 and ends_at > $1`, at))
	if errors.Is(err, pgx.ErrNoRows) {
		return Season{}, ErrNoSeason
	}
	return season, err
}

func (s *Store) CloseSeason(ctx context.Context, id string, mode rules.Mode, now time.Time) (Closed, error) {
	config, _, err := s.Config(mode)
	if err != nil {
		return Closed{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Closed{}, err
	}
	defer tx.Rollback(ctx)
	season, err := scanSeason(tx.QueryRow(ctx, `select `+seasonColumns+` from ranked_seasons where id = $1 for update`, id))
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Closed{}, ErrSeasonUnknown
	case err != nil:
		return Closed{}, err
	case season.ClosedFor(mode):
		return Closed{Skipped: true}, nil
	case now.Before(season.End):
		return Closed{}, fmt.Errorf("%w: %s ends at %s", ErrSeasonOpen, id, season.End.Format(time.RFC3339))
	}
	next, err := s.nextSeason(ctx, tx, season, config, now)
	if err != nil {
		return Closed{}, err
	}
	apex, _, err := s.assignApex(ctx, tx, mode, config)
	if err != nil {
		return Closed{}, err
	}
	placed, err := s.snapshotSeason(ctx, tx, season, mode, config)
	if err != nil {
		return Closed{}, err
	}
	if _, err := tx.Exec(ctx, `update ratings set season_id = $3, placed = false, placement_played = 0, placement_penalty = 0, lp = 0, wins = 0, apex_tier = null, lp_at = null, updated_at = $4,
		rank_index = coalesce(reset_index, 0) where mode = $1 and (season_id = $2 or season_id is null)`, mode.String(), id, next.ID, now); err != nil {
		return Closed{}, err
	}
	column := "closed_1v1_at"
	if mode == rules.Mode2v2 {
		column = "closed_2v2_at"
	}
	if _, err := tx.Exec(ctx, `update ranked_seasons set `+column+` = $2 where id = $1`, id, now); err != nil {
		return Closed{}, err
	}
	return Closed{Placed: placed, Next: next.ID, Apex: apex}, tx.Commit(ctx)
}

func (s *Store) nextSeason(ctx context.Context, tx pgx.Tx, season Season, config Config, now time.Time) (Season, error) {
	next, err := scanSeason(tx.QueryRow(ctx, `select `+seasonColumns+` from ranked_seasons where starts_at = $1`, season.End))
	if err == nil {
		return next, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Season{}, err
	}
	end := season.End.AddDate(0, config.SeasonLengthMonths, 0)
	for _, id := range []string{season.End.Format("season.2006_01"), fmt.Sprintf("season.%d", season.End.Unix())} {
		var exists bool
		if err := tx.QueryRow(ctx, `select exists (select 1 from ranked_seasons where id = $1)`, id).Scan(&exists); err != nil {
			return Season{}, err
		}
		if !exists {
			return insertSeason(ctx, tx, Season{ID: id, Start: season.End, End: end}, now)
		}
	}
	return Season{}, fmt.Errorf("%w: no free id for the season after %s", ErrSeason, season.ID)
}

func (s *Store) snapshotSeason(ctx context.Context, tx pgx.Tx, season Season, mode rules.Mode, config Config) (int, error) {
	rows, err := tx.Query(ctx, `select account_id, rank_index, apex_tier, lp, wins, mmr from ratings where mode = $1 and (season_id = $2 or season_id is null) and placed
		order by rank_index desc, case apex_tier when $3 then 0 when $4 then 1 else 2 end, lp desc, lp_at asc nulls last, wins desc, account_id asc for update`, mode.String(), season.ID, ApexChallenger, ApexGrandmaster)
	if err != nil {
		return 0, err
	}
	var results [][]any
	var resets []struct {
		account string
		reset   int
	}
	for rows.Next() {
		var account string
		var index, lp, wins, mmr int
		var apex *string
		if err := rows.Scan(&account, &index, &apex, &lp, &wins, &mmr); err != nil {
			rows.Close()
			return 0, err
		}
		reset := config.ResetIndex(index)
		results = append(results, []any{season.ID, mode.String(), account, len(results) + 1, index, apex, lp, wins, mmr, reset})
		resets = append(resets, struct {
			account string
			reset   int
		}{account, reset})
	}
	rows.Close()
	if rows.Err() != nil {
		return 0, rows.Err()
	}
	columns := []string{"season_id", "mode", "account_id", "position", "rank_index", "apex_tier", "lp", "wins", "mmr", "reset_index"}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"season_results"}, columns, pgx.CopyFromRows(results)); err != nil {
		return 0, err
	}
	for _, r := range resets {
		if _, err := tx.Exec(ctx, `update ratings set reset_index = $3 where account_id = $1 and mode = $2`, r.account, mode.String(), r.reset); err != nil {
			return 0, err
		}
	}
	if _, err := tx.Exec(ctx, `update ratings set reset_index = null where mode = $1 and (season_id = $2 or season_id is null) and not placed`, mode.String(), season.ID); err != nil {
		return 0, err
	}
	return len(results), nil
}

func (s *Store) CloseDue(ctx context.Context, now time.Time) ([]Closed, error) {
	rows, err := s.pool.Query(ctx, `select `+seasonColumns+` from ranked_seasons where ends_at <= $1 and (closed_1v1_at is null or closed_2v2_at is null) order by ends_at`, now)
	if err != nil {
		return nil, err
	}
	var due []Season
	for rows.Next() {
		season, err := scanSeason(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		due = append(due, season)
	}
	rows.Close()
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	var closed []Closed
	for _, season := range due {
		for _, mode := range modes {
			if season.ClosedFor(mode) {
				continue
			}
			result, err := s.CloseSeason(ctx, season.ID, mode, now)
			if err != nil {
				return closed, fmt.Errorf("close %s %s: %w", season.ID, mode, err)
			}
			closed = append(closed, result)
		}
	}
	return closed, nil
}

func (s *Store) RunSeasonEnd(ctx context.Context, logger *slog.Logger, every time.Duration, now func() time.Time) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		closed, err := s.CloseDue(ctx, now())
		if err != nil {
			logger.Error("season end failed", "error", err)
		}
		for _, c := range closed {
			logger.Info("season closed", "placed", c.Placed, "next", c.Next, "challenger", c.Apex.Challenger, "grandmaster", c.Apex.Grandmaster)
		}
	}
}

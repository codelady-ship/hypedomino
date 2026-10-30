package rating

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/rules"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
)

var (
	ErrNoConfig = errors.New("rating.no_config")
	ErrResult   = errors.New("rating: invalid result")
)

type Participant struct {
	Account  string
	Offender bool
}

type Result struct {
	MatchID string
	Mode    rules.Mode
	Teams   [2][]Participant
	Winner  int
	At      time.Time
}

type Applied struct {
	Duplicate     bool
	ConfigVersion int64
	Changes       [2][]Change
}

type Rating struct {
	Account          string    `json:"account_id"`
	Mode             string    `json:"mode"`
	MMR              int       `json:"mmr"`
	Matches          int       `json:"matches"`
	Wins             int       `json:"wins"`
	PlacementPlayed  int       `json:"placement_played"`
	PlacementPenalty int       `json:"placement_penalty"`
	Placed           bool      `json:"placed"`
	Rank             Rank      `json:"rank"`
	Tier             string    `json:"tier"`
	LPAt             time.Time `json:"lp_at"`
	apexTier         *string
}

type Recalculated struct {
	Skipped     bool
	Challenger  int
	Grandmaster int
	Master      int
}

type Store struct {
	pool    *pgxpool.Pool
	configs *snapshot.Watcher[*Snapshot]
}

func NewWatcher(pool *pgxpool.Pool, logger *slog.Logger) *snapshot.Watcher[*Snapshot] {
	watcher := snapshot.New(pool, Domain, Parse, logger)
	watcher.Seed(Default(), 0)
	return watcher
}

func NewStore(pool *pgxpool.Pool, configs *snapshot.Watcher[*Snapshot]) *Store {
	return &Store{pool: pool, configs: configs}
}

func (s *Store) Config(mode rules.Mode) (Config, int64, error) {
	current, version, ok := s.configs.Current()
	if !ok {
		return Config{}, 0, ErrNoConfig
	}
	config, ok := current.Mode(mode)
	if !ok {
		return Config{}, 0, fmt.Errorf("%w: %s", ErrNoConfig, mode)
	}
	return config, version, nil
}

func (r Result) validate() error {
	switch {
	case r.MatchID == "" || r.At.IsZero():
		return fmt.Errorf("%w: match id and time are required", ErrResult)
	case r.Winner != 0 && r.Winner != 1:
		return fmt.Errorf("%w: winner %d", ErrResult, r.Winner)
	}
	size := 1
	if r.Mode == rules.Mode2v2 {
		size = 2
	}
	seen := map[string]bool{}
	for _, team := range r.Teams {
		if len(team) != size {
			return fmt.Errorf("%w: %s teams hold %d players", ErrResult, r.Mode, size)
		}
		for _, p := range team {
			if p.Account == "" || seen[p.Account] {
				return fmt.Errorf("%w: accounts must be unique and non-empty", ErrResult)
			}
			seen[p.Account] = true
		}
	}
	return nil
}

func (s *Store) Apply(ctx context.Context, r Result) (Applied, error) {
	if err := r.validate(); err != nil {
		return Applied{}, err
	}
	config, version, err := s.Config(r.Mode)
	if err != nil {
		return Applied{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Applied{}, err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `insert into rating_results (match_id, mode, config_version, applied_at) values ($1, $2, $3, $4) on conflict (match_id) do nothing`, r.MatchID, r.Mode.String(), version, r.At)
	if err != nil {
		return Applied{}, err
	}
	if tag.RowsAffected() == 0 {
		return Applied{Duplicate: true, ConfigVersion: version}, nil
	}
	ratings, err := s.lock(ctx, tx, r, config)
	if err != nil {
		return Applied{}, err
	}
	var match Match
	match.Winner = r.Winner
	for team, members := range r.Teams {
		for _, p := range members {
			current := ratings[p.Account]
			match.Teams[team] = append(match.Teams[team], Player{Account: p.Account, MMR: current.MMR, Matches: current.Matches, Placement: !current.Placed, Rank: current.Rank, Offender: p.Offender})
		}
	}
	changes, err := config.Resolve(match)
	if err != nil {
		return Applied{}, err
	}
	for team := range changes {
		for i := range changes[team] {
			if err := s.record(ctx, tx, r, config, ratings[changes[team][i].Account], &changes[team][i]); err != nil {
				return Applied{}, err
			}
		}
	}
	return Applied{ConfigVersion: version, Changes: changes}, tx.Commit(ctx)
}

const columns = `account_id, mode, mmr, matches, wins, placement_played, placement_penalty, placed, rank_index, lp, apex_tier, lp_at`

func (s *Store) lock(ctx context.Context, tx pgx.Tx, r Result, config Config) (map[string]Rating, error) {
	accounts := make([]string, 0, 4)
	for _, team := range r.Teams {
		for _, p := range team {
			accounts = append(accounts, p.Account)
		}
	}
	sort.Strings(accounts)
	ratings := make(map[string]Rating, len(accounts))
	for _, account := range accounts {
		if _, err := tx.Exec(ctx, `insert into ratings (account_id, mode, mmr, updated_at) values ($1, $2, $3, $4) on conflict (account_id, mode) do nothing`, account, r.Mode.String(), config.InitialMmr, r.At); err != nil {
			return nil, err
		}
		rating, err := scan(tx.QueryRow(ctx, `select `+columns+` from ratings where account_id = $1 and mode = $2 for update`, account, r.Mode.String()))
		if err != nil {
			return nil, err
		}
		ratings[account] = rating
	}
	return ratings, nil
}

func (s *Store) record(ctx context.Context, tx pgx.Tx, r Result, config Config, before Rating, change *Change) error {
	after := before
	after.MMR, after.Matches, after.Rank = change.MMR, before.Matches+1, change.Rank
	if change.Won {
		after.Wins++
	}
	if change.Placement {
		after.PlacementPlayed++
		after.PlacementPenalty += change.Penalty
		if after.PlacementPlayed >= config.PlacementMatches {
			after.Placed = true
			after.Rank = config.Place(after.MMR, after.PlacementPenalty)
			change.Rank = after.Rank
		}
	}
	if after.Placed && (after.Rank != before.Rank || !before.Placed) {
		after.LPAt = r.At
	}
	after.apexTier = apexTier(before, after)
	if _, err := tx.Exec(ctx, `update ratings set mmr = $3, matches = $4, wins = $5, placement_played = $6, placement_penalty = $7, placed = $8, rank_index = $9, lp = $10, apex_tier = $11, lp_at = $12, updated_at = $13 where account_id = $1 and mode = $2`,
		before.Account, r.Mode.String(), after.MMR, after.Matches, after.Wins, after.PlacementPlayed, after.PlacementPenalty, after.Placed, after.Rank.Index, after.Rank.LP, after.apexTier, nullable(after.LPAt), r.At); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `insert into rating_changes (match_id, account_id, won, offender, placement, k, expected, mmr_before, mmr_after, lp_delta, rank_before, lp_before, rank_after, lp_after)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		r.MatchID, change.Account, change.Won, change.Offender, change.Placement, change.K, change.Expected, change.MMRBefore, change.MMR, change.LPDelta, change.RankBefore.Index, change.RankBefore.LP, change.Rank.Index, change.Rank.LP)
	return err
}

func apexTier(before, after Rating) *string {
	switch {
	case !after.Rank.Apex():
		return nil
	case before.apexTier != nil && before.Rank.Apex():
		return before.apexTier
	}
	master := ApexMaster
	return &master
}

func nullable(at time.Time) *time.Time {
	if at.IsZero() {
		return nil
	}
	return &at
}

func scan(row pgx.Row) (Rating, error) {
	var r Rating
	var at *time.Time
	if err := row.Scan(&r.Account, &r.Mode, &r.MMR, &r.Matches, &r.Wins, &r.PlacementPlayed, &r.PlacementPenalty, &r.Placed, &r.Rank.Index, &r.Rank.LP, &r.apexTier, &at); err != nil {
		return Rating{}, err
	}
	if at != nil {
		r.LPAt = at.UTC()
	}
	r.Tier = r.Rank.Tier()
	if r.apexTier != nil {
		r.Tier = TierName(*r.apexTier)
	}
	return r, nil
}

func (s *Store) Get(ctx context.Context, account string, mode rules.Mode) (Rating, bool, error) {
	rating, err := scan(s.pool.QueryRow(ctx, `select `+columns+` from ratings where account_id = $1 and mode = $2`, account, mode.String()))
	if errors.Is(err, pgx.ErrNoRows) {
		return Rating{}, false, nil
	}
	return rating, err == nil, err
}

func (s *Store) RecalculateApex(ctx context.Context, mode rules.Mode) (Recalculated, error) {
	config, _, err := s.Config(mode)
	if err != nil {
		return Recalculated{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Recalculated{}, err
	}
	defer tx.Rollback(ctx)
	var locked bool
	if err := tx.QueryRow(ctx, `select pg_try_advisory_xact_lock(hashtext($1))`, "rating.apex."+mode.String()).Scan(&locked); err != nil {
		return Recalculated{}, err
	}
	if !locked {
		return Recalculated{Skipped: true}, nil
	}
	rows, err := tx.Query(ctx, `select account_id, lp, apex_tier from ratings where mode = $1 and placed and rank_index = $2 order by lp desc, lp_at asc nulls last, wins desc, account_id asc for update`, mode.String(), ApexIndex)
	if err != nil {
		return Recalculated{}, err
	}
	type apex struct {
		account string
		lp      int
		tier    *string
	}
	var players []apex
	for rows.Next() {
		var p apex
		if err := rows.Scan(&p.account, &p.lp, &p.tier); err != nil {
			rows.Close()
			return Recalculated{}, err
		}
		players = append(players, p)
	}
	rows.Close()
	if rows.Err() != nil {
		return Recalculated{}, rows.Err()
	}
	var result Recalculated
	challenger, grandmaster := config.ChallengerSlots, config.GrandmasterSlots
	for _, p := range players {
		tier := ApexMaster
		switch {
		case challenger > 0 && p.lp >= config.ChallengerMinLp:
			tier, challenger = ApexChallenger, challenger-1
			result.Challenger++
		case grandmaster > 0 && p.lp >= config.GrandmasterMinLp:
			tier, grandmaster = ApexGrandmaster, grandmaster-1
			result.Grandmaster++
		default:
			result.Master++
		}
		if p.tier != nil && *p.tier == tier {
			continue
		}
		if _, err := tx.Exec(ctx, `update ratings set apex_tier = $3 where account_id = $1 and mode = $2`, p.account, mode.String(), tier); err != nil {
			return Recalculated{}, err
		}
	}
	return result, tx.Commit(ctx)
}

func NextApexRun(now time.Time, hourUTC int) time.Time {
	now = now.UTC()
	next := time.Date(now.Year(), now.Month(), now.Day(), hourUTC, 0, 0, 0, time.UTC)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

func (s *Store) RunApexRecalc(ctx context.Context, logger *slog.Logger, now func() time.Time) {
	modes := []rules.Mode{rules.Mode1v1, rules.Mode2v2}
	for {
		var soonest time.Time
		var due []rules.Mode
		for _, mode := range modes {
			config, _, err := s.Config(mode)
			if err != nil {
				continue
			}
			next := NextApexRun(now(), config.ApexRecalcHourUtc)
			switch {
			case soonest.IsZero() || next.Before(soonest):
				soonest, due = next, []rules.Mode{mode}
			case next.Equal(soonest):
				due = append(due, mode)
			}
		}
		if soonest.IsZero() {
			soonest = now().Add(time.Hour)
		}
		timer := time.NewTimer(soonest.Sub(now()))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		for _, mode := range due {
			result, err := s.RecalculateApex(ctx, mode)
			if err != nil {
				logger.Error("apex recalculation failed", "mode", mode.String(), "error", err)
				continue
			}
			logger.Info("apex tiers recalculated", "mode", mode.String(), "skipped", result.Skipped, "challenger", result.Challenger, "grandmaster", result.Grandmaster, "master", result.Master)
		}
	}
}

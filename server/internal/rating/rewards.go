package rating

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

const SourceRankReward = "rank_reward"

var ErrNoRewarder = errors.New("rating: no reward engine")

type Rewarder interface {
	ApplyIn(ctx context.Context, tx pgx.Tx, grant reward.Grant, now time.Time) (reward.Report, error)
}

type Emblem struct {
	Season    string    `json:"season"`
	Mode      string    `json:"mode"`
	Board     string    `json:"board"`
	Bracket   int       `json:"bracket"`
	Position  int       `json:"position"`
	AwardedAt time.Time `json:"awarded_at"`
}

func (e Emblem) String() string {
	return fmt.Sprintf("%s:%d", e.Board, e.Bracket)
}

type Reward struct {
	Season   string    `json:"season"`
	Mode     string    `json:"mode"`
	Tier     string    `json:"tier"`
	Rank     Rank      `json:"rank"`
	Coins    int64     `json:"coins"`
	BundleID string    `json:"bundle_id,omitempty"`
	Emblems  []string  `json:"emblems"`
	At       time.Time `json:"at"`
	Seen     bool      `json:"seen"`
}

type Distributed struct {
	Season   string
	Mode     rules.Mode
	Rewarded int
	Skipped  bool
}

type finalRow struct {
	account  string
	rank     Rank
	tier     string
	position int
	country  string
}

func (s *Store) SetRewards(engine Rewarder) {
	s.rewards = engine
}

func bracketOf(brackets []int, position int) (int, bool) {
	for _, bracket := range brackets {
		if position <= bracket {
			return bracket, true
		}
	}
	return 0, false
}

func awardEmblems(snapshot *Snapshot, rows []finalRow) map[string][]Emblem {
	out := map[string][]Emblem{}
	counts := map[string]int{}
	for _, row := range rows {
		boards := []string{"global"}
		if row.country != "" {
			boards = append(boards, "country:"+row.country)
			if region, ok := snapshot.Region(row.country); ok {
				boards = append(boards, "region:"+region)
			}
		}
		for _, board := range boards {
			counts[board]++
			if bracket, ok := bracketOf(snapshot.EmblemBrackets, counts[board]); ok {
				out[row.account] = append(out[row.account], Emblem{Board: board, Bracket: bracket, Position: counts[board]})
			}
		}
	}
	return out
}

func rankBundle(config Config, snapshot *Snapshot, tier string, version int64) (reward.Bundle, bool) {
	bundle := reward.Bundle{ID: "bundle.rank_" + strings.ToLower(tier), Revision: int(max(version, 1)), Coins: snapshot.RankRewardCoins[tier]}
	if extra, ok := config.RankRewards[tier]; ok {
		bundle.ID, bundle.Revision = extra.ID, extra.Revision
		bundle.Coins += extra.Coins
		bundle.Cosmetics, bundle.Medals, bundle.PassXP = extra.Cosmetics, extra.Medals, extra.PassXP
	}
	return bundle, bundle.Coins > 0 || len(bundle.Cosmetics) > 0 || len(bundle.Medals) > 0 || bundle.PassXP > 0
}

func (s *Store) finals(ctx context.Context, season string, mode rules.Mode) ([]finalRow, error) {
	rows, err := s.pool.Query(ctx, `select account_id, rank_index, lp, apex_tier, position, coalesce(country, '') from season_results where season_id = $1 and mode = $2 order by position`, season, mode.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []finalRow
	for rows.Next() {
		var row finalRow
		var apex *string
		if err := rows.Scan(&row.account, &row.rank.Index, &row.rank.LP, &apex, &row.position, &row.country); err != nil {
			return nil, err
		}
		row.tier = tierOf(row.rank, apex)
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) DistributeRewards(ctx context.Context, id string, mode rules.Mode, now time.Time) (Distributed, error) {
	if s.rewards == nil {
		return Distributed{}, ErrNoRewarder
	}
	season, err := scanSeason(s.pool.QueryRow(ctx, `select `+seasonColumns+` from ranked_seasons where id = $1`, id))
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Distributed{}, ErrSeasonUnknown
	case err != nil:
		return Distributed{}, err
	case season.RewardedFor(mode):
		return Distributed{Season: id, Mode: mode, Skipped: true}, nil
	case !season.ClosedFor(mode):
		return Distributed{}, fmt.Errorf("%w: %s is not closed for %s", ErrSeasonOpen, id, mode)
	}
	config, version, err := s.Config(mode)
	if err != nil {
		return Distributed{}, err
	}
	snapshot, _, _ := s.configs.Current()
	result := Distributed{Season: id, Mode: mode}
	if !season.NoRewards {
		rows, err := s.finals(ctx, id, mode)
		if err != nil {
			return Distributed{}, err
		}
		emblems := awardEmblems(snapshot, rows)
		for _, row := range rows {
			granted, err := s.rewardOne(ctx, id, mode, row, emblems[row.account], config, snapshot, version, now)
			if err != nil {
				return result, fmt.Errorf("reward %s: %w", row.account, err)
			}
			if granted {
				result.Rewarded++
			}
		}
	}
	column := "rewarded_1v1_at"
	if mode == rules.Mode2v2 {
		column = "rewarded_2v2_at"
	}
	_, err = s.pool.Exec(ctx, `update ranked_seasons set `+column+` = $2 where id = $1`, id, now)
	return result, err
}

func (s *Store) rewardOne(ctx context.Context, season string, mode rules.Mode, row finalRow, emblems []Emblem, config Config, snapshot *Snapshot, version int64, now time.Time) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	bundle, grants := rankBundle(config, snapshot, row.tier, version)
	names := make([]string, 0, len(emblems))
	for _, e := range emblems {
		names = append(names, e.String())
	}
	var bundleID *string
	if grants {
		bundleID = &bundle.ID
	}
	tag, err := tx.Exec(ctx, `insert into season_rewards (season_id, mode, account_id, tier, rank_index, coins, bundle_id, emblems, created_at) values ($1, $2, $3, $4, $5, $6, $7, $8, $9) on conflict do nothing`,
		season, mode.String(), row.account, row.tier, row.rank.Index, bundle.Coins, bundleID, names, now)
	if err != nil || tag.RowsAffected() == 0 {
		return false, err
	}
	if grants {
		report, err := s.rewards.ApplyIn(ctx, tx, reward.Grant{Account: row.account, Bundle: bundle, Source: cosmetics.Source{Type: SourceRankReward, Ref: bundle.ID}, Reason: wallet.RankReward, Key: "rank:" + season + ":" + mode.String(), ConfigVersion: int(version)}, now)
		if err != nil {
			return false, err
		}
		if _, err := tx.Exec(ctx, `update season_rewards set grant_id = $4 where season_id = $1 and mode = $2 and account_id = $3`, season, mode.String(), row.account, report.ID); err != nil {
			return false, err
		}
	}
	for _, e := range emblems {
		if _, err := tx.Exec(ctx, `insert into leaderboard_emblems (account_id, mode, season_id, board, bracket, position, awarded_at) values ($1, $2, $3, $4, $5, $6, $7) on conflict do nothing`, row.account, mode.String(), season, e.Board, e.Bracket, e.Position, now); err != nil {
			return false, err
		}
	}
	return true, tx.Commit(ctx)
}

func (s *Store) DistributeDue(ctx context.Context, now time.Time) ([]Distributed, error) {
	rows, err := s.pool.Query(ctx, `select `+seasonColumns+` from ranked_seasons where (closed_1v1_at is not null and rewarded_1v1_at is null) or (closed_2v2_at is not null and rewarded_2v2_at is null) order by ends_at`)
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
	var done []Distributed
	for _, season := range due {
		for _, mode := range modes {
			if !season.ClosedFor(mode) || season.RewardedFor(mode) {
				continue
			}
			result, err := s.DistributeRewards(ctx, season.ID, mode, now)
			if err != nil {
				return done, fmt.Errorf("distribute %s %s: %w", season.ID, mode, err)
			}
			done = append(done, result)
		}
	}
	return done, nil
}

func (s *Store) Rewards(ctx context.Context, account string) ([]Reward, error) {
	rows, err := s.pool.Query(ctx, `select season_id, mode, tier, rank_index, coins, coalesce(bundle_id, ''), emblems, created_at, seen_at is not null from season_rewards where account_id = $1 order by created_at desc, mode`, account)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Reward{}
	for rows.Next() {
		var r Reward
		if err := rows.Scan(&r.Season, &r.Mode, &r.Tier, &r.Rank.Index, &r.Coins, &r.BundleID, &r.Emblems, &r.At, &r.Seen); err != nil {
			return nil, err
		}
		r.At = r.At.UTC()
		if r.Emblems == nil {
			r.Emblems = []string{}
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) MarkRewardsSeen(ctx context.Context, account string, now time.Time) (int64, error) {
	tag, err := s.pool.Exec(ctx, `update season_rewards set seen_at = $2 where account_id = $1 and seen_at is null`, account, now)
	return tag.RowsAffected(), err
}

func (s *Store) Emblems(ctx context.Context, account string) ([]Emblem, error) {
	rows, err := s.pool.Query(ctx, `select season_id, mode, board, bracket, position, awarded_at from leaderboard_emblems where account_id = $1 and revoked_at is null order by awarded_at desc, mode, board`, account)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Emblem{}
	for rows.Next() {
		var e Emblem
		if err := rows.Scan(&e.Season, &e.Mode, &e.Board, &e.Bracket, &e.Position, &e.AwardedAt); err != nil {
			return nil, err
		}
		e.AwardedAt = e.AwardedAt.UTC()
		out = append(out, e)
	}
	return out, rows.Err()
}

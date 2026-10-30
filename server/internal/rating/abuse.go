package rating

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

const (
	SignalTopic = "rating_signals"

	SignalPairFrequency = "pair_frequency"
	SignalFeeding       = "feeding"
	SignalWinTrade      = "win_trade"
)

type Abuse struct {
	RepeatPairWindowHours int `json:"repeatPairWindowHours"`
	RepeatPairMax         int `json:"repeatPairMax"`
	PairWindowHours       int `json:"pairWindowHours"`
	PairMinMatches        int `json:"pairMinMatches"`
	FeedingSharePercent   int `json:"feedingSharePercent"`
	WinTradeSharePercent  int `json:"winTradeSharePercent"`
}

func DefaultAbuse() Abuse {
	return Abuse{RepeatPairWindowHours: 24, RepeatPairMax: 3, PairWindowHours: 168, PairMinMatches: 6, FeedingSharePercent: 80, WinTradeSharePercent: 35}
}

func (a Abuse) Validate() error {
	switch {
	case a.RepeatPairWindowHours < 1 || a.PairWindowHours < 1:
		return fmt.Errorf("%w: abuse windows must be at least 1 hour", ErrInvalidConfig)
	case a.RepeatPairMax < 1 || a.PairMinMatches < 1:
		return fmt.Errorf("%w: abuse match counts must be at least 1", ErrInvalidConfig)
	case a.FeedingSharePercent < 51 || a.FeedingSharePercent > 100:
		return fmt.Errorf("%w: feedingSharePercent must be between 51 and 100", ErrInvalidConfig)
	case a.WinTradeSharePercent < 1 || a.WinTradeSharePercent > 50:
		return fmt.Errorf("%w: winTradeSharePercent must be between 1 and 50", ErrInvalidConfig)
	}
	return nil
}

type Signal struct {
	Account     string    `json:"account_id"`
	Kind        string    `json:"kind"`
	Counterpart string    `json:"counterpart"`
	Mode        string    `json:"mode"`
	Matches     int       `json:"matches"`
	Wins        int       `json:"wins"`
	Offenses    int       `json:"offenses"`
	WindowHours int       `json:"window_hours"`
	FirstSeen   time.Time `json:"first_seen"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type pairStats struct {
	matches  int
	wins     int
	offenses int
}

func classify(abuse Abuse, stats pairStats) []string {
	if stats.matches < abuse.PairMinMatches {
		return nil
	}
	kinds := []string{SignalPairFrequency}
	share := stats.wins * 100 / stats.matches
	switch {
	case share >= abuse.FeedingSharePercent || 100-share >= abuse.FeedingSharePercent:
		kinds = append(kinds, SignalFeeding)
	case share >= abuse.WinTradeSharePercent && 100-share >= abuse.WinTradeSharePercent:
		kinds = append(kinds, SignalWinTrade)
	}
	return kinds
}

func (s *Store) pair(ctx context.Context, a, b string, mode rules.Mode, since time.Time) (pairStats, error) {
	var stats pairStats
	err := s.pool.QueryRow(ctx, `select count(*), count(*) filter (where a.won), count(*) filter (where a.offender or b.offender)
		from rating_changes a
		join rating_changes b on b.match_id = a.match_id and b.account_id = $2
		join rating_results r on r.match_id = a.match_id
		where a.account_id = $1 and r.mode = $3 and r.applied_at > $4 and a.won <> b.won`, a, b, mode.String(), since).Scan(&stats.matches, &stats.wins, &stats.offenses)
	return stats, err
}

func (s *Store) inspect(ctx context.Context, r Result) error {
	snapshot, _, ok := s.configs.Current()
	if !ok {
		return ErrNoConfig
	}
	abuse := snapshot.Abuse
	since := r.At.Add(-time.Duration(abuse.PairWindowHours) * time.Hour)
	for _, a := range r.Teams[0] {
		for _, b := range r.Teams[1] {
			stats, err := s.pair(ctx, a.Account, b.Account, r.Mode, since)
			if err != nil {
				return err
			}
			kinds := classify(abuse, stats)
			if len(kinds) == 0 {
				continue
			}
			if err := s.flag(ctx, r, a.Account, b.Account, stats, kinds, abuse.PairWindowHours); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Store) flag(ctx context.Context, r Result, a, b string, stats pairStats, kinds []string, window int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, kind := range kinds {
		for _, side := range [2]Signal{
			{Account: a, Kind: kind, Counterpart: b, Mode: r.Mode.String(), Matches: stats.matches, Wins: stats.wins, Offenses: stats.offenses, WindowHours: window, FirstSeen: r.At, UpdatedAt: r.At},
			{Account: b, Kind: kind, Counterpart: a, Mode: r.Mode.String(), Matches: stats.matches, Wins: stats.matches - stats.wins, Offenses: stats.offenses, WindowHours: window, FirstSeen: r.At, UpdatedAt: r.At},
		} {
			if _, err := tx.Exec(ctx, `insert into rating_signals (account_id, kind, counterpart, mode, matches, wins, offenses, window_hours, first_seen, updated_at) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
				on conflict (account_id, kind, counterpart, mode) do update set matches = excluded.matches, wins = excluded.wins, offenses = excluded.offenses, window_hours = excluded.window_hours, updated_at = excluded.updated_at`,
				side.Account, side.Kind, side.Counterpart, side.Mode, side.Matches, side.Wins, side.Offenses, side.WindowHours, r.At); err != nil {
				return err
			}
			payload, err := json.Marshal(side)
			if err != nil {
				return err
			}
			key := fmt.Sprintf("%s:%s:%s:%s:%d", side.Account, side.Kind, side.Counterpart, side.Mode, side.Matches)
			if _, err := tx.Exec(ctx, `insert into outbox (topic, key, payload) values ($1, $2, $3) on conflict (topic, key) do nothing`, SignalTopic, key, payload); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) Signals(ctx context.Context, account string) ([]Signal, error) {
	rows, err := s.pool.Query(ctx, `select account_id, kind, counterpart, mode, matches, wins, offenses, window_hours, first_seen, updated_at from rating_signals where account_id = $1 order by updated_at desc, kind, counterpart`, account)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Signal{}
	for rows.Next() {
		var signal Signal
		if err := rows.Scan(&signal.Account, &signal.Kind, &signal.Counterpart, &signal.Mode, &signal.Matches, &signal.Wins, &signal.Offenses, &signal.WindowHours, &signal.FirstSeen, &signal.UpdatedAt); err != nil {
			return nil, err
		}
		signal.FirstSeen, signal.UpdatedAt = signal.FirstSeen.UTC(), signal.UpdatedAt.UTC()
		out = append(out, signal)
	}
	return out, rows.Err()
}

func (s *Store) Avoid(ctx context.Context, account string, mode rules.Mode, now time.Time) ([]string, error) {
	snapshot, _, ok := s.configs.Current()
	if !ok {
		return nil, ErrNoConfig
	}
	since := now.Add(-time.Duration(snapshot.Abuse.RepeatPairWindowHours) * time.Hour)
	rows, err := s.pool.Query(ctx, `select o.account_id from rating_changes c
		join rating_results r on r.match_id = c.match_id
		join rating_changes o on o.match_id = c.match_id and o.account_id <> c.account_id
		where c.account_id = $1 and r.mode = $2 and r.applied_at > $3
		group by o.account_id having count(*) >= $4 order by o.account_id`, account, mode.String(), since, snapshot.Abuse.RepeatPairMax)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var other string
		if err := rows.Scan(&other); err != nil {
			return nil, err
		}
		out = append(out, other)
	}
	return out, rows.Err()
}

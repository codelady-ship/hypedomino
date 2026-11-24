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

	SignalPairFrequency    = "pair_frequency"
	SignalFeeding          = "feeding"
	SignalWinTrade         = "win_trade"
	SignalPartnerFrequency = "partner_frequency"
	SignalPartnerFit       = "partner_fit"
)

type Abuse struct {
	RepeatPairWindowHours int `json:"repeatPairWindowHours"`
	RepeatPairMax         int `json:"repeatPairMax"`
	PairWindowHours       int `json:"pairWindowHours"`
	PairMinMatches        int `json:"pairMinMatches"`
	FeedingSharePercent   int `json:"feedingSharePercent"`
	WinTradeSharePercent  int `json:"winTradeSharePercent"`
	PartnerMinMatches     int `json:"partnerMinMatches"`
	FitMinDecisions       int `json:"fitMinDecisions"`
	FitMarginPercent      int `json:"fitMarginPercent"`
}

func DefaultAbuse() Abuse {
	return Abuse{RepeatPairWindowHours: 24, RepeatPairMax: 3, PairWindowHours: 168, PairMinMatches: 6, FeedingSharePercent: 80, WinTradeSharePercent: 35, PartnerMinMatches: 3, FitMinDecisions: 100, FitMarginPercent: 20}
}

func (a Abuse) Validate() error {
	switch {
	case a.RepeatPairWindowHours < 1 || a.PairWindowHours < 1:
		return fmt.Errorf("%w: abuse windows must be at least 1 hour", ErrInvalidConfig)
	case a.RepeatPairMax < 1 || a.PairMinMatches < 1 || a.PartnerMinMatches < 1 || a.FitMinDecisions < 1:
		return fmt.Errorf("%w: abuse match counts must be at least 1", ErrInvalidConfig)
	case a.FeedingSharePercent < 51 || a.FeedingSharePercent > 100:
		return fmt.Errorf("%w: feedingSharePercent must be between 51 and 100", ErrInvalidConfig)
	case a.WinTradeSharePercent < 1 || a.WinTradeSharePercent > 50:
		return fmt.Errorf("%w: winTradeSharePercent must be between 1 and 50", ErrInvalidConfig)
	case a.FitMarginPercent < 1 || a.FitMarginPercent > 100:
		return fmt.Errorf("%w: fitMarginPercent must be between 1 and 100", ErrInvalidConfig)
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

func (s *Store) pair(ctx context.Context, a, b string, mode rules.Mode, since time.Time, partners bool) (pairStats, error) {
	var stats pairStats
	err := s.pool.QueryRow(ctx, `select count(*), count(*) filter (where a.won), count(*) filter (where a.offender or b.offender)
		from rating_changes a
		join rating_changes b on b.match_id = a.match_id and b.account_id = $2
		join rating_results r on r.match_id = a.match_id
		where a.account_id = $1 and r.mode = $3 and r.applied_at > $4 and (a.won = b.won) = $5`, a, b, mode.String(), since, partners).Scan(&stats.matches, &stats.wins, &stats.offenses)
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
			stats, err := s.pair(ctx, a.Account, b.Account, r.Mode, since, false)
			if err != nil {
				return err
			}
			kinds := classify(abuse, stats)
			if len(kinds) == 0 {
				continue
			}
			if err := s.flag(ctx, r.At, opponentSignals(r, a.Account, b.Account, stats, kinds, abuse.PairWindowHours)); err != nil {
				return err
			}
		}
	}
	return s.inspectPartners(ctx, r, abuse, since)
}

func (s *Store) inspectPartners(ctx context.Context, r Result, abuse Abuse, since time.Time) error {
	for _, team := range r.Teams {
		if len(team) != 2 {
			continue
		}
		stats, err := s.pair(ctx, team[0].Account, team[1].Account, r.Mode, since, true)
		if err != nil {
			return err
		}
		if stats.matches < abuse.PartnerMinMatches {
			continue
		}
		signals := partnerSignals(r, team[0].Account, team[1].Account, stats, abuse.PairWindowHours)
		for _, side := range signals {
			fits, err := s.fitsPartner(ctx, side.Account, side.Counterpart, r.Mode, since, abuse)
			if err != nil {
				return err
			}
			if fits {
				fit := side
				fit.Kind = SignalPartnerFit
				signals = append(signals, fit)
			}
		}
		if err := s.flag(ctx, r.At, signals); err != nil {
			return err
		}
	}
	return nil
}

func opponentSignals(r Result, a, b string, stats pairStats, kinds []string, window int) []Signal {
	var signals []Signal
	for _, kind := range kinds {
		signals = append(signals,
			Signal{Account: a, Kind: kind, Counterpart: b, Mode: r.Mode.String(), Matches: stats.matches, Wins: stats.wins, Offenses: stats.offenses, WindowHours: window, FirstSeen: r.At, UpdatedAt: r.At},
			Signal{Account: b, Kind: kind, Counterpart: a, Mode: r.Mode.String(), Matches: stats.matches, Wins: stats.matches - stats.wins, Offenses: stats.offenses, WindowHours: window, FirstSeen: r.At, UpdatedAt: r.At},
		)
	}
	return signals
}

func partnerSignals(r Result, a, b string, stats pairStats, window int) []Signal {
	side := Signal{Kind: SignalPartnerFrequency, Mode: r.Mode.String(), Matches: stats.matches, Wins: stats.wins, Offenses: stats.offenses, WindowHours: window, FirstSeen: r.At, UpdatedAt: r.At}
	first, second := side, side
	first.Account, first.Counterpart = a, b
	second.Account, second.Counterpart = b, a
	return []Signal{first, second}
}

func (s *Store) flag(ctx context.Context, at time.Time, signals []Signal) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, side := range signals {
		if _, err := tx.Exec(ctx, `insert into rating_signals (account_id, kind, counterpart, mode, matches, wins, offenses, window_hours, first_seen, updated_at) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
				on conflict (account_id, kind, counterpart, mode) do update set matches = excluded.matches, wins = excluded.wins, offenses = excluded.offenses, window_hours = excluded.window_hours, updated_at = excluded.updated_at`,
			side.Account, side.Kind, side.Counterpart, side.Mode, side.Matches, side.Wins, side.Offenses, side.WindowHours, at); err != nil {
			return err
		}
		if err := s.reportCollusion(ctx, tx, side, at); err != nil {
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

type fitStats struct {
	decisions, fits int64
}

func (f fitStats) rate(scale int64) int64 {
	return f.fits * 100 * scale
}

func (s *Store) fitsPartner(ctx context.Context, account, partner string, mode rules.Mode, since time.Time, abuse Abuse) (bool, error) {
	var with, others fitStats
	err := s.pool.QueryRow(ctx, `select coalesce(sum(m.partner_decisions) filter (where p.account_id = $2), 0), coalesce(sum(m.partner_fits) filter (where p.account_id = $2), 0),
			coalesce(sum(m.partner_decisions) filter (where p.account_id <> $2), 0), coalesce(sum(m.partner_fits) filter (where p.account_id <> $2), 0)
		from match_participants m
		join match_participants p on p.match_id = m.match_id and p.started_at = m.started_at and p.team = m.team and p.seat <> m.seat
		join rating_results r on r.match_id = m.match_id
		where m.account_id = $1 and m.partner_decisions is not null and r.mode = $3 and r.applied_at > $4`, account, partner, mode.String(), since).Scan(&with.decisions, &with.fits, &others.decisions, &others.fits)
	if err != nil {
		return false, err
	}
	minimum := int64(abuse.FitMinDecisions)
	if with.decisions < minimum || others.decisions < minimum {
		return false, nil
	}
	return with.rate(others.decisions)-others.rate(with.decisions) >= int64(abuse.FitMarginPercent)*with.decisions*others.decisions, nil
}

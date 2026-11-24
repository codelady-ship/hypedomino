package pass

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/analytics"
	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
)

const (
	XpTopic     = "pass_xp"
	SourceMatch = "match"
)

var (
	ErrUnknownSeason = errors.New("pass: unknown season")
	ErrPremiumOwned  = errors.New("pass.premium_owned")
)

type xpEvent struct {
	Amount int
	Source cosmetics.Source
	Key    string
	Match  bool
}

type XpAdded struct {
	Account string           `json:"account_id"`
	Season  string           `json:"season"`
	Amount  int              `json:"amount"`
	Source  cosmetics.Source `json:"source"`
	Key     string           `json:"key"`
	XpTotal int64            `json:"xp_total"`
	Level   int              `json:"level"`
	At      time.Time        `json:"at"`
}

func (s *Store) addXpIn(ctx context.Context, tx pgx.Tx, account string, season Season, version int64, event xpEvent, now time.Time) (bool, error) {
	progress, err := s.enterIn(ctx, tx, account, season, version, now)
	if err != nil {
		return false, err
	}
	tag, err := tx.Exec(ctx, `insert into pass_xp_events (account_id, season_id, amount, source_type, source_ref, idempotency_key, match_xp, economy_day, created_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9) on conflict (account_id, idempotency_key) do nothing`,
		account, season.ID, event.Amount, event.Source.Type, event.Source.Ref, event.Key, event.Match, EconomyDay(now), now)
	if err != nil || tag.RowsAffected() == 0 {
		return false, err
	}
	total := progress.XP + int64(event.Amount)
	if _, err := tx.Exec(ctx, `update pass_progress set xp_total = $3 where account_id = $1 and season_id = $2`, account, season.ID, total); err != nil {
		return false, err
	}
	before, after := season.Curve.LevelFor(progress.XP), season.Curve.LevelFor(total)
	if _, err := s.grantLevels(ctx, tx, account, season, version, unlocked(progress), before+1, after, now); err != nil {
		return false, err
	}
	if after > before {
		if err := analytics.RecordServerIn(ctx, tx, analytics.PassLevel, account, after-before, now); err != nil {
			return false, err
		}
	}
	payload, err := json.Marshal(XpAdded{Account: account, Season: season.ID, Amount: event.Amount, Source: event.Source, Key: event.Key, XpTotal: total, Level: after, At: now})
	if err != nil {
		return false, err
	}
	_, err = tx.Exec(ctx, `insert into outbox (topic, key, payload) values ($1, $2, $3) on conflict (topic, key) do nothing`, XpTopic, account+":"+event.Key, payload)
	return err == nil, err
}

func (s *Store) AddIn(ctx context.Context, tx pgx.Tx, account string, xp int, source cosmetics.Source, key string, now time.Time) (bool, error) {
	current, version, err := s.current()
	if err != nil {
		return false, err
	}
	season, ok := current.Active(now)
	if !ok {
		return false, nil
	}
	if _, err := s.addXpIn(ctx, tx, account, season, version, xpEvent{Amount: xp, Source: source, Key: key}, now); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) AwardMatch(ctx context.Context, match Match) (int, error) {
	current, version, err := s.current()
	if err != nil {
		return 0, err
	}
	xp := current.MatchXp(match)
	season, ok := current.Active(match.EndedAt)
	if xp == 0 || !ok {
		return 0, nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	if err := reward.LockIn(ctx, tx, match.Account); err != nil {
		return 0, err
	}
	var earned int
	if err := tx.QueryRow(ctx, `select coalesce(sum(amount), 0) from pass_xp_events where account_id = $1 and match_xp and economy_day = $2`, match.Account, EconomyDay(match.EndedAt)).Scan(&earned); err != nil {
		return 0, err
	}
	amount := capToRemainder(xp, earned, current.DailyMatchXpCap)
	if amount == 0 {
		return 0, tx.Commit(ctx)
	}
	added, err := s.addXpIn(ctx, tx, match.Account, season, version, xpEvent{Amount: amount, Source: cosmetics.Source{Type: SourceMatch, Ref: match.ID}, Key: "match:" + match.ID, Match: true}, match.EndedAt)
	if err != nil || !added {
		return 0, err
	}
	return amount, tx.Commit(ctx)
}

func (s *Store) UnlockPremiumIn(ctx context.Context, tx pgx.Tx, account, seasonID, ref string, now time.Time) (int, error) {
	current, version, err := s.current()
	if err != nil {
		return 0, err
	}
	season, ok := current.Season(seasonID)
	if !ok {
		return 0, ErrUnknownSeason
	}
	progress, err := s.enterIn(ctx, tx, account, season, version, now)
	if err != nil {
		return 0, err
	}
	var owner *string
	if err := tx.QueryRow(ctx, `select premium_ref from pass_progress where account_id = $1 and season_id = $2`, account, season.ID).Scan(&owner); err != nil {
		return 0, err
	}
	switch {
	case owner != nil && *owner == ref:
		return 0, nil
	case owner != nil:
		return 0, ErrPremiumOwned
	}
	if _, err := tx.Exec(ctx, `update pass_progress set premium_ref = $3, premium_at = $4 where account_id = $1 and season_id = $2`, account, season.ID, ref, now); err != nil {
		return 0, err
	}
	return s.grantLevels(ctx, tx, account, season, version, []string{Premium}, 1, season.Curve.LevelFor(progress.XP), now)
}

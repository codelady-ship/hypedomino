package pass

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

const SourcePremiumPlus = "pass_premium_plus"

type PremiumState struct {
	Season     string
	Premium    bool
	Level      int
	Levels     int
	PlusLevels int
}

func (s *Store) ActiveSeason(at time.Time) (string, bool) {
	current, _, err := s.current()
	if err != nil {
		return "", false
	}
	season, ok := current.Active(at)
	return season.ID, ok
}

func (s *Store) Premium(ctx context.Context, account string, now time.Time) (PremiumState, error) {
	current, _, err := s.current()
	if err != nil {
		return PremiumState{}, err
	}
	season, ok := current.Active(now)
	if !ok {
		return PremiumState{}, ErrNoSeason
	}
	state := PremiumState{Season: season.ID, Level: 1, Levels: season.Curve.Levels, PlusLevels: current.PremiumPlusLevels}
	var xp int64
	err = s.pool.QueryRow(ctx, `select xp_total, premium_ref is not null from pass_progress where account_id = $1 and season_id = $2`, account, season.ID).Scan(&xp, &state.Premium)
	if errors.Is(err, pgx.ErrNoRows) {
		return state, nil
	}
	state.Level = season.Curve.LevelFor(xp)
	return state, err
}

func (s *Store) AddPremiumPlusIn(ctx context.Context, tx pgx.Tx, account, seasonID, key string, now time.Time) (int64, error) {
	current, version, err := s.current()
	if err != nil {
		return 0, err
	}
	season, ok := current.Season(seasonID)
	if !ok {
		return 0, ErrUnknownSeason
	}
	progress, err := progressIn(ctx, tx, account, season.ID)
	if err != nil {
		return 0, err
	}
	level := season.Curve.LevelFor(progress.XP)
	reached := min(level+current.PremiumPlusLevels, season.Curve.Levels)
	overflow := int64(level+current.PremiumPlusLevels-reached) * current.LevelCrystalPrice
	amount := season.Curve.XpToReach(reached) - season.Curve.XpToReach(level)
	if amount == 0 {
		return overflow, nil
	}
	event := xpEvent{Amount: int(amount), Source: cosmetics.Source{Type: SourcePremiumPlus, Ref: key}, Key: key}
	_, err = s.addXpIn(ctx, tx, account, season, version, event, now)
	return overflow, err
}

func (s *Store) RevokePremiumIn(ctx context.Context, tx pgx.Tx, account, seasonID, ref, revokeKey string, now time.Time) error {
	current, _, err := s.current()
	if err != nil {
		return err
	}
	season, ok := current.Season(seasonID)
	if !ok {
		return ErrUnknownSeason
	}
	var owner *string
	var xp int64
	err = tx.QueryRow(ctx, `select premium_ref, xp_total from pass_progress where account_id = $1 and season_id = $2 for update`, account, season.ID).Scan(&owner, &xp)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && (owner == nil || *owner != ref)) {
		return nil
	}
	if err != nil {
		return err
	}
	var added int64
	if err := tx.QueryRow(ctx, `delete from pass_xp_events where account_id = $1 and idempotency_key = $2 returning amount`, account, ref).Scan(&added); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	remaining := max(0, xp-added)
	if _, err := tx.Exec(ctx, `update pass_progress set premium_ref = null, premium_at = null, xp_total = $3 where account_id = $1 and season_id = $2`, account, season.ID, remaining); err != nil {
		return err
	}
	keep := season.Curve.LevelFor(remaining)
	for level := 1; level <= season.Curve.Levels; level++ {
		if err := s.revokeLevel(ctx, tx, account, season.ID, Premium, level, revokeKey, now); err != nil {
			return err
		}
		if level > keep {
			if err := s.revokeLevel(ctx, tx, account, season.ID, Free, level, revokeKey, now); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Store) revokeLevel(ctx context.Context, tx pgx.Tx, account, season, track string, level int, revokeKey string, now time.Time) error {
	key := fmt.Sprintf("pass:%s:%s:%d", season, track, level)
	_, err := s.rewards.RevokeIn(ctx, tx, account, key, wallet.RefundRevoke, revokeKey+":"+key, now)
	if errors.Is(err, reward.ErrUnknownGrant) || errors.Is(err, reward.ErrAlreadyRevoke) {
		return nil
	}
	return err
}

package account

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrCountryInvalid  = errors.New("country.invalid")
	ErrCountryCooldown = errors.New("country.cooldown")
	ErrUnknown         = errors.New("account: unknown")
)

type Country struct {
	Code      string
	ChangedAt time.Time
}

func ValidCountry(code string) bool {
	if len(code) != 2 {
		return false
	}
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}

func (s *Store) Country(ctx context.Context, id string) (Country, error) {
	var c Country
	var changed *time.Time
	err := s.pool.QueryRow(ctx, `select coalesce(country, ''), country_changed_at from accounts where id = $1`, id).Scan(&c.Code, &changed)
	if errors.Is(err, pgx.ErrNoRows) {
		return Country{}, ErrUnknown
	}
	if changed != nil {
		c.ChangedAt = changed.UTC()
	}
	return c, err
}

func (s *Store) AdoptCountry(ctx context.Context, id, code string) error {
	if !ValidCountry(code) {
		return nil
	}
	_, err := s.pool.Exec(ctx, `update accounts set country = $2 where id = $1 and country is null`, id, code)
	return err
}

func (s *Store) SetCountry(ctx context.Context, id, code string, now time.Time, cooldown time.Duration) (Country, error) {
	if !ValidCountry(code) {
		return Country{}, fmt.Errorf("%w: %q", ErrCountryInvalid, code)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Country{}, err
	}
	defer tx.Rollback(ctx)
	var current string
	var changed *time.Time
	err = tx.QueryRow(ctx, `select coalesce(country, ''), country_changed_at from accounts where id = $1 for update`, id).Scan(&current, &changed)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Country{}, ErrUnknown
	case err != nil:
		return Country{}, err
	case current == code:
		return Country{Code: code, ChangedAt: changedAt(changed)}, nil
	case changed != nil && now.Before(changed.Add(cooldown)):
		return Country{}, fmt.Errorf("%w: until %s", ErrCountryCooldown, changed.Add(cooldown).UTC().Format(time.RFC3339))
	}
	if _, err := tx.Exec(ctx, `update accounts set country = $2, country_changed_at = $3 where id = $1`, id, code, now); err != nil {
		return Country{}, err
	}
	return Country{Code: code, ChangedAt: now.UTC()}, tx.Commit(ctx)
}

func changedAt(at *time.Time) time.Time {
	if at == nil {
		return time.Time{}
	}
	return at.UTC()
}

package push

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Register(ctx context.Context, account string, device Device, now time.Time) error {
	if err := device.Validate(); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `insert into push_devices (token, account_id, platform, registered_at) values ($1, $2, $3, $4)
		on conflict (token) do update set account_id = excluded.account_id, platform = excluded.platform, registered_at = excluded.registered_at`,
		device.Token, account, device.Platform, now)
	return err
}

func (s *Store) Unregister(ctx context.Context, account, token string) error {
	if token == "" {
		return ErrNoToken
	}
	_, err := s.pool.Exec(ctx, `delete from push_devices where token = $1 and account_id = $2`, token, account)
	return err
}

func (s *Store) Forget(ctx context.Context, token string) error {
	_, err := s.pool.Exec(ctx, `delete from push_devices where token = $1`, token)
	return err
}

func (s *Store) Devices(ctx context.Context, account string) ([]Device, error) {
	rows, err := s.pool.Query(ctx, `select token, platform from push_devices where account_id = $1 order by registered_at, token`, account)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Device])
}

func (s *Store) Settings(ctx context.Context, account string) (Settings, error) {
	var settings Settings
	err := s.pool.QueryRow(ctx, `select gameplay, promotional from push_settings where account_id = $1`, account).Scan(&settings.Gameplay, &settings.Promotional)
	if errors.Is(err, pgx.ErrNoRows) {
		return DefaultSettings, nil
	}
	return settings, err
}

func (s *Store) SaveSettings(ctx context.Context, account string, change SettingsChange, now time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	previous := DefaultSettings
	err = tx.QueryRow(ctx, `select promotional from push_settings where account_id = $1 for update`, account).Scan(&previous.Promotional)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if change.Promotional != previous.Promotional {
		if err := recordPromotionalConsent(ctx, tx, account, change, now); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `insert into push_settings (account_id, gameplay, promotional, updated_at) values ($1, $2, $3, $4)
		on conflict (account_id) do update set gameplay = excluded.gameplay, promotional = excluded.promotional, updated_at = excluded.updated_at`,
		account, change.Gameplay, change.Promotional, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func recordPromotionalConsent(ctx context.Context, tx pgx.Tx, account string, change SettingsChange, now time.Time) error {
	if change.PolicyVersion == "" || change.AppVersion == "" {
		return ErrConsentContext
	}
	_, err := tx.Exec(ctx, `insert into consent_records (account_id, purpose, granted, policy_version, source, app_version, recorded_at)
		values ($1, 'promotional_push', $2, $3, 'settings', $4, $5)`, account, change.Promotional, change.PolicyVersion, change.AppVersion, now)
	return err
}

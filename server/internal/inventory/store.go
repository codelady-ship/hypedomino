package inventory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/profile"
)

var (
	ErrUnknownItem     = errors.New("item.unknown")
	ErrNotOwned        = errors.New("item.not_owned")
	ErrWrongType       = errors.New("item.wrong_type")
	ErrDisabled        = errors.New("item.disabled")
	ErrDefaultItem     = errors.New("item.default")
	ErrSourceNotListed = errors.New("item.source_not_listed")
	ErrUnknownSlot     = errors.New("slot.unknown")
	ErrTooMany         = errors.New("slot.too_many")
	ErrNoProfile       = errors.New("profile.setup_required")
)

type Config struct {
	EmojiSlots  int
	ActionSlots int
}

func DefaultConfig() Config {
	return Config{EmojiSlots: 6, ActionSlots: 4}
}

type Owned struct {
	ID        string
	Type      string
	Source    cosmetics.Source
	GrantedAt time.Time
}

type Store struct {
	pool    *pgxpool.Pool
	catalog *cosmetics.Catalog
	config  Config
}

func NewStore(pool *pgxpool.Pool, catalog *cosmetics.Catalog, config Config) *Store {
	return &Store{pool: pool, catalog: catalog, config: config}
}

func (s *Store) Catalog() *cosmetics.Catalog {
	return s.catalog
}

func (s *Store) Owned(ctx context.Context, account string) ([]Owned, error) {
	var owned []Owned
	for _, item := range s.catalog.Defaults() {
		owned = append(owned, Owned{ID: item.ID, Type: item.Type, Source: cosmetics.Source{Type: cosmetics.SourceDefault}})
	}
	rows, err := s.pool.Query(ctx, `select item_id, source_type, source_ref, granted_at from inventory_items where account_id = $1 and revoked_at is null order by granted_at, item_id`, account)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var o Owned
		if err := rows.Scan(&o.ID, &o.Source.Type, &o.Source.Ref, &o.GrantedAt); err != nil {
			return nil, err
		}
		o.Type = cosmetics.TypeOf(o.ID)
		o.GrantedAt = o.GrantedAt.UTC()
		owned = append(owned, o)
	}
	return owned, rows.Err()
}

func (s *Store) Owns(ctx context.Context, account, id string) (bool, error) {
	return s.ownsIn(ctx, s.pool, account, id)
}

func (s *Store) OwnsIn(ctx context.Context, tx pgx.Tx, account, id string) (bool, error) {
	return s.ownsIn(ctx, tx, account, id)
}

func (s *Store) ownsIn(ctx context.Context, q querier, account, id string) (bool, error) {
	item, ok := s.catalog.Item(id)
	if !ok {
		return false, nil
	}
	if item.Default() && item.Availability == cosmetics.Live {
		return true, nil
	}
	return s.held(ctx, q, account, id)
}

func (s *Store) Grant(ctx context.Context, account, id string, source cosmetics.Source, now time.Time) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	granted, err := s.GrantIn(ctx, tx, account, id, source, now)
	if err != nil {
		return false, err
	}
	return granted, tx.Commit(ctx)
}

func (s *Store) GrantIn(ctx context.Context, tx pgx.Tx, account, id string, source cosmetics.Source, now time.Time) (bool, error) {
	item, ok := s.catalog.Item(id)
	switch {
	case !ok:
		return false, ErrUnknownItem
	case source.Type == cosmetics.SourceDefault:
		return false, ErrDefaultItem
	case !item.Grantable(source):
		return false, fmt.Errorf("%w: %s %s does not grant %s", ErrSourceNotListed, source.Type, source.Ref, id)
	}
	tag, err := tx.Exec(ctx, `insert into inventory_items (account_id, item_id, source_type, source_ref, granted_at) values ($1, $2, $3, $4, $5) on conflict do nothing`,
		account, id, source.Type, source.Ref, now)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 1 {
		return true, nil
	}
	tag, err = tx.Exec(ctx, `update inventory_items set source_type = $3, source_ref = $4, granted_at = $5, revoked_at = null where account_id = $1 and item_id = $2 and revoked_at is not null`,
		account, id, source.Type, source.Ref, now)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (s *Store) Revoke(ctx context.Context, account, id string, now time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := s.RevokeIn(ctx, tx, account, id, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) RevokeIn(ctx context.Context, tx pgx.Tx, account, id string, now time.Time) error {
	if item, ok := s.catalog.Item(id); ok && item.Default() {
		return ErrDefaultItem
	}
	tag, err := tx.Exec(ctx, `update inventory_items set revoked_at = $3 where account_id = $1 and item_id = $2 and revoked_at is null`, account, id, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotOwned
	}
	loadout, found, err := lock(ctx, tx, account)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	return s.save(ctx, tx, account, s.resolve(ctx, tx, account, loadout), now)
}

func (s *Store) Equip(ctx context.Context, account, slot string, ids []string, now time.Time) (profile.Loadout, error) {
	if err := s.fits(slot, ids); err != nil {
		return profile.Loadout{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return profile.Loadout{}, err
	}
	defer tx.Rollback(ctx)
	for _, id := range ids {
		if err := s.equippable(ctx, tx, account, slot, id); err != nil {
			return profile.Loadout{}, err
		}
	}
	loadout, found, err := lock(ctx, tx, account)
	if err != nil {
		return profile.Loadout{}, err
	}
	if !found {
		return profile.Loadout{}, ErrNoProfile
	}
	set(&loadout, slot, ids)
	if err := s.save(ctx, tx, account, loadout, now); err != nil {
		return profile.Loadout{}, err
	}
	return loadout, tx.Commit(ctx)
}

func (s *Store) Freeze(ctx context.Context, account string) (profile.Loadout, error) {
	var raw []byte
	err := s.pool.QueryRow(ctx, `select loadout from profiles where account_id = $1`, account).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return s.resolve(ctx, s.pool, account, profile.Loadout{}), nil
	}
	if err != nil {
		return profile.Loadout{}, err
	}
	var loadout profile.Loadout
	if err := json.Unmarshal(raw, &loadout); err != nil {
		return profile.Loadout{}, err
	}
	return s.resolve(ctx, s.pool, account, loadout), nil
}

func (s *Store) fits(slot string, ids []string) error {
	switch slot {
	case cosmetics.TypeAvatar, cosmetics.TypeFrame, cosmetics.TypeCharacter, cosmetics.TypeTable:
		if len(ids) != 1 {
			return fmt.Errorf("%w: %s takes exactly one item", ErrTooMany, slot)
		}
	case cosmetics.TypeEmoji:
		if len(ids) > s.config.EmojiSlots {
			return fmt.Errorf("%w: at most %d emojis", ErrTooMany, s.config.EmojiSlots)
		}
	case cosmetics.TypeAction:
		if len(ids) > s.config.ActionSlots {
			return fmt.Errorf("%w: at most %d actions", ErrTooMany, s.config.ActionSlots)
		}
	default:
		return fmt.Errorf("%w: %q", ErrUnknownSlot, slot)
	}
	if len(ids) != len(slices.Compact(slices.Sorted(slices.Values(ids)))) {
		return fmt.Errorf("%w: an item is listed twice", ErrTooMany)
	}
	return nil
}

type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (s *Store) equippable(ctx context.Context, q querier, account, slot, id string) error {
	item, ok := s.catalog.Item(id)
	switch {
	case !ok:
		return fmt.Errorf("%w: %s", ErrUnknownItem, id)
	case item.Type != slot:
		return fmt.Errorf("%w: %s is not a %s", ErrWrongType, id, slot)
	case !item.Equippable():
		return fmt.Errorf("%w: %s", ErrDisabled, id)
	case item.Default():
		return nil
	}
	held, err := s.held(ctx, q, account, id)
	if err != nil {
		return err
	}
	if !held {
		return fmt.Errorf("%w: %s", ErrNotOwned, id)
	}
	return nil
}

func (s *Store) held(ctx context.Context, q querier, account, id string) (bool, error) {
	var held bool
	err := q.QueryRow(ctx, `select exists (select 1 from inventory_items where account_id = $1 and item_id = $2 and revoked_at is null)`, account, id).Scan(&held)
	return held, err
}

func (s *Store) resolve(ctx context.Context, q querier, account string, loadout profile.Loadout) profile.Loadout {
	for _, slot := range []struct {
		name string
		id   *string
	}{{cosmetics.TypeAvatar, &loadout.Avatar}, {cosmetics.TypeFrame, &loadout.Frame}, {cosmetics.TypeCharacter, &loadout.Character}, {cosmetics.TypeTable, &loadout.Table}} {
		if *slot.id == "" || s.equippable(ctx, q, account, slot.name, *slot.id) != nil {
			if def, ok := s.catalog.Default(slot.name); ok {
				*slot.id = def.ID
			} else {
				*slot.id = ""
			}
		}
	}
	loadout.Emojis = s.keep(ctx, q, account, cosmetics.TypeEmoji, loadout.Emojis)
	loadout.Actions = s.keep(ctx, q, account, cosmetics.TypeAction, loadout.Actions)
	return loadout
}

func (s *Store) keep(ctx context.Context, q querier, account, slot string, ids []string) []string {
	var kept []string
	for _, id := range ids {
		if s.equippable(ctx, q, account, slot, id) == nil && !slices.Contains(kept, id) {
			kept = append(kept, id)
		}
	}
	return kept
}

func set(loadout *profile.Loadout, slot string, ids []string) {
	switch slot {
	case cosmetics.TypeAvatar:
		loadout.Avatar = ids[0]
	case cosmetics.TypeFrame:
		loadout.Frame = ids[0]
	case cosmetics.TypeCharacter:
		loadout.Character = ids[0]
	case cosmetics.TypeTable:
		loadout.Table = ids[0]
	case cosmetics.TypeEmoji:
		loadout.Emojis = slices.Clone(ids)
	case cosmetics.TypeAction:
		loadout.Actions = slices.Clone(ids)
	}
}

func lock(ctx context.Context, tx pgx.Tx, account string) (profile.Loadout, bool, error) {
	var raw []byte
	err := tx.QueryRow(ctx, `select loadout from profiles where account_id = $1 for update`, account).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return profile.Loadout{}, false, nil
	}
	if err != nil {
		return profile.Loadout{}, false, err
	}
	var loadout profile.Loadout
	if err := json.Unmarshal(raw, &loadout); err != nil {
		return profile.Loadout{}, false, err
	}
	return loadout, true, nil
}

func (s *Store) save(ctx context.Context, tx pgx.Tx, account string, loadout profile.Loadout, now time.Time) error {
	encoded, err := json.Marshal(loadout)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `update profiles set loadout = $2, updated_at = $3 where account_id = $1`, account, encoded, now)
	return err
}

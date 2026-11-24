package medal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
)

var (
	ErrNoCatalog       = errors.New("medal.unavailable")
	ErrUnknownMedal    = errors.New("medal.unknown")
	ErrSourceNotListed = errors.New("medal.source_not_listed")
	ErrNotOwned        = errors.New("medal.not_owned")
	ErrShowcase        = errors.New("medal.showcase_invalid")
)

type Owned struct {
	Medal    string    `json:"medal"`
	EarnedAt time.Time `json:"earned_at"`
}

type ShownMedal struct {
	Medal
	Group string `json:"group,omitempty"`
}

type Collection struct {
	Version  int64        `json:"version"`
	Medals   []ShownMedal `json:"medals"`
	Owned    []Owned      `json:"owned"`
	Showcase []string     `json:"showcase"`
	Slots    int          `json:"slots"`
}

type Store struct {
	pool        *pgxpool.Pool
	catalog     *snapshot.Watcher[*Catalog]
	missionType func(id string) string
}

func NewWatcher(pool *pgxpool.Pool, logger *slog.Logger) *snapshot.Watcher[*Catalog] {
	watcher := snapshot.New(pool, Domain, Parse, logger)
	watcher.Seed(Default(), 0)
	return watcher
}

func NewStore(pool *pgxpool.Pool, catalog *snapshot.Watcher[*Catalog]) *Store {
	return &Store{pool: pool, catalog: catalog}
}

func (s *Store) SetMissionTypes(missionType func(id string) string) {
	s.missionType = missionType
}

func (s *Store) current() (*Catalog, int64, error) {
	catalog, version, ok := s.catalog.Current()
	if !ok {
		return nil, 0, ErrNoCatalog
	}
	return catalog, version, nil
}

func (s *Store) GrantIn(ctx context.Context, tx pgx.Tx, account, id string, source cosmetics.Source, now time.Time) (bool, error) {
	catalog, _, err := s.current()
	if err != nil {
		return false, err
	}
	medal, ok := catalog.Medal(id)
	switch {
	case !ok:
		return false, fmt.Errorf("%w: %s", ErrUnknownMedal, id)
	case medal.Availability != cosmetics.Live || !medal.Lists(source):
		return false, fmt.Errorf("%w: %s %s does not grant %s", ErrSourceNotListed, source.Type, source.Ref, id)
	}
	tag, err := tx.Exec(ctx, `insert into medal_ownership (account_id, medal_id, source_type, source_ref, granted_at) values ($1, $2, $3, $4, $5) on conflict do nothing`, account, id, source.Type, source.Ref, now)
	if err != nil || tag.RowsAffected() == 1 {
		return err == nil, err
	}
	tag, err = tx.Exec(ctx, `update medal_ownership set source_type = $3, source_ref = $4, granted_at = $5, revoked_at = null where account_id = $1 and medal_id = $2 and revoked_at is not null`, account, id, source.Type, source.Ref, now)
	return err == nil && tag.RowsAffected() == 1, err
}

func (s *Store) RevokeIn(ctx context.Context, tx pgx.Tx, account, id string, now time.Time) error {
	if _, err := tx.Exec(ctx, `update medal_ownership set revoked_at = $3 where account_id = $1 and medal_id = $2 and revoked_at is null`, account, id, now); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `update medal_showcases set medals = array_remove(medals, $2), updated_at = $3 where account_id = $1`, account, id, now)
	return err
}

func (s *Store) owned(ctx context.Context, q pgx.Tx, account string) ([]Owned, error) {
	rows, err := q.Query(ctx, `select medal_id, granted_at from medal_ownership where account_id = $1 and revoked_at is null order by granted_at, medal_id`, account)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Owned, error) {
		var o Owned
		return o, row.Scan(&o.Medal, &o.EarnedAt)
	})
}

func (s *Store) Collection(ctx context.Context, account string) (Collection, error) {
	catalog, version, err := s.current()
	if err != nil {
		return Collection{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Collection{}, err
	}
	defer tx.Rollback(ctx)
	owned, err := s.owned(ctx, tx, account)
	if err != nil {
		return Collection{}, err
	}
	held := map[string]bool{}
	for _, o := range owned {
		held[o.Medal] = true
	}
	out := Collection{Version: version, Owned: owned, Slots: catalog.ProfileMedalSlots, Medals: []ShownMedal{}, Showcase: []string{}}
	for _, medal := range catalog.Medals {
		if medal.Availability == cosmetics.Live || (medal.Availability == cosmetics.Retired && held[medal.ID]) {
			out.Medals = append(out.Medals, ShownMedal{Medal: medal, Group: medal.Group(s.missionType)})
		}
	}
	err = tx.QueryRow(ctx, `select medals from medal_showcases where account_id = $1`, account).Scan(&out.Showcase)
	if errors.Is(err, pgx.ErrNoRows) {
		err = nil
	}
	return out, err
}

func (s *Store) SetShowcase(ctx context.Context, account string, medals []string, now time.Time) ([]string, error) {
	catalog, _, err := s.current()
	if err != nil {
		return nil, err
	}
	if len(medals) > catalog.ProfileMedalSlots || hasDuplicates(medals) {
		return nil, ErrShowcase
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	owned, err := s.owned(ctx, tx, account)
	if err != nil {
		return nil, err
	}
	for _, id := range medals {
		if !slices.ContainsFunc(owned, func(o Owned) bool { return o.Medal == id }) {
			return nil, fmt.Errorf("%w: %s", ErrNotOwned, id)
		}
	}
	showcase := append([]string{}, medals...)
	if _, err := tx.Exec(ctx, `insert into medal_showcases (account_id, medals, updated_at) values ($1, $2, $3) on conflict (account_id) do update set medals = excluded.medals, updated_at = excluded.updated_at`, account, showcase, now); err != nil {
		return nil, err
	}
	return showcase, tx.Commit(ctx)
}

func hasDuplicates(ids []string) bool {
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			return true
		}
		seen[id] = true
	}
	return false
}

package promo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

const StateDraft = "draft"

type Template struct {
	ID       string         `json:"id"`
	Campaign string         `json:"campaign"`
	Kind     string         `json:"kind"`
	Bundle   *reward.Bundle `json:"bundle"`
	Template bool           `json:"template"`
	State    string         `json:"state"`
	Creator  string         `json:"creator"`
	Updated  time.Time      `json:"updated"`
}

func validateTemplate(t Template) error {
	fail := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
	}
	switch {
	case !promoID.MatchString(t.ID):
		return fail("id %q", t.ID)
	case !campaignID.MatchString(t.Campaign):
		return fail("campaign %q", t.Campaign)
	case !kinds[t.Kind]:
		return fail("kind %q", t.Kind)
	case t.Creator == "":
		return fail("creator is required")
	case t.Bundle == nil:
		return nil
	case len(t.Bundle.Medals) > 0:
		return fail("a code never carries a medal")
	}
	if err := t.Bundle.ValidateFor(wallet.Promo); err != nil {
		return fail("%v", err)
	}
	return nil
}

func (s *Store) CreateTemplate(ctx context.Context, t Template, now time.Time) (Template, error) {
	if err := validateTemplate(t); err != nil {
		return Template{}, err
	}
	encoded, err := encodeBundle(t.Bundle)
	if err != nil {
		return Template{}, err
	}
	_, err = s.db.Exec(ctx, `insert into promo_templates (promo_id, campaign_id, kind, bundle, created_by, created_at, updated_at) values ($1, $2, $3, $4, $5, $6, $6)`,
		t.ID, t.Campaign, t.Kind, encoded, t.Creator, now)
	if isUniqueViolation(err) {
		return Template{}, fmt.Errorf("%w: template %s already exists", ErrDuplicate, t.ID)
	}
	if err != nil {
		return Template{}, err
	}
	return drafted(t, now), nil
}

func (s *Store) EditTemplate(ctx context.Context, t Template, now time.Time) (Template, Template, error) {
	before, err := s.lockedTemplate(ctx, t.ID)
	if err != nil {
		return Template{}, Template{}, err
	}
	t.Creator = before.Creator
	if err := validateTemplate(t); err != nil {
		return Template{}, Template{}, err
	}
	encoded, err := encodeBundle(t.Bundle)
	if err != nil {
		return Template{}, Template{}, err
	}
	if _, err := s.db.Exec(ctx, `update promo_templates set campaign_id = $2, kind = $3, bundle = $4, updated_at = $5 where promo_id = $1`, t.ID, t.Campaign, t.Kind, encoded, now); err != nil {
		return Template{}, Template{}, err
	}
	return before, drafted(t, now), nil
}

func (s *Store) Templates(ctx context.Context) ([]Template, error) {
	rows, err := s.db.Query(ctx, `select `+templateColumns+` from promo_templates order by promo_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	templates := []Template{}
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

func (s *Store) lockedTemplate(ctx context.Context, id string) (Template, error) {
	t, err := scanTemplate(s.db.QueryRow(ctx, `select `+templateColumns+` from promo_templates where promo_id = $1 for update`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Template{}, ErrUnknown
	}
	return t, err
}

const templateColumns = `promo_id, campaign_id, kind, bundle, created_by, updated_at`

func scanTemplate(row pgx.Row) (Template, error) {
	var t Template
	var bundle []byte
	if err := row.Scan(&t.ID, &t.Campaign, &t.Kind, &bundle, &t.Creator, &t.Updated); err != nil {
		return Template{}, err
	}
	if bundle != nil {
		t.Bundle = &reward.Bundle{}
		if err := json.Unmarshal(bundle, t.Bundle); err != nil {
			return Template{}, err
		}
	}
	return drafted(t, t.Updated), nil
}

func drafted(t Template, updated time.Time) Template {
	t.Template, t.State, t.Updated = true, StateDraft, updated.UTC()
	return t
}

func encodeBundle(bundle *reward.Bundle) ([]byte, error) {
	if bundle == nil {
		return nil, nil
	}
	return json.Marshal(bundle)
}

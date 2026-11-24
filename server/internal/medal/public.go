package medal

import (
	"context"
	"errors"
	"slices"

	"github.com/jackc/pgx/v5"
)

func (s *Store) Public(ctx context.Context, account string) ([]Medal, error) {
	catalog, _, err := s.current()
	if err != nil {
		return nil, err
	}
	var chosen []string
	err = s.pool.QueryRow(ctx, `select medals from medal_showcases where account_id = $1`, account).Scan(&chosen)
	if errors.Is(err, pgx.ErrNoRows) {
		return []Medal{}, nil
	}
	if err != nil {
		return nil, err
	}
	shown := []Medal{}
	for _, id := range chosen {
		if at := slices.IndexFunc(catalog.Medals, func(m Medal) bool { return m.ID == id }); at >= 0 {
			shown = append(shown, catalog.Medals[at])
		}
	}
	return shown, nil
}

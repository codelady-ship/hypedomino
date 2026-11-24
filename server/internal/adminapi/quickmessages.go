package adminapi

import (
	"context"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/gamestrings"
	"github.com/Alievs-corp/hypedomino/server/internal/quickmessage"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

func quickMessagesDomain() configDomain {
	return configDomain{name: quickmessage.Domain, object: rbac.GameStrings, prepare: requireTranslatedKeys, accept: func(_, after []byte) error {
		_, err := quickmessage.Parse(after)
		return err
	}}
}

func requireTranslatedKeys(ctx context.Context, tx pgx.Tx, _, after []byte) ([]byte, error) {
	catalog, err := quickmessage.Parse(after)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `select v.key from game_string_values v join game_strings s on s.key = v.key
		where v.key = any($1) and v.status = 'approved' and not s.deprecated and v.language = any($2)
		group by v.key having count(*) = $3`, catalog.Keys, languages(), len(gamestrings.Languages))
	if err != nil {
		return nil, err
	}
	translated, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}
	var missing []string
	for _, key := range catalog.Keys {
		if !slices.Contains(translated, key) {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("%w: keys without an approved value in every language: %v", quickmessage.ErrInvalid, missing)
	}
	return after, nil
}

func languages() []string {
	codes := make([]string, 0, len(gamestrings.Languages))
	for _, language := range gamestrings.Languages {
		codes = append(codes, string(language))
	}
	return codes
}

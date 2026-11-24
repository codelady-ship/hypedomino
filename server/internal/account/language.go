package account

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/Alievs-corp/hypedomino/server/internal/gamestrings"
	"github.com/jackc/pgx/v5"
)

var ErrLanguageInvalid = errors.New("language.invalid")

func ValidLanguage(code string) bool {
	return slices.Contains(gamestrings.Languages, gamestrings.Language(code))
}

func (s *Store) Language(ctx context.Context, id string) (string, error) {
	var language string
	err := s.pool.QueryRow(ctx, `select coalesce(language, '') from accounts where id = $1`, id).Scan(&language)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrUnknown
	}
	return language, err
}

func (s *Store) SetLanguage(ctx context.Context, id, code string) (string, error) {
	if !ValidLanguage(code) {
		return "", fmt.Errorf("%w: %q", ErrLanguageInvalid, code)
	}
	tag, err := s.pool.Exec(ctx, `update accounts set language = $2 where id = $1`, id, code)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() == 0 {
		return "", ErrUnknown
	}
	return code, nil
}

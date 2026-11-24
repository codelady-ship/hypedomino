package friends

import (
	"context"
	"crypto/rand"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	codeAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	codeLength   = 8
	codeAttempts = 5
)

var codeReading = strings.NewReplacer("-", "", " ", "", "O", "0", "I", "1", "L", "1")

func newCode() (string, error) {
	raw := make([]byte, codeLength)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	for i, b := range raw {
		raw[i] = codeAlphabet[int(b)%len(codeAlphabet)]
	}
	return string(raw), nil
}

func normalizeCode(input string) (string, bool) {
	code := codeReading.Replace(strings.ToUpper(strings.TrimSpace(input)))
	if len(code) != codeLength {
		return "", false
	}
	for _, r := range code {
		if !strings.ContainsRune(codeAlphabet, r) {
			return "", false
		}
	}
	return code, true
}

func DisplayCode(code string) string {
	return code[:codeLength/2] + "-" + code[codeLength/2:]
}

func (s *Store) Code(ctx context.Context, account string) (string, error) {
	for range codeAttempts {
		var code string
		err := s.pool.QueryRow(ctx, `select code from friend_codes where account_id = $1`, account).Scan(&code)
		if err == nil {
			return code, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return "", err
		}
		if code, err = newCode(); err != nil {
			return "", err
		}
		_, err = s.pool.Exec(ctx, `insert into friend_codes (account_id, code) values ($1, $2) on conflict (account_id) do nothing`, account, code)
		var pgErr *pgconn.PgError
		if err != nil && !(errors.As(err, &pgErr) && pgErr.Code == "23505") {
			return "", err
		}
	}
	return "", errors.New("friends: no free friend code")
}

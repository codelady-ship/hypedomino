package appeal

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Alievs-corp/hypedomino/server/internal/sanction"
)

const (
	StateOpen   = "open"
	StateUpheld = "upheld"
	StateLifted = "lifted"
)

var (
	ErrTextRequired  = errors.New("text_required")
	ErrTextTooLong   = errors.New("text_too_long")
	ErrNotAppealable = errors.New("not_appealable")
	ErrEnded         = errors.New("sanction_ended")
	ErrOpen          = errors.New("appeal_open")
	ErrRejected      = errors.New("appeal_rejected")
	ErrNoAppeal      = errors.New("appeal: no open appeal with that id")
	ErrDecision      = errors.New("appeal: unknown decision")
)

var appealable = []sanction.Kind{sanction.TemporaryBan, sanction.PermanentBan, sanction.RankedBan, sanction.PenaltyBan, sanction.ForcedRename}

func Appealable(kind sanction.Kind) bool {
	return slices.Contains(appealable, kind)
}

func CleanText(raw string, maxLength int) (string, error) {
	text := strings.TrimSpace(raw)
	switch {
	case text == "":
		return "", ErrTextRequired
	case !utf8.ValidString(text) || utf8.RuneCountInString(text) > maxLength:
		return "", ErrTextTooLong
	}
	return text, nil
}

func Admissible(s sanction.Sanction, existing string, now time.Time) error {
	switch {
	case !Appealable(s.Kind):
		return ErrNotAppealable
	case existing == StateOpen:
		return ErrOpen
	case existing == StateUpheld:
		return ErrRejected
	case existing == StateLifted, !s.Running(now):
		return ErrEnded
	}
	return nil
}

type Appeal struct {
	ID         int64      `json:"id"`
	AccountID  string     `json:"account_id"`
	SanctionID int64      `json:"sanction_id"`
	Text       string     `json:"text"`
	State      string     `json:"state"`
	CreatedAt  time.Time  `json:"created_at"`
	DecidedAt  *time.Time `json:"decided_at"`
}

const columns = `id, account_id::text, sanction_id, text, state, created_at, decided_at`

func scan(row pgx.Row) (Appeal, error) {
	var a Appeal
	err := row.Scan(&a.ID, &a.AccountID, &a.SanctionID, &a.Text, &a.State, &a.CreatedAt, &a.DecidedAt)
	a.CreatedAt = a.CreatedAt.UTC()
	if a.DecidedAt != nil {
		decided := a.DecidedAt.UTC()
		a.DecidedAt = &decided
	}
	return a, err
}

type Beginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

func Submit(ctx context.Context, db Beginner, account string, sanctionID int64, text string, now time.Time) (Appeal, error) {
	var filed Appeal
	err := pgx.BeginFunc(ctx, db, func(tx pgx.Tx) error {
		s, err := sanction.Lock(ctx, tx, account, sanctionID)
		if err != nil {
			return err
		}
		var existing string
		if err := tx.QueryRow(ctx, `select state from appeals where sanction_id = $1`, sanctionID).Scan(&existing); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if err := Admissible(s, existing, now); err != nil {
			return err
		}
		filed, err = scan(tx.QueryRow(ctx, `insert into appeals (account_id, sanction_id, text, created_at) values ($1, $2, $3, $4) returning `+columns,
			account, sanctionID, text, now))
		return err
	})
	if isUniqueViolation(err) {
		return Appeal{}, ErrOpen
	}
	return filed, err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

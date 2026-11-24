package sanction

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
)

type Kind string

const (
	TemporaryBan Kind = "temporary_ban"
	PermanentBan Kind = "permanent_ban"
	RankedBan    Kind = "ranked_ban"
	PenaltyBan   Kind = "penalty_ban"
	ForcedRename Kind = "forced_rename"
)

type Category string

const (
	Cheating      Category = "cheating"
	Collusion     Category = "collusion"
	AfkLeaving    Category = "afk_leaving"
	OffensiveName Category = "offensive_name"
	PaymentFraud  Category = "payment_fraud"
	Other         Category = "other"
)

var Categories = []Category{Cheating, Collusion, AfkLeaving, OffensiveName, PaymentFraud, Other}

var (
	ErrInvalidCategory = errors.New("sanction: unknown ban reason category")
	ErrUnknown         = errors.New("sanction: no such sanction on the account")
)

var reasons = map[Category]pb.BanReason{
	Cheating:      pb.BanReason_BAN_REASON_CHEATING,
	Collusion:     pb.BanReason_BAN_REASON_COLLUSION,
	AfkLeaving:    pb.BanReason_BAN_REASON_AFK_LEAVING,
	OffensiveName: pb.BanReason_BAN_REASON_OFFENSIVE_NAME,
	PaymentFraud:  pb.BanReason_BAN_REASON_PAYMENT_FRAUD,
	Other:         pb.BanReason_BAN_REASON_OTHER,
}

func ParseCategory(name string) (Category, error) {
	category := Category(name)
	if !slices.Contains(Categories, category) {
		return "", ErrInvalidCategory
	}
	return category, nil
}

func (c Category) Reason() pb.BanReason {
	if reason, ok := reasons[c]; ok {
		return reason
	}
	return pb.BanReason_BAN_REASON_OTHER
}

type Sanction struct {
	ID       int64      `json:"id"`
	Kind     Kind       `json:"kind"`
	Category Category   `json:"category"`
	StartsAt time.Time  `json:"starts_at"`
	EndsAt   *time.Time `json:"ends_at"`
	LiftedAt *time.Time `json:"lifted_at"`
}

type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Execer interface {
	Querier
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

const columns = `id, kind, category, starts_at, ends_at, lifted_at`

const running = `lifted_at is null and (ends_at is null or ends_at > $3)`

func scan(row pgx.Row) (Sanction, error) {
	var s Sanction
	err := row.Scan(&s.ID, &s.Kind, &s.Category, &s.StartsAt, &s.EndsAt, &s.LiftedAt)
	return s, err
}

func Impose(ctx context.Context, db Execer, account string, kind Kind, category Category, now time.Time, ends *time.Time) (Sanction, error) {
	if _, err := Lift(ctx, db, account, kind, now); err != nil {
		return Sanction{}, err
	}
	return scan(db.QueryRow(ctx, `insert into account_sanctions (account_id, kind, category, starts_at, ends_at) values ($1, $2, $3, $4, $5)
		returning `+columns, account, string(kind), string(category), now, ends))
}

func Lift(ctx context.Context, db Execer, account string, kind Kind, now time.Time) (int64, error) {
	tag, err := db.Exec(ctx, `update account_sanctions set lifted_at = $3 where account_id = $1 and kind = $2 and `+running, account, string(kind), now)
	return tag.RowsAffected(), err
}

func (s Sanction) Running(now time.Time) bool {
	return s.LiftedAt == nil && (s.EndsAt == nil || s.EndsAt.After(now))
}

func Lock(ctx context.Context, db Querier, account string, id int64) (Sanction, error) {
	s, err := scan(db.QueryRow(ctx, `select `+columns+` from account_sanctions where id = $1 and account_id = $2 for update`, id, account))
	if errors.Is(err, pgx.ErrNoRows) {
		return Sanction{}, ErrUnknown
	}
	return s, err
}

func LiftByID(ctx context.Context, db Execer, id int64, now time.Time) (bool, error) {
	tag, err := db.Exec(ctx, `update account_sanctions set lifted_at = $2 where id = $1 and lifted_at is null and (ends_at is null or ends_at > $2)`, id, now)
	return tag.RowsAffected() == 1, err
}

func End(ctx context.Context, db Execer, account string, kind Kind, now time.Time) error {
	_, err := db.Exec(ctx, `update account_sanctions set ends_at = greatest($3, starts_at + interval '1 microsecond') where account_id = $1 and kind = $2 and `+running, account, string(kind), now)
	return err
}

func Current(ctx context.Context, db Querier, account string, now time.Time, kinds ...Kind) (Sanction, bool, error) {
	names := make([]string, len(kinds))
	for i, kind := range kinds {
		names[i] = string(kind)
	}
	s, err := scan(db.QueryRow(ctx, `select `+columns+` from account_sanctions where account_id = $1 and kind = any($2) and `+running+`
		order by starts_at desc, id desc limit 1`, account, names, now))
	if errors.Is(err, pgx.ErrNoRows) {
		return Sanction{}, false, nil
	}
	return s, err == nil, err
}

func List(ctx context.Context, db Querier, account string, limit int) ([]Sanction, error) {
	rows, err := db.Query(ctx, `select `+columns+` from account_sanctions where account_id = $1 order by starts_at desc, id desc limit $2`, account, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Sanction, error) { return scan(row) })
}

type RankedBanError struct {
	Sanction Sanction
}

func (e *RankedBanError) Error() string {
	return "sanction: ranked ban"
}

func RankedBanCheck(db Querier) func(ctx context.Context, account string, now time.Time) error {
	return func(ctx context.Context, account string, now time.Time) error {
		s, banned, err := Current(ctx, db, account, now, RankedBan)
		if err != nil || !banned {
			return err
		}
		return &RankedBanError{Sanction: s}
	}
}

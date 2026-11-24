package approval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

var (
	ErrInvalid      = errors.New("approval: invalid proposal")
	ErrNotAllowed   = errors.New("approval: the roles do not allow this")
	ErrSelfDecision = errors.New("approval: the proposer cannot decide their own proposal")
	ErrDecided      = errors.New("approval: proposal already decided")
	ErrExpired      = errors.New("approval: proposal expired")
	ErrNotFound     = errors.New("approval: no such proposal")
)

type Kind string

const (
	PlayerGrant     Kind = "player_grant"
	PromoCode       Kind = "promo_code"
	PermanentBan    Kind = "permanent_ban"
	LiveControl     Kind = "live_control"
	SuperAdminGrant Kind = "super_admin_grant"
	BreakGlass      Kind = "break_glass"
)

type rule struct {
	object    rbac.Object
	proposers []rbac.Role
	approvers []rbac.Role
	confirmed bool
}

var rules = map[Kind]rule{
	PlayerGrant:     {object: rbac.Players, proposers: []rbac.Role{rbac.Support}, approvers: []rbac.Role{rbac.Support, rbac.SuperAdmin}},
	PromoCode:       {object: rbac.PromoCodes, proposers: []rbac.Role{rbac.LiveOps}, approvers: []rbac.Role{rbac.LiveOps, rbac.SuperAdmin}},
	PermanentBan:    {object: rbac.Players, proposers: []rbac.Role{rbac.Support}, approvers: []rbac.Role{rbac.Support, rbac.SuperAdmin}},
	LiveControl:     {object: rbac.LiveControls, proposers: []rbac.Role{rbac.LiveOps, rbac.SuperAdmin}, approvers: []rbac.Role{rbac.LiveOps, rbac.SuperAdmin}},
	SuperAdminGrant: {object: rbac.AdminAccounts, proposers: []rbac.Role{rbac.SuperAdmin}, approvers: []rbac.Role{rbac.SuperAdmin}},
	BreakGlass:      {object: rbac.LiveControls, proposers: []rbac.Role{rbac.LiveOps, rbac.SuperAdmin}, approvers: []rbac.Role{rbac.LiveOps, rbac.SuperAdmin}, confirmed: true},
}

type State string

const (
	Pending  State = "pending"
	Approved State = "approved"
	Rejected State = "rejected"
)

type Proposal struct {
	ID         int64
	Kind       Kind
	Object     audit.Object
	Change     json.RawMessage
	Reason     string
	Proposer   audit.Actor
	ProposedAt time.Time
	ExpiresAt  time.Time
	State      State
	Decider    audit.Actor
	DecidedAt  time.Time
}

type Config struct {
	TTL time.Duration
}

func DefaultConfig() Config {
	return Config{TTL: 24 * time.Hour}
}

type database interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Store struct {
	db     database
	config Config
}

func NewStore(pool *pgxpool.Pool, config Config) *Store {
	return &Store{db: pool, config: config}
}

func (s *Store) In(tx pgx.Tx) *Store {
	return &Store{db: tx, config: s.config}
}

func holdsAny(actor audit.Actor, roles []rbac.Role) bool {
	return slices.ContainsFunc(actor.Roles, func(role rbac.Role) bool { return slices.Contains(roles, role) })
}

func Decides(roles []rbac.Role, kind Kind) bool {
	return holdsAny(audit.Actor{Roles: roles}, rules[kind].approvers)
}

func (s *Store) Propose(ctx context.Context, kind Kind, objectID string, change json.RawMessage, reason string, proposer audit.Actor, now time.Time) (Proposal, error) {
	rule, known := rules[kind]
	switch {
	case !known:
		return Proposal{}, fmt.Errorf("%w: kind %q", ErrInvalid, kind)
	case objectID == "" || strings.TrimSpace(reason) == "" || proposer.ID == "":
		return Proposal{}, fmt.Errorf("%w: object, reason and proposer are required", ErrInvalid)
	case !json.Valid(change):
		return Proposal{}, fmt.Errorf("%w: change must be JSON", ErrInvalid)
	case !holdsAny(proposer, rule.proposers):
		return Proposal{}, fmt.Errorf("%w: %v cannot propose %s", ErrNotAllowed, proposer.Roles, kind)
	}
	p := Proposal{
		Kind: kind, Object: audit.Object{Type: rule.object, ID: objectID}, Change: change, Reason: reason,
		Proposer: proposer, ProposedAt: now.UTC().Truncate(time.Microsecond), ExpiresAt: now.Add(s.config.TTL).UTC().Truncate(time.Microsecond), State: Pending,
	}
	err := s.db.QueryRow(ctx, `insert into admin_approvals
		(kind, object_type, object_id, change, reason, proposer_id, proposer_roles, proposed_at, expires_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9) returning id`,
		p.Kind, p.Object.Type, p.Object.ID, string(p.Change), p.Reason, p.Proposer.ID, p.Proposer.Roles, p.ProposedAt, p.ExpiresAt,
	).Scan(&p.ID)
	return p, err
}

func (s *Store) Approve(ctx context.Context, id int64, approver audit.Actor, now time.Time) (Proposal, error) {
	return s.decide(ctx, id, approver, Approved, now)
}

func (s *Store) Reject(ctx context.Context, id int64, approver audit.Actor, now time.Time) (Proposal, error) {
	return s.decide(ctx, id, approver, Rejected, now)
}

func (s *Store) decide(ctx context.Context, id int64, decider audit.Actor, state State, now time.Time) (Proposal, error) {
	var decided Proposal
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		p, err := scan(tx.QueryRow(ctx, `select `+columns+` from admin_approvals where id = $1 for update`, id))
		if err != nil {
			return err
		}
		if err := p.decidableBy(decider, now); err != nil {
			return err
		}
		p.State, p.Decider, p.DecidedAt = state, decider, now.UTC().Truncate(time.Microsecond)
		tag, err := tx.Exec(ctx, `update admin_approvals set state = $2, decider_id = $3, decider_roles = $4, decided_at = $5 where id = $1 and state = 'pending'`,
			p.ID, p.State, p.Decider.ID, p.Decider.Roles, p.DecidedAt)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return fmt.Errorf("%w: %d", ErrDecided, p.ID)
		}
		decided = p
		return nil
	})
	return decided, err
}

func (p Proposal) decidableBy(decider audit.Actor, now time.Time) error {
	switch {
	case p.State != Pending:
		return fmt.Errorf("%w: %d is %s", ErrDecided, p.ID, p.State)
	case !now.Before(p.ExpiresAt) && !rules[p.Kind].confirmed:
		return fmt.Errorf("%w: %d expired at %s", ErrExpired, p.ID, p.ExpiresAt)
	case decider.ID == p.Proposer.ID:
		return ErrSelfDecision
	case !holdsAny(decider, rules[p.Kind].approvers):
		return fmt.Errorf("%w: %v cannot decide %s", ErrNotAllowed, decider.Roles, p.Kind)
	}
	return nil
}

func (s *Store) Get(ctx context.Context, id int64) (Proposal, error) {
	return scan(s.db.QueryRow(ctx, `select `+columns+` from admin_approvals where id = $1`, id))
}

func (s *Store) Pending(ctx context.Context, now time.Time) ([]Proposal, error) {
	return s.list(ctx, `select `+columns+` from admin_approvals where state = 'pending' and (expires_at > $1 or kind = $2) order by id`, now.UTC(), BreakGlass)
}

func (s *Store) Unconfirmed(ctx context.Context) ([]Proposal, error) {
	return s.list(ctx, `select `+columns+` from admin_approvals where state = 'pending' and kind = $1 order by id`, BreakGlass)
}

func (s *Store) list(ctx context.Context, sql string, args ...any) ([]Proposal, error) {
	rows, err := s.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Proposal, error) { return scan(row) })
}

const columns = `id, kind, object_type, object_id, change, reason, proposer_id, proposer_roles, proposed_at, expires_at,
	state, coalesce(decider_id, ''), coalesce(decider_roles, '{}'), decided_at`

func scan(row pgx.Row) (Proposal, error) {
	var p Proposal
	var change []byte
	var decidedAt *time.Time
	err := row.Scan(&p.ID, &p.Kind, &p.Object.Type, &p.Object.ID, &change, &p.Reason, &p.Proposer.ID, &p.Proposer.Roles,
		&p.ProposedAt, &p.ExpiresAt, &p.State, &p.Decider.ID, &p.Decider.Roles, &decidedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Proposal{}, ErrNotFound
	}
	if err != nil {
		return Proposal{}, err
	}
	p.Change = change
	p.ProposedAt, p.ExpiresAt = p.ProposedAt.UTC(), p.ExpiresAt.UTC()
	if len(p.Decider.Roles) == 0 {
		p.Decider.Roles = nil
	}
	if decidedAt != nil {
		p.DecidedAt = decidedAt.UTC()
	}
	return p, nil
}

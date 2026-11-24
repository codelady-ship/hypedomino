package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

var (
	ErrInvalid   = errors.New("audit: invalid entry")
	ErrForbidden = errors.New("audit: the roles cannot view the audit log")
)

var playerObjects = []rbac.Object{rbac.Players, rbac.Queues, rbac.RightsRequests}

const (
	Staging    = "staging"
	Production = "production"
)

const (
	DefaultLimit = 50
	MaxLimit     = 200
)

type Actor struct {
	ID    string
	Roles []rbac.Role
}

type Object struct {
	Type rbac.Object
	ID   string
}

type Source struct {
	SessionID string
	IP        netip.Addr
	UserAgent string
}

type Entry struct {
	Actor         Actor
	Environment   string
	Action        string
	Object        Object
	Before        json.RawMessage
	After         json.RawMessage
	Reason        string
	ApprovalID    int64
	Ref           string
	ChangeSet     string
	ConfigVersion int64
	Source        Source
}

type Recorded struct {
	Entry
	ID int64
	At time.Time
}

type Filter struct {
	ActorID        string
	Object         Object
	Action         string
	From, To       time.Time
	ReasonContains string
	ConfigVersion  int64
	BeforeID       int64
	Limit          int
}

type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Store struct {
	db querier
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{db: pool}
}

func (s *Store) In(tx pgx.Tx) *Store {
	return &Store{db: tx}
}

func (e Entry) Validate() error {
	switch {
	case e.Actor.ID == "" || len(e.Actor.Roles) == 0:
		return fmt.Errorf("%w: actor id and roles are required", ErrInvalid)
	case slices.ContainsFunc(e.Actor.Roles, func(role rbac.Role) bool { return !rbac.IsRole(role) }):
		return fmt.Errorf("%w: unknown role in %v", ErrInvalid, e.Actor.Roles)
	case e.Environment != Staging && e.Environment != Production:
		return fmt.Errorf("%w: environment %q", ErrInvalid, e.Environment)
	case e.Action == "" || e.Object.Type == "" || e.Object.ID == "":
		return fmt.Errorf("%w: action and object are required", ErrInvalid)
	case !rbac.IsObject(e.Object.Type):
		return fmt.Errorf("%w: unknown object type %q", ErrInvalid, e.Object.Type)
	case e.Source.SessionID == "":
		return fmt.Errorf("%w: session id is required", ErrInvalid)
	case !validJSON(e.Before) || !validJSON(e.After):
		return fmt.Errorf("%w: before and after must be JSON", ErrInvalid)
	}
	return nil
}

func validJSON(raw json.RawMessage) bool {
	return len(raw) == 0 || json.Valid(raw)
}

func (s *Store) Append(ctx context.Context, entry Entry, now time.Time) (Recorded, error) {
	if err := entry.Validate(); err != nil {
		return Recorded{}, err
	}
	var id int64
	err := s.db.QueryRow(ctx, `insert into admin_audit_entries
		(created_at, actor_id, actor_roles, environment, action, object_type, object_id, before, after,
		 reason, approval_id, ref, change_set, config_version, session_id, source_ip, user_agent)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		returning id`,
		now.UTC(), entry.Actor.ID, entry.Actor.Roles, entry.Environment, entry.Action, entry.Object.Type, entry.Object.ID,
		nullableJSON(entry.Before), nullableJSON(entry.After), entry.Reason, nullableID(entry.ApprovalID), entry.Ref,
		entry.ChangeSet, nullableID(entry.ConfigVersion), entry.Source.SessionID, nullableIP(entry.Source.IP), entry.Source.UserAgent,
	).Scan(&id)
	if err != nil {
		return Recorded{}, err
	}
	return Recorded{Entry: entry, ID: id, At: now.UTC().Truncate(time.Microsecond)}, nil
}

func VisibleObjects(roles []rbac.Role) []rbac.Object {
	if !rbac.Allows(roles, rbac.AuditLog, rbac.View) {
		return nil
	}
	seesPlayers := slices.Contains(roles, rbac.SuperAdmin)
	var visible []rbac.Object
	for _, object := range rbac.Objects() {
		if slices.Contains(playerObjects, object) && !seesPlayers {
			continue
		}
		if rbac.Allows(roles, object, rbac.View) {
			visible = append(visible, object)
		}
	}
	return visible
}

func (s *Store) Search(ctx context.Context, viewer []rbac.Role, filter Filter) ([]Recorded, error) {
	visible := VisibleObjects(viewer)
	if len(visible) == 0 {
		return nil, ErrForbidden
	}
	where, args := filter.conditions(visible)
	args = append(args, filter.limit())
	rows, err := s.db.Query(ctx, `select id, created_at, actor_id, actor_roles, environment, action, object_type, object_id,
		before, after, reason, coalesce(approval_id, 0), ref, change_set, coalesce(config_version, 0), session_id, source_ip, user_agent
		from admin_audit_entries`+where+` order by id desc limit $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, scanRecorded)
}

func (f Filter) conditions(visible []rbac.Object) (string, []any) {
	var clauses []string
	var args []any
	add := func(clause string, value any) {
		args = append(args, value)
		clauses = append(clauses, strings.ReplaceAll(clause, "?", "$"+strconv.Itoa(len(args))))
	}
	add("object_type = any(?)", visible)
	if f.ActorID != "" {
		add("actor_id = ?", f.ActorID)
	}
	if f.Object.Type != "" {
		add("object_type = ?", f.Object.Type)
	}
	if f.Object.ID != "" {
		add("object_id = ?", f.Object.ID)
	}
	if f.Action != "" {
		add("action = ?", f.Action)
	}
	if !f.From.IsZero() {
		add("created_at >= ?", f.From.UTC())
	}
	if !f.To.IsZero() {
		add("created_at < ?", f.To.UTC())
	}
	if f.ReasonContains != "" {
		add("strpos(lower(reason), lower(?)) > 0", f.ReasonContains)
	}
	if f.ConfigVersion > 0 {
		add("config_version = ?", f.ConfigVersion)
	}
	if f.BeforeID > 0 {
		add("id < ?", f.BeforeID)
	}
	return " where " + strings.Join(clauses, " and "), args
}

func (f Filter) limit() int {
	if f.Limit <= 0 {
		return DefaultLimit
	}
	return min(f.Limit, MaxLimit)
}

func scanRecorded(row pgx.CollectableRow) (Recorded, error) {
	var r Recorded
	var before, after []byte
	var ip *netip.Addr
	err := row.Scan(&r.ID, &r.At, &r.Actor.ID, &r.Actor.Roles, &r.Environment, &r.Action, &r.Object.Type, &r.Object.ID,
		&before, &after, &r.Reason, &r.ApprovalID, &r.Ref, &r.ChangeSet, &r.ConfigVersion, &r.Source.SessionID, &ip, &r.Source.UserAgent)
	if err != nil {
		return Recorded{}, err
	}
	r.At = r.At.UTC()
	r.Before, r.After = before, after
	if ip != nil {
		r.Source.IP = *ip
	}
	return r, nil
}

func nullableJSON(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	return string(raw)
}

func nullableID(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

func nullableIP(ip netip.Addr) any {
	if !ip.IsValid() {
		return nil
	}
	return ip
}

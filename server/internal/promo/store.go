package promo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

const (
	SourceType = "promo"
	Topic      = "promo"

	SingleUse = "single_use"
	MultiUse  = "multi_use"
	Targeted  = "targeted"

	StatePending  = "pending_approval"
	StateActive   = "active"
	StateDisabled = "disabled"

	MythicRarity = "mythic"
)

var (
	ErrInvalid         = errors.New("promo.invalid")
	ErrUnknown         = errors.New("promo.unknown")
	ErrNotActive       = errors.New("promo.not_active")
	ErrDisabled        = errors.New("promo.disabled")
	ErrNotStarted      = errors.New("promo.not_started")
	ErrExpired         = errors.New("promo.expired")
	ErrExhausted       = errors.New("promo.exhausted")
	ErrAlreadyRedeemed = errors.New("promo.already_redeemed")
	ErrNotEligible     = errors.New("promo.not_eligible")
	ErrRateLimited     = errors.New("promo.rate_limited")
	ErrLocked          = errors.New("promo.locked")
	ErrDuplicate       = errors.New("promo.duplicate")
	ErrApprover        = errors.New("promo.approver")

	promoID    = regexp.MustCompile(`^promo\.[a-z0-9_]+$`)
	campaignID = regexp.MustCompile(`^campaign\.[a-z0-9_]+$`)
	kinds      = map[string]bool{SingleUse: true, MultiUse: true, Targeted: true}
)

type Config struct {
	CodeLength               int
	MaxValidityDays          int
	AttemptsPerHour          int
	FailedAttemptsBeforeLock int
	LockMinutes              int
	CoinsApprovalThreshold   int
}

func DefaultConfig() Config {
	return Config{CodeLength: 12, MaxValidityDays: 90, AttemptsPerHour: 10, FailedAttemptsBeforeLock: 5, LockMinutes: 60, CoinsApprovalThreshold: 2000}
}

func (c Config) Validate() error {
	switch {
	case c.CodeLength < MinCodeLength || c.CodeLength > MaxCodeLength:
		return fmt.Errorf("promo: code length between %d and %d", MinCodeLength, MaxCodeLength)
	case c.MaxValidityDays < 1 || c.AttemptsPerHour < 1 || c.FailedAttemptsBeforeLock < 1 || c.LockMinutes < 1:
		return errors.New("promo: validity days, attempts per hour, failed attempts before lock and lock minutes must be at least 1")
	case c.CoinsApprovalThreshold < 0:
		return errors.New("promo: approval threshold must not be negative")
	}
	return nil
}

type Definition struct {
	ID       string
	Campaign string
	Kind     string
	Bundle   reward.Bundle
	Start    time.Time
	End      time.Time
	Cap      int
	Code     string
	Accounts []string
	Creator  string
}

type Code struct {
	ID          string        `json:"id"`
	Campaign    string        `json:"campaign"`
	Kind        string        `json:"kind"`
	Bundle      reward.Bundle `json:"bundle"`
	Start       time.Time     `json:"start"`
	End         time.Time     `json:"end"`
	Cap         int           `json:"cap"`
	State       string        `json:"state"`
	Creator     string        `json:"creator"`
	Approver    string        `json:"approver,omitempty"`
	Redemptions int           `json:"redemptions"`
}

type Created struct {
	Code      Code
	Plaintext string
}

type Redemption struct {
	Promo    string        `json:"promo"`
	Campaign string        `json:"campaign"`
	Grant    reward.Report `json:"grant"`
}

type Store struct {
	pool    *pgxpool.Pool
	engine  *reward.Engine
	catalog *cosmetics.Catalog
	hasher  Hasher
	config  Config
}

func NewStore(pool *pgxpool.Pool, engine *reward.Engine, catalog *cosmetics.Catalog, hasher Hasher, config Config) *Store {
	return &Store{pool: pool, engine: engine, catalog: catalog, hasher: hasher, config: config}
}

func (s *Store) validate(d Definition) error {
	fail := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
	}
	switch {
	case !promoID.MatchString(d.ID):
		return fail("id %q", d.ID)
	case !campaignID.MatchString(d.Campaign):
		return fail("campaign %q", d.Campaign)
	case !kinds[d.Kind]:
		return fail("kind %q", d.Kind)
	case d.Creator == "":
		return fail("creator is required")
	case d.Start.IsZero() || !d.End.After(d.Start):
		return fail("window must end after it starts")
	case d.End.Sub(d.Start) > time.Duration(s.config.MaxValidityDays)*24*time.Hour:
		return fail("window longer than %d days", s.config.MaxValidityDays)
	case d.Kind == MultiUse && d.Cap < 1:
		return fail("a multi_use code needs a cap")
	case d.Kind != MultiUse && d.Cap != 0:
		return fail("the cap of a %s code is set by the server", d.Kind)
	case d.Kind != MultiUse && d.Code != "":
		return fail("only a multi_use code may be chosen by the admin")
	case d.Kind == Targeted && len(d.Accounts) == 0:
		return fail("a targeted code lists its accounts")
	case d.Kind != Targeted && len(d.Accounts) > 0:
		return fail("only a targeted code lists accounts")
	}
	if err := d.Bundle.Validate(); err != nil {
		return fail("%v", err)
	}
	if len(d.Bundle.Medals) > 0 {
		return fail("a code never carries a medal")
	}
	seen := make(map[string]bool, len(d.Accounts))
	for _, account := range d.Accounts {
		if account == "" || seen[account] {
			return fail("accounts must be unique and non-empty")
		}
		seen[account] = true
	}
	return nil
}

func (s *Store) needsApproval(bundle reward.Bundle) bool {
	if bundle.Coins > int64(s.config.CoinsApprovalThreshold) {
		return true
	}
	for _, id := range bundle.Cosmetics {
		if item, ok := s.catalog.Item(id); ok && item.Rarity == MythicRarity {
			return true
		}
	}
	return false
}

func (s *Store) cap(d Definition) int {
	switch d.Kind {
	case SingleUse:
		return 1
	case Targeted:
		return len(d.Accounts)
	}
	return d.Cap
}

func (s *Store) plaintext(d Definition) (string, error) {
	if d.Code != "" {
		return Normalize(d.Code)
	}
	return Generate(s.config.CodeLength)
}

func (s *Store) Create(ctx context.Context, d Definition, now time.Time) (Created, error) {
	if err := s.validate(d); err != nil {
		return Created{}, err
	}
	plaintext, err := s.plaintext(d)
	if err != nil {
		return Created{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Created{}, err
	}
	defer tx.Rollback(ctx)
	code, err := s.insert(ctx, tx, d, plaintext, now)
	if err != nil {
		return Created{}, err
	}
	for _, account := range d.Accounts {
		_, err := tx.Exec(ctx, `insert into promo_targets (promo_id, account_id) values ($1, $2)`, d.ID, account)
		if isForeignKeyViolation(err) {
			return Created{}, fmt.Errorf("%w: unknown account %s", ErrInvalid, account)
		}
		if err != nil {
			return Created{}, err
		}
	}
	return Created{Code: code, Plaintext: plaintext}, tx.Commit(ctx)
}

func (s *Store) insert(ctx context.Context, tx pgx.Tx, d Definition, plaintext string, now time.Time) (Code, error) {
	encoded, err := json.Marshal(d.Bundle)
	if err != nil {
		return Code{}, err
	}
	state := StateActive
	if s.needsApproval(d.Bundle) {
		state = StatePending
	}
	code := Code{ID: d.ID, Campaign: d.Campaign, Kind: d.Kind, Bundle: d.Bundle, Start: d.Start.UTC(), End: d.End.UTC(), Cap: s.cap(d), State: state, Creator: d.Creator}
	_, err = tx.Exec(ctx, `insert into promo_codes (promo_id, campaign_id, kind, bundle, code_hash, starts_at, ends_at, cap, state, created_by, created_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		code.ID, code.Campaign, code.Kind, encoded, s.hasher.Hash(plaintext), code.Start, code.End, code.Cap, code.State, code.Creator, now)
	if isUniqueViolation(err) {
		return Code{}, fmt.Errorf("%w: id or code already exists", ErrDuplicate)
	}
	if err != nil {
		return Code{}, err
	}
	return code, nil
}

func (s *Store) Batch(ctx context.Context, d Definition, count int, now time.Time) ([]Created, error) {
	if d.Kind != SingleUse || count < 1 {
		return nil, fmt.Errorf("%w: a batch generates at least one single_use code", ErrInvalid)
	}
	if err := s.validate(d); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(d.Bundle)
	if err != nil {
		return nil, err
	}
	state := StateActive
	if s.needsApproval(d.Bundle) {
		state = StatePending
	}
	created := make([]Created, 0, count)
	rows := make([][]any, 0, count)
	width := len(fmt.Sprint(count))
	for i := 1; i <= count; i++ {
		plaintext, err := Generate(s.config.CodeLength)
		if err != nil {
			return nil, err
		}
		code := Code{ID: fmt.Sprintf("%s_%0*d", d.ID, width, i), Campaign: d.Campaign, Kind: SingleUse, Bundle: d.Bundle, Start: d.Start.UTC(), End: d.End.UTC(), Cap: 1, State: state, Creator: d.Creator}
		created = append(created, Created{Code: code, Plaintext: plaintext})
		rows = append(rows, []any{code.ID, code.Campaign, code.Kind, encoded, s.hasher.Hash(plaintext), code.Start, code.End, code.Cap, code.State, code.Creator, now})
	}
	columns := []string{"promo_id", "campaign_id", "kind", "bundle", "code_hash", "starts_at", "ends_at", "cap", "state", "created_by", "created_at"}
	if _, err := s.pool.CopyFrom(ctx, pgx.Identifier{"promo_codes"}, columns, pgx.CopyFromRows(rows)); err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("%w: id or code already exists", ErrDuplicate)
		}
		return nil, err
	}
	return created, nil
}

func (s *Store) Approve(ctx context.Context, id, approver string, now time.Time) (Code, error) {
	code, err := s.Get(ctx, id)
	if err != nil {
		return Code{}, err
	}
	switch {
	case code.State != StatePending:
		return Code{}, fmt.Errorf("%w: %s is %s", ErrNotActive, id, code.State)
	case approver == "" || approver == code.Creator:
		return Code{}, fmt.Errorf("%w: a second admin approves", ErrApprover)
	}
	if _, err := s.pool.Exec(ctx, `update promo_codes set state = $2, approved_by = $3 where promo_id = $1`, id, StateActive, approver); err != nil {
		return Code{}, err
	}
	code.State, code.Approver = StateActive, approver
	return code, nil
}

func (s *Store) Disable(ctx context.Context, id string, now time.Time) (Code, error) {
	code, err := s.Get(ctx, id)
	if err != nil {
		return Code{}, err
	}
	if _, err := s.pool.Exec(ctx, `update promo_codes set state = $2, disabled_at = $3 where promo_id = $1`, id, StateDisabled, now); err != nil {
		return Code{}, err
	}
	code.State = StateDisabled
	return code, nil
}

const codeColumns = `c.promo_id, c.campaign_id, c.kind, c.bundle, c.starts_at, c.ends_at, c.cap, c.state, c.created_by, c.approved_by,
	(select count(*) from promo_redemptions r where r.promo_id = c.promo_id)`

func scanCode(row pgx.Row) (Code, error) {
	var code Code
	var bundle []byte
	if err := row.Scan(&code.ID, &code.Campaign, &code.Kind, &bundle, &code.Start, &code.End, &code.Cap, &code.State, &code.Creator, &code.Approver, &code.Redemptions); err != nil {
		return Code{}, err
	}
	if err := json.Unmarshal(bundle, &code.Bundle); err != nil {
		return Code{}, err
	}
	return code, nil
}

func (s *Store) Get(ctx context.Context, id string) (Code, error) {
	code, err := scanCode(s.pool.QueryRow(ctx, `select `+codeColumns+` from promo_codes c where c.promo_id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Code{}, ErrUnknown
	}
	return code, err
}

func (s *Store) Redeem(ctx context.Context, account, raw string, now time.Time) (Redemption, error) {
	normalized, err := Normalize(raw)
	if err != nil {
		return Redemption{}, err
	}
	if err := s.throttle(ctx, account, now); err != nil {
		return Redemption{}, err
	}
	redemption, err := s.redeem(ctx, account, s.hasher.Hash(normalized), now)
	if err != nil && !refused(err) {
		return Redemption{}, err
	}
	if err != nil {
		if _, recordErr := s.pool.Exec(ctx, `insert into promo_attempts (account_id, ok, attempted_at) values ($1, false, $2)`, account, now); recordErr != nil {
			return Redemption{}, recordErr
		}
		return Redemption{}, err
	}
	return redemption, nil
}

func refused(err error) bool {
	for _, known := range []error{ErrUnknown, ErrNotActive, ErrDisabled, ErrNotStarted, ErrExpired, ErrExhausted, ErrAlreadyRedeemed, ErrNotEligible} {
		if errors.Is(err, known) {
			return true
		}
	}
	return false
}

func (s *Store) throttle(ctx context.Context, account string, now time.Time) error {
	var attempts, failed int
	hour := now.Add(-time.Hour)
	lock := now.Add(-time.Duration(s.config.LockMinutes) * time.Minute)
	if err := s.pool.QueryRow(ctx, `select count(*) filter (where attempted_at > $2), count(*) filter (where not ok and attempted_at > $3) from promo_attempts where account_id = $1 and attempted_at > least($2, $3)`, account, hour, lock).Scan(&attempts, &failed); err != nil {
		return err
	}
	switch {
	case failed >= s.config.FailedAttemptsBeforeLock:
		return ErrLocked
	case attempts >= s.config.AttemptsPerHour:
		return ErrRateLimited
	}
	return nil
}

func (s *Store) redeem(ctx context.Context, account string, hash []byte, now time.Time) (Redemption, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Redemption{}, err
	}
	defer tx.Rollback(ctx)
	if err := reward.LockIn(ctx, tx, account); err != nil {
		return Redemption{}, err
	}
	code, err := scanCode(tx.QueryRow(ctx, `select `+codeColumns+` from promo_codes c where c.code_hash = $1 for update of c`, hash))
	if errors.Is(err, pgx.ErrNoRows) {
		return Redemption{}, ErrUnknown
	}
	if err != nil {
		return Redemption{}, err
	}
	if err := s.check(ctx, tx, code, account, now); err != nil {
		return Redemption{}, err
	}
	key := "promo:" + code.ID + ":" + account
	source := cosmetics.Source{Type: SourceType, Ref: code.Campaign}
	report, err := s.engine.ApplyIn(ctx, tx, reward.Grant{Account: account, Bundle: code.Bundle, Source: source, Reason: wallet.Promo, Key: key}, now)
	if err != nil {
		return Redemption{}, err
	}
	if _, err := tx.Exec(ctx, `insert into promo_redemptions (promo_id, account_id, grant_key, redeemed_at) values ($1, $2, $3, $4)`, code.ID, account, key, now); err != nil {
		return Redemption{}, err
	}
	if _, err := tx.Exec(ctx, `insert into promo_attempts (account_id, ok, attempted_at) values ($1, true, $2)`, account, now); err != nil {
		return Redemption{}, err
	}
	return Redemption{Promo: code.ID, Campaign: code.Campaign, Grant: report}, tx.Commit(ctx)
}

func (s *Store) check(ctx context.Context, tx pgx.Tx, code Code, account string, now time.Time) error {
	switch {
	case code.State == StateDisabled:
		return ErrDisabled
	case code.State != StateActive:
		return ErrNotActive
	case now.Before(code.Start):
		return ErrNotStarted
	case !now.Before(code.End):
		return ErrExpired
	}
	var redeemed, listed bool
	var redemptions int
	if err := tx.QueryRow(ctx, `select (select count(*) from promo_redemptions where promo_id = $1), exists (select 1 from promo_redemptions where promo_id = $1 and account_id = $2), exists (select 1 from promo_targets where promo_id = $1 and account_id = $2)`, code.ID, account).Scan(&redemptions, &redeemed, &listed); err != nil {
		return err
	}
	switch {
	case redeemed:
		return ErrAlreadyRedeemed
	case code.Kind == Targeted && !listed:
		return ErrNotEligible
	case redemptions >= code.Cap:
		return ErrExhausted
	}
	return nil
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}

func isForeignKeyViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23503")
}

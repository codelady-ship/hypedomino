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
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/pass"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/risk"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

const (
	SourceType = "promo"
	Topic      = "promo"
	SignalLock = "promo_lock"

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
	ErrNoPassSeason    = errors.New("promo.no_pass_season")
	ErrPassOwned       = errors.New("promo.pass_premium_owned")

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
	LockSignalWeight         int
	SpikePercent             int
	SpikeWindowMinutes       int
}

func DefaultConfig() Config {
	return Config{CodeLength: 12, MaxValidityDays: 90, AttemptsPerHour: 10, FailedAttemptsBeforeLock: 5, LockMinutes: 60, CoinsApprovalThreshold: 2000, LockSignalWeight: 5, SpikePercent: 50, SpikeWindowMinutes: 60}
}

func (c Config) Validate() error {
	switch {
	case c.CodeLength < MinCodeLength || c.CodeLength > MaxCodeLength:
		return fmt.Errorf("promo: code length between %d and %d", MinCodeLength, MaxCodeLength)
	case c.MaxValidityDays < 1 || c.AttemptsPerHour < 1 || c.FailedAttemptsBeforeLock < 1 || c.LockMinutes < 1:
		return errors.New("promo: validity days, attempts per hour, failed attempts before lock and lock minutes must be at least 1")
	case c.CoinsApprovalThreshold < 0:
		return errors.New("promo: approval threshold must not be negative")
	case c.LockSignalWeight < 0:
		return errors.New("promo: lock signal weight must not be negative")
	case c.SpikePercent < 1 || c.SpikePercent > 100 || c.SpikeWindowMinutes < 1:
		return errors.New("promo: spike percent between 1 and 100 and spike window at least 1 minute")
	}
	return nil
}

type Definition struct {
	ID          string
	Campaign    string
	Kind        string
	Bundle      reward.Bundle
	Start       time.Time
	End         time.Time
	Cap         int
	Code        string
	Accounts    []string
	Creator     string
	PassPremium bool
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
	Batch       string        `json:"batch,omitempty"`
	Redemptions int           `json:"redemptions"`
	SpikeAt     *time.Time    `json:"spike_at,omitempty"`
	PassPremium bool          `json:"passPremium"`
}

type CampaignUsage struct {
	Campaign    string    `json:"campaign"`
	Codes       int       `json:"codes"`
	Active      int       `json:"active"`
	Pending     int       `json:"pending"`
	Disabled    int       `json:"disabled"`
	Cap         int       `json:"cap"`
	Redemptions int       `json:"redemptions"`
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
}

type Created struct {
	Code      Code
	Plaintext string
}

type Redemption struct {
	Promo       string        `json:"promo"`
	Campaign    string        `json:"campaign"`
	Grant       reward.Report `json:"grant"`
	PassPremium bool          `json:"pass_premium"`
	Spiked      bool          `json:"-"`
}

type database interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
}

type Redeemed struct {
	Account string    `json:"account_id"`
	At      time.Time `json:"redeemed_at"`
}

type TargetChange struct {
	Added   []string
	Removed []string
	Count   int
}

type Pass interface {
	ActiveSeason(at time.Time) (string, bool)
	UnlockPremiumIn(ctx context.Context, tx pgx.Tx, account, season, ref string, now time.Time) (int, error)
}

type Store struct {
	db      database
	engine  *reward.Engine
	catalog *cosmetics.Catalog
	hasher  Hasher
	config  Config
	pass    Pass
}

func NewStore(pool *pgxpool.Pool, engine *reward.Engine, catalog *cosmetics.Catalog, hasher Hasher, config Config) *Store {
	return &Store{db: pool, engine: engine, catalog: catalog, hasher: hasher, config: config}
}

func (s *Store) WithPass(p Pass) {
	s.pass = p
}

func (s *Store) In(tx pgx.Tx) *Store {
	copied := *s
	copied.db = tx
	return &copied
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
	case d.Kind != Targeted && len(d.Accounts) > 0:
		return fail("only a targeted code lists accounts")
	}
	if err := d.Bundle.ValidateFor(wallet.Promo); err != nil {
		return fail("%v", err)
	}
	if len(d.Bundle.Medals) > 0 {
		return fail("a code never carries a medal")
	}
	return uniqueAccounts(d.Accounts)
}

func uniqueAccounts(accounts []string) error {
	seen := make(map[string]bool, len(accounts))
	for _, account := range accounts {
		if account == "" || seen[account] {
			return fmt.Errorf("%w: accounts must be unique and non-empty", ErrInvalid)
		}
		seen[account] = true
	}
	return nil
}

func (s *Store) needsApproval(d Definition) bool {
	if d.PassPremium || d.Bundle.Coins > int64(s.config.CoinsApprovalThreshold) {
		return true
	}
	for _, id := range d.Bundle.Cosmetics {
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
	tx, err := s.db.Begin(ctx)
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
	if s.needsApproval(d) {
		state = StatePending
	}
	code := Code{ID: d.ID, Campaign: d.Campaign, Kind: d.Kind, Bundle: d.Bundle, Start: d.Start.UTC(), End: d.End.UTC(), Cap: s.cap(d), State: state, Creator: d.Creator, PassPremium: d.PassPremium}
	_, err = tx.Exec(ctx, `insert into promo_codes (promo_id, campaign_id, kind, bundle, code_hash, starts_at, ends_at, cap, state, created_by, created_at, pass_premium)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		code.ID, code.Campaign, code.Kind, encoded, s.hasher.Hash(plaintext), code.Start, code.End, code.Cap, code.State, code.Creator, now, code.PassPremium)
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
	if s.needsApproval(d) {
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
		code := Code{ID: fmt.Sprintf("%s_%0*d", d.ID, width, i), Campaign: d.Campaign, Kind: SingleUse, Bundle: d.Bundle, Start: d.Start.UTC(), End: d.End.UTC(), Cap: 1, State: state, Creator: d.Creator, Batch: d.ID, PassPremium: d.PassPremium}
		created = append(created, Created{Code: code, Plaintext: plaintext})
		rows = append(rows, []any{code.ID, code.Campaign, code.Kind, encoded, s.hasher.Hash(plaintext), code.Start, code.End, code.Cap, code.State, code.Creator, now, code.Batch, code.PassPremium})
	}
	columns := []string{"promo_id", "campaign_id", "kind", "bundle", "code_hash", "starts_at", "ends_at", "cap", "state", "created_by", "created_at", "batch_id", "pass_premium"}
	if _, err := s.db.CopyFrom(ctx, pgx.Identifier{"promo_codes"}, columns, pgx.CopyFromRows(rows)); err != nil {
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
	if _, err := s.db.Exec(ctx, `update promo_codes set state = $2, approved_by = $3 where promo_id = $1`, id, StateActive, approver); err != nil {
		return Code{}, err
	}
	code.State, code.Approver = StateActive, approver
	return code, nil
}

func (s *Store) ApproveBatch(ctx context.Context, batch, approver string) (int, error) {
	var creator string
	err := s.db.QueryRow(ctx, `select created_by from promo_codes where batch_id = $1 and batch_id <> '' and state = $2 limit 1`, batch, StatePending).Scan(&creator)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("%w: batch %s has no code pending approval", ErrNotActive, batch)
	}
	if err != nil {
		return 0, err
	}
	if approver == "" || approver == creator {
		return 0, fmt.Errorf("%w: a second admin approves", ErrApprover)
	}
	tag, err := s.db.Exec(ctx, `update promo_codes set state = $3, approved_by = $4 where batch_id = $1 and state = $2`, batch, StatePending, StateActive, approver)
	return int(tag.RowsAffected()), err
}

func (s *Store) DisableBatch(ctx context.Context, batch string, now time.Time) (int, error) {
	if batch == "" {
		return 0, ErrUnknown
	}
	tag, err := s.db.Exec(ctx, `update promo_codes set state = $2, disabled_at = $3 where batch_id = $1 and state <> $2`, batch, StateDisabled, now)
	return int(tag.RowsAffected()), err
}

func (s *Store) Targets(ctx context.Context, id string) ([]string, error) {
	return s.accounts(ctx, `select account_id::text from promo_targets where promo_id = $1 order by account_id`, id)
}

func (s *Store) accounts(ctx context.Context, query, id string) ([]string, error) {
	rows, err := s.db.Query(ctx, query, id)
	if err != nil {
		return nil, err
	}
	accounts, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if accounts == nil {
		accounts = []string{}
	}
	return accounts, err
}

func (s *Store) Redemptions(ctx context.Context, id string) ([]Redeemed, error) {
	rows, err := s.db.Query(ctx, `select account_id::text, redeemed_at from promo_redemptions where promo_id = $1 order by redeemed_at, account_id`, id)
	if err != nil {
		return nil, err
	}
	redeemed, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Redeemed])
	if redeemed == nil {
		redeemed = []Redeemed{}
	}
	return redeemed, err
}

func (s *Store) SetTargets(ctx context.Context, id string, accounts []string) (TargetChange, error) {
	if err := uniqueAccounts(accounts); err != nil {
		return TargetChange{}, err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return TargetChange{}, err
	}
	defer tx.Rollback(ctx)
	var kind, state string
	err = tx.QueryRow(ctx, `select kind, state from promo_codes where promo_id = $1 for update`, id).Scan(&kind, &state)
	if errors.Is(err, pgx.ErrNoRows) {
		return TargetChange{}, ErrUnknown
	}
	switch {
	case err != nil:
		return TargetChange{}, err
	case kind != Targeted:
		return TargetChange{}, fmt.Errorf("%w: only a targeted code lists accounts", ErrInvalid)
	case state == StateDisabled:
		return TargetChange{}, fmt.Errorf("%w: %s is disabled", ErrDisabled, id)
	}
	stored := s.In(tx)
	current, err := stored.Targets(ctx, id)
	if err != nil {
		return TargetChange{}, err
	}
	redeemed, err := stored.accounts(ctx, `select account_id::text from promo_redemptions where promo_id = $1`, id)
	if err != nil {
		return TargetChange{}, err
	}
	change := TargetChange{Added: missingFrom(current, accounts), Removed: missingFrom(accounts, current), Count: len(accounts)}
	if kept := missingFrom(change.Removed, redeemed); len(kept) < len(redeemed) {
		return TargetChange{}, fmt.Errorf("%w: an account that redeemed the code stays listed", ErrInvalid)
	}
	if _, err := tx.Exec(ctx, `delete from promo_targets where promo_id = $1 and account_id::text = any($2)`, id, change.Removed); err != nil {
		return TargetChange{}, err
	}
	for _, account := range change.Added {
		if _, err := tx.Exec(ctx, `insert into promo_targets (promo_id, account_id) values ($1, $2)`, id, account); err != nil {
			if isForeignKeyViolation(err) || isInvalidText(err) {
				return TargetChange{}, fmt.Errorf("%w: unknown account %s", ErrInvalid, account)
			}
			return TargetChange{}, err
		}
	}
	if _, err := tx.Exec(ctx, `update promo_codes set cap = $2 where promo_id = $1`, id, len(accounts)); err != nil {
		return TargetChange{}, err
	}
	return change, tx.Commit(ctx)
}

func missingFrom(present, wanted []string) []string {
	have := make(map[string]bool, len(present))
	for _, account := range present {
		have[account] = true
	}
	missing := []string{}
	for _, account := range wanted {
		if !have[account] {
			missing = append(missing, account)
		}
	}
	return missing
}

type ListQuery struct {
	Campaign string
	State    string
	IDPrefix string
	Before   string
	Limit    int
}

func (s *Store) List(ctx context.Context, q ListQuery) ([]Code, error) {
	rows, err := s.db.Query(ctx, `select `+codeColumns+` from promo_codes c
		where ($1 = '' or c.campaign_id = $1) and ($2 = '' or c.state = $2) and starts_with(c.promo_id, $3)
			and ($4 = '' or (c.created_at, c.promo_id) < (select created_at, promo_id from promo_codes where promo_id = $4))
		order by c.created_at desc, c.promo_id desc limit $5`, q.Campaign, q.State, q.IDPrefix, q.Before, q.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	codes := []Code{}
	for rows.Next() {
		code, err := scanCode(rows)
		if err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	return codes, rows.Err()
}

type CampaignBundle struct {
	Campaign string
	Bundle   reward.Bundle
}

func (s *Store) CampaignBundles(ctx context.Context) ([]CampaignBundle, error) {
	rows, err := s.db.Query(ctx, `select distinct campaign_id, bundle from promo_codes order by campaign_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	found := []CampaignBundle{}
	for rows.Next() {
		var held CampaignBundle
		var bundle []byte
		if err := rows.Scan(&held.Campaign, &bundle); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(bundle, &held.Bundle); err != nil {
			return nil, err
		}
		found = append(found, held)
	}
	return found, rows.Err()
}

func (s *Store) Campaigns(ctx context.Context) ([]CampaignUsage, error) {
	rows, err := s.db.Query(ctx, `select c.campaign_id, count(*), count(*) filter (where c.state = $1), count(*) filter (where c.state = $2), count(*) filter (where c.state = $3),
		sum(c.cap), coalesce(sum(r.redemptions), 0), min(c.starts_at), max(c.ends_at)
		from promo_codes c left join (select promo_id, count(*) redemptions from promo_redemptions group by promo_id) r on r.promo_id = c.promo_id
		group by c.campaign_id order by max(c.created_at) desc, c.campaign_id`, StateActive, StatePending, StateDisabled)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	usage := []CampaignUsage{}
	for rows.Next() {
		var u CampaignUsage
		if err := rows.Scan(&u.Campaign, &u.Codes, &u.Active, &u.Pending, &u.Disabled, &u.Cap, &u.Redemptions, &u.Start, &u.End); err != nil {
			return nil, err
		}
		usage = append(usage, u)
	}
	return usage, rows.Err()
}

func (s *Store) Disable(ctx context.Context, id string, now time.Time) (Code, error) {
	code, err := s.Get(ctx, id)
	if err != nil {
		return Code{}, err
	}
	if _, err := s.db.Exec(ctx, `update promo_codes set state = $2, disabled_at = $3 where promo_id = $1`, id, StateDisabled, now); err != nil {
		return Code{}, err
	}
	code.State = StateDisabled
	return code, nil
}

const codeColumns = `c.promo_id, c.campaign_id, c.kind, c.bundle, c.starts_at, c.ends_at, c.cap, c.state, c.created_by, c.approved_by, c.batch_id, c.spike_at, c.pass_premium,
	(select count(*) from promo_redemptions r where r.promo_id = c.promo_id)`

func scanCode(row pgx.Row) (Code, error) {
	var code Code
	var bundle []byte
	if err := row.Scan(&code.ID, &code.Campaign, &code.Kind, &bundle, &code.Start, &code.End, &code.Cap, &code.State, &code.Creator, &code.Approver, &code.Batch, &code.SpikeAt, &code.PassPremium, &code.Redemptions); err != nil {
		return Code{}, err
	}
	if err := json.Unmarshal(bundle, &code.Bundle); err != nil {
		return Code{}, err
	}
	return code, nil
}

func (s *Store) Get(ctx context.Context, id string) (Code, error) {
	code, err := scanCode(s.db.QueryRow(ctx, `select `+codeColumns+` from promo_codes c where c.promo_id = $1`, id))
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
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Redemption{}, err
	}
	defer tx.Rollback(ctx)
	if err := reward.LockIn(ctx, tx, account); err != nil {
		return Redemption{}, err
	}
	if err := s.throttle(ctx, tx, account, now); err != nil {
		return Redemption{}, err
	}
	attempt, err := tx.Begin(ctx)
	if err != nil {
		return Redemption{}, err
	}
	redemption, err := s.redeem(ctx, attempt, account, s.hasher.Hash(normalized), now)
	if err != nil && !refused(err) {
		return Redemption{}, err
	}
	if err != nil {
		if rollbackErr := attempt.Rollback(ctx); rollbackErr != nil {
			return Redemption{}, rollbackErr
		}
		if _, recordErr := tx.Exec(ctx, `insert into promo_attempts (account_id, ok, attempted_at) values ($1, false, $2)`, account, now); recordErr != nil {
			return Redemption{}, recordErr
		}
		if lockErr := s.flagLock(ctx, tx, account, now); lockErr != nil {
			return Redemption{}, lockErr
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			return Redemption{}, commitErr
		}
		return Redemption{}, err
	}
	if err := attempt.Commit(ctx); err != nil {
		return Redemption{}, err
	}
	return redemption, tx.Commit(ctx)
}

func refused(err error) bool {
	for _, known := range []error{ErrUnknown, ErrNotActive, ErrDisabled, ErrNotStarted, ErrExpired, ErrExhausted, ErrAlreadyRedeemed, ErrNotEligible} {
		if errors.Is(err, known) {
			return true
		}
	}
	return false
}

func (s *Store) throttle(ctx context.Context, tx pgx.Tx, account string, now time.Time) error {
	var attempts, failed int
	hour := now.Add(-time.Hour)
	lock := now.Add(-time.Duration(s.config.LockMinutes) * time.Minute)
	if err := tx.QueryRow(ctx, `select count(*) filter (where attempted_at > $2), count(*) filter (where not ok and attempted_at > $3) from promo_attempts where account_id = $1 and attempted_at > least($2, $3)`, account, hour, lock).Scan(&attempts, &failed); err != nil {
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

func (s *Store) redeem(ctx context.Context, tx pgx.Tx, account string, hash []byte, now time.Time) (Redemption, error) {
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
	if err := s.unlockPremium(ctx, tx, code, account, now); err != nil {
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
	spiked, err := s.markSpike(ctx, tx, code, now)
	if err != nil {
		return Redemption{}, err
	}
	return Redemption{Promo: code.ID, Campaign: code.Campaign, Grant: report, PassPremium: code.PassPremium, Spiked: spiked}, nil
}

func (s *Store) unlockPremium(ctx context.Context, tx pgx.Tx, code Code, account string, now time.Time) error {
	if !code.PassPremium {
		return nil
	}
	if s.pass == nil {
		return ErrNoPassSeason
	}
	season, ok := s.pass.ActiveSeason(now)
	if !ok {
		return ErrNoPassSeason
	}
	_, err := s.pass.UnlockPremiumIn(ctx, tx, account, season, "promo:"+code.ID, now)
	if errors.Is(err, pass.ErrPremiumOwned) {
		return ErrPassOwned
	}
	return err
}

func (s *Store) flagLock(ctx context.Context, tx pgx.Tx, account string, now time.Time) error {
	var failed int
	since := now.Add(-time.Duration(s.config.LockMinutes) * time.Minute)
	if err := tx.QueryRow(ctx, `select count(*) from promo_attempts where account_id = $1 and not ok and attempted_at > $2`, account, since).Scan(&failed); err != nil {
		return err
	}
	if failed != s.config.FailedAttemptsBeforeLock {
		return nil
	}
	signal := risk.Signal{
		Account: account, Kind: SignalLock, Weight: s.config.LockSignalWeight, Dedupe: now.UTC().Format(time.RFC3339Nano), At: now,
		Evidence: map[string]any{"failed": failed, "lock_minutes": s.config.LockMinutes},
	}
	_, err := risk.RecordIn(ctx, tx, signal)
	return err
}

func (s *Store) markSpike(ctx context.Context, tx pgx.Tx, code Code, now time.Time) (bool, error) {
	if code.Kind != MultiUse || code.SpikeAt != nil {
		return false, nil
	}
	since := now.Add(-time.Duration(s.config.SpikeWindowMinutes) * time.Minute)
	tag, err := tx.Exec(ctx, `update promo_codes set spike_at = $3 where promo_id = $1 and spike_at is null
		and (select count(*) from promo_redemptions where promo_id = $1 and redeemed_at > $2) * 100 >= cap * $4`, code.ID, since, now, s.config.SpikePercent)
	return tag.RowsAffected() == 1, err
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

func isInvalidText(err error) bool {
	return err != nil && strings.Contains(err.Error(), "22P02")
}

func isForeignKeyViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23503")
}

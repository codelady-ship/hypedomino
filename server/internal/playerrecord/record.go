package playerrecord

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/profile"
	"github.com/Alievs-corp/hypedomino/server/internal/rating"
	"github.com/Alievs-corp/hypedomino/server/internal/sanction"
)

const (
	ListLimit = 50
)

var (
	ErrNotFound   = errors.New("playerrecord: no such account")
	ErrEmptyQuery = errors.New("playerrecord: empty search")
	likeWildcards = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
)

type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Summary struct {
	ID        string    `json:"id"`
	Nickname  string    `json:"nickname"`
	State     string    `json:"state"`
	Country   string    `json:"country"`
	CreatedAt time.Time `json:"created_at"`
}

type Identity struct {
	Provider string `json:"provider"`
	Subject  string `json:"subject"`
}

type Account struct {
	Summary
	Emails              []string   `json:"emails"`
	Identities          []Identity `json:"identities,omitempty"`
	BannedUntil         *time.Time `json:"banned_until"`
	ShadowbannedAt      *time.Time `json:"shadowbanned_at"`
	MutedUntil          *time.Time `json:"muted_until"`
	DeletionRequestedAt *time.Time `json:"deletion_requested_at"`
	DeletedAt           *time.Time `json:"deleted_at"`
}

type Profile struct {
	RenameRequired bool            `json:"rename_required"`
	Level          int             `json:"level"`
	XP             int64           `json:"xp"`
	Loadout        json.RawMessage `json:"loadout"`
	Stats          json.RawMessage `json:"stats"`
}

type FormerNickname struct {
	Nickname   string    `json:"nickname"`
	ReplacedAt time.Time `json:"replaced_at"`
}

type Item struct {
	ItemID     string     `json:"item_id"`
	SourceType string     `json:"source_type"`
	SourceRef  string     `json:"source_ref"`
	GrantedAt  time.Time  `json:"granted_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
}

type LedgerEntry struct {
	ID            int64     `json:"id"`
	Currency      string    `json:"currency"`
	Amount        int64     `json:"amount"`
	Reason        string    `json:"reason"`
	SourceType    string    `json:"source_type"`
	SourceRef     string    `json:"source_ref"`
	ConfigVersion int64     `json:"config_version"`
	BalanceAfter  int64     `json:"balance_after"`
	At            time.Time `json:"at"`
}

type Grant struct {
	ID            int64      `json:"id"`
	BundleID      string     `json:"bundle_id"`
	Revision      int        `json:"revision"`
	SourceType    string     `json:"source_type"`
	SourceRef     string     `json:"source_ref"`
	ConfigVersion int64      `json:"config_version"`
	At            time.Time  `json:"at"`
	RevokedAt     *time.Time `json:"revoked_at"`
}

type Purchase struct {
	Platform      string     `json:"platform"`
	TransactionID string     `json:"transaction_id"`
	OrderID       string     `json:"order_id"`
	OfferID       string     `json:"offer_id"`
	ProductID     string     `json:"product_id"`
	Environment   string     `json:"environment"`
	State         string     `json:"state"`
	PurchasedAt   time.Time  `json:"purchased_at"`
	GrantedAt     *time.Time `json:"granted_at"`
	RefundedAt    *time.Time `json:"refunded_at"`
}

type Match struct {
	MatchID    string     `json:"match_id"`
	StartedAt  time.Time  `json:"started_at"`
	EndedAt    *time.Time `json:"ended_at"`
	Mode       string     `json:"mode"`
	Kind       string     `json:"kind"`
	Seat       int        `json:"seat"`
	Team       int        `json:"team"`
	Result     string     `json:"result"`
	ScoreA     int        `json:"score_a"`
	ScoreB     int        `json:"score_b"`
	Cancelled  bool       `json:"cancelled"`
	Suspicious []string   `json:"suspicious"`
}

type PairSignal struct {
	Kind        string    `json:"kind"`
	Counterpart string    `json:"counterpart"`
	Mode        string    `json:"mode"`
	Matches     int       `json:"matches"`
	Wins        int       `json:"wins"`
	Offenses    int       `json:"offenses"`
	WindowHours int       `json:"window_hours"`
	FirstSeen   time.Time `json:"first_seen"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type MissionAssignment struct {
	MissionID   string     `json:"mission_id"`
	Type        string     `json:"type"`
	Slot        int        `json:"slot"`
	Period      string     `json:"period"`
	Progress    int64      `json:"progress"`
	Target      int64      `json:"target"`
	AssignedAt  time.Time  `json:"assigned_at"`
	CompletedAt *time.Time `json:"completed_at"`
	ReplacedAt  *time.Time `json:"replaced_at"`
}

type Report struct {
	ID        int64      `json:"id"`
	Reporter  string     `json:"reporter"`
	Target    string     `json:"target"`
	Reason    string     `json:"reason"`
	NameText  string     `json:"name_text"`
	MatchID   *string    `json:"match_id"`
	State     string     `json:"state"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DecidedAt *time.Time `json:"decided_at"`
}

type Record struct {
	Account         Account             `json:"account"`
	Profile         *Profile            `json:"profile"`
	NicknameHistory []FormerNickname    `json:"nickname_history"`
	Inventory       []Item              `json:"inventory"`
	Coins           int64               `json:"coins"`
	Crystals        int64               `json:"crystals"`
	Ledger          []LedgerEntry       `json:"ledger"`
	Grants          []Grant             `json:"grants"`
	Purchases       []Purchase          `json:"purchases"`
	Matches         []Match             `json:"matches"`
	PairSignals     []PairSignal        `json:"pair_signals"`
	Missions        []MissionAssignment `json:"missions"`
	Reports         []Report            `json:"reports"`
	Sanctions       []sanction.Sanction `json:"sanctions"`
}

const summaryColumns = `select a.id::text, coalesce(p.nickname, ''), a.state, coalesce(a.country, ''), a.created_at
	from accounts a left join profiles p on p.account_id = a.id `

type SearchQuery struct {
	Text  string
	State string
	After string
	Limit int
}

func Search(ctx context.Context, db Querier, q SearchQuery) ([]Summary, error) {
	text := strings.TrimSpace(q.Text)
	var rows pgx.Rows
	var err error
	switch {
	case text == "" && q.State == "":
		return nil, ErrEmptyQuery
	case text == "":
		rows, err = db.Query(ctx, summaryColumns+`where a.state = $1
			and ($2 = '' or (a.created_at, a.id) < (select created_at, id from accounts where id = nullif($2, '')::uuid))
			order by a.created_at desc, a.id desc limit $3`, q.State, q.After, q.Limit)
	case isAccountID(text):
		rows, err = db.Query(ctx, summaryColumns+`where a.id = $1::uuid and ($2 = '' or a.state = $2) and $3 = ''`, text, q.State, q.After)
	case strings.Contains(text, "@"):
		rows, err = db.Query(ctx, summaryColumns+`where a.id in (select account_id from auth_identities where lower(email) = lower($1))
			and ($2 = '' or a.state = $2)
			and ($3 = '' or (a.created_at, a.id) > (select created_at, id from accounts where id = nullif($3, '')::uuid))
			order by a.created_at, a.id limit $4`, text, q.State, q.After, q.Limit)
	default:
		rows, err = db.Query(ctx, summaryColumns+`where p.nickname_key like $1 escape '\' and ($2 = '' or a.state = $2)
			and ($3 = '' or p.nickname_key > (select nickname_key from profiles where account_id = nullif($3, '')::uuid))
			order by p.nickname_key limit $4`,
			likeWildcards.Replace(profile.Key(text))+"%", q.State, q.After, q.Limit)
	}
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Summary, error) {
		var s Summary
		err := row.Scan(&s.ID, &s.Nickname, &s.State, &s.Country, &s.CreatedAt)
		return s, err
	})
}

func SearchIdentity(ctx context.Context, db Querier, provider, subject string) ([]Summary, error) {
	rows, err := db.Query(ctx, summaryColumns+`where a.id in (select account_id from auth_identities where provider = $1 and subject = $2)`, provider, subject)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Summary, error) {
		var s Summary
		err := row.Scan(&s.ID, &s.Nickname, &s.State, &s.Country, &s.CreatedAt)
		return s, err
	})
}

func isAccountID(query string) bool {
	if len(query) != 36 {
		return false
	}
	for i, r := range query {
		dash := i == 8 || i == 13 || i == 18 || i == 23
		if dash != (r == '-') || (!dash && !strings.ContainsRune("0123456789abcdefABCDEF", r)) {
			return false
		}
	}
	return true
}

func Lock(ctx context.Context, tx pgx.Tx, accountID string) error {
	if !isAccountID(accountID) {
		return ErrNotFound
	}
	var state string
	err := tx.QueryRow(ctx, `select state from accounts where id = $1::uuid for update`, accountID).Scan(&state)
	if errors.Is(err, pgx.ErrNoRows) || state == "deleted" {
		return ErrNotFound
	}
	return err
}

func Load(ctx context.Context, db Querier, accountID string, withIdentities bool) (Record, error) {
	if !isAccountID(accountID) {
		return Record{}, ErrNotFound
	}
	var r Record
	err := db.QueryRow(ctx, `select a.id::text, coalesce(p.nickname, ''), a.state, coalesce(a.country, ''), a.created_at,
		a.banned_until, a.shadowbanned_at, a.muted_until, a.deletion_requested_at, a.deleted_at, coalesce(c.balance, 0), coalesce(k.balance, 0)
		from accounts a left join profiles p on p.account_id = a.id
		left join wallets c on c.account_id = a.id and c.currency = 'coin'
		left join wallets k on k.account_id = a.id and k.currency = 'crystal'
		where a.id = $1::uuid`, accountID).Scan(&r.Account.ID, &r.Account.Nickname, &r.Account.State, &r.Account.Country,
		&r.Account.CreatedAt, &r.Account.BannedUntil, &r.Account.ShadowbannedAt, &r.Account.MutedUntil, &r.Account.DeletionRequestedAt, &r.Account.DeletedAt, &r.Coins, &r.Crystals)
	if errors.Is(err, pgx.ErrNoRows) {
		return Record{}, ErrNotFound
	}
	if err != nil {
		return Record{}, err
	}
	if err := loadIdentities(ctx, db, &r, withIdentities); err != nil {
		return Record{}, err
	}
	for _, load := range []func(context.Context, Querier, *Record) error{loadProfile, loadNicknames, loadInventory, loadLedger, loadGrants, loadPurchases, loadMatches, loadPairSignals, loadMissions, loadReports, loadSanctions} {
		if err := load(ctx, db, &r); err != nil {
			return Record{}, err
		}
	}
	return r, nil
}

func loadSanctions(ctx context.Context, db Querier, r *Record) error {
	sanctions, err := sanction.List(ctx, db, r.Account.ID, ListLimit)
	r.Sanctions = sanctions
	return err
}

func collect[T any](ctx context.Context, db Querier, into *[]T, sql string, scan func(row pgx.CollectableRow, item *T) error, args ...any) error {
	rows, err := db.Query(ctx, sql, args...)
	if err != nil {
		return err
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (T, error) {
		var item T
		err := scan(row, &item)
		return item, err
	})
	if err != nil {
		return err
	}
	*into = items
	return nil
}

type identityRow struct {
	Identity
	email string
}

func loadIdentities(ctx context.Context, db Querier, r *Record, withIdentities bool) error {
	var identities []identityRow
	err := collect(ctx, db, &identities, `select provider, subject, coalesce(email, '') from auth_identities where account_id = $1::uuid order by created_at`,
		func(row pgx.CollectableRow, i *identityRow) error { return row.Scan(&i.Provider, &i.Subject, &i.email) }, r.Account.ID)
	if err != nil {
		return err
	}
	r.Account.Emails = []string{}
	for _, i := range identities {
		if i.email != "" {
			r.Account.Emails = append(r.Account.Emails, i.email)
		}
		if withIdentities {
			r.Account.Identities = append(r.Account.Identities, i.Identity)
		}
	}
	return nil
}

func loadProfile(ctx context.Context, db Querier, r *Record) error {
	var p Profile
	err := db.QueryRow(ctx, `select rename_required, loadout, stats from profiles where account_id = $1::uuid`, r.Account.ID).Scan(&p.RenameRequired, &p.Loadout, &p.Stats)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err == nil {
		r.Profile = &p
	}
	return err
}

func loadNicknames(ctx context.Context, db Querier, r *Record) error {
	return collect(ctx, db, &r.NicknameHistory, `select nickname, replaced_at from nickname_history where account_id = $1::uuid order by replaced_at desc, id desc limit $2`,
		func(row pgx.CollectableRow, n *FormerNickname) error { return row.Scan(&n.Nickname, &n.ReplacedAt) }, r.Account.ID, ListLimit)
}

func loadInventory(ctx context.Context, db Querier, r *Record) error {
	return collect(ctx, db, &r.Inventory, `select item_id, source_type, source_ref, granted_at, revoked_at from inventory_items where account_id = $1::uuid order by granted_at desc, item_id`,
		func(row pgx.CollectableRow, i *Item) error {
			return row.Scan(&i.ItemID, &i.SourceType, &i.SourceRef, &i.GrantedAt, &i.RevokedAt)
		}, r.Account.ID)
}

func loadLedger(ctx context.Context, db Querier, r *Record) error {
	return collect(ctx, db, &r.Ledger, `select id, currency, amount, reason, source_type, source_ref, config_version, balance_after, created_at
		from ledger_entries where account_id = $1::uuid order by created_at desc, id desc limit $2`,
		func(row pgx.CollectableRow, e *LedgerEntry) error {
			return row.Scan(&e.ID, &e.Currency, &e.Amount, &e.Reason, &e.SourceType, &e.SourceRef, &e.ConfigVersion, &e.BalanceAfter, &e.At)
		}, r.Account.ID, ListLimit)
}

func loadGrants(ctx context.Context, db Querier, r *Record) error {
	return collect(ctx, db, &r.Grants, `select id, bundle_id, revision, source_type, source_ref, config_version, created_at, revoked_at
		from reward_grants where account_id = $1::uuid order by created_at desc, id desc limit $2`,
		func(row pgx.CollectableRow, g *Grant) error {
			return row.Scan(&g.ID, &g.BundleID, &g.Revision, &g.SourceType, &g.SourceRef, &g.ConfigVersion, &g.At, &g.RevokedAt)
		}, r.Account.ID, ListLimit)
}

func loadPurchases(ctx context.Context, db Querier, r *Record) error {
	return collect(ctx, db, &r.Purchases, `select platform, transaction_id, order_id, offer_id, product_id, environment, state, purchased_at, granted_at, refunded_at
		from purchases where account_id = $1::uuid order by created_at desc, id desc limit $2`,
		func(row pgx.CollectableRow, p *Purchase) error {
			return row.Scan(&p.Platform, &p.TransactionID, &p.OrderID, &p.OfferID, &p.ProductID, &p.Environment, &p.State, &p.PurchasedAt, &p.GrantedAt, &p.RefundedAt)
		}, r.Account.ID, ListLimit)
}

func loadMatches(ctx context.Context, db Querier, r *Record) error {
	return collect(ctx, db, &r.Matches, `select m.id::text, m.started_at, m.ended_at, m.mode, m.kind, mp.seat, mp.team, coalesce(mp.result, ''), m.score_a, m.score_b, m.cancelled,
		array(select distinct s.kind from rating_signals s join match_participants o on o.match_id = m.id and o.started_at = m.started_at and o.account_id = s.counterpart
			where s.account_id = mp.account_id and s.mode = m.mode and s.kind = any($3) and m.started_at > s.updated_at - make_interval(hours => s.window_hours) order by s.kind)
		from match_participants mp join matches m on m.id = mp.match_id and m.started_at = mp.started_at
		where mp.account_id = $1::uuid order by mp.started_at desc limit $2`,
		func(row pgx.CollectableRow, m *Match) error {
			return row.Scan(&m.MatchID, &m.StartedAt, &m.EndedAt, &m.Mode, &m.Kind, &m.Seat, &m.Team, &m.Result, &m.ScoreA, &m.ScoreB, &m.Cancelled, &m.Suspicious)
		}, r.Account.ID, ListLimit, rating.CollusionSignals)
}

func loadPairSignals(ctx context.Context, db Querier, r *Record) error {
	return collect(ctx, db, &r.PairSignals, `select kind, counterpart::text, mode, matches, wins, offenses, window_hours, first_seen, updated_at
		from rating_signals where account_id = $1::uuid order by updated_at desc, kind, counterpart limit $2`,
		func(row pgx.CollectableRow, s *PairSignal) error {
			return row.Scan(&s.Kind, &s.Counterpart, &s.Mode, &s.Matches, &s.Wins, &s.Offenses, &s.WindowHours, &s.FirstSeen, &s.UpdatedAt)
		}, r.Account.ID, ListLimit)
}

func loadMissions(ctx context.Context, db Querier, r *Record) error {
	return collect(ctx, db, &r.Missions, `select mission_id, mission_type, slot, period, progress, assigned_at, completed_at, replaced_at
		from mission_assignments where account_id = $1::uuid order by assigned_at desc, mission_type, slot, mission_id limit $2`,
		func(row pgx.CollectableRow, m *MissionAssignment) error {
			return row.Scan(&m.MissionID, &m.Type, &m.Slot, &m.Period, &m.Progress, &m.AssignedAt, &m.CompletedAt, &m.ReplacedAt)
		}, r.Account.ID, ListLimit)
}

func loadReports(ctx context.Context, db Querier, r *Record) error {
	return collect(ctx, db, &r.Reports, `select id, reporter::text, target::text, reason, name_text, match_id::text, state, created_at, updated_at, decided_at
		from player_reports where target = $1::uuid or reporter = $1::uuid order by updated_at desc, id desc limit $2`,
		func(row pgx.CollectableRow, p *Report) error {
			return row.Scan(&p.ID, &p.Reporter, &p.Target, &p.Reason, &p.NameText, &p.MatchID, &p.State, &p.CreatedAt, &p.UpdatedAt, &p.DecidedAt)
		}, r.Account.ID, ListLimit)
}

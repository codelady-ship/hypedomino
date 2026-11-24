package shop

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/inventory"
	"github.com/Alievs-corp/hypedomino/server/internal/pass"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/secrand"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

var (
	ErrNoCatalog       = errors.New("shop.unavailable")
	ErrUnknownOffer    = errors.New("offer.unknown")
	ErrUnavailable     = errors.New("offer.unavailable")
	ErrLimit           = errors.New("offer.limit_reached")
	ErrNotEligible     = errors.New("offer.not_eligible")
	ErrOwned           = errors.New("offer.owned")
	ErrStoreOnly       = errors.New("offer.store_only")
	ErrInGameOnly      = errors.New("offer.in_game_only")
	ErrUnsupported     = errors.New("offer.unsupported")
	ErrRequest         = errors.New("request.invalid")
	ErrBlocked         = errors.New("purchase.blocked")
	ErrPurchasesClosed = errors.New("purchases.disabled")
)

type Gate interface {
	Blocked(ctx context.Context, account string, now time.Time) (bool, error)
}

const MaxRequestID = 64

var accountTokenSpace = uuid.MustParse("5f0d6a2e-3c7b-4e1a-9d8f-2b6c4a1e7f30")

type Store struct {
	pool      *pgxpool.Pool
	wallet    *wallet.Store
	inventory *inventory.Store
	engine    *reward.Engine
	catalog   *snapshot.Watcher[*Catalog]
	config    Config
	gate      Gate
	pass      PassStatus
	random    Random
}

func (s *Store) Guard(gate Gate) {
	s.gate = gate
}

type PassStatus interface {
	Premium(ctx context.Context, account string, now time.Time) (pass.PremiumState, error)
}

func (s *Store) WithPass(status PassStatus) {
	s.pass = status
}

func (s *Store) WithRandom(random Random) {
	s.random = random
}

func NewStore(pool *pgxpool.Pool, wallet *wallet.Store, inventory *inventory.Store, engine *reward.Engine, catalog *snapshot.Watcher[*Catalog], config Config) *Store {
	return &Store{pool: pool, wallet: wallet, inventory: inventory, engine: engine, catalog: catalog, config: config, random: secrand.Source{}}
}

type OfferView struct {
	ID                     string           `json:"id"`
	Kind                   string           `json:"kind"`
	NameKey                string           `json:"name_key"`
	Variant                string           `json:"variant,omitempty"`
	PriceCoins             int64            `json:"price_coins,omitempty"`
	PriceCrystals          int64            `json:"price_crystals,omitempty"`
	ProductID              string           `json:"product_id,omitempty"`
	ReferencePriceCoins    int64            `json:"reference_price_coins,omitempty"`
	ReferencePriceCrystals int64            `json:"reference_price_crystals,omitempty"`
	Coins                  int64            `json:"coins,omitempty"`
	Crystals               int64            `json:"crystals,omitempty"`
	Cosmetics              []string         `json:"cosmetics"`
	PassXP                 int              `json:"pass_xp,omitempty"`
	EndsAt                 string           `json:"ends_at,omitempty"`
	Limit                  int              `json:"limit,omitempty"`
	Bought                 int              `json:"bought"`
	Owned                  []string         `json:"owned"`
	Compensation           map[string]int64 `json:"compensation"`
	Odds                   []ChestOdds      `json:"odds,omitempty"`
	OpensToPity            int              `json:"opens_to_pity,omitempty"`
	Purchasable            bool             `json:"purchasable"`
	Reason                 string           `json:"reason,omitempty"`
}

type SectionView struct {
	Section string      `json:"section"`
	EndsAt  string      `json:"ends_at,omitempty"`
	Offers  []OfferView `json:"offers"`
}

type View struct {
	Version  int64         `json:"version"`
	Sections []SectionView `json:"sections"`
}

type Purchase struct {
	Offer    string          `json:"offer"`
	Currency wallet.Currency `json:"currency"`
	Balance  int64           `json:"balance"`
	Grant    reward.Report   `json:"grant"`
	Chest    *ChestOpen      `json:"chest,omitempty"`
	Repeated bool            `json:"repeated"`
}

type Checkout struct {
	Offer        string `json:"offer"`
	ProductID    string `json:"product_id"`
	AccountToken string `json:"account_token"`
}

func AccountToken(account string) string {
	return uuid.NewSHA1(accountTokenSpace, []byte(account)).String()
}

type standing struct {
	created   time.Time
	blocked   bool
	purchases int
	counts    map[string]int
	owns      func(id string) (bool, error)
	premium   *pass.PremiumState
	noSeason  bool
	pity      int
}

func (s *Store) standing(ctx context.Context, q wallet.Querier, account string, owns func(id string) (bool, error), now time.Time) (standing, error) {
	st := standing{owns: owns}
	if err := q.QueryRow(ctx, `select created_at from accounts where id = $1`, account).Scan(&st.created); err != nil {
		return st, err
	}
	purchases, err := s.wallet.CountIn(ctx, q, account, wallet.Purchase)
	if err != nil {
		return st, err
	}
	st.purchases = purchases
	if s.gate != nil {
		if st.blocked, err = s.gate.Blocked(ctx, account, now); err != nil {
			return st, err
		}
	}
	if s.pass != nil {
		premium, err := s.pass.Premium(ctx, account, now)
		switch {
		case errors.Is(err, pass.ErrNoSeason):
			st.noSeason = true
		case err != nil:
			return st, err
		default:
			st.premium = &premium
		}
	}
	if st.pity, err = pityIn(ctx, q, account); err != nil {
		return st, err
	}
	st.counts, err = reward.CountsIn(ctx, q, account, SourceType)
	return st, err
}

type status struct {
	owned        []string
	compensation map[string]int64
	bought       int
	refusal      error
}

func (s *Store) status(offer Offer, st standing, now time.Time) (status, error) {
	if offer.Kind == KindChest {
		return s.chestStatus(offer, st)
	}
	out := status{owned: []string{}, compensation: map[string]int64{}, bought: st.counts[offer.ID]}
	for _, id := range offer.Bundle.Cosmetics {
		held, err := st.owns(id)
		if err != nil {
			return out, err
		}
		if !held {
			continue
		}
		out.owned = append(out.owned, id)
		item, _ := s.inventory.Catalog().Item(id)
		out.compensation[id] = s.engine.Config().Compensation(item)
	}
	age := now.Sub(st.created)
	switch {
	case offer.Kind == KindPassLevels || (offer.Kind == KindPass && st.premium == nil && !st.noSeason):
		out.refusal = ErrUnsupported
	case offer.Kind == KindPass && st.noSeason:
		out.refusal = ErrNotEligible
	case offer.Kind == KindPass && st.premium.Premium:
		out.refusal = ErrOwned
	case offer.Kind == KindPass && offer.ProductID == ProductPassPremiumPlus && st.premium.Level > st.premium.Levels-st.premium.PlusLevels:
		out.refusal = ErrNotEligible
	case offer.ProductID != "" && st.blocked:
		out.refusal = ErrBlocked
	case offer.LimitPerAccount > 0 && out.bought >= offer.LimitPerAccount:
		out.refusal = ErrLimit
	case offer.Eligibility.MaxAccountAgeHours > 0 && age > time.Duration(offer.Eligibility.MaxAccountAgeHours)*time.Hour:
		out.refusal = ErrNotEligible
	case offer.Eligibility.FirstPurchaseOnly && st.purchases > 0:
		out.refusal = ErrNotEligible
	case offer.Bundle.Coins == 0 && offer.Bundle.Crystals == 0 && offer.Bundle.PassXP == 0 && len(offer.Bundle.Cosmetics) > 0 && len(out.owned) == len(offer.Bundle.Cosmetics):
		out.refusal = ErrOwned
	}
	return out, nil
}

func (s *Store) View(ctx context.Context, account string, now time.Time) (View, error) {
	catalog, version, ok := s.catalog.Current()
	if !ok {
		return View{}, ErrNoCatalog
	}
	owned, err := s.inventory.Owned(ctx, account)
	if err != nil {
		return View{}, err
	}
	held := make(map[string]bool, len(owned))
	for _, o := range owned {
		held[o.ID] = true
	}
	st, err := s.standing(ctx, s.pool, account, func(id string) (bool, error) { return held[id], nil }, now)
	if err != nil {
		return View{}, err
	}
	view := View{Version: version, Sections: make([]SectionView, 0, len(Sections))}
	for _, section := range catalog.Sections(s.config, now) {
		section = section.For(account)
		sv := SectionView{Section: section.Name, EndsAt: stamp(section.EndsAt), Offers: make([]OfferView, 0, len(section.Offers))}
		for _, offer := range section.Offers {
			ov, err := s.offerView(offer, catalog.Chests, st, now)
			if err != nil {
				return View{}, err
			}
			if variant, ok := offer.VariantFor(account); ok {
				ov.Variant = variant.ID
			}
			sv.Offers = append(sv.Offers, ov)
		}
		view.Sections = append(view.Sections, sv)
	}
	return view, nil
}

func (s *Store) offerView(offer Offer, chests ChestRules, st standing, now time.Time) (OfferView, error) {
	status, err := s.status(offer, st, now)
	if err != nil {
		return OfferView{}, err
	}
	view := OfferView{
		ID: offer.ID, Kind: offer.Kind, NameKey: offer.NameKey,
		PriceCoins: offer.PriceCoins, PriceCrystals: offer.PriceCrystals, ProductID: offer.ProductID,
		ReferencePriceCoins: offer.ReferencePriceCoins, ReferencePriceCrystals: offer.ReferencePriceCrystals,
		Coins: offer.Bundle.Coins, Crystals: offer.Bundle.Crystals, Cosmetics: append([]string{}, offer.Bundle.Cosmetics...), PassXP: offer.Bundle.PassXP,
		EndsAt: stamp(offer.Window.End), Limit: offer.LimitPerAccount, Bought: status.bought,
		Owned: status.owned, Compensation: status.compensation, Purchasable: status.refusal == nil,
	}
	if status.refusal != nil {
		view.Reason = status.refusal.Error()
	}
	if offer.Kind == KindChest {
		view.Odds = chests.Odds()
		view.OpensToPity = chests.OpensToPity(st.pity)
	}
	return view, nil
}

func (s *Store) offer(id string, now time.Time) (Offer, *Catalog, int64, error) {
	catalog, version, ok := s.catalog.Current()
	if !ok {
		return Offer{}, nil, 0, ErrNoCatalog
	}
	offer, ok := catalog.Offer(id)
	if !ok {
		return Offer{}, nil, 0, ErrUnknownOffer
	}
	if !catalog.Available(s.config, offer, now) {
		return Offer{}, nil, 0, ErrUnavailable
	}
	return offer, catalog, version, nil
}

func (s *Store) Buy(ctx context.Context, account, offerID, requestID string, now time.Time) (Purchase, error) {
	if requestID == "" || len(requestID) > MaxRequestID {
		return Purchase{}, ErrRequest
	}
	offer, catalog, version, err := s.offer(offerID, now)
	if err != nil {
		return Purchase{}, err
	}
	if offer.ProductID != "" {
		return Purchase{}, ErrStoreOnly
	}
	if offer.Kind == KindChest {
		return s.open(ctx, account, offer, catalog.Chests, version, requestID, now)
	}
	currency, price := offer.Price()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Purchase{}, err
	}
	defer tx.Rollback(ctx)
	if err := reward.LockIn(ctx, tx, account); err != nil {
		return Purchase{}, err
	}
	key := "shop:" + requestID
	earlier, err := reward.GetIn(ctx, tx, account, key)
	if err == nil {
		balance, err := s.wallet.BalanceIn(ctx, tx, account, currency)
		if err != nil {
			return Purchase{}, err
		}
		return Purchase{Offer: earlier.Source.Ref, Currency: currency, Balance: balance, Grant: earlier, Repeated: true}, tx.Commit(ctx)
	}
	if !errors.Is(err, reward.ErrUnknownGrant) {
		return Purchase{}, err
	}
	st, err := s.standing(ctx, tx, account, func(id string) (bool, error) { return s.inventory.OwnsIn(ctx, tx, account, id) }, now)
	if err != nil {
		return Purchase{}, err
	}
	status, err := s.status(offer, st, now)
	if err != nil {
		return Purchase{}, err
	}
	if status.refusal != nil {
		return Purchase{}, status.refusal
	}
	source := cosmetics.Source{Type: SourceType, Ref: offer.ID}
	spend := wallet.Entry{Account: account, Currency: currency, Amount: -price, Reason: wallet.ShopPurchase, Source: wallet.Source{Type: source.Type, Ref: source.Ref}, Key: key + ":spend", ConfigVersion: int(version)}
	applied, err := s.wallet.ApplyIn(ctx, tx, spend, now)
	if err != nil {
		return Purchase{}, err
	}
	grant := reward.Grant{Account: account, Bundle: offer.Bundle, Source: source, Reason: wallet.ShopPurchase, Key: key, ConfigVersion: int(version)}
	report, err := s.engine.ApplyIn(ctx, tx, grant, now)
	if err != nil {
		return Purchase{}, err
	}
	return Purchase{Offer: offer.ID, Currency: currency, Balance: applied.BalanceAfter, Grant: report}, tx.Commit(ctx)
}

func (s *Store) Begin(ctx context.Context, account, offerID string, now time.Time) (Checkout, error) {
	offer, _, _, err := s.offer(offerID, now)
	if err != nil {
		return Checkout{}, err
	}
	if offer.ProductID == "" {
		return Checkout{}, ErrInGameOnly
	}
	st, err := s.standing(ctx, s.pool, account, func(id string) (bool, error) { return s.inventory.Owns(ctx, account, id) }, now)
	if err != nil {
		return Checkout{}, err
	}
	status, err := s.status(offer, st, now)
	if err != nil {
		return Checkout{}, err
	}
	if status.refusal != nil {
		return Checkout{}, status.refusal
	}
	return Checkout{Offer: offer.ID, ProductID: offer.ProductID, AccountToken: AccountToken(account)}, nil
}

func stamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

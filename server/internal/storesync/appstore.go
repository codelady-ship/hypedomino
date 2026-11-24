package storesync

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"maps"
	"math"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/storeprice"
)

const (
	DefaultAppleEndpoint = "https://api.appstoreconnect.apple.com"
	AppleAudience        = "appstoreconnect-v1"
	AppleTokenLife       = 15 * time.Minute
	baseStorefront       = "USA"
)

type AppleConfig struct {
	IssuerID       string
	KeyID          string
	PrivateKeyFile string
	AppID          string
	Endpoint       string
	Client         *http.Client
	Now            func() time.Time
	Pause          time.Duration
}

type Apple struct {
	endpoint string
	app      string
	issuer   string
	keyID    string
	key      *ecdsa.PrivateKey
	now      func() time.Time
	caller   caller
	mu       sync.Mutex
	token    string
	expires  time.Time
}

func NewApple(config AppleConfig) (*Apple, error) {
	if config.IssuerID == "" || config.KeyID == "" || config.PrivateKeyFile == "" || config.AppID == "" {
		return nil, errors.New("storesync: issuer id, key id, private key and app id are required")
	}
	key, err := loadAppleKey(config.PrivateKeyFile)
	if err != nil {
		return nil, err
	}
	if config.Endpoint == "" {
		config.Endpoint = DefaultAppleEndpoint
	}
	if config.Client == nil {
		config.Client = &http.Client{Timeout: 20 * time.Second}
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.Pause == 0 {
		config.Pause = DefaultPause
	}
	a := &Apple{endpoint: config.Endpoint, app: config.AppID, issuer: config.IssuerID, keyID: config.KeyID, key: key, now: config.Now}
	a.caller = caller{client: config.Client, authorize: a.bearer, pause: config.Pause}
	return a, nil
}

func loadAppleKey(file string) (*ecdsa.PrivateKey, error) {
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, errors.New("storesync: the App Store Connect key is not PEM")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("storesync: the App Store Connect key: %w", err)
	}
	key, ok := parsed.(*ecdsa.PrivateKey)
	if !ok || key.Curve != elliptic.P256() {
		return nil, errors.New("storesync: the App Store Connect key is not a P-256 key")
	}
	return key, nil
}

func (a *Apple) bearer(context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := a.now()
	if a.token != "" && now.Add(time.Minute).Before(a.expires) {
		return "Bearer " + a.token, nil
	}
	token, err := a.sign(now)
	if err != nil {
		return "", err
	}
	a.token, a.expires = token, now.Add(AppleTokenLife)
	return "Bearer " + token, nil
}

func (a *Apple) sign(now time.Time) (string, error) {
	header, err := json.Marshal(map[string]string{"alg": "ES256", "kid": a.keyID, "typ": "JWT"})
	if err != nil {
		return "", err
	}
	claims, err := json.Marshal(map[string]any{"iss": a.issuer, "iat": now.Unix(), "exp": now.Add(AppleTokenLife).Unix(), "aud": AppleAudience})
	if err != nil {
		return "", err
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(claims)
	digest := sha256.Sum256([]byte(input))
	r, s, err := ecdsa.Sign(rand.Reader, a.key, digest[:])
	if err != nil {
		return "", err
	}
	signature := make([]byte, 64)
	r.FillBytes(signature[:32])
	s.FillBytes(signature[32:])
	return input + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

type resource struct {
	Type       string `json:"type"`
	ID         string `json:"id"`
	Attributes struct {
		ProductID     string  `json:"productId"`
		CustomerPrice string  `json:"customerPrice"`
		Currency      string  `json:"currency"`
		EndDate       *string `json:"endDate"`
	} `json:"attributes"`
	Relationships map[string]struct {
		Data json.RawMessage `json:"data"`
	} `json:"relationships"`
}

func (r resource) related(name string) string {
	var link struct {
		ID string `json:"id"`
	}
	json.Unmarshal(r.Relationships[name].Data, &link)
	return link.ID
}

type document struct {
	Data     json.RawMessage `json:"data"`
	Included []resource      `json:"included"`
	Links    struct {
		Next string `json:"next"`
	} `json:"links"`
}

type included map[string]resource

func (i included) of(kind, id string) resource {
	return i[kind+"/"+id]
}

type applePoint struct {
	ID       string
	Price    string
	Currency string
}

func (p applePoint) shown() *Price {
	return &Price{Currency: p.Currency, Amount: p.Price}
}

type schedule struct {
	id     string
	base   string
	manual map[string]applePoint
}

func (a *Apple) Sync(ctx context.Context, targets []storeprice.Target, write bool) []Row {
	products, groups := byProduct(targets)
	rows := []Row{}
	for _, product := range products {
		rows = append(rows, a.syncProduct(ctx, product, groups[product], write)...)
	}
	return rows
}

func (a *Apple) syncProduct(ctx context.Context, product string, targets []storeprice.Target, write bool) []Row {
	purchase, err := a.purchase(ctx, product)
	if err != nil {
		return rowsFor(targets, StatusError, err.Error())
	}
	if purchase == "" {
		return rowsFor(targets, StatusNotInStore, "in-app purchase not found")
	}
	prices, err := a.schedule(ctx, purchase)
	if err != nil {
		return rowsFor(targets, StatusError, err.Error())
	}
	if _, priced := prices.manual[prices.base]; !priced {
		return rowsFor(targets, StatusError, "no base price in App Store Connect")
	}
	current, err := a.currentPrices(ctx, prices, targets)
	if err != nil {
		return rowsFor(targets, StatusError, err.Error())
	}
	points, err := a.basePoints(ctx, purchase)
	if err != nil {
		return rowsFor(targets, StatusError, err.Error())
	}
	desired := a.desiredPoints(ctx, points, targets)
	rows := make([]Row, 0, len(targets))
	for _, target := range targets {
		rows = append(rows, appleRow(target, current, desired[target.Cents]))
	}
	if !write || changes(rows) == 0 {
		return rows
	}
	manual := maps.Clone(prices.manual)
	for _, row := range rows {
		if row.Status == StatusChange {
			territory, _ := Territory(row.Country)
			manual[territory] = desired[row.TargetCents].points[territory]
		}
	}
	settle(rows, a.replaceSchedule(ctx, purchase, prices.base, manual))
	return rows
}

type equalized struct {
	points map[string]applePoint
	err    error
}

func appleRow(target storeprice.Target, current map[string]applePoint, wanted equalized) Row {
	row := Row{Product: target.Product, Country: target.Country, TargetCents: target.Cents}
	territory, known := Territory(target.Country)
	price, priced := current[territory]
	if priced {
		row.Current = price.shown()
	}
	next, available := wanted.points[territory]
	switch {
	case wanted.err != nil:
		row.Status, row.Detail = StatusError, wanted.err.Error()
	case !known || !available:
		row.Status, row.Detail = StatusNotInStore, "territory has no equalized price point"
	case priced && price.ID == next.ID:
		row.Next, row.Status = next.shown(), StatusUnchanged
	default:
		row.Next, row.Status = next.shown(), StatusChange
	}
	return row
}

func (a *Apple) purchase(ctx context.Context, product string) (string, error) {
	query := url.Values{"filter[productId]": {product}, "limit": {"200"}}
	found, _, err := a.list(ctx, a.endpoint+"/v1/apps/"+url.PathEscape(a.app)+"/inAppPurchasesV2?"+query.Encode())
	if err != nil {
		return "", err
	}
	for _, item := range found {
		if item.Attributes.ProductID == product {
			return item.ID, nil
		}
	}
	return "", nil
}

func (a *Apple) schedule(ctx context.Context, purchase string) (schedule, error) {
	target := a.endpoint + "/v1/inAppPurchasesV2/" + url.PathEscape(purchase) + "/iapPriceSchedule?include=baseTerritory"
	status, body, err := a.caller.do(ctx, http.MethodGet, target, nil)
	switch {
	case err != nil:
		return schedule{}, err
	case status == http.StatusNotFound:
		return schedule{}, errors.New("no price schedule in App Store Connect")
	case status != http.StatusOK:
		return schedule{}, unexpected(http.MethodGet, target, status)
	}
	var doc document
	var current resource
	if err := json.Unmarshal(body, &doc); err != nil {
		return schedule{}, fmt.Errorf("%w: %v", ErrStore, err)
	}
	if err := json.Unmarshal(doc.Data, &current); err != nil {
		return schedule{}, fmt.Errorf("%w: %v", ErrStore, err)
	}
	base := current.related("baseTerritory")
	if base == "" {
		base = baseStorefront
	}
	manual, err := a.prices(ctx, a.endpoint+"/v1/inAppPurchasePriceSchedules/"+url.PathEscape(current.ID)+"/manualPrices", nil)
	return schedule{id: current.ID, base: base, manual: manual}, err
}

func (a *Apple) currentPrices(ctx context.Context, prices schedule, targets []storeprice.Target) (map[string]applePoint, error) {
	current := maps.Clone(prices.manual)
	var missing []string
	for _, target := range targets {
		territory, known := Territory(target.Country)
		if _, priced := current[territory]; known && !priced && !slices.Contains(missing, territory) {
			missing = append(missing, territory)
		}
	}
	if len(missing) == 0 {
		return current, nil
	}
	automatic, err := a.prices(ctx, a.endpoint+"/v1/inAppPurchasePriceSchedules/"+url.PathEscape(prices.id)+"/automaticPrices", missing)
	for territory, point := range automatic {
		current[territory] = point
	}
	return current, err
}

func (a *Apple) prices(ctx context.Context, path string, territories []string) (map[string]applePoint, error) {
	query := url.Values{"include": {"inAppPurchasePricePoint,territory"}, "limit": {"200"}}
	if len(territories) > 0 {
		query.Set("filter[territory]", strings.Join(territories, ","))
	}
	found, extra, err := a.list(ctx, path+"?"+query.Encode())
	if err != nil {
		return nil, err
	}
	points := map[string]applePoint{}
	for _, price := range found {
		if price.Attributes.EndDate != nil {
			continue
		}
		territory, point := price.related("territory"), price.related("inAppPurchasePricePoint")
		points[territory] = applePoint{ID: point, Price: extra.of("inAppPurchasePricePoints", point).Attributes.CustomerPrice, Currency: extra.of("territories", territory).Attributes.Currency}
	}
	return points, nil
}

func (a *Apple) basePoints(ctx context.Context, purchase string) ([]applePoint, error) {
	query := url.Values{"filter[territory]": {baseStorefront}, "limit": {"8000"}}
	found, _, err := a.list(ctx, a.endpoint+"/v2/inAppPurchases/"+url.PathEscape(purchase)+"/pricePoints?"+query.Encode())
	if err != nil {
		return nil, err
	}
	points := make([]applePoint, 0, len(found))
	for _, point := range found {
		points = append(points, applePoint{ID: point.ID, Price: point.Attributes.CustomerPrice, Currency: "USD"})
	}
	return points, nil
}

func (a *Apple) desiredPoints(ctx context.Context, points []applePoint, targets []storeprice.Target) map[int64]equalized {
	wanted := map[int64][]string{}
	for _, target := range targets {
		if territory, known := Territory(target.Country); known && !slices.Contains(wanted[target.Cents], territory) {
			wanted[target.Cents] = append(wanted[target.Cents], territory)
		}
	}
	desired := map[int64]equalized{}
	for cents, territories := range wanted {
		reference, found := nearest(points, cents)
		if !found {
			desired[cents] = equalized{err: errors.New("no USA price point")}
			continue
		}
		equal, err := a.equalizations(ctx, reference.ID, territories)
		if slices.Contains(territories, baseStorefront) {
			equal[baseStorefront] = reference
		}
		desired[cents] = equalized{points: equal, err: err}
	}
	return desired
}

func nearest(points []applePoint, cents int64) (applePoint, bool) {
	var best applePoint
	bestGap, bestCents := int64(math.MaxInt64), int64(0)
	for _, point := range points {
		price, err := strconv.ParseFloat(point.Price, 64)
		if err != nil {
			continue
		}
		pointCents := int64(math.Round(price * 100))
		gap := pointCents - cents
		if gap < 0 {
			gap = -gap
		}
		if gap < bestGap || gap == bestGap && pointCents < bestCents {
			best, bestGap, bestCents = point, gap, pointCents
		}
	}
	return best, bestGap != math.MaxInt64
}

func (a *Apple) equalizations(ctx context.Context, point string, territories []string) (map[string]applePoint, error) {
	query := url.Values{"filter[territory]": {strings.Join(territories, ",")}, "include": {"territory"}, "limit": {"200"}}
	found, extra, err := a.list(ctx, a.endpoint+"/v1/inAppPurchasePricePoints/"+url.PathEscape(point)+"/equalizations?"+query.Encode())
	equal := map[string]applePoint{}
	for _, item := range found {
		territory := item.related("territory")
		equal[territory] = applePoint{ID: item.ID, Price: item.Attributes.CustomerPrice, Currency: extra.of("territories", territory).Attributes.Currency}
	}
	return equal, err
}

func (a *Apple) list(ctx context.Context, target string) ([]resource, included, error) {
	var found []resource
	extra := included{}
	for target != "" {
		status, body, err := a.caller.do(ctx, http.MethodGet, target, nil)
		if err != nil {
			return nil, nil, err
		}
		if status != http.StatusOK {
			return nil, nil, unexpected(http.MethodGet, target, status)
		}
		var doc document
		var page []resource
		if err := json.Unmarshal(body, &doc); err != nil {
			return nil, nil, fmt.Errorf("%w: %v", ErrStore, err)
		}
		if err := json.Unmarshal(doc.Data, &page); err != nil {
			return nil, nil, fmt.Errorf("%w: %v", ErrStore, err)
		}
		found = append(found, page...)
		for _, item := range doc.Included {
			extra[item.Type+"/"+item.ID] = item
		}
		target = doc.Links.Next
	}
	return found, extra, nil
}

type relation struct {
	Data any `json:"data"`
}

type link struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

func (a *Apple) replaceSchedule(ctx context.Context, purchase, base string, manual map[string]applePoint) error {
	territories := make([]string, 0, len(manual))
	for territory := range manual {
		territories = append(territories, territory)
	}
	slices.Sort(territories)
	prices, inline := []link{}, []any{}
	for _, territory := range territories {
		local := "${price-" + territory + "}"
		prices = append(prices, link{Type: "inAppPurchasePrices", ID: local})
		inline = append(inline, map[string]any{
			"type": "inAppPurchasePrices", "id": local,
			"attributes": map[string]any{"startDate": nil},
			"relationships": map[string]relation{
				"inAppPurchaseV2":         {Data: link{Type: "inAppPurchases", ID: purchase}},
				"inAppPurchasePricePoint": {Data: link{Type: "inAppPurchasePricePoints", ID: manual[territory].ID}},
			},
		})
	}
	body, err := json.Marshal(map[string]any{
		"data": map[string]any{
			"type": "inAppPurchasePriceSchedules",
			"relationships": map[string]relation{
				"inAppPurchase": {Data: link{Type: "inAppPurchases", ID: purchase}},
				"baseTerritory": {Data: link{Type: "territories", ID: base}},
				"manualPrices":  {Data: prices},
			},
		},
		"included": inline,
	})
	if err != nil {
		return err
	}
	target := a.endpoint + "/v1/inAppPurchasePriceSchedules"
	status, _, err := a.caller.do(ctx, http.MethodPost, target, body)
	if err != nil {
		return err
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return unexpected(http.MethodPost, target, status)
	}
	return nil
}

package storesync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/googleauth"
	"github.com/Alievs-corp/hypedomino/server/internal/storeprice"
)

const (
	DefaultGoogleEndpoint = "https://androidpublisher.googleapis.com"
	googleScope           = "https://www.googleapis.com/auth/androidpublisher"
)

type GoogleConfig struct {
	PackageName        string
	ServiceAccountFile string
	Endpoint           string
	Client             *http.Client
	Now                func() time.Time
	Pause              time.Duration
}

type Google struct {
	base   string
	caller caller
}

func NewGoogle(config GoogleConfig) (*Google, error) {
	if config.PackageName == "" || config.ServiceAccountFile == "" {
		return nil, errors.New("storesync: package name and service account are required")
	}
	if config.Endpoint == "" {
		config.Endpoint = DefaultGoogleEndpoint
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
	source, err := googleauth.Load(config.ServiceAccountFile, googleScope, config.Client, config.Now)
	if err != nil {
		return nil, err
	}
	authorize := func(ctx context.Context) (string, error) {
		token, err := source.Token(ctx)
		return "Bearer " + token, err
	}
	base := config.Endpoint + "/androidpublisher/v3/applications/" + url.PathEscape(config.PackageName)
	return &Google{base: base, caller: caller{client: config.Client, authorize: authorize, pause: config.Pause}}, nil
}

type playPrice struct {
	PriceMicros string `json:"priceMicros"`
	Currency    string `json:"currency"`
}

type money struct {
	CurrencyCode string      `json:"currencyCode"`
	Units        json.Number `json:"units,omitempty"`
	Nanos        int64       `json:"nanos,omitempty"`
}

type conversion struct {
	prices map[string]playPrice
	err    error
}

func (g *Google) Sync(ctx context.Context, targets []storeprice.Target, write bool) []Row {
	conversions := map[int64]conversion{}
	products, groups := byProduct(targets)
	rows := []Row{}
	for _, product := range products {
		rows = append(rows, g.syncProduct(ctx, product, groups[product], conversions, write)...)
	}
	return rows
}

func (g *Google) syncProduct(ctx context.Context, product string, targets []storeprice.Target, conversions map[int64]conversion, write bool) []Row {
	listing, found, err := g.product(ctx, product)
	if err != nil {
		return rowsFor(targets, StatusError, err.Error())
	}
	if !found {
		return rowsFor(targets, StatusNotInStore, "product not found")
	}
	prices := map[string]playPrice{}
	if raw, set := listing["prices"]; set {
		if err := json.Unmarshal(raw, &prices); err != nil {
			return rowsFor(targets, StatusError, err.Error())
		}
	}
	rows := make([]Row, 0, len(targets))
	for _, target := range targets {
		converted, cached := conversions[target.Cents]
		if !cached {
			converted.prices, converted.err = g.convert(ctx, target.Cents)
			conversions[target.Cents] = converted
		}
		rows = append(rows, playRow(target, prices, converted))
	}
	if !write || changes(rows) == 0 {
		return rows
	}
	for _, row := range rows {
		if row.Status == StatusChange {
			prices[row.Country] = conversions[row.TargetCents].prices[row.Country]
		}
	}
	settle(rows, g.patch(ctx, product, listing, prices))
	return rows
}

func playRow(target storeprice.Target, current map[string]playPrice, converted conversion) Row {
	row := Row{Product: target.Product, Country: target.Country, TargetCents: target.Cents}
	if price, set := current[target.Country]; set {
		row.Current = shownPlayPrice(price)
	}
	next, available := converted.prices[target.Country]
	switch {
	case converted.err != nil:
		row.Status, row.Detail = StatusError, converted.err.Error()
	case !available:
		row.Status, row.Detail = StatusNotInStore, "region not offered by the conversion"
	case row.Current != nil && current[target.Country] == next:
		row.Next, row.Status = shownPlayPrice(next), StatusUnchanged
	default:
		row.Next, row.Status = shownPlayPrice(next), StatusChange
	}
	return row
}

func shownPlayPrice(price playPrice) *Price {
	micros, err := strconv.ParseInt(price.PriceMicros, 10, 64)
	if err != nil {
		return &Price{Currency: price.Currency, Amount: price.PriceMicros}
	}
	return &Price{Currency: price.Currency, Amount: decimal(micros, 6)}
}

func decimal(value int64, places int) string {
	scale := int64(1)
	for range places {
		scale *= 10
	}
	fraction := fmt.Sprintf("%0*d", places, value%scale)
	for len(fraction) > 2 && strings.HasSuffix(fraction, "0") {
		fraction = fraction[:len(fraction)-1]
	}
	return strconv.FormatInt(value/scale, 10) + "." + fraction
}

func (g *Google) product(ctx context.Context, product string) (map[string]json.RawMessage, bool, error) {
	target := g.base + "/inappproducts/" + url.PathEscape(product)
	status, body, err := g.caller.do(ctx, http.MethodGet, target, nil)
	switch {
	case err != nil:
		return nil, false, err
	case status == http.StatusNotFound:
		return nil, false, nil
	case status != http.StatusOK:
		return nil, false, unexpected(http.MethodGet, target, status)
	}
	listing := map[string]json.RawMessage{}
	if err := json.Unmarshal(body, &listing); err != nil {
		return nil, false, fmt.Errorf("%w: %v", ErrStore, err)
	}
	return listing, true, nil
}

func (g *Google) convert(ctx context.Context, cents int64) (map[string]playPrice, error) {
	request, err := json.Marshal(map[string]money{"price": {CurrencyCode: "USD", Units: json.Number(strconv.FormatInt(cents/100, 10)), Nanos: cents % 100 * 10_000_000}})
	if err != nil {
		return nil, err
	}
	target := g.base + "/pricing:convertRegionPrices"
	status, body, err := g.caller.do(ctx, http.MethodPost, target, request)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, unexpected(http.MethodPost, target, status)
	}
	var reply struct {
		ConvertedRegionPrices map[string]struct {
			Price money `json:"price"`
		} `json:"convertedRegionPrices"`
	}
	if err := json.Unmarshal(body, &reply); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStore, err)
	}
	prices := map[string]playPrice{}
	for region, converted := range reply.ConvertedRegionPrices {
		micros, err := microsOf(converted.Price)
		if err != nil {
			return nil, fmt.Errorf("%w: price of %s: %v", ErrStore, region, err)
		}
		prices[region] = playPrice{PriceMicros: strconv.FormatInt(micros, 10), Currency: converted.Price.CurrencyCode}
	}
	return prices, nil
}

func microsOf(price money) (int64, error) {
	if price.Units == "" {
		return price.Nanos / 1000, nil
	}
	units, err := price.Units.Int64()
	return units*1_000_000 + price.Nanos/1000, err
}

func (g *Google) patch(ctx context.Context, product string, listing map[string]json.RawMessage, prices map[string]playPrice) error {
	raw, err := json.Marshal(prices)
	if err != nil {
		return err
	}
	listing["prices"] = raw
	body, err := json.Marshal(listing)
	if err != nil {
		return err
	}
	target := g.base + "/inappproducts/" + url.PathEscape(product) + "?autoConvertMissingPrices=false"
	status, _, err := g.caller.do(ctx, http.MethodPatch, target, body)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return unexpected(http.MethodPatch, target, status)
	}
	return nil
}

package storesync

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/googleauth/googleauthtest"
	"github.com/Alievs-corp/hypedomino/server/internal/storeprice"
)

type fakePlay struct {
	mu       sync.Mutex
	products map[string]map[string]any
	failures map[string][]int
	patches  []string
}

var playRates = map[string]struct {
	currency string
	micros   int64
}{"AZ": {"AZN", 17_000}, "TR": {"TRY", 400_000}, "DE": {"EUR", 9_200}}

func newFakePlay(t *testing.T) (*fakePlay, *Google) {
	t.Helper()
	server := googleauthtest.New(t, googleScope)
	play := &fakePlay{products: map[string]map[string]any{
		"crystals_tier1": {
			"packageName": "com.example.game", "sku": "crystals_tier1", "status": "active", "purchaseType": "managedUser",
			"defaultPrice": map[string]any{"priceMicros": "990000", "currency": "USD"},
			"listings":     map[string]any{"en-US": map[string]any{"title": "Crystals", "description": "A pack"}},
			"prices": map[string]any{
				"AZ": map[string]any{"priceMicros": "1003000", "currency": "AZN"},
				"RU": map[string]any{"priceMicros": "89000000", "currency": "RUB"},
				"US": map[string]any{"priceMicros": "990000", "currency": "USD"},
			},
		},
		"pass_premium": {"sku": "pass_premium", "defaultPrice": map[string]any{"priceMicros": "4990000", "currency": "USD"}},
	}, failures: map[string][]int{}}
	base := "/androidpublisher/v3/applications/com.example.game"
	server.Mux.HandleFunc("POST "+base+"/pricing:convertRegionPrices", play.convert)
	server.Mux.HandleFunc("GET "+base+"/inappproducts/{sku}", play.get)
	server.Mux.HandleFunc("PATCH "+base+"/inappproducts/{sku}", play.patch)
	google, err := NewGoogle(GoogleConfig{PackageName: "com.example.game", ServiceAccountFile: server.File, Endpoint: server.URL, Client: server.Client(), Pause: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	return play, google
}

func (p *fakePlay) failing(w http.ResponseWriter, key string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	queue := p.failures[key]
	if len(queue) == 0 {
		return false
	}
	p.failures[key] = queue[1:]
	w.WriteHeader(queue[0])
	return true
}

func (p *fakePlay) convert(w http.ResponseWriter, r *http.Request) {
	if !googleauthtest.Authorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	var request struct {
		Price money `json:"price"`
	}
	if json.NewDecoder(r.Body).Decode(&request) != nil || request.Price.CurrencyCode != "USD" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	micros, _ := microsOf(request.Price)
	cents := micros / 10_000
	converted := map[string]any{}
	for region, rate := range playRates {
		local := cents * rate.micros
		converted[region] = map[string]any{"regionCode": region, "price": map[string]any{"currencyCode": rate.currency, "units": strconv.FormatInt(local/1_000_000, 10), "nanos": local % 1_000_000 * 1000}}
	}
	json.NewEncoder(w).Encode(map[string]any{"convertedRegionPrices": converted, "regionVersion": map[string]string{"version": "2022/02"}})
}

func (p *fakePlay) get(w http.ResponseWriter, r *http.Request) {
	sku := r.PathValue("sku")
	if !googleauthtest.Authorized(r) || p.failing(w, "get "+sku) {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	product, found := p.products[sku]
	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(product)
}

func (p *fakePlay) patch(w http.ResponseWriter, r *http.Request) {
	sku := r.PathValue("sku")
	if !googleauthtest.Authorized(r) || r.URL.Query().Get("autoConvertMissingPrices") != "false" || p.failing(w, "patch "+sku) {
		return
	}
	var product map[string]any
	if json.NewDecoder(r.Body).Decode(&product) != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.products[sku], p.patches = product, append(p.patches, sku)
	json.NewEncoder(w).Encode(product)
}

func (p *fakePlay) price(sku, region string) map[string]any {
	p.mu.Lock()
	defer p.mu.Unlock()
	prices, _ := p.products[sku]["prices"].(map[string]any)
	price, _ := prices[region].(map[string]any)
	return price
}

func target(product, country string, cents int64) storeprice.Target {
	return storeprice.Target{Product: product, Country: country, Cents: cents}
}

func rowOf(t *testing.T, rows []Row, product, country string) Row {
	t.Helper()
	for _, row := range rows {
		if row.Product == product && row.Country == country {
			return row
		}
	}
	t.Fatalf("no row for %s %s in %+v", product, country, rows)
	return Row{}
}

var playTargets = []storeprice.Target{
	target("crystals_tier1", "AZ", 59), target("crystals_tier1", "TR", 59), target("crystals_tier1", "PL", 79), target("crystals_tier1", "DE", 99),
	target("pass_premium", "AZ", 299), target("offer_starter", "AZ", 119),
}

func TestThePlayPlanComparesEachConvertedRegionPriceAndWritesNothing(t *testing.T) {
	play, google := newFakePlay(t)
	rows := google.Sync(context.Background(), playTargets, false)

	if row := rowOf(t, rows, "crystals_tier1", "AZ"); row.Status != StatusUnchanged || *row.Current != (Price{"AZN", "1.003"}) || *row.Next != (Price{"AZN", "1.003"}) {
		t.Fatalf("AZ already at the converted target: %+v", row)
	}
	if row := rowOf(t, rows, "crystals_tier1", "TR"); row.Status != StatusChange || row.Current != nil || *row.Next != (Price{"TRY", "23.60"}) {
		t.Fatalf("TR has no price yet: %+v", row)
	}
	if row := rowOf(t, rows, "crystals_tier1", "PL"); row.Status != StatusNotInStore {
		t.Fatalf("PL is not converted: %+v", row)
	}
	if row := rowOf(t, rows, "crystals_tier1", "DE"); row.Status != StatusChange || *row.Next != (Price{"EUR", "0.9108"}) {
		t.Fatalf("DE: %+v", row)
	}
	if row := rowOf(t, rows, "pass_premium", "AZ"); row.Status != StatusChange || row.TargetCents != 299 {
		t.Fatalf("pass AZ: %+v", row)
	}
	if row := rowOf(t, rows, "offer_starter", "AZ"); row.Status != StatusNotInStore || row.Detail != "product not found" {
		t.Fatalf("missing product: %+v", row)
	}
	if len(play.patches) != 0 {
		t.Fatalf("a plan wrote %v", play.patches)
	}
}

func TestAPlayApplyWritesOnlyTheChangesKeepsEverythingElseAndIsIdempotent(t *testing.T) {
	play, google := newFakePlay(t)
	rows := google.Sync(context.Background(), playTargets, true)

	if row := rowOf(t, rows, "crystals_tier1", "TR"); row.Status != StatusWritten {
		t.Fatalf("TR: %+v", row)
	}
	if row := rowOf(t, rows, "crystals_tier1", "AZ"); row.Status != StatusUnchanged {
		t.Fatalf("AZ: %+v", row)
	}
	if len(play.patches) != 2 {
		t.Fatalf("one patch per changed product, got %v", play.patches)
	}
	if got := play.price("crystals_tier1", "TR"); got["priceMicros"] != "23600000" || got["currency"] != "TRY" {
		t.Fatalf("TR written as %v", got)
	}
	if got := play.price("crystals_tier1", "RU"); got["priceMicros"] != "89000000" {
		t.Fatalf("RU touched: %v", got)
	}
	if got := play.price("crystals_tier1", "US"); got["priceMicros"] != "990000" {
		t.Fatalf("US touched: %v", got)
	}
	product := play.products["crystals_tier1"]
	if product["defaultPrice"].(map[string]any)["priceMicros"] != "990000" || product["listings"] == nil || product["status"] != "active" {
		t.Fatalf("other fields lost: %v", product)
	}

	again := google.Sync(context.Background(), playTargets, true)
	if len(play.patches) != 2 {
		t.Fatalf("a second apply wrote %v", play.patches)
	}
	for _, row := range again {
		if row.Status == StatusChange || row.Status == StatusWritten {
			t.Fatalf("still changing after apply: %+v", row)
		}
	}
}

func TestPlayRetriesARateLimitAndReportsAPersistentFailurePerProduct(t *testing.T) {
	play, google := newFakePlay(t)
	play.failures["get crystals_tier1"] = []int{http.StatusTooManyRequests, http.StatusServiceUnavailable}
	play.failures["get pass_premium"] = []int{500, 500, 500, 500}
	rows := google.Sync(context.Background(), playTargets, true)

	if row := rowOf(t, rows, "crystals_tier1", "TR"); row.Status != StatusWritten {
		t.Fatalf("retried product: %+v", row)
	}
	if row := rowOf(t, rows, "pass_premium", "AZ"); row.Status != StatusError || row.Detail == "" {
		t.Fatalf("failing product: %+v", row)
	}
	if len(play.patches) != 1 || play.patches[0] != "crystals_tier1" {
		t.Fatalf("patches %v", play.patches)
	}
}

func TestAFailedPlayWriteIsReportedOnItsRows(t *testing.T) {
	play, google := newFakePlay(t)
	play.failures["patch pass_premium"] = []int{400}
	rows := google.Sync(context.Background(), playTargets, true)
	if row := rowOf(t, rows, "pass_premium", "AZ"); row.Status != StatusFailed || row.Detail == "" {
		t.Fatalf("refused write: %+v", row)
	}
	if row := rowOf(t, rows, "crystals_tier1", "TR"); row.Status != StatusWritten {
		t.Fatalf("other product: %+v", row)
	}
}

func TestPlayPricesAreShownAsDecimalAmounts(t *testing.T) {
	for micros, want := range map[int64]string{990000: "0.99", 23600000: "23.60", 1003000: "1.003", 5_000_000: "5.00", 123: "0.000123"} {
		if got := decimal(micros, 6); got != want {
			t.Errorf("%d: %s, want %s", micros, got, want)
		}
	}
}

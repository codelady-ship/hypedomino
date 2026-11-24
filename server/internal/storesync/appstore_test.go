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
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/storeprice"
)

const (
	testIssuer = "57246542-96fe-1a63-e053-0824d011072a"
	testKeyID  = "2X9R4HXF34"
	testApp    = "1234567890"
)

var appleNow = time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

var appleRates = map[string]struct {
	currency string
	percent  int64
}{"AZE": {"AZN", 170}, "TUR": {"TRY", 4000}, "RUS": {"RUB", 9000}}

var usaPoints = []int64{29, 49, 59, 69, 99, 119, 199, 249, 299, 399, 499, 599, 999, 1999, 4999}

type fakeAppStore struct {
	t         *testing.T
	key       *ecdsa.PublicKey
	mu        sync.Mutex
	purchases map[string]string
	manual    map[string]map[string]string
	failures  map[string][]int
	posts     []map[string]any
}

func newFakeAppStore(t *testing.T) (*fakeAppStore, *Apple) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "AuthKey.p8")
	if err := os.WriteFile(file, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	store := &fakeAppStore{t: t, key: &key.PublicKey, purchases: map[string]string{"crystals_tier1": "iap-1", "pass_premium": "iap-2"}, failures: map[string][]int{},
		manual: map[string]map[string]string{
			"iap-1": {"USA": "USA-99", "AZE": "AZE-59", "RUS": "RUS-99"},
			"iap-2": {"USA": "USA-499"},
		}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/apps/{app}/inAppPurchasesV2", store.signed(store.findPurchase))
	mux.HandleFunc("GET /v1/inAppPurchasesV2/{iap}/iapPriceSchedule", store.signed(store.schedule))
	mux.HandleFunc("GET /v1/inAppPurchasePriceSchedules/{schedule}/manualPrices", store.signed(store.manualPrices))
	mux.HandleFunc("GET /v1/inAppPurchasePriceSchedules/{schedule}/automaticPrices", store.signed(store.automaticPrices))
	mux.HandleFunc("GET /v2/inAppPurchases/{iap}/pricePoints", store.signed(store.pricePoints))
	mux.HandleFunc("GET /v1/inAppPurchasePricePoints/{point}/equalizations", store.signed(store.equalizations))
	mux.HandleFunc("POST /v1/inAppPurchasePriceSchedules", store.signed(store.replace))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	apple, err := NewApple(AppleConfig{IssuerID: testIssuer, KeyID: testKeyID, PrivateKeyFile: file, AppID: testApp, Endpoint: server.URL, Client: server.Client(), Now: func() time.Time { return appleNow }, Pause: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	return store, apple
}

func verifyAppleToken(token string, key *ecdsa.PublicKey) (map[string]any, map[string]any, error) {
	segments := strings.Split(token, ".")
	if len(segments) != 3 {
		return nil, nil, fmt.Errorf("token has %d segments", len(segments))
	}
	signature, err := base64.RawURLEncoding.DecodeString(segments[2])
	if err != nil || len(signature) != 64 {
		return nil, nil, fmt.Errorf("signature %d bytes: %v", len(signature), err)
	}
	digest := sha256.Sum256([]byte(segments[0] + "." + segments[1]))
	if !ecdsa.Verify(key, digest[:], new(big.Int).SetBytes(signature[:32]), new(big.Int).SetBytes(signature[32:])) {
		return nil, nil, fmt.Errorf("signature does not verify")
	}
	var header, claims map[string]any
	rawHeader, _ := base64.RawURLEncoding.DecodeString(segments[0])
	rawClaims, _ := base64.RawURLEncoding.DecodeString(segments[1])
	if json.Unmarshal(rawHeader, &header) != nil || json.Unmarshal(rawClaims, &claims) != nil {
		return nil, nil, fmt.Errorf("token is not JSON")
	}
	return header, claims, nil
}

func (s *fakeAppStore) signed(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, _ := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		header, claims, err := verifyAppleToken(token, s.key)
		if err != nil || header["alg"] != "ES256" || header["kid"] != testKeyID || claims["iss"] != testIssuer || claims["aud"] != AppleAudience {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if s.failing(w, r.Method+" "+r.URL.Path) {
			return
		}
		next(w, r)
	}
}

func (s *fakeAppStore) failing(w http.ResponseWriter, key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	queue := s.failures[key]
	if len(queue) == 0 {
		return false
	}
	s.failures[key] = queue[1:]
	w.WriteHeader(queue[0])
	return true
}

func pointPrice(point string) (string, string) {
	territory, raw, _ := strings.Cut(point, "-")
	usdCents, _ := strconv.ParseInt(raw, 10, 64)
	if territory == "USA" {
		return "USD", decimal(usdCents, 2)
	}
	rate := appleRates[territory]
	return rate.currency, decimal(usdCents*rate.percent/100, 2)
}

func pointResource(point string) map[string]any {
	_, price := pointPrice(point)
	return map[string]any{"type": "inAppPurchasePricePoints", "id": point, "attributes": map[string]any{"customerPrice": price, "proceeds": price}}
}

func territoryResource(territory string) map[string]any {
	currency := "USD"
	if rate, known := appleRates[territory]; known {
		currency = rate.currency
	}
	return map[string]any{"type": "territories", "id": territory, "attributes": map[string]any{"currency": currency}}
}

func priceResource(id, territory, point string, ended bool) map[string]any {
	attributes := map[string]any{"startDate": nil, "endDate": nil}
	if ended {
		attributes["endDate"] = "2026-01-01"
	}
	return map[string]any{
		"type": "inAppPurchasePrices", "id": id, "attributes": attributes,
		"relationships": map[string]any{
			"inAppPurchasePricePoint": map[string]any{"data": map[string]any{"type": "inAppPurchasePricePoints", "id": point}},
			"territory":               map[string]any{"data": map[string]any{"type": "territories", "id": territory}},
		},
	}
}

func answer(w http.ResponseWriter, data any, included []map[string]any) {
	json.NewEncoder(w).Encode(map[string]any{"data": data, "included": included, "links": map[string]any{}})
}

func (s *fakeAppStore) findPurchase(w http.ResponseWriter, r *http.Request) {
	product := r.URL.Query().Get("filter[productId]")
	data := []map[string]any{}
	if id, found := s.purchases[product]; found && r.PathValue("app") == testApp {
		data = append(data, map[string]any{"type": "inAppPurchases", "id": id, "attributes": map[string]any{"productId": product, "name": product}})
	}
	answer(w, data, nil)
}

func (s *fakeAppStore) schedule(w http.ResponseWriter, r *http.Request) {
	iap := r.PathValue("iap")
	answer(w, map[string]any{"type": "inAppPurchasePriceSchedules", "id": "schedule-" + iap, "relationships": map[string]any{
		"baseTerritory": map[string]any{"data": map[string]any{"type": "territories", "id": "USA"}},
		"manualPrices":  map[string]any{"links": map[string]any{"related": "..."}},
	}}, []map[string]any{territoryResource("USA")})
}

func (s *fakeAppStore) manualPrices(w http.ResponseWriter, r *http.Request) {
	iap := strings.TrimPrefix(r.PathValue("schedule"), "schedule-")
	s.mu.Lock()
	defer s.mu.Unlock()
	data, included := []map[string]any{priceResource("old", "AZE", "AZE-99", true)}, []map[string]any{pointResource("AZE-99"), territoryResource("AZE")}
	for territory, point := range s.manual[iap] {
		data = append(data, priceResource("price-"+territory, territory, point, false))
		included = append(included, pointResource(point), territoryResource(territory))
	}
	answer(w, data, included)
}

func (s *fakeAppStore) automaticPrices(w http.ResponseWriter, r *http.Request) {
	iap := strings.TrimPrefix(r.PathValue("schedule"), "schedule-")
	s.mu.Lock()
	base := s.manual[iap]["USA"]
	s.mu.Unlock()
	_, usdCents, _ := strings.Cut(base, "-")
	data, included := []map[string]any{}, []map[string]any{}
	for _, territory := range strings.Split(r.URL.Query().Get("filter[territory]"), ",") {
		if _, offered := appleRates[territory]; offered {
			point := territory + "-" + usdCents
			data = append(data, priceResource("auto-"+territory, territory, point, false))
			included = append(included, pointResource(point), territoryResource(territory))
		}
	}
	answer(w, data, included)
}

func (s *fakeAppStore) pricePoints(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("filter[territory]") != "USA" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	data := []map[string]any{}
	for _, cents := range usaPoints {
		data = append(data, pointResource("USA-"+strconv.FormatInt(cents, 10)))
	}
	answer(w, data, nil)
}

func (s *fakeAppStore) equalizations(w http.ResponseWriter, r *http.Request) {
	_, usdCents, _ := strings.Cut(r.PathValue("point"), "-")
	data, included := []map[string]any{}, []map[string]any{}
	for _, territory := range strings.Split(r.URL.Query().Get("filter[territory]"), ",") {
		if _, offered := appleRates[territory]; offered {
			point := pointResource(territory + "-" + usdCents)
			point["relationships"] = map[string]any{"territory": map[string]any{"data": map[string]any{"type": "territories", "id": territory}}}
			data = append(data, point)
			included = append(included, territoryResource(territory))
		}
	}
	answer(w, data, included)
}

func (s *fakeAppStore) replace(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Data struct {
			Relationships map[string]struct {
				Data json.RawMessage `json:"data"`
			} `json:"relationships"`
		} `json:"data"`
		Included []resource `json:"included"`
	}
	if json.NewDecoder(r.Body).Decode(&body) != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var purchase, base link
	json.Unmarshal(body.Data.Relationships["inAppPurchase"].Data, &purchase)
	json.Unmarshal(body.Data.Relationships["baseTerritory"].Data, &base)
	manual := map[string]string{}
	for _, price := range body.Included {
		if price.Type != "inAppPurchasePrices" || price.related("inAppPurchaseV2") != purchase.ID {
			w.WriteHeader(http.StatusConflict)
			return
		}
		point := price.related("inAppPurchasePricePoint")
		territory, _, _ := strings.Cut(point, "-")
		manual[territory] = point
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.manual[purchase.ID] = manual
	s.posts = append(s.posts, map[string]any{"purchase": purchase.ID, "base": base.ID, "manual": manual})
	w.WriteHeader(http.StatusCreated)
	answer(w, map[string]any{"type": "inAppPurchasePriceSchedules", "id": "schedule-" + purchase.ID}, nil)
}

var appleTargets = []storeprice.Target{
	target("crystals_tier1", "AZ", 59), target("crystals_tier1", "TR", 64), target("crystals_tier1", "GB", 59), target("crystals_tier1", "XK", 59),
	target("pass_premium", "AZ", 299), target("offer_starter", "AZ", 119),
}

func TestTheAppStorePlanEqualizesTheNearestUSAPointAndWritesNothing(t *testing.T) {
	store, apple := newFakeAppStore(t)
	rows := apple.Sync(context.Background(), appleTargets, false)

	if row := rowOf(t, rows, "crystals_tier1", "AZ"); row.Status != StatusUnchanged || *row.Current != (Price{"AZN", "1.00"}) {
		t.Fatalf("AZ manual price already at the point: %+v", row)
	}
	if row := rowOf(t, rows, "crystals_tier1", "TR"); row.Status != StatusChange || *row.Current != (Price{"TRY", "39.60"}) || *row.Next != (Price{"TRY", "23.60"}) {
		t.Fatalf("TR from its automatic price to the 0.59 equalization: %+v", row)
	}
	if row := rowOf(t, rows, "crystals_tier1", "GB"); row.Status != StatusNotInStore {
		t.Fatalf("GB has no equalization: %+v", row)
	}
	if row := rowOf(t, rows, "crystals_tier1", "XK"); row.Status != StatusNotInStore {
		t.Fatalf("XK has no territory: %+v", row)
	}
	if row := rowOf(t, rows, "pass_premium", "AZ"); row.Status != StatusChange || *row.Current != (Price{"AZN", "8.48"}) || *row.Next != (Price{"AZN", "5.08"}) {
		t.Fatalf("pass AZ: %+v", row)
	}
	if row := rowOf(t, rows, "offer_starter", "AZ"); row.Status != StatusNotInStore {
		t.Fatalf("missing purchase: %+v", row)
	}
	if len(store.posts) != 0 {
		t.Fatalf("a plan wrote %v", store.posts)
	}
}

func TestAnAppStoreApplyKeepsTheBaseAndOtherManualPricesAndIsIdempotent(t *testing.T) {
	store, apple := newFakeAppStore(t)
	rows := apple.Sync(context.Background(), appleTargets, true)

	if row := rowOf(t, rows, "crystals_tier1", "TR"); row.Status != StatusWritten {
		t.Fatalf("TR: %+v", row)
	}
	if len(store.posts) != 2 {
		t.Fatalf("one schedule per changed purchase, got %v", store.posts)
	}
	first := store.posts[0]
	want := map[string]string{"USA": "USA-99", "AZE": "AZE-59", "RUS": "RUS-99", "TUR": "TUR-59"}
	if first["purchase"] != "iap-1" || first["base"] != "USA" || fmt.Sprint(first["manual"]) != fmt.Sprint(want) {
		t.Fatalf("schedule %v", first)
	}

	again := apple.Sync(context.Background(), appleTargets, true)
	if len(store.posts) != 2 {
		t.Fatalf("a second apply wrote %v", store.posts[2:])
	}
	for _, row := range again {
		if row.Status == StatusChange || row.Status == StatusWritten {
			t.Fatalf("still changing after apply: %+v", row)
		}
	}
}

func TestAppStoreRetriesARateLimitAndReportsAPersistentFailurePerProduct(t *testing.T) {
	store, apple := newFakeAppStore(t)
	store.failures["GET /v2/inAppPurchases/iap-1/pricePoints"] = []int{http.StatusTooManyRequests}
	store.failures["GET /v1/inAppPurchasesV2/iap-2/iapPriceSchedule"] = []int{502, 502, 502, 502}
	rows := apple.Sync(context.Background(), appleTargets, true)

	if row := rowOf(t, rows, "crystals_tier1", "TR"); row.Status != StatusWritten {
		t.Fatalf("retried product: %+v", row)
	}
	if row := rowOf(t, rows, "pass_premium", "AZ"); row.Status != StatusError || row.Detail == "" {
		t.Fatalf("failing product: %+v", row)
	}
	if len(store.posts) != 1 {
		t.Fatalf("posts %v", store.posts)
	}
}

func TestTheNearestUSAPointWinsAndATieTakesTheLowerPoint(t *testing.T) {
	points := []applePoint{{ID: "a", Price: "0.69"}, {ID: "b", Price: "0.59"}, {ID: "c", Price: "0.99"}, {ID: "d", Price: "bad"}}
	for cents, want := range map[int64]string{64: "b", 59: "b", 66: "a", 84: "a", 85: "c", 500: "c", 1: "b"} {
		if got, ok := nearest(points, cents); !ok || got.ID != want {
			t.Errorf("%d cents: %s, want %s", cents, got.ID, want)
		}
	}
	if _, ok := nearest(nil, 99); ok {
		t.Error("no points found one")
	}
}

func TestTheAppStoreTokenIsAShortES256TokenWithTheTeamClaims(t *testing.T) {
	store, apple := newFakeAppStore(t)
	authorization, err := apple.bearer(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	header, claims, err := verifyAppleToken(strings.TrimPrefix(authorization, "Bearer "), store.key)
	if err != nil {
		t.Fatal(err)
	}
	if header["alg"] != "ES256" || header["kid"] != testKeyID || header["typ"] != "JWT" {
		t.Fatalf("header %v", header)
	}
	issued, expires := int64(claims["iat"].(float64)), int64(claims["exp"].(float64))
	if claims["iss"] != testIssuer || claims["aud"] != AppleAudience || issued != appleNow.Unix() || expires-issued >= int64(20*time.Minute/time.Second) || expires <= issued {
		t.Fatalf("claims %v", claims)
	}
	again, _ := apple.bearer(context.Background())
	if again != authorization {
		t.Fatal("the token is not reused while valid")
	}
}

func TestAnAppStoreKeyMustBeAP256PrivateKey(t *testing.T) {
	file := filepath.Join(t.TempDir(), "AuthKey.p8")
	os.WriteFile(file, []byte("not a key"), 0o600)
	if _, err := NewApple(AppleConfig{IssuerID: testIssuer, KeyID: testKeyID, PrivateKeyFile: file, AppID: testApp}); err == nil {
		t.Fatal("a broken key was accepted")
	}
	if _, err := NewApple(AppleConfig{IssuerID: testIssuer, KeyID: testKeyID, AppID: testApp}); err == nil {
		t.Fatal("a missing key was accepted")
	}
}

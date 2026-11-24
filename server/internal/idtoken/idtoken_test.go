package idtoken

import (
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	googleClient = "web-client.apps.googleusercontent.com"
	appleClient  = "com.example.hypedomino"
)

var start = time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)

var (
	keysOnce sync.Once
	keyA     *rsa.PrivateKey
	keyB     *rsa.PrivateKey
	keyC     *rsa.PrivateKey
)

func testKeys(t *testing.T) {
	t.Helper()
	keysOnce.Do(func() {
		keyA = generateKey()
		keyB = generateKey()
		keyC = generateKey()
	})
}

func generateKey() *rsa.PrivateKey {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	return key
}

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

type keyServer struct {
	mu           sync.Mutex
	keys         map[string]*rsa.PrivateKey
	cacheControl string
	status       int
	fetches      atomic.Int32
	server       *httptest.Server
}

func newKeyServer(t *testing.T, keys map[string]*rsa.PrivateKey) *keyServer {
	t.Helper()
	k := &keyServer{keys: keys, cacheControl: "public, max-age=3600", status: http.StatusOK}
	k.server = httptest.NewServer(http.HandlerFunc(k.serve))
	t.Cleanup(k.server.Close)
	return k
}

func (k *keyServer) serve(w http.ResponseWriter, _ *http.Request) {
	k.fetches.Add(1)
	k.mu.Lock()
	defer k.mu.Unlock()
	w.Header().Set("Cache-Control", k.cacheControl)
	if k.status != http.StatusOK {
		w.WriteHeader(k.status)
		return
	}
	set := struct {
		Keys []jsonWebKey `json:"keys"`
	}{}
	for kid, key := range k.keys {
		set.Keys = append(set.Keys, publicJWK(kid, &key.PublicKey))
	}
	_ = json.NewEncoder(w).Encode(set)
}

func (k *keyServer) set(keys map[string]*rsa.PrivateKey, status int) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.keys, k.status = keys, status
}

func publicJWK(kid string, key *rsa.PublicKey) jsonWebKey {
	return jsonWebKey{
		KeyType:   "RSA",
		KeyID:     kid,
		Use:       "sig",
		Algorithm: "RS256",
		Modulus:   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		Exponent:  base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}
}

type fixture struct {
	verifier *Verifier
	clock    *fakeClock
	nonces   *MemoryNonces
	google   *keyServer
	apple    *keyServer
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	testKeys(t)
	f := &fixture{
		clock:  &fakeClock{now: start},
		nonces: NewMemoryNonces(),
		google: newKeyServer(t, map[string]*rsa.PrivateKey{"google-a": keyA}),
		apple:  newKeyServer(t, map[string]*rsa.PrivateKey{"apple-a": keyB}),
	}
	config := DefaultConfig()
	config.Google.KeySetURL = f.google.server.URL
	config.Google.Audiences = []string{googleClient}
	config.Apple.KeySetURL = f.apple.server.URL
	config.Apple.Audiences = []string{appleClient}
	f.verifier = New(Options{Config: config, Clock: f.clock, HTTPClient: f.google.server.Client(), Nonces: f.nonces})
	return f
}

func (f *fixture) nonce(t *testing.T) string {
	t.Helper()
	nonce, err := f.verifier.IssueNonce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return nonce
}

func googleClaims(nonce string) map[string]any {
	return map[string]any{
		"iss":            "https://accounts.google.com",
		"aud":            googleClient,
		"azp":            "android-client.apps.googleusercontent.com",
		"sub":            "110169484474386276334",
		"email":          "player@example.com",
		"email_verified": true,
		"iat":            start.Add(-time.Minute).Unix(),
		"nbf":            start.Add(-time.Minute).Unix(),
		"exp":            start.Add(time.Hour).Unix(),
		"nonce":          nonce,
	}
}

func appleClaims(nonce string) map[string]any {
	digest := sha256.Sum256([]byte(nonce))
	return map[string]any{
		"iss":              "https://appleid.apple.com",
		"aud":              appleClient,
		"sub":              "001234.abcdef0123456789abcdef0123456789.1234",
		"email":            "relay123@privaterelay.appleid.com",
		"email_verified":   "true",
		"is_private_email": "true",
		"nonce_supported":  true,
		"iat":              start.Add(-time.Minute).Unix(),
		"exp":              start.Add(10 * time.Minute).Unix(),
		"nonce":            hex.EncodeToString(digest[:]),
	}
}

func issuedAt(claims map[string]any, now time.Time) map[string]any {
	claims["iat"] = now.Add(-time.Minute).Unix()
	claims["nbf"] = now.Add(-time.Minute).Unix()
	claims["exp"] = now.Add(time.Hour).Unix()
	return claims
}

func encodeSegment(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(data)
}

func signRS256(t *testing.T, header, claims map[string]any, key *rsa.PrivateKey) string {
	t.Helper()
	input := encodeSegment(t, header) + "." + encodeSegment(t, claims)
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func rsHeader(kid string) map[string]any {
	return map[string]any{"alg": "RS256", "kid": kid, "typ": "JWT"}
}

func TestVerify(t *testing.T) {
	tests := []struct {
		name     string
		provider Provider
		claims   func(nonce string) map[string]any
		header   map[string]any
		key      func() *rsa.PrivateKey
		want     error
		check    func(t *testing.T, identity Identity)
	}{
		{
			name:     "valid google token",
			provider: ProviderGoogle,
			claims:   googleClaims,
			check: func(t *testing.T, identity Identity) {
				want := Identity{Provider: ProviderGoogle, Subject: "110169484474386276334", Email: "player@example.com", EmailVerified: true}
				if identity != want {
					t.Fatalf("identity = %+v, want %+v", identity, want)
				}
			},
		},
		{
			name:     "valid apple token with relay email",
			provider: ProviderApple,
			claims:   appleClaims,
			check: func(t *testing.T, identity Identity) {
				want := Identity{Provider: ProviderApple, Subject: "001234.abcdef0123456789abcdef0123456789.1234", Email: "relay123@privaterelay.appleid.com", EmailVerified: true, PrivateEmail: true}
				if identity != want {
					t.Fatalf("identity = %+v, want %+v", identity, want)
				}
			},
		},
		{
			name:     "apple boolean claims",
			provider: ProviderApple,
			claims: with(appleClaims, func(c map[string]any) {
				c["email_verified"] = false
				c["is_private_email"] = "false"
			}),
			check: func(t *testing.T, identity Identity) {
				if identity.EmailVerified || identity.PrivateEmail {
					t.Fatalf("identity = %+v, want unverified real email", identity)
				}
			},
		},
		{
			name:     "google workspace token carries its hosted domain",
			provider: ProviderGoogle,
			claims:   with(googleClaims, func(c map[string]any) { c["hd"] = "example.com" }),
			check: func(t *testing.T, identity Identity) {
				if identity.HostedDomain != "example.com" {
					t.Fatalf("identity = %+v, want hosted domain example.com", identity)
				}
			},
		},
		{
			name:     "google issuer without scheme",
			provider: ProviderGoogle,
			claims:   with(googleClaims, func(c map[string]any) { c["iss"] = "accounts.google.com" }),
		},
		{
			name:     "audience as list of trusted clients",
			provider: ProviderGoogle,
			claims:   with(googleClaims, func(c map[string]any) { c["aud"] = []string{googleClient} }),
		},
		{
			name:     "expired inside clock skew",
			provider: ProviderGoogle,
			claims:   with(googleClaims, func(c map[string]any) { c["exp"] = start.Add(-30 * time.Second).Unix() }),
		},
		{
			name:     "expired",
			provider: ProviderGoogle,
			claims:   with(googleClaims, func(c map[string]any) { c["exp"] = start.Add(-61 * time.Second).Unix() }),
			want:     ErrExpired,
		},
		{
			name:     "not yet valid",
			provider: ProviderGoogle,
			claims:   with(googleClaims, func(c map[string]any) { c["nbf"] = start.Add(2 * time.Minute).Unix() }),
			want:     ErrNotYetValid,
		},
		{
			name:     "issued in the future",
			provider: ProviderApple,
			claims:   with(appleClaims, func(c map[string]any) { c["iat"] = start.Add(2 * time.Minute).Unix() }),
			want:     ErrNotYetValid,
		},
		{
			name:     "missing expiry",
			provider: ProviderGoogle,
			claims:   with(googleClaims, func(c map[string]any) { delete(c, "exp") }),
			want:     ErrMalformed,
		},
		{
			name:     "expiry as string",
			provider: ProviderGoogle,
			claims:   with(googleClaims, func(c map[string]any) { c["exp"] = "9999999999" }),
			want:     ErrMalformed,
		},
		{
			name:     "email_verified with unknown value",
			provider: ProviderApple,
			claims:   with(appleClaims, func(c map[string]any) { c["email_verified"] = "yes" }),
			want:     ErrMalformed,
		},
		{
			name:     "wrong issuer",
			provider: ProviderGoogle,
			claims:   with(googleClaims, func(c map[string]any) { c["iss"] = "https://appleid.apple.com" }),
			want:     ErrIssuer,
		},
		{
			name:     "google issuer under the apple key set",
			provider: ProviderApple,
			claims:   googleClaims,
			want:     ErrIssuer,
		},
		{
			name:     "google key presented as apple",
			provider: ProviderApple,
			claims:   appleClaims,
			header:   rsHeader("google-a"),
			key:      func() *rsa.PrivateKey { return keyA },
			want:     ErrUnknownKey,
		},
		{
			name:     "wrong audience",
			provider: ProviderApple,
			claims:   with(appleClaims, func(c map[string]any) { c["aud"] = "com.other.app" }),
			want:     ErrAudience,
		},
		{
			name:     "untrusted extra audience",
			provider: ProviderGoogle,
			claims:   with(googleClaims, func(c map[string]any) { c["aud"] = []string{googleClient, "other"} }),
			want:     ErrAudience,
		},
		{
			name:     "missing subject",
			provider: ProviderGoogle,
			claims:   with(googleClaims, func(c map[string]any) { delete(c, "sub") }),
			want:     ErrSubject,
		},
		{
			name:     "bad signature from a foreign key with a known kid",
			provider: ProviderGoogle,
			claims:   googleClaims,
			key:      func() *rsa.PrivateKey { return keyC },
			want:     ErrSignature,
		},
		{
			name:     "unknown kid",
			provider: ProviderGoogle,
			claims:   googleClaims,
			header:   rsHeader("forged"),
			want:     ErrUnknownKey,
		},
		{
			name:     "missing kid",
			provider: ProviderGoogle,
			claims:   googleClaims,
			header:   map[string]any{"alg": "RS256"},
			want:     ErrUnknownKey,
		},
		{
			name:     "RS384 not allowed",
			provider: ProviderGoogle,
			claims:   googleClaims,
			header:   map[string]any{"alg": "RS384", "kid": "google-a"},
			want:     ErrAlgorithm,
		},
		{
			name:     "critical header",
			provider: ProviderGoogle,
			claims:   googleClaims,
			header:   map[string]any{"alg": "RS256", "kid": "google-a", "crit": []string{"exp"}},
			want:     ErrMalformed,
		},
		{
			name:     "nonce of another login",
			provider: ProviderGoogle,
			claims:   with(googleClaims, func(c map[string]any) { c["nonce"] = "another" }),
			want:     ErrNonce,
		},
		{
			name:     "missing nonce claim",
			provider: ProviderApple,
			claims:   with(appleClaims, func(c map[string]any) { delete(c, "nonce") }),
			want:     ErrNonce,
		},
		{
			name:     "apple raw nonce where hash is configured",
			provider: ProviderApple,
			claims: func(nonce string) map[string]any {
				c := appleClaims(nonce)
				c["nonce"] = nonce
				return c
			},
			want: ErrNonce,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newFixture(t)
			header := test.header
			if header == nil {
				header = rsHeader(defaultKeyID(test.provider))
			}
			key := defaultKey(test.provider)
			if test.key != nil {
				key = test.key()
			}
			nonce := f.nonce(t)
			token := signRS256(t, header, test.claims(nonce), key)
			identity, err := f.verifier.Verify(context.Background(), test.provider, token, nonce)
			if !errors.Is(err, test.want) {
				t.Fatalf("err = %v, want %v", err, test.want)
			}
			if test.check != nil {
				test.check(t, identity)
			}
		})
	}
}

func with(base func(string) map[string]any, change func(map[string]any)) func(string) map[string]any {
	return func(nonce string) map[string]any {
		claims := base(nonce)
		change(claims)
		return claims
	}
}

func defaultKeyID(provider Provider) string {
	if provider == ProviderApple {
		return "apple-a"
	}
	return "google-a"
}

func defaultKey(provider Provider) *rsa.PrivateKey {
	if provider == ProviderApple {
		return keyB
	}
	return keyA
}

func TestVerifyRejectsForgedAlgorithms(t *testing.T) {
	f := newFixture(t)
	nonce := f.nonce(t)
	payload := encodeSegment(t, googleClaims(nonce))
	der, err := x509.MarshalPKIXPublicKey(&keyA.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	hmacInput := encodeSegment(t, map[string]any{"alg": "HS256", "kid": "google-a"}) + "." + payload
	mac := hmac.New(sha256.New, der)
	mac.Write([]byte(hmacInput))
	tokens := map[string]string{
		"alg none":                        encodeSegment(t, map[string]any{"alg": "none", "kid": "google-a"}) + "." + payload + ".",
		"alg none lower case":             encodeSegment(t, map[string]any{"alg": "None"}) + "." + payload + ".c2ln",
		"HS256 with public key as secret": hmacInput + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)),
	}
	for name, token := range tokens {
		t.Run(name, func(t *testing.T) {
			if _, err := f.verifier.Verify(context.Background(), ProviderGoogle, token, nonce); !errors.Is(err, ErrAlgorithm) {
				t.Fatalf("err = %v, want %v", err, ErrAlgorithm)
			}
		})
	}
}

func TestVerifyRejectsMalformedTokens(t *testing.T) {
	f := newFixture(t)
	nonce := f.nonce(t)
	valid := signRS256(t, rsHeader("google-a"), googleClaims(nonce), keyA)
	segments := strings.Split(valid, ".")
	tokens := map[string]string{
		"empty":               "",
		"two segments":        segments[0] + "." + segments[1],
		"four segments":       valid + ".extra",
		"padded header":       segments[0] + "=." + segments[1] + "." + segments[2],
		"header not json":     base64.RawURLEncoding.EncodeToString([]byte("{")) + "." + segments[1] + "." + segments[2],
		"payload not base64":  segments[0] + ".!!!." + segments[2],
		"empty signature":     segments[0] + "." + segments[1] + ".",
		"oversized":           segments[0] + "." + strings.Repeat("a", DefaultConfig().MaxTokenBytes) + "." + segments[2],
		"payload is an array": signRS256Raw(t, rsHeader("google-a"), []byte("[1]"), keyA),
	}
	for name, token := range tokens {
		t.Run(name, func(t *testing.T) {
			if _, err := f.verifier.Verify(context.Background(), ProviderGoogle, token, nonce); !errors.Is(err, ErrMalformed) {
				t.Fatalf("err = %v, want %v", err, ErrMalformed)
			}
		})
	}
	if _, err := f.verifier.Verify(context.Background(), Provider(9), valid, nonce); !errors.Is(err, ErrUnknownProvider) {
		t.Fatalf("unknown provider err = %v", err)
	}
}

func signRS256Raw(t *testing.T, header map[string]any, payload []byte, key *rsa.PrivateKey) string {
	t.Helper()
	input := encodeSegment(t, header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func TestNonceIsSingleUse(t *testing.T) {
	f := newFixture(t)
	nonce := f.nonce(t)
	token := signRS256(t, rsHeader("apple-a"), appleClaims(nonce), keyB)
	if _, err := f.verifier.Verify(context.Background(), ProviderApple, token, nonce); err != nil {
		t.Fatalf("first use: %v", err)
	}
	if _, err := f.verifier.Verify(context.Background(), ProviderApple, token, nonce); !errors.Is(err, ErrNonceUsed) {
		t.Fatalf("replay err = %v, want %v", err, ErrNonceUsed)
	}
}

func TestNonceMustBeIssuedAndFresh(t *testing.T) {
	f := newFixture(t)
	unissued := "client-made-nonce"
	token := signRS256(t, rsHeader("google-a"), googleClaims(unissued), keyA)
	if _, err := f.verifier.Verify(context.Background(), ProviderGoogle, token, unissued); !errors.Is(err, ErrNonceUsed) {
		t.Fatalf("unissued err = %v, want %v", err, ErrNonceUsed)
	}
	stale := f.nonce(t)
	token = signRS256(t, rsHeader("google-a"), googleClaims(stale), keyA)
	f.clock.advance(DefaultConfig().NonceTTL)
	if _, err := f.verifier.Verify(context.Background(), ProviderGoogle, token, stale); !errors.Is(err, ErrNonceUsed) {
		t.Fatalf("stale err = %v, want %v", err, ErrNonceUsed)
	}
}

func TestRejectedTokenKeepsNonce(t *testing.T) {
	f := newFixture(t)
	nonce := f.nonce(t)
	forged := signRS256(t, rsHeader("google-a"), googleClaims(nonce), keyC)
	if _, err := f.verifier.Verify(context.Background(), ProviderGoogle, forged, nonce); !errors.Is(err, ErrSignature) {
		t.Fatalf("forged err = %v", err)
	}
	token := signRS256(t, rsHeader("google-a"), googleClaims(nonce), keyA)
	if _, err := f.verifier.Verify(context.Background(), ProviderGoogle, token, nonce); err != nil {
		t.Fatalf("genuine token after forged attempt: %v", err)
	}
}

func TestKeyRotationRefetchesOnce(t *testing.T) {
	f := newFixture(t)
	verify := func(kid string, key *rsa.PrivateKey) error {
		nonce := f.nonce(t)
		_, err := f.verifier.Verify(context.Background(), ProviderGoogle, signRS256(t, rsHeader(kid), googleClaims(nonce), key), nonce)
		return err
	}
	if err := verify("google-a", keyA); err != nil {
		t.Fatal(err)
	}
	f.google.set(map[string]*rsa.PrivateKey{"google-b": keyC}, http.StatusOK)
	f.clock.advance(DefaultConfig().KeyRefetchMinInterval)
	if err := verify("google-b", keyC); err != nil {
		t.Fatalf("rotated key: %v", err)
	}
	if got := f.google.fetches.Load(); got != 2 {
		t.Fatalf("fetches after rotation = %d, want 2", got)
	}
	for _, kid := range []string{"forged-1", "forged-2", "forged-3"} {
		if err := verify(kid, keyC); !errors.Is(err, ErrUnknownKey) {
			t.Fatalf("%s err = %v, want %v", kid, err, ErrUnknownKey)
		}
	}
	if got := f.google.fetches.Load(); got != 2 {
		t.Fatalf("forged kids fetched %d times, want no refetch inside the interval", got-2)
	}
	if err := verify("google-a", keyA); !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("retired key err = %v, want %v", err, ErrUnknownKey)
	}
	f.clock.advance(DefaultConfig().KeyRefetchMinInterval)
	if err := verify("forged-4", keyC); !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("forged-4 err = %v", err)
	}
	if got := f.google.fetches.Load(); got != 3 {
		t.Fatalf("fetches after the interval = %d, want 3", got)
	}
}

func TestKeyCacheHonoursMaxAge(t *testing.T) {
	f := newFixture(t)
	verify := func() error {
		nonce := f.nonce(t)
		claims := issuedAt(googleClaims(nonce), f.clock.Now())
		_, err := f.verifier.Verify(context.Background(), ProviderGoogle, signRS256(t, rsHeader("google-a"), claims, keyA), nonce)
		return err
	}
	if err := verify(); err != nil {
		t.Fatal(err)
	}
	f.clock.advance(59 * time.Minute)
	if err := verify(); err != nil {
		t.Fatal(err)
	}
	if got := f.google.fetches.Load(); got != 1 {
		t.Fatalf("fetches inside max-age = %d, want 1", got)
	}
	f.clock.advance(2 * time.Minute)
	if err := verify(); err != nil {
		t.Fatal(err)
	}
	if got := f.google.fetches.Load(); got != 2 {
		t.Fatalf("fetches after max-age = %d, want 2", got)
	}
}

func TestKeyFetchFailure(t *testing.T) {
	f := newFixture(t)
	f.google.set(nil, http.StatusInternalServerError)
	nonce := f.nonce(t)
	token := signRS256(t, rsHeader("google-a"), googleClaims(nonce), keyA)
	if _, err := f.verifier.Verify(context.Background(), ProviderGoogle, token, nonce); !errors.Is(err, ErrKeyFetch) {
		t.Fatalf("err = %v, want %v", err, ErrKeyFetch)
	}
}

func TestStaleKeysServeWhileProviderIsDown(t *testing.T) {
	f := newFixture(t)
	verify := func() error {
		nonce := f.nonce(t)
		claims := issuedAt(googleClaims(nonce), f.clock.Now())
		_, err := f.verifier.Verify(context.Background(), ProviderGoogle, signRS256(t, rsHeader("google-a"), claims, keyA), nonce)
		return err
	}
	if err := verify(); err != nil {
		t.Fatal(err)
	}
	f.google.set(nil, http.StatusServiceUnavailable)
	f.clock.advance(2 * time.Hour)
	if err := verify(); err != nil {
		t.Fatalf("stale key while provider is down: %v", err)
	}
	if got := f.google.fetches.Load(); got != 2 {
		t.Fatalf("fetches = %d, want 2", got)
	}
}

func TestConcurrentVerifyFetchesOnce(t *testing.T) {
	f := newFixture(t)
	type login struct{ nonce, token string }
	logins := make([]login, 32)
	for i := range logins {
		nonce := f.nonce(t)
		logins[i] = login{nonce, signRS256(t, rsHeader("apple-a"), appleClaims(nonce), keyB)}
	}
	var wait sync.WaitGroup
	failures := make(chan error, len(logins)*2)
	for _, l := range logins {
		for range 2 {
			wait.Add(1)
			go func() {
				defer wait.Done()
				_, err := f.verifier.Verify(context.Background(), ProviderApple, l.token, l.nonce)
				failures <- err
			}()
		}
	}
	wait.Wait()
	close(failures)
	accepted, replayed := 0, 0
	for err := range failures {
		switch {
		case err == nil:
			accepted++
		case errors.Is(err, ErrNonceUsed):
			replayed++
		default:
			t.Fatalf("unexpected err %v", err)
		}
	}
	if accepted != len(logins) || replayed != len(logins) {
		t.Fatalf("accepted %d replayed %d, want %d each", accepted, replayed, len(logins))
	}
	if got := f.apple.fetches.Load(); got != 1 {
		t.Fatalf("fetches = %d, want 1", got)
	}
}

func TestCacheLifetime(t *testing.T) {
	config := DefaultConfig()
	tests := []struct {
		cacheControl string
		want         time.Duration
	}{
		{"public, max-age=19600, must-revalidate, no-transform", 19600 * time.Second},
		{"no-store", config.KeyCacheMinTTL},
		{"", config.KeyCacheMinTTL},
		{"max-age=10", config.KeyCacheMinTTL},
		{"MAX-AGE=7200", 2 * time.Hour},
		{`max-age="7200"`, 2 * time.Hour},
		{"max-age=999999999999999999", config.KeyCacheMaxTTL},
		{"max-age=-5", config.KeyCacheMinTTL},
		{"max-age=abc", config.KeyCacheMinTTL},
	}
	for _, test := range tests {
		got := clampDuration(cacheMaxAge(test.cacheControl), config.KeyCacheMinTTL, config.KeyCacheMaxTTL)
		if got != test.want {
			t.Errorf("%q: got %v, want %v", test.cacheControl, got, test.want)
		}
	}
}

func TestKeySetSkipsUnusableKeys(t *testing.T) {
	testKeys(t)
	small, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	good := publicJWK("good", &keyA.PublicKey)
	weak := publicJWK("weak", &small.PublicKey)
	encryption := publicJWK("encryption", &keyA.PublicKey)
	encryption.Use = "enc"
	otherAlgorithm := publicJWK("ps256", &keyA.PublicKey)
	otherAlgorithm.Algorithm = "PS256"
	elliptic := publicJWK("ec", &keyA.PublicKey)
	elliptic.KeyType = "EC"
	evenExponent := publicJWK("even", &keyA.PublicKey)
	evenExponent.Exponent = base64.RawURLEncoding.EncodeToString([]byte{4})
	body, err := json.Marshal(map[string]any{"keys": []jsonWebKey{good, weak, encryption, otherAlgorithm, elliptic, evenExponent}})
	if err != nil {
		t.Fatal(err)
	}
	keys, err := parseKeySet(body, DefaultConfig().MinRSAKeyBits)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys["good"] == nil {
		t.Fatalf("keys = %v, want only good", keys)
	}
	if _, err := parseKeySet([]byte(`{"keys":[]}`), 2048); err == nil {
		t.Fatal("empty key set accepted")
	}
}

func TestMemoryNoncesPruneExpired(t *testing.T) {
	nonces := NewMemoryNonces()
	ctx := context.Background()
	ttl := time.Minute
	_ = nonces.Save(ctx, "first", start, ttl)
	_ = nonces.Save(ctx, "second", start.Add(30*time.Second), ttl)
	_ = nonces.Save(ctx, "third", start.Add(ttl), ttl)
	if len(nonces.expiries) != 2 {
		t.Fatalf("pending = %d, want 2 after pruning the expired first nonce", len(nonces.expiries))
	}
	if ok, _ := nonces.Consume(ctx, "second", start.Add(ttl)); !ok {
		t.Fatal("unexpired nonce refused")
	}
}

func TestIssuedNoncesAreUnique(t *testing.T) {
	f := newFixture(t)
	seen := map[string]bool{}
	for range 1000 {
		nonce := f.nonce(t)
		if seen[nonce] || len(nonce) < 26 {
			t.Fatalf("nonce %q repeated or short", nonce)
		}
		seen[nonce] = true
	}
}

package idtoken

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"net/http"
	"time"
)

var (
	ErrUnknownProvider = errors.New("idtoken: unknown provider")
	ErrMalformed       = errors.New("idtoken: malformed token")
	ErrAlgorithm       = errors.New("idtoken: algorithm not allowed")
	ErrUnknownKey      = errors.New("idtoken: unknown signing key")
	ErrKeyFetch        = errors.New("idtoken: signing keys unavailable")
	ErrSignature       = errors.New("idtoken: bad signature")
	ErrIssuer          = errors.New("idtoken: issuer not trusted")
	ErrAudience        = errors.New("idtoken: audience not trusted")
	ErrExpired         = errors.New("idtoken: token expired")
	ErrNotYetValid     = errors.New("idtoken: token not yet valid")
	ErrSubject         = errors.New("idtoken: subject missing")
	ErrNonce           = errors.New("idtoken: nonce does not match")
	ErrNonceUsed       = errors.New("idtoken: nonce unknown, expired or already used")
)

type Provider uint8

const (
	ProviderGoogle Provider = iota + 1
	ProviderApple
)

func (p Provider) String() string {
	switch p {
	case ProviderGoogle:
		return "google"
	case ProviderApple:
		return "apple"
	}
	return "invalid"
}

type NonceForm uint8

const (
	NonceRaw NonceForm = iota + 1
	NonceSHA256Hex
)

type ProviderConfig struct {
	KeySetURL string
	Issuers   []string
	Audiences []string
	NonceForm NonceForm
}

type Config struct {
	Google                ProviderConfig
	Apple                 ProviderConfig
	ClockSkew             time.Duration
	NonceTTL              time.Duration
	KeyCacheMinTTL        time.Duration
	KeyCacheMaxTTL        time.Duration
	KeyRefetchMinInterval time.Duration
	KeyFetchTimeout       time.Duration
	MaxTokenBytes         int
	MaxKeySetBytes        int64
	MinRSAKeyBits         int
}

func DefaultConfig() Config {
	return Config{
		Google: ProviderConfig{
			KeySetURL: "https://www.googleapis.com/oauth2/v3/certs",
			Issuers:   []string{"https://accounts.google.com", "accounts.google.com"},
			NonceForm: NonceRaw,
		},
		Apple: ProviderConfig{
			KeySetURL: "https://appleid.apple.com/auth/keys",
			Issuers:   []string{"https://appleid.apple.com"},
			NonceForm: NonceSHA256Hex,
		},
		ClockSkew:             60 * time.Second,
		NonceTTL:              10 * time.Minute,
		KeyCacheMinTTL:        15 * time.Minute,
		KeyCacheMaxTTL:        24 * time.Hour,
		KeyRefetchMinInterval: time.Minute,
		KeyFetchTimeout:       5 * time.Second,
		MaxTokenBytes:         8192,
		MaxKeySetBytes:        65536,
		MinRSAKeyBits:         2048,
	}
}

type Identity struct {
	Provider      Provider
	Subject       string
	Email         string
	EmailVerified bool
	PrivateEmail  bool
	HostedDomain  string
}

type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

type Options struct {
	Config     Config
	Clock      Clock
	HTTPClient *http.Client
	Nonces     NonceStore
}

type Verifier struct {
	config Config
	clock  Clock
	nonces NonceStore
	keys   map[Provider]*keyCache
}

func New(options Options) *Verifier {
	if options.Clock == nil {
		options.Clock = systemClock{}
	}
	if options.HTTPClient == nil {
		options.HTTPClient = &http.Client{}
	}
	if options.Nonces == nil {
		options.Nonces = NewMemoryNonces()
	}
	config := options.Config
	return &Verifier{
		config: config,
		clock:  options.Clock,
		nonces: options.Nonces,
		keys: map[Provider]*keyCache{
			ProviderGoogle: newKeyCache(config.Google.KeySetURL, config, options.HTTPClient),
			ProviderApple:  newKeyCache(config.Apple.KeySetURL, config, options.HTTPClient),
		},
	}
}

func (v *Verifier) IssueNonce(ctx context.Context) (string, error) {
	nonce := rand.Text()
	if err := v.nonces.Save(ctx, nonce, v.clock.Now(), v.config.NonceTTL); err != nil {
		return "", err
	}
	return nonce, nil
}

func (v *Verifier) Verify(ctx context.Context, provider Provider, token, rawNonce string) (Identity, error) {
	providerConfig, ok := v.providerConfig(provider)
	if !ok {
		return Identity{}, ErrUnknownProvider
	}
	if len(token) > v.config.MaxTokenBytes {
		return Identity{}, ErrMalformed
	}
	parsed, err := parse(token)
	if err != nil {
		return Identity{}, err
	}
	now := v.clock.Now()
	key, err := v.keys[provider].lookup(ctx, parsed.header.KeyID, now)
	if err != nil {
		return Identity{}, err
	}
	if err := parsed.verifySignature(key); err != nil {
		return Identity{}, err
	}
	claims, err := decodeClaims(parsed.payload)
	if err != nil {
		return Identity{}, err
	}
	if err := claims.validate(providerConfig, now, v.config.ClockSkew); err != nil {
		return Identity{}, err
	}
	if err := v.consumeNonce(ctx, providerConfig.NonceForm, claims.Nonce, rawNonce, now); err != nil {
		return Identity{}, err
	}
	return claims.identity(provider), nil
}

func (v *Verifier) providerConfig(provider Provider) (ProviderConfig, bool) {
	switch provider {
	case ProviderGoogle:
		return v.config.Google, true
	case ProviderApple:
		return v.config.Apple, true
	}
	return ProviderConfig{}, false
}

func (v *Verifier) consumeNonce(ctx context.Context, form NonceForm, claimed, rawNonce string, now time.Time) error {
	expected, ok := form.encode(rawNonce)
	if !ok || rawNonce == "" || claimed == "" {
		return ErrNonce
	}
	if subtle.ConstantTimeCompare([]byte(expected), []byte(claimed)) != 1 {
		return ErrNonce
	}
	consumed, err := v.nonces.Consume(ctx, rawNonce, now)
	if err != nil {
		return err
	}
	if !consumed {
		return ErrNonceUsed
	}
	return nil
}

package token

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrNoKeys     = errors.New("token: no signing key")
	ErrWeakKey    = errors.New("token: signing key shorter than 32 bytes")
	ErrBadKeySpec = errors.New("token: key spec is not id:base64")
	ErrMalformed  = errors.New("token: malformed access token")
	ErrSignature  = errors.New("token: bad signature")
	ErrUnknownKey = errors.New("token: unknown signing key")
	ErrExpired    = errors.New("token: access token expired")
	ErrIssuer     = errors.New("token: issuer not trusted")
	ErrUse        = errors.New("token: not valid for this use")
)

const (
	Issuer          = "hypedomino"
	MinKeyBytes     = 32
	algorithm       = "HS256"
	maxTokenSize    = 4096
	requestKeyLabel = "hd-request-mac"
)

var segment = base64.RawURLEncoding.Strict()

type Key struct {
	ID     string
	Secret []byte
}

func ParseKeys(spec string) ([]Key, error) {
	var keys []Key
	seen := map[string]bool{}
	for _, item := range strings.Split(spec, ",") {
		if item = strings.TrimSpace(item); item == "" {
			continue
		}
		id, encoded, ok := strings.Cut(item, ":")
		if !ok || id == "" {
			return nil, fmt.Errorf("%w: %q", ErrBadKeySpec, id)
		}
		secret, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("%w: key %q", ErrBadKeySpec, id)
		}
		if len(secret) < MinKeyBytes {
			return nil, fmt.Errorf("%w: key %q", ErrWeakKey, id)
		}
		if seen[id] {
			return nil, fmt.Errorf("%w: key %q listed twice", ErrBadKeySpec, id)
		}
		seen[id] = true
		keys = append(keys, Key{ID: id, Secret: secret})
	}
	if len(keys) == 0 {
		return nil, ErrNoKeys
	}
	return keys, nil
}

type Claims struct {
	Issuer    string `json:"iss"`
	Account   string `json:"sub"`
	Session   string `json:"sid"`
	Device    string `json:"dev"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
	Use       string `json:"use,omitempty"`
	Sanction  int64  `json:"san,omitempty"`
	KeyID     string `json:"-"`
}

type header struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
	KeyID     string `json:"kid"`
}

type Signer struct {
	keys map[string]Key
	sign Key
	skew time.Duration
}

func NewSigner(keys []Key, skew time.Duration) (*Signer, error) {
	if len(keys) == 0 {
		return nil, ErrNoKeys
	}
	s := &Signer{keys: map[string]Key{}, sign: keys[0], skew: skew}
	for _, key := range keys {
		if len(key.Secret) < MinKeyBytes {
			return nil, fmt.Errorf("%w: key %q", ErrWeakKey, key.ID)
		}
		s.keys[key.ID] = key
	}
	return s, nil
}

func (s *Signer) Sign(claims Claims) (string, error) {
	head, err := json.Marshal(header{Algorithm: algorithm, Type: "JWT", KeyID: s.sign.ID})
	if err != nil {
		return "", err
	}
	body, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	input := segment.EncodeToString(head) + "." + segment.EncodeToString(body)
	return input + "." + segment.EncodeToString(mac(s.sign.Secret, input)), nil
}

func (s *Signer) Verify(token string, now time.Time) (Claims, error) {
	claims, err := s.verify(token, now)
	if err != nil {
		return Claims{}, err
	}
	if claims.Session == "" || claims.Use != "" {
		return Claims{}, ErrMalformed
	}
	return claims, nil
}

func (s *Signer) verify(token string, now time.Time) (Claims, error) {
	if len(token) > maxTokenSize {
		return Claims{}, ErrMalformed
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrMalformed
	}
	headBytes, err := segment.DecodeString(parts[0])
	if err != nil {
		return Claims{}, ErrMalformed
	}
	var head header
	if json.Unmarshal(headBytes, &head) != nil || head.Algorithm != algorithm {
		return Claims{}, ErrMalformed
	}
	key, ok := s.keys[head.KeyID]
	if !ok {
		return Claims{}, ErrUnknownKey
	}
	signature, err := segment.DecodeString(parts[2])
	if err != nil || !hmac.Equal(signature, mac(key.Secret, parts[0]+"."+parts[1])) {
		return Claims{}, ErrSignature
	}
	body, err := segment.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrMalformed
	}
	var claims Claims
	if json.Unmarshal(body, &claims) != nil || claims.Account == "" {
		return Claims{}, ErrMalformed
	}
	claims.KeyID = head.KeyID
	if claims.Issuer != Issuer {
		return Claims{}, ErrIssuer
	}
	if !now.Add(-s.skew).Before(time.Unix(claims.ExpiresAt, 0)) {
		return Claims{}, ErrExpired
	}
	return claims, nil
}

func mac(secret []byte, input string) []byte {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(input))
	return h.Sum(nil)
}

func (s *Signer) RequestKey(claims Claims) ([]byte, error) {
	key, ok := s.keys[claims.KeyID]
	if !ok {
		return nil, ErrUnknownKey
	}
	return requestKey(key.Secret, claims.Session), nil
}

func requestKey(secret []byte, session string) []byte {
	return mac(secret, requestKeyLabel+"\x00"+session)
}

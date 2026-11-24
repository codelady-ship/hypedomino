package googleauth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	Grant     = "urn:ietf:params:oauth:grant-type:jwt-bearer"
	tokenLife = time.Hour
	slack     = time.Minute
	maxBody   = 64 * 1024
)

var ErrUnavailable = errors.New("googleauth: token endpoint unavailable")

type serviceAccount struct {
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
	TokenURI    string `json:"token_uri"`
}

type Source struct {
	scope    string
	client   *http.Client
	now      func() time.Time
	email    string
	key      *rsa.PrivateKey
	tokenURI string
	mu       sync.Mutex
	token    string
	expires  time.Time
}

func Load(file, scope string, client *http.Client, now func() time.Time) (*Source, error) {
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var account serviceAccount
	if err := json.Unmarshal(raw, &account); err != nil {
		return nil, fmt.Errorf("service account: %w", err)
	}
	key, err := parsePrivateKey(account.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("service account: %w", err)
	}
	if account.ClientEmail == "" || account.TokenURI == "" {
		return nil, errors.New("service account: client_email and token_uri are required")
	}
	return &Source{scope: scope, client: client, now: now, email: account.ClientEmail, key: key, tokenURI: account.TokenURI}, nil
}

func parsePrivateKey(encoded string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(encoded))
	if block == nil {
		return nil, errors.New("private_key is not PEM")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rsaKey, ok := key.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}
		return nil, errors.New("private_key is not RSA")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func (s *Source) Token(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	if s.token != "" && now.Add(slack).Before(s.expires) {
		return s.token, nil
	}
	assertion, err := s.assertion(now)
	if err != nil {
		return "", err
	}
	form := url.Values{"grant_type": {Grant}, "assertion": {assertion}}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.tokenURI, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := s.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(response.Body, maxBody))
	if err != nil || response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: answered %d", ErrUnavailable, response.StatusCode)
	}
	var issued struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(payload, &issued); err != nil || issued.AccessToken == "" {
		return "", fmt.Errorf("%w: answered no token", ErrUnavailable)
	}
	s.token = issued.AccessToken
	s.expires = now.Add(time.Duration(issued.ExpiresIn) * time.Second)
	return s.token, nil
}

func (s *Source) assertion(now time.Time) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	claims, err := json.Marshal(map[string]any{"iss": s.email, "scope": s.scope, "aud": s.tokenURI, "iat": now.Unix(), "exp": now.Add(tokenLife).Unix()})
	if err != nil {
		return "", err
	}
	signingInput := header + "." + base64.RawURLEncoding.EncodeToString(claims)
	digest := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, s.key, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

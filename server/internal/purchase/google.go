package purchase

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
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DefaultGoogleEndpoint = "https://androidpublisher.googleapis.com"
	googleScope           = "https://www.googleapis.com/auth/androidpublisher"
	googleGrant           = "urn:ietf:params:oauth:grant-type:jwt-bearer"
	googleTokenLife       = time.Hour
	googleTokenSlack      = time.Minute
	googleMaxBody         = 64 * 1024
)

type GoogleConfig struct {
	PackageName        string
	ServiceAccountFile string
	AllowTest          bool
	Endpoint           string
	Client             *http.Client
	Now                func() time.Time
}

type serviceAccount struct {
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
	TokenURI    string `json:"token_uri"`
}

type Google struct {
	pkg       string
	allowTest bool
	endpoint  string
	client    *http.Client
	now       func() time.Time
	email     string
	key       *rsa.PrivateKey
	tokenURI  string
	mu        sync.Mutex
	token     string
	expires   time.Time
}

func NewGoogle(config GoogleConfig) (*Google, error) {
	if config.PackageName == "" || config.ServiceAccountFile == "" {
		return nil, fmt.Errorf("%w: package name and service account are required", ErrMalformed)
	}
	raw, err := os.ReadFile(config.ServiceAccountFile)
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
	g := &Google{pkg: config.PackageName, allowTest: config.AllowTest, endpoint: config.Endpoint, client: config.Client, now: config.Now, email: account.ClientEmail, key: key, tokenURI: account.TokenURI}
	if g.endpoint == "" {
		g.endpoint = DefaultGoogleEndpoint
	}
	if g.client == nil {
		g.client = &http.Client{Timeout: 10 * time.Second}
	}
	if g.now == nil {
		g.now = time.Now
	}
	return g, nil
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

type productPurchase struct {
	PurchaseState        int    `json:"purchaseState"`
	OrderID              string `json:"orderId"`
	PurchaseTimeMillis   string `json:"purchaseTimeMillis"`
	ObfuscatedAccountID  string `json:"obfuscatedExternalAccountId"`
	PurchaseType         *int   `json:"purchaseType"`
	ProductID            string `json:"productId"`
	AcknowledgementState int    `json:"acknowledgementState"`
}

func (g *Google) Verify(ctx context.Context, productID, token string) (Receipt, error) {
	if productID == "" || token == "" || strings.ContainsAny(token, "/?#") {
		return Receipt{}, ErrMalformed
	}
	status, body, err := g.call(ctx, http.MethodGet, g.purchaseURL(productID, token), nil)
	if err != nil {
		return Receipt{}, err
	}
	switch {
	case status == http.StatusBadRequest, status == http.StatusNotFound, status == http.StatusGone:
		return Receipt{}, fmt.Errorf("%w: store answered %d", ErrNotFound, status)
	case status != http.StatusOK:
		return Receipt{}, fmt.Errorf("%w: store answered %d", ErrStore, status)
	}
	var purchase productPurchase
	if err := json.Unmarshal(body, &purchase); err != nil {
		return Receipt{}, fmt.Errorf("%w: %v", ErrStore, err)
	}
	environment := EnvironmentProduction
	if purchase.PurchaseType != nil {
		if !g.allowTest {
			return Receipt{}, fmt.Errorf("%w: test purchase", ErrEnvironment)
		}
		environment = EnvironmentSandbox
	}
	state, ok := map[int]State{0: StatePurchased, 1: StateCancelled, 2: StatePending}[purchase.PurchaseState]
	if !ok {
		return Receipt{}, fmt.Errorf("%w: purchase state %d", ErrStore, purchase.PurchaseState)
	}
	millis, _ := strconv.ParseInt(purchase.PurchaseTimeMillis, 10, 64)
	if purchase.ProductID == "" {
		purchase.ProductID = productID
	}
	return Receipt{
		Platform:      PlatformGoogle,
		TransactionID: token,
		OrderID:       purchase.OrderID,
		ProductID:     purchase.ProductID,
		AccountToken:  strings.ToLower(purchase.ObfuscatedAccountID),
		State:         state,
		Environment:   environment,
		PurchasedAt:   time.UnixMilli(millis).UTC(),
	}, nil
}

func (g *Google) Finish(ctx context.Context, receipt Receipt) error {
	status, _, err := g.call(ctx, http.MethodPost, g.purchaseURL(receipt.ProductID, receipt.TransactionID)+":consume", strings.NewReader(""))
	if err != nil {
		return err
	}
	if status != http.StatusOK && status != http.StatusNoContent {
		return fmt.Errorf("%w: consume answered %d", ErrStore, status)
	}
	return nil
}

func (g *Google) purchaseURL(productID, token string) string {
	return g.endpoint + "/androidpublisher/v3/applications/" + url.PathEscape(g.pkg) + "/purchases/products/" + url.PathEscape(productID) + "/tokens/" + url.PathEscape(token)
}

func (g *Google) call(ctx context.Context, method, target string, body io.Reader) (int, []byte, error) {
	token, err := g.accessToken(ctx)
	if err != nil {
		return 0, nil, err
	}
	request, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return 0, nil, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/json")
	response, err := g.client.Do(request)
	if err != nil {
		return 0, nil, fmt.Errorf("%w: %v", ErrStore, err)
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(response.Body, googleMaxBody))
	if err != nil {
		return 0, nil, fmt.Errorf("%w: %v", ErrStore, err)
	}
	return response.StatusCode, payload, nil
}

func (g *Google) accessToken(ctx context.Context) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := g.now()
	if g.token != "" && now.Add(googleTokenSlack).Before(g.expires) {
		return g.token, nil
	}
	assertion, err := g.assertion(now)
	if err != nil {
		return "", err
	}
	form := url.Values{"grant_type": {googleGrant}, "assertion": {assertion}}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, g.tokenURI, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := g.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("%w: token: %v", ErrStore, err)
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(response.Body, googleMaxBody))
	if err != nil || response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: token endpoint answered %d", ErrStore, response.StatusCode)
	}
	var issued struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(payload, &issued); err != nil || issued.AccessToken == "" {
		return "", fmt.Errorf("%w: token endpoint answered no token", ErrStore)
	}
	g.token = issued.AccessToken
	g.expires = now.Add(time.Duration(issued.ExpiresIn) * time.Second)
	return g.token, nil
}

func (g *Google) assertion(now time.Time) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	claims, err := json.Marshal(map[string]any{"iss": g.email, "scope": googleScope, "aud": g.tokenURI, "iat": now.Unix(), "exp": now.Add(googleTokenLife).Unix()})
	if err != nil {
		return "", err
	}
	signingInput := header + "." + base64.RawURLEncoding.EncodeToString(claims)
	digest := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, g.key, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

type pubsubPush struct {
	Message struct {
		Data      string `json:"data"`
		MessageID string `json:"messageId"`
	} `json:"message"`
}

type developerNotification struct {
	PackageName string `json:"packageName"`
	OneTime     *struct {
		NotificationType int    `json:"notificationType"`
		PurchaseToken    string `json:"purchaseToken"`
		SKU              string `json:"sku"`
	} `json:"oneTimeProductNotification"`
	Voided *struct {
		PurchaseToken string `json:"purchaseToken"`
		OrderID       string `json:"orderId"`
	} `json:"voidedPurchaseNotification"`
}

func (g *Google) Notice(body []byte) (Notice, error) {
	var push pubsubPush
	if err := json.Unmarshal(body, &push); err != nil || push.Message.MessageID == "" || push.Message.Data == "" {
		return Notice{}, ErrMalformed
	}
	data, err := base64.StdEncoding.DecodeString(push.Message.Data)
	if err != nil {
		return Notice{}, ErrMalformed
	}
	var notification developerNotification
	if err := json.Unmarshal(data, &notification); err != nil {
		return Notice{}, ErrMalformed
	}
	if notification.PackageName != g.pkg {
		return Notice{}, fmt.Errorf("%w: package %q", ErrApp, notification.PackageName)
	}
	notice := Notice{ID: push.Message.MessageID, Kind: KindOther, Environment: EnvironmentProduction}
	switch {
	case notification.Voided != nil:
		notice.Kind, notice.TransactionID = KindRefund, notification.Voided.PurchaseToken
	case notification.OneTime != nil && notification.OneTime.NotificationType == 1:
		notice.Kind, notice.TransactionID, notice.ProductID = KindPurchased, notification.OneTime.PurchaseToken, notification.OneTime.SKU
	case notification.OneTime != nil:
		notice.TransactionID, notice.ProductID = notification.OneTime.PurchaseToken, notification.OneTime.SKU
	}
	return notice, nil
}

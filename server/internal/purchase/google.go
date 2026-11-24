package purchase

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/googleauth"
)

const (
	DefaultGoogleEndpoint = "https://androidpublisher.googleapis.com"
	googleScope           = "https://www.googleapis.com/auth/androidpublisher"
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

type Google struct {
	pkg       string
	allowTest bool
	endpoint  string
	client    *http.Client
	now       func() time.Time
	source    *googleauth.Source
}

func NewGoogle(config GoogleConfig) (*Google, error) {
	if config.PackageName == "" || config.ServiceAccountFile == "" {
		return nil, fmt.Errorf("%w: package name and service account are required", ErrMalformed)
	}
	g := &Google{pkg: config.PackageName, allowTest: config.AllowTest, endpoint: config.Endpoint, client: config.Client, now: config.Now}
	if g.endpoint == "" {
		g.endpoint = DefaultGoogleEndpoint
	}
	if g.client == nil {
		g.client = &http.Client{Timeout: 10 * time.Second}
	}
	if g.now == nil {
		g.now = time.Now
	}
	source, err := googleauth.Load(config.ServiceAccountFile, googleScope, g.client, g.now)
	if err != nil {
		return nil, err
	}
	g.source = source
	return g, nil
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
	token, err := g.source.Token(ctx)
	if err != nil {
		return 0, nil, fmt.Errorf("%w: %v", ErrStore, err)
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

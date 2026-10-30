package purchase

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	_ "embed"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"
)

//go:embed apple_root_g3.cer
var appleRootDER []byte

var appleReceiptSigning = asn1.ObjectIdentifier{1, 2, 840, 113635, 100, 6, 11, 1}

const (
	appleEnvironmentProduction = "Production"
	appleEnvironmentSandbox    = "Sandbox"
)

type AppleConfig struct {
	BundleID     string
	AllowSandbox bool
	Roots        *x509.CertPool
	Now          func() time.Time
}

type Apple struct {
	bundle  string
	sandbox bool
	roots   *x509.CertPool
	now     func() time.Time
}

func NewApple(config AppleConfig) (*Apple, error) {
	if config.BundleID == "" {
		return nil, fmt.Errorf("%w: bundle id is required", ErrMalformed)
	}
	roots := config.Roots
	if roots == nil {
		root, err := x509.ParseCertificate(appleRootDER)
		if err != nil {
			return nil, err
		}
		roots = x509.NewCertPool()
		roots.AddCert(root)
	}
	now := config.Now
	if now == nil {
		now = time.Now
	}
	return &Apple{bundle: config.BundleID, sandbox: config.AllowSandbox, roots: roots, now: now}, nil
}

type appleTransaction struct {
	TransactionID         string `json:"transactionId"`
	OriginalTransactionID string `json:"originalTransactionId"`
	BundleID              string `json:"bundleId"`
	ProductID             string `json:"productId"`
	PurchaseDate          int64  `json:"purchaseDate"`
	RevocationDate        int64  `json:"revocationDate"`
	Environment           string `json:"environment"`
	AppAccountToken       string `json:"appAccountToken"`
}

func (a *Apple) Verify(_ context.Context, _ string, token string) (Receipt, error) {
	payload, err := a.decode(token)
	if err != nil {
		return Receipt{}, err
	}
	var tx appleTransaction
	if err := json.Unmarshal(payload, &tx); err != nil || tx.TransactionID == "" || tx.ProductID == "" {
		return Receipt{}, ErrMalformed
	}
	if tx.BundleID != a.bundle {
		return Receipt{}, fmt.Errorf("%w: bundle %q", ErrApp, tx.BundleID)
	}
	environment, err := a.environment(tx.Environment)
	if err != nil {
		return Receipt{}, err
	}
	state := StatePurchased
	if tx.RevocationDate > 0 {
		state = StateCancelled
	}
	return Receipt{
		Platform:      PlatformApple,
		TransactionID: tx.TransactionID,
		OrderID:       tx.OriginalTransactionID,
		ProductID:     tx.ProductID,
		AccountToken:  strings.ToLower(tx.AppAccountToken),
		State:         state,
		Environment:   environment,
		PurchasedAt:   time.UnixMilli(tx.PurchaseDate).UTC(),
	}, nil
}

func (a *Apple) Finish(context.Context, Receipt) error {
	return nil
}

type appleNotification struct {
	Type string `json:"notificationType"`
	UUID string `json:"notificationUUID"`
	Data struct {
		BundleID              string `json:"bundleId"`
		Environment           string `json:"environment"`
		SignedTransactionInfo string `json:"signedTransactionInfo"`
	} `json:"data"`
}

var appleKinds = map[string]Kind{"REFUND": KindRefund, "REFUND_REVERSED": KindReversal, "ONE_TIME_CHARGE": KindPurchased}

func (a *Apple) Notice(signedPayload []byte) (Notice, error) {
	payload, err := a.decode(string(signedPayload))
	if err != nil {
		return Notice{}, err
	}
	var notification appleNotification
	if err := json.Unmarshal(payload, &notification); err != nil || notification.UUID == "" || notification.Type == "" {
		return Notice{}, ErrMalformed
	}
	if notification.Data.BundleID != a.bundle {
		return Notice{}, fmt.Errorf("%w: bundle %q", ErrApp, notification.Data.BundleID)
	}
	environment, err := a.environment(notification.Data.Environment)
	if err != nil {
		return Notice{}, err
	}
	notice := Notice{ID: notification.UUID, Kind: KindOther, Environment: environment}
	if kind, known := appleKinds[notification.Type]; known {
		notice.Kind = kind
	}
	if notification.Data.SignedTransactionInfo == "" {
		return notice, nil
	}
	receipt, err := a.Verify(context.Background(), "", notification.Data.SignedTransactionInfo)
	if err != nil {
		return Notice{}, err
	}
	notice.TransactionID, notice.ProductID = receipt.TransactionID, receipt.ProductID
	return notice, nil
}

func (a *Apple) environment(value string) (Environment, error) {
	switch {
	case value == appleEnvironmentProduction:
		return EnvironmentProduction, nil
	case value == appleEnvironmentSandbox && a.sandbox:
		return EnvironmentSandbox, nil
	case value == appleEnvironmentSandbox:
		return "", fmt.Errorf("%w: sandbox", ErrEnvironment)
	}
	return "", fmt.Errorf("%w: environment %q", ErrMalformed, value)
}

type jwsHeader struct {
	Algorithm string   `json:"alg"`
	Chain     []string `json:"x5c"`
}

func (a *Apple) decode(token string) ([]byte, error) {
	segments := strings.Split(token, ".")
	if len(segments) != 3 {
		return nil, ErrMalformed
	}
	rawHeader, err := base64.RawURLEncoding.DecodeString(segments[0])
	if err != nil {
		return nil, ErrMalformed
	}
	var header jwsHeader
	if err := json.Unmarshal(rawHeader, &header); err != nil || header.Algorithm != "ES256" || len(header.Chain) < 2 {
		return nil, ErrMalformed
	}
	payload, err := base64.RawURLEncoding.DecodeString(segments[1])
	if err != nil {
		return nil, ErrMalformed
	}
	signature, err := base64.RawURLEncoding.DecodeString(segments[2])
	if err != nil || len(signature) != 64 {
		return nil, ErrMalformed
	}
	leaf, err := a.verifyChain(header.Chain)
	if err != nil {
		return nil, err
	}
	key, ok := leaf.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, ErrSignature
	}
	digest := sha256.Sum256([]byte(segments[0] + "." + segments[1]))
	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:])
	if !ecdsa.Verify(key, digest[:], r, s) {
		return nil, ErrSignature
	}
	return payload, nil
}

func (a *Apple) verifyChain(chain []string) (*x509.Certificate, error) {
	certs := make([]*x509.Certificate, 0, len(chain))
	for _, encoded := range chain {
		der, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, ErrMalformed
		}
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			return nil, ErrMalformed
		}
		certs = append(certs, cert)
	}
	intermediates := x509.NewCertPool()
	for _, cert := range certs[1:] {
		intermediates.AddCert(cert)
	}
	leaf := certs[0]
	options := x509.VerifyOptions{Roots: a.roots, Intermediates: intermediates, CurrentTime: a.now(), KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny}}
	if _, err := leaf.Verify(options); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSignature, err)
	}
	for _, extension := range leaf.Extensions {
		if extension.Id.Equal(appleReceiptSigning) {
			return leaf, nil
		}
	}
	return nil, fmt.Errorf("%w: not a receipt signing certificate", ErrSignature)
}

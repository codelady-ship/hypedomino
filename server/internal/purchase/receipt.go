package purchase

import (
	"context"
	"errors"
	"time"
)

type Platform string

const (
	PlatformGoogle Platform = "google"
	PlatformApple  Platform = "apple"
)

type State string

const (
	StatePurchased State = "purchased"
	StatePending   State = "pending"
	StateCancelled State = "cancelled"
)

type Environment string

const (
	EnvironmentProduction Environment = "production"
	EnvironmentSandbox    Environment = "sandbox"
)

type Receipt struct {
	Platform      Platform
	TransactionID string
	OrderID       string
	ProductID     string
	AccountToken  string
	State         State
	Environment   Environment
	PurchasedAt   time.Time
}

type Validator interface {
	Verify(ctx context.Context, productID, token string) (Receipt, error)
	Finish(ctx context.Context, receipt Receipt) error
}

type Kind string

const (
	KindPurchased Kind = "purchased"
	KindRefund    Kind = "refund"
	KindReversal  Kind = "reversal"
	KindOther     Kind = "other"
)

type Notice struct {
	ID            string
	Kind          Kind
	TransactionID string
	ProductID     string
	Environment   Environment
}

var (
	ErrMalformed   = errors.New("receipt.malformed")
	ErrSignature   = errors.New("receipt.signature")
	ErrApp         = errors.New("receipt.app")
	ErrEnvironment = errors.New("receipt.environment")
	ErrNotFound    = errors.New("receipt.unknown")
	ErrStore       = errors.New("receipt.store_unavailable")
)

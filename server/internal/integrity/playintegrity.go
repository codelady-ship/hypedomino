package integrity

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	playRecognized       = "PLAY_RECOGNIZED"
	licensed             = "LICENSED"
	meetsBasicIntegrity  = "MEETS_BASIC_INTEGRITY"
	meetsDeviceIntegrity = "MEETS_DEVICE_INTEGRITY"
	meetsStrongIntegrity = "MEETS_STRONG_INTEGRITY"
)

var ErrTokenRejected = errors.New("integrity: token rejected by the attestation service")

type PlayIntegrityPayload struct {
	RequestDetails  RequestDetails  `json:"requestDetails"`
	AppIntegrity    AppIntegrity    `json:"appIntegrity"`
	DeviceIntegrity DeviceIntegrity `json:"deviceIntegrity"`
	AccountDetails  AccountDetails  `json:"accountDetails"`
}

type RequestDetails struct {
	RequestPackageName string `json:"requestPackageName"`
	Nonce              string `json:"nonce"`
	RequestHash        string `json:"requestHash"`
	TimestampMillis    string `json:"timestampMillis"`
}

type AppIntegrity struct {
	AppRecognitionVerdict   string   `json:"appRecognitionVerdict"`
	PackageName             string   `json:"packageName"`
	CertificateSha256Digest []string `json:"certificateSha256Digest"`
	VersionCode             string   `json:"versionCode"`
}

type DeviceIntegrity struct {
	DeviceRecognitionVerdict []string `json:"deviceRecognitionVerdict"`
}

type AccountDetails struct {
	AppLicensingVerdict string `json:"appLicensingVerdict"`
}

type Decoder interface {
	Decode(ctx context.Context, token string) (PlayIntegrityPayload, error)
}

type PlayIntegrityConfig struct {
	PackageName        string
	CertificateDigests []string
	TokenMaxAge        time.Duration
}

type PlayIntegrity struct {
	config  PlayIntegrityConfig
	digests map[string]bool
	decoder Decoder
	nonces  *Nonces
	clock   Clock
}

func NewPlayIntegrity(config PlayIntegrityConfig, decoder Decoder, nonces *Nonces, clock Clock) (*PlayIntegrity, error) {
	switch {
	case config.PackageName == "":
		return nil, errors.New("integrity: package name is required")
	case len(config.CertificateDigests) == 0:
		return nil, errors.New("integrity: signing certificate allowlist is empty")
	case config.TokenMaxAge <= 0:
		return nil, errors.New("integrity: token max age must be positive")
	case decoder == nil || nonces == nil || clock == nil:
		return nil, errors.New("integrity: decoder, nonces and clock are required")
	}
	digests := map[string]bool{}
	for _, encoded := range config.CertificateDigests {
		digest, err := decodeDigest(encoded)
		if err != nil || len(digest) != sha256.Size {
			return nil, fmt.Errorf("integrity: invalid signing certificate digest %q", encoded)
		}
		digests[string(digest)] = true
	}
	return &PlayIntegrity{config: config, digests: digests, decoder: decoder, nonces: nonces, clock: clock}, nil
}

func (p *PlayIntegrity) Check(ctx context.Context, policy Policy, binding Binding, token string) (Decision, error) {
	if token == "" {
		return decide(policy, binding.Purpose, []Reason{ReasonTokenRejected}), nil
	}
	payload, err := p.decoder.Decode(ctx, token)
	if errors.Is(err, ErrTokenRejected) {
		return decide(policy, binding.Purpose, []Reason{ReasonTokenRejected}), nil
	}
	if err != nil {
		return Decision{}, fmt.Errorf("integrity: decode token: %w", err)
	}
	reasons, err := p.evaluate(ctx, policy, binding, payload)
	if err != nil {
		return Decision{}, err
	}
	return decide(policy, binding.Purpose, reasons), nil
}

func (p *PlayIntegrity) evaluate(ctx context.Context, policy Policy, binding Binding, payload PlayIntegrityPayload) ([]Reason, error) {
	nonceReason, err := p.nonces.redeem(ctx, requestNonce(payload.RequestDetails), binding)
	if err != nil {
		return nil, err
	}
	var reasons []Reason
	reasons = appendReason(reasons, nonceReason)
	reasons = appendReason(reasons, p.packageReason(payload))
	reasons = appendReason(reasons, p.freshnessReason(payload.RequestDetails.TimestampMillis))
	if payload.AppIntegrity.AppRecognitionVerdict != playRecognized {
		reasons = append(reasons, ReasonUnrecognizedApp)
	}
	if !p.allowedCertificates(payload.AppIntegrity.CertificateSha256Digest) {
		reasons = append(reasons, ReasonUnknownSigningCertificate)
	}
	if deviceLevel(payload.DeviceIntegrity.DeviceRecognitionVerdict) < policy.MinDeviceLevel {
		reasons = append(reasons, ReasonWeakDevice)
	}
	if policy.RequireLicensed && payload.AccountDetails.AppLicensingVerdict != licensed {
		reasons = append(reasons, ReasonUnlicensed)
	}
	return reasons, nil
}

func requestNonce(details RequestDetails) string {
	if details.Nonce != "" {
		return details.Nonce
	}
	return details.RequestHash
}

func (p *PlayIntegrity) packageReason(payload PlayIntegrityPayload) Reason {
	if payload.RequestDetails.RequestPackageName != p.config.PackageName || payload.AppIntegrity.PackageName != p.config.PackageName {
		return ReasonPackageMismatch
	}
	return 0
}

func (p *PlayIntegrity) freshnessReason(timestampMillis string) Reason {
	millis, err := strconv.ParseInt(timestampMillis, 10, 64)
	if err != nil {
		return ReasonTokenMalformed
	}
	age := p.clock.Now().Sub(time.UnixMilli(millis))
	if age > p.config.TokenMaxAge || age < -p.config.TokenMaxAge {
		return ReasonTokenStale
	}
	return 0
}

func (p *PlayIntegrity) allowedCertificates(encodedDigests []string) bool {
	if len(encodedDigests) == 0 {
		return false
	}
	for _, encoded := range encodedDigests {
		digest, err := decodeDigest(encoded)
		if err != nil || !p.digests[string(digest)] {
			return false
		}
	}
	return true
}

func decodeDigest(encoded string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(strings.TrimRight(encoded, "="))
}

func deviceLevel(labels []string) DeviceLevel {
	level := DeviceLevelNone
	for _, label := range labels {
		level = max(level, labelLevel(label))
	}
	return level
}

func labelLevel(label string) DeviceLevel {
	switch label {
	case meetsBasicIntegrity:
		return DeviceLevelBasic
	case meetsDeviceIntegrity:
		return DeviceLevelDevice
	case meetsStrongIntegrity:
		return DeviceLevelStrong
	}
	return DeviceLevelNone
}

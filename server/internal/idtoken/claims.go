package idtoken

import (
	"encoding/json"
	"errors"
	"slices"
	"time"
)

var errClaimType = errors.New("idtoken: claim has the wrong type")

type numericDate struct {
	seconds float64
	present bool
}

func (d *numericDate) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var seconds float64
	if err := json.Unmarshal(data, &seconds); err != nil {
		return err
	}
	d.seconds, d.present = seconds, true
	return nil
}

type flexibleBool bool

func (b *flexibleBool) UnmarshalJSON(data []byte) error {
	switch string(data) {
	case "true", `"true"`:
		*b = true
	case "false", `"false"`, "null":
		*b = false
	default:
		return errClaimType
	}
	return nil
}

type audience []string

func (a *audience) UnmarshalJSON(data []byte) error {
	var single string
	if json.Unmarshal(data, &single) == nil {
		*a = audience{single}
		return nil
	}
	var many []string
	if err := json.Unmarshal(data, &many); err != nil {
		return err
	}
	*a = many
	return nil
}

type claims struct {
	Issuer        string       `json:"iss"`
	Subject       string       `json:"sub"`
	Audience      audience     `json:"aud"`
	ExpiresAt     numericDate  `json:"exp"`
	IssuedAt      numericDate  `json:"iat"`
	NotBefore     numericDate  `json:"nbf"`
	Nonce         string       `json:"nonce"`
	Email         string       `json:"email"`
	EmailVerified flexibleBool `json:"email_verified"`
	PrivateEmail  flexibleBool `json:"is_private_email"`
	HostedDomain  string       `json:"hd"`
}

func decodeClaims(payload []byte) (claims, error) {
	var decoded claims
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return claims{}, ErrMalformed
	}
	if !decoded.ExpiresAt.present || !decoded.IssuedAt.present {
		return claims{}, ErrMalformed
	}
	return decoded, nil
}

func (c claims) validate(config ProviderConfig, now time.Time, skew time.Duration) error {
	if !slices.Contains(config.Issuers, c.Issuer) {
		return ErrIssuer
	}
	if !trustedAudience(c.Audience, config.Audiences) {
		return ErrAudience
	}
	nowSeconds := float64(now.UnixNano()) / float64(time.Second)
	skewSeconds := skew.Seconds()
	if c.ExpiresAt.seconds+skewSeconds <= nowSeconds {
		return ErrExpired
	}
	if c.IssuedAt.seconds-skewSeconds > nowSeconds {
		return ErrNotYetValid
	}
	if c.NotBefore.present && c.NotBefore.seconds-skewSeconds > nowSeconds {
		return ErrNotYetValid
	}
	if c.Subject == "" {
		return ErrSubject
	}
	return nil
}

func trustedAudience(claimed audience, trusted []string) bool {
	if len(claimed) == 0 {
		return false
	}
	for _, value := range claimed {
		if value == "" || !slices.Contains(trusted, value) {
			return false
		}
	}
	return true
}

func (c claims) identity(provider Provider) Identity {
	return Identity{
		Provider:      provider,
		Subject:       c.Subject,
		Email:         c.Email,
		EmailVerified: bool(c.EmailVerified),
		PrivateEmail:  bool(c.PrivateEmail),
		HostedDomain:  c.HostedDomain,
	}
}

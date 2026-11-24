package promo

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
)

const (
	Alphabet      = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	MinCodeLength = 4
	MaxCodeLength = 32
	MinKeyLength  = 16
)

var (
	ErrCodeFormat = errors.New("promo.code_invalid")
	ErrKey        = errors.New("promo: key must be at least 16 bytes")
)

func Normalize(code string) (string, error) {
	var out strings.Builder
	for _, r := range strings.ToUpper(code) {
		switch {
		case r == ' ' || r == '-' || r == '_':
			continue
		case !strings.ContainsRune(Alphabet, r):
			return "", fmt.Errorf("%w: %q", ErrCodeFormat, r)
		}
		out.WriteRune(r)
	}
	if out.Len() < MinCodeLength || out.Len() > MaxCodeLength {
		return "", fmt.Errorf("%w: length %d", ErrCodeFormat, out.Len())
	}
	return out.String(), nil
}

func Generate(length int) (string, error) {
	if length < MinCodeLength || length > MaxCodeLength {
		return "", fmt.Errorf("%w: length %d", ErrCodeFormat, length)
	}
	raw := make([]byte, length*2)
	code := make([]byte, 0, length)
	for len(code) < length {
		if _, err := rand.Read(raw); err != nil {
			return "", err
		}
		for _, b := range raw {
			if int(b) < 256-256%len(Alphabet) {
				code = append(code, Alphabet[int(b)%len(Alphabet)])
			}
			if len(code) == length {
				break
			}
		}
	}
	return string(code), nil
}

type Hasher struct {
	key []byte
}

func NewHasher(key string) (Hasher, error) {
	if len(key) < MinKeyLength {
		return Hasher{}, ErrKey
	}
	return Hasher{key: []byte(key)}, nil
}

func (h Hasher) Hash(normalized string) []byte {
	mac := hmac.New(sha256.New, h.key)
	mac.Write([]byte(normalized))
	return mac.Sum(nil)
}

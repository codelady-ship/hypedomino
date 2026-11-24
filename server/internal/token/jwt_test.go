package token

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"
)

var start = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

func keySpec(id string, fill byte) string {
	return id + ":" + base64.StdEncoding.EncodeToString([]byte(strings.Repeat(string(fill), MinKeyBytes)))
}

func signer(t *testing.T, spec string) *Signer {
	t.Helper()
	keys, err := ParseKeys(spec)
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSigner(keys, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func claimsAt(now time.Time) Claims {
	return Claims{Issuer: Issuer, Account: "acc-1", Session: "sid-1", Device: "dev-1", IssuedAt: now.Unix(), ExpiresAt: now.Add(15 * time.Minute).Unix()}
}

func TestSignedTokensVerifyUntilTheyExpire(t *testing.T) {
	s := signer(t, keySpec("k1", 'a'))
	token, err := s.Sign(claimsAt(start))
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.Verify(token, start.Add(14*time.Minute))
	want := claimsAt(start)
	want.KeyID = "k1"
	if err != nil || got != want {
		t.Fatalf("verify: %+v %v", got, err)
	}
	if _, err := s.Verify(token, start.Add(15*time.Minute+30*time.Second)); err != nil {
		t.Fatalf("inside the clock skew: %v", err)
	}
	if _, err := s.Verify(token, start.Add(17*time.Minute)); !errors.Is(err, ErrExpired) {
		t.Fatalf("after expiry: %v", err)
	}
}

func TestTheFirstKeySignsAndEveryKeyVerifies(t *testing.T) {
	old := signer(t, keySpec("k1", 'a'))
	token, err := old.Sign(claimsAt(start))
	if err != nil {
		t.Fatal(err)
	}
	rotated := signer(t, keySpec("k2", 'b')+","+keySpec("k1", 'a'))
	if _, err := rotated.Verify(token, start); err != nil {
		t.Fatalf("a token of the previous key: %v", err)
	}
	fresh, err := rotated.Sign(claimsAt(start))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := old.Verify(fresh, start); !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("a token of an unknown key: %v", err)
	}
}

func TestTamperedAndForgedTokensAreRefused(t *testing.T) {
	s := signer(t, keySpec("k1", 'a'))
	token, err := s.Sign(claimsAt(start))
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(token, ".")
	forgedBody := segment.EncodeToString([]byte(`{"iss":"hypedomino","sub":"acc-2","sid":"sid-1","iat":1,"exp":9999999999}`))
	noneHeader := segment.EncodeToString([]byte(`{"alg":"none","typ":"JWT","kid":"k1"}`))
	other := signer(t, keySpec("k1", 'z'))
	otherToken, _ := other.Sign(claimsAt(start))
	cases := map[string]error{
		parts[0] + "." + forgedBody + "." + parts[2]: ErrSignature,
		noneHeader + "." + parts[1] + ".":            ErrMalformed,
		noneHeader + "." + parts[1] + "." + parts[2]: ErrMalformed,
		otherToken:                          ErrSignature,
		"not.a.token":                       ErrMalformed,
		"":                                  ErrMalformed,
		parts[0] + "." + parts[1]:           ErrMalformed,
		strings.Repeat("x", maxTokenSize+1): ErrMalformed,
		parts[0] + "." + parts[1] + "." + parts[2] + "xx": ErrSignature,
	}
	for input, want := range cases {
		if _, err := s.Verify(input, start); !errors.Is(err, want) {
			t.Fatalf("%.40q: %v, want %v", input, err, want)
		}
	}
	foreign, _ := s.Sign(Claims{Issuer: "other", Account: "a", Session: "s", ExpiresAt: start.Add(time.Hour).Unix()})
	if _, err := s.Verify(foreign, start); !errors.Is(err, ErrIssuer) {
		t.Fatalf("foreign issuer: %v", err)
	}
	anonymous, _ := s.Sign(Claims{Issuer: Issuer, ExpiresAt: start.Add(time.Hour).Unix()})
	if _, err := s.Verify(anonymous, start); !errors.Is(err, ErrMalformed) {
		t.Fatalf("token without a subject: %v", err)
	}
}

func TestKeySpecsAreValidated(t *testing.T) {
	cases := map[string]error{
		"":                              ErrNoKeys,
		" , ":                           ErrNoKeys,
		"k1":                            ErrBadKeySpec,
		":" + strings.Repeat("YQ==", 1): ErrBadKeySpec,
		"k1:not base64":                 ErrBadKeySpec,
		"k1:" + base64.StdEncoding.EncodeToString([]byte("short")): ErrWeakKey,
		keySpec("k1", 'a') + "," + keySpec("k1", 'b'):              ErrBadKeySpec,
	}
	for spec, want := range cases {
		if _, err := ParseKeys(spec); !errors.Is(err, want) {
			t.Fatalf("%q: %v, want %v", spec, err, want)
		}
	}
	keys, err := ParseKeys(" " + keySpec("k2", 'b') + " , " + keySpec("k1", 'a') + ",")
	if err != nil || len(keys) != 2 || keys[0].ID != "k2" || keys[1].ID != "k1" {
		t.Fatalf("two keys: %+v %v", keys, err)
	}
	if _, err := NewSigner(nil, 0); !errors.Is(err, ErrNoKeys) {
		t.Fatalf("signer without keys: %v", err)
	}
}

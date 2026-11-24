package token

import (
	"errors"
	"testing"
	"time"
)

func TestAnAppealTokenServesOnlyAppealsUntilItExpires(t *testing.T) {
	s := signer(t, keySpec("k1", 'a'))
	appeal, expires, err := s.SignAppeal("acc-1", 41, start, 30*time.Minute)
	if err != nil || !expires.Equal(start.Add(30*time.Minute)) {
		t.Fatalf("sign: %v %v", expires, err)
	}
	access, err := s.Sign(claimsAt(start))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name    string
		token   string
		at      time.Time
		want    AppealClaims
		wantErr error
	}{
		{"fresh", appeal, start, AppealClaims{Account: "acc-1", Sanction: 41}, nil},
		{"inside the clock skew", appeal, start.Add(30*time.Minute + 30*time.Second), AppealClaims{Account: "acc-1", Sanction: 41}, nil},
		{"expired", appeal, start.Add(32 * time.Minute), AppealClaims{}, ErrExpired},
		{"an access token", access, start, AppealClaims{}, ErrUse},
		{"tampered", appeal + "x", start, AppealClaims{}, ErrSignature},
	}
	for _, c := range cases {
		got, err := s.VerifyAppeal(c.token, c.at)
		if got != c.want || !errors.Is(err, c.wantErr) {
			t.Fatalf("%s: %+v %v", c.name, got, err)
		}
	}
	if _, err := s.Verify(appeal, start); !errors.Is(err, ErrMalformed) {
		t.Fatalf("an appeal token passed as an access token: %v", err)
	}
}

func TestAnAppealTokenNeedsASanction(t *testing.T) {
	s := signer(t, keySpec("k1", 'a'))
	unbound, err := s.Sign(Claims{Issuer: Issuer, Account: "acc-1", Use: AppealUse, IssuedAt: start.Unix(), ExpiresAt: start.Add(time.Minute).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.VerifyAppeal(unbound, start); !errors.Is(err, ErrUse) {
		t.Fatalf("an appeal token without a sanction verified: %v", err)
	}
}

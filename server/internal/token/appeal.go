package token

import "time"

const AppealUse = "appeal"

type AppealClaims struct {
	Account  string
	Sanction int64
}

func (s *Signer) SignAppeal(account string, sanction int64, now time.Time, ttl time.Duration) (string, time.Time, error) {
	expires := now.Add(ttl).Truncate(time.Second)
	signed, err := s.Sign(Claims{Issuer: Issuer, Account: account, Use: AppealUse, Sanction: sanction, IssuedAt: now.Unix(), ExpiresAt: expires.Unix()})
	return signed, expires, err
}

func (s *Signer) VerifyAppeal(token string, now time.Time) (AppealClaims, error) {
	claims, err := s.verify(token, now)
	if err != nil {
		return AppealClaims{}, err
	}
	if claims.Use != AppealUse || claims.Session != "" || claims.Sanction <= 0 {
		return AppealClaims{}, ErrUse
	}
	return AppealClaims{Account: claims.Account, Sanction: claims.Sanction}, nil
}

func (s *Service) IssueAppeal(account string, sanction int64, ttl time.Duration) (string, time.Time, error) {
	return s.signer.SignAppeal(account, sanction, s.now(), ttl)
}

func (s *Service) VerifyAppeal(token string) (AppealClaims, error) {
	return s.signer.VerifyAppeal(token, s.now())
}

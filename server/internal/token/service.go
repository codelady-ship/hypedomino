package token

import (
	"context"
	"time"
)

type Config struct {
	Keys       string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	ReuseGrace time.Duration
	ClockSkew  time.Duration
}

func DefaultConfig() Config {
	return Config{
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 30 * 24 * time.Hour,
		ReuseGrace: 30 * time.Second,
		ClockSkew:  60 * time.Second,
	}
}

type Pair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	Session      string
	RequestKey   []byte
}

type Service struct {
	signer *Signer
	store  *Store
	config Config
	now    func() time.Time
}

func NewService(config Config, store *Store, now func() time.Time) (*Service, error) {
	keys, err := ParseKeys(config.Keys)
	if err != nil {
		return nil, err
	}
	signer, err := NewSigner(keys, config.ClockSkew)
	if err != nil {
		return nil, err
	}
	return &Service{signer: signer, store: store, config: config, now: now}, nil
}

func (s *Service) Issue(ctx context.Context, account, device string) (Pair, error) {
	f, refresh, err := s.store.Open(ctx, account, device, s.now(), s.config.RefreshTTL)
	if err != nil {
		return Pair{}, err
	}
	return s.pair(f, refresh)
}

func (s *Service) Refresh(ctx context.Context, refresh, device string) (Pair, error) {
	f, next, err := s.store.Rotate(ctx, refresh, device, s.now(), s.config.RefreshTTL, s.config.ReuseGrace)
	if err != nil {
		return Pair{}, err
	}
	return s.pair(f, next)
}

func (s *Service) pair(f Family, refresh string) (Pair, error) {
	now := s.now()
	access, err := s.signer.Sign(Claims{
		Issuer:    Issuer,
		Account:   f.Account,
		Session:   f.ID,
		Device:    f.Device,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(s.config.AccessTTL).Unix(),
	})
	if err != nil {
		return Pair{}, err
	}
	return Pair{AccessToken: access, RefreshToken: refresh, ExpiresIn: int(s.config.AccessTTL / time.Second), Session: f.ID, RequestKey: requestKey(s.signer.sign.Secret, f.ID)}, nil
}

func (s *Service) Logout(ctx context.Context, refresh string) error {
	return s.store.Revoke(ctx, refresh, s.now())
}

func (s *Service) LogoutAll(ctx context.Context, account string) (int, error) {
	return s.store.RevokeAccount(ctx, account, s.now())
}

func (s *Service) Verify(access string) (Claims, error) {
	return s.signer.Verify(access, s.now())
}

func (s *Service) Authenticate(access string) (string, string, error) {
	claims, err := s.Verify(access)
	if err != nil {
		return "", "", err
	}
	return claims.Account, claims.Session, nil
}

func (s *Service) RequestKey(claims Claims) ([]byte, error) {
	return s.signer.RequestKey(claims)
}

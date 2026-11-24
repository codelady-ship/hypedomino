package appeal

import (
	"context"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/liveops"
	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
	"github.com/Alievs-corp/hypedomino/server/internal/sanction"
)

type Grant struct {
	SanctionID int64     `json:"sanction_id"`
	Token      string    `json:"token"`
	ExpiresAt  time.Time `json:"expires_at"`
}

func (g *Grant) Proto() *pb.AppealGrant {
	if g == nil {
		return nil
	}
	return &pb.AppealGrant{SanctionId: uint64(g.SanctionID), Token: g.Token, ExpiresUnixMs: uint64(g.ExpiresAt.UnixMilli())}
}

type Signer interface {
	IssueAppeal(account string, sanction int64, ttl time.Duration) (string, time.Time, error)
}

type Granter struct {
	signer   Signer
	settings func() liveops.Appeals
}

func NewGranter(signer Signer, settings func() liveops.Appeals) *Granter {
	return &Granter{signer: signer, settings: settings}
}

func (g *Granter) Grant(account string, sanctionID int64) (*Grant, error) {
	if g == nil || sanctionID <= 0 {
		return nil, nil
	}
	signed, expires, err := g.signer.IssueAppeal(account, sanctionID, g.settings().TokenLifetime)
	if err != nil {
		return nil, err
	}
	return &Grant{SanctionID: sanctionID, Token: signed, ExpiresAt: expires.UTC()}, nil
}

func (g *Granter) Running(ctx context.Context, db sanction.Querier, account string, kind sanction.Kind, now time.Time) (*Grant, error) {
	if g == nil {
		return nil, nil
	}
	running, found, err := sanction.Current(ctx, db, account, now, kind)
	if err != nil || !found {
		return nil, err
	}
	return g.Grant(account, running.ID)
}

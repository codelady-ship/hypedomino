package integrity

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

const (
	PlayTokenHeader       = "X-HD-Play-Integrity"
	AttestKeyHeader       = "X-HD-Attest-Key"
	AttestNonceHeader     = "X-HD-Attest-Nonce"
	AttestAssertionHeader = "X-HD-Attest-Assertion"
)

var ErrKeyNotFound = errors.New("integrity: no attested key for the account")
var ErrStaleCounter = errors.New("integrity: assertion counter did not move forward")

type AttestedKeys interface {
	Key(ctx context.Context, account string, keyID []byte) (AttestedKey, error)
	Advance(ctx context.Context, account string, keyID []byte, counter uint32) error
}

type SignalSink interface {
	RecordIntegrity(ctx context.Context, account, purpose, kind string, weight int, reasons []string)
}

type Proof struct {
	PlayToken string
	KeyID     []byte
	Nonce     string
	Assertion []byte
}

type Gate struct {
	policy  Policy
	play    *PlayIntegrity
	attest  *AppAttest
	keys    AttestedKeys
	signals SignalSink
	logger  *slog.Logger
}

func NewGate(policy Policy, signals SignalSink, logger *slog.Logger) *Gate {
	return &Gate{policy: policy, signals: signals, logger: logger}
}

func (g *Gate) WithPlay(play *PlayIntegrity) {
	g.play = play
}

func (g *Gate) WithAppAttest(attest *AppAttest, keys AttestedKeys) {
	g.attest, g.keys = attest, keys
}

func (g *Gate) Enabled() bool {
	return g.play != nil || g.attest != nil
}

func (g *Gate) Check(ctx context.Context, binding Binding, proof Proof) (Decision, error) {
	decision, err := g.decide(ctx, binding, proof)
	if err != nil {
		return Decision{}, err
	}
	for _, signal := range decision.Signals {
		names := Decision{Reasons: signal.Reasons}.ReasonNames()
		g.signals.RecordIntegrity(ctx, binding.Account, binding.Purpose.String(), signal.Kind.String(), signal.Weight, names)
	}
	return decision, nil
}

func (g *Gate) decide(ctx context.Context, binding Binding, proof Proof) (Decision, error) {
	switch {
	case proof.PlayToken != "" && g.play != nil:
		return g.play.Check(ctx, g.policy, binding, proof.PlayToken)
	case len(proof.KeyID) > 0 && g.attest != nil:
		return g.assert(ctx, binding, proof)
	}
	return decide(g.policy, binding.Purpose, []Reason{ReasonTokenRejected}), nil
}

func (g *Gate) assert(ctx context.Context, binding Binding, proof Proof) (Decision, error) {
	key, err := g.keys.Key(ctx, binding.Account, proof.KeyID)
	if errors.Is(err, ErrKeyNotFound) {
		return decide(g.policy, binding.Purpose, []Reason{ReasonKeyIDMismatch}), nil
	}
	if err != nil {
		return Decision{}, err
	}
	decision, counter, err := g.attest.Assert(ctx, g.policy, binding, proof.Nonce, key, proof.Assertion)
	if err != nil || !decision.Passed() {
		return decision, err
	}
	err = g.keys.Advance(ctx, binding.Account, proof.KeyID, counter)
	if errors.Is(err, ErrStaleCounter) {
		return decide(g.policy, binding.Purpose, []Reason{ReasonCounterReplay}), nil
	}
	return decision, err
}

func (g *Gate) Require(verifier token.Verifier, purpose Purpose, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := token.Authorize(verifier, r)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		proof, err := proofFrom(r)
		if err != nil {
			reply(w, http.StatusBadRequest, errorResponse{Error: "bad_integrity_proof"})
			return
		}
		decision, err := g.Check(r.Context(), Binding{Account: claims.Account, Session: claims.Session, Purpose: purpose}, proof)
		if err != nil {
			g.logger.Error("integrity not checked", "account", claims.Account, "purpose", purpose.String(), "error", err)
			reply(w, http.StatusServiceUnavailable, errorResponse{Error: "integrity_unavailable"})
			return
		}
		if !decision.Allowed() {
			g.logger.Warn("integrity refused", "account", claims.Account, "purpose", purpose.String(), "reasons", decision.ReasonNames())
			reply(w, http.StatusForbidden, errorResponse{Error: "integrity_failed"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func proofFrom(r *http.Request) (Proof, error) {
	proof := Proof{PlayToken: r.Header.Get(PlayTokenHeader), Nonce: r.Header.Get(AttestNonceHeader)}
	var err error
	if proof.KeyID, err = base64.StdEncoding.DecodeString(r.Header.Get(AttestKeyHeader)); err != nil {
		return Proof{}, fmt.Errorf("integrity: key id: %w", err)
	}
	if proof.Assertion, err = base64.StdEncoding.DecodeString(r.Header.Get(AttestAssertionHeader)); err != nil {
		return Proof{}, fmt.Errorf("integrity: assertion: %w", err)
	}
	return proof, nil
}

func (g *Gate) SessionStart(ctx context.Context, account, session string, proof *pb.IntegrityProof) bool {
	binding := Binding{Account: account, Session: session, Purpose: PurposeSessionStart}
	decision, err := g.Check(ctx, binding, proofFromHello(proof))
	if err != nil {
		g.logger.Error("integrity not checked at session start", "account", account, "error", err)
		return true
	}
	if !decision.Allowed() {
		g.logger.Warn("integrity refused", "account", account, "purpose", binding.Purpose.String(), "reasons", decision.ReasonNames())
	}
	return decision.Allowed()
}

func (g *Gate) MatchStart(ctx context.Context, account, session string, queue pb.QueueKind, proof *pb.IntegrityProof) pb.QueueRefusal {
	binding := Binding{Account: account, Session: session, Purpose: matchPurpose(queue)}
	decision, err := g.Check(ctx, binding, proofFromHello(proof))
	switch {
	case err != nil && binding.Purpose == PurposeRankedMatch:
		g.logger.Error("integrity not checked at ranked match start", "account", account, "error", err)
		return pb.QueueRefusal_QUEUE_REFUSAL_INTEGRITY_UNAVAILABLE
	case err != nil:
		g.logger.Error("integrity not checked at casual match start", "account", account, "error", err)
		return pb.QueueRefusal_QUEUE_REFUSAL_UNSPECIFIED
	case !decision.Allowed():
		g.logger.Warn("integrity refused", "account", account, "purpose", binding.Purpose.String(), "reasons", decision.ReasonNames())
		return pb.QueueRefusal_QUEUE_REFUSAL_INTEGRITY_FAILED
	}
	return pb.QueueRefusal_QUEUE_REFUSAL_UNSPECIFIED
}

func matchPurpose(queue pb.QueueKind) Purpose {
	if queue == pb.QueueKind_QUEUE_KIND_RANKED {
		return PurposeRankedMatch
	}
	return PurposeCasualMatch
}

func proofFromHello(proof *pb.IntegrityProof) Proof {
	if attest := proof.GetAppAttest(); attest != nil {
		return Proof{KeyID: attest.GetKeyId(), Nonce: attest.GetNonce(), Assertion: attest.GetAssertion()}
	}
	return Proof{PlayToken: proof.GetPlayIntegrityToken()}
}

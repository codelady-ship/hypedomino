package integrity

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
)

type recordedSignal struct {
	account, purpose, kind string
	weight                 int
	reasons                []string
}

type signalLog struct{ signals []recordedSignal }

func (l *signalLog) RecordIntegrity(_ context.Context, account, purpose, kind string, weight int, reasons []string) {
	l.signals = append(l.signals, recordedSignal{account, purpose, kind, weight, reasons})
}

type memoryKeys struct {
	keys     map[string]AttestedKey
	advanced []uint32
	stale    bool
}

func (m *memoryKeys) Key(_ context.Context, account string, keyID []byte) (AttestedKey, error) {
	key, ok := m.keys[account+"/"+string(keyID)]
	if !ok {
		return AttestedKey{}, ErrKeyNotFound
	}
	return key, nil
}

func (m *memoryKeys) Advance(_ context.Context, account string, keyID []byte, counter uint32) error {
	if m.stale {
		return ErrStaleCounter
	}
	key := m.keys[account+"/"+string(keyID)]
	key.Counter = counter
	m.keys[account+"/"+string(keyID)] = key
	m.advanced = append(m.advanced, counter)
	return nil
}

type gateFixture struct {
	play    *playFixture
	attest  *attestFixture
	keys    *memoryKeys
	signals *signalLog
	handler http.Handler
	reached int
}

func newGateFixture(t *testing.T) *gateFixture {
	t.Helper()
	f := &gateFixture{play: newPlayFixture(t), attest: newAttestFixture(t, false), signals: &signalLog{}}
	key := f.attest.attestedKey(t, 0)
	f.keys = &memoryKeys{keys: map[string]AttestedKey{"account-1/" + string(key.KeyID): key}}
	gate := NewGate(lenientPolicy(), f.signals, slog.New(slog.NewTextHandler(io.Discard, nil)))
	gate.WithPlay(f.play.verify)
	gate.WithAppAttest(f.attest.verifier, f.keys)
	f.guard(gate, http.StatusNoContent)
	return f
}

func (f *gateFixture) guard(gate *Gate, status int) {
	f.handler = gate.Require(fixtureVerifier{}, PurposePurchase, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.reached++
		w.WriteHeader(status)
	}))
}

func (f *gateFixture) call(access string, headers map[string]string) int {
	r := httptest.NewRequest(http.MethodPost, "/shop/complete", nil)
	r.Header.Set("Authorization", "Bearer "+access)
	for name, value := range headers {
		r.Header.Set(name, value)
	}
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, r)
	return w.Code
}

func purchaseBinding() Binding {
	binding := sessionBinding()
	binding.Purpose = PurposePurchase
	return binding
}

func (f *gateFixture) playHeaders(t *testing.T, binding Binding, edit func(*PlayIntegrityPayload)) map[string]string {
	t.Helper()
	return map[string]string{PlayTokenHeader: f.play.token(t, binding, edit)}
}

func (f *gateFixture) assertionHeaders(t *testing.T, counter uint32) map[string]string {
	t.Helper()
	nonce := f.attest.issueNonce(t, purchaseBinding())
	assertion := signAssertion(t, f.attest.credential, assertionAuthData(testAppID, counter), []byte(nonce))
	return map[string]string{
		AttestKeyHeader:       base64.StdEncoding.EncodeToString(keyIDOf(t, f.attest.credential)),
		AttestNonceHeader:     nonce,
		AttestAssertionHeader: base64.StdEncoding.EncodeToString(assertion),
	}
}

func TestPurchaseGateAcceptsGenuineDevicesAndFlagsTheRest(t *testing.T) {
	cases := []struct {
		name    string
		headers func(t *testing.T, f *gateFixture) map[string]string
		status  int
		signal  string
		weight  int
		reasons []string
	}{
		{name: "genuine android", status: http.StatusNoContent, headers: func(t *testing.T, f *gateFixture) map[string]string {
			return f.playHeaders(t, purchaseBinding(), nil)
		}},
		{name: "re-signed android build", status: http.StatusForbidden, signal: "integrity_failure", weight: 10, reasons: []string{"unknown_signing_certificate"}, headers: func(t *testing.T, f *gateFixture) map[string]string {
			return f.playHeaders(t, purchaseBinding(), func(p *PlayIntegrityPayload) { p.AppIntegrity.CertificateSha256Digest = []string{digestOf("other")} })
		}},
		{name: "weak android device", status: http.StatusNoContent, signal: "device_risk", weight: 3, reasons: []string{"weak_device"}, headers: func(t *testing.T, f *gateFixture) map[string]string {
			return f.playHeaders(t, purchaseBinding(), func(p *PlayIntegrityPayload) { p.DeviceIntegrity.DeviceRecognitionVerdict = nil })
		}},
		{name: "android token for a session nonce", status: http.StatusForbidden, signal: "integrity_failure", weight: 10, reasons: []string{"nonce_binding_mismatch"}, headers: func(t *testing.T, f *gateFixture) map[string]string {
			return f.playHeaders(t, sessionBinding(), nil)
		}},
		{name: "no proof", status: http.StatusForbidden, signal: "integrity_failure", weight: 10, reasons: []string{"token_rejected"}, headers: func(t *testing.T, f *gateFixture) map[string]string {
			return nil
		}},
		{name: "genuine ios", status: http.StatusNoContent, headers: func(t *testing.T, f *gateFixture) map[string]string {
			return f.assertionHeaders(t, 1)
		}},
		{name: "ios key of nobody", status: http.StatusForbidden, signal: "integrity_failure", weight: 10, reasons: []string{"key_id_mismatch"}, headers: func(t *testing.T, f *gateFixture) map[string]string {
			headers := f.assertionHeaders(t, 1)
			headers[AttestKeyHeader] = base64.StdEncoding.EncodeToString([]byte("unknown"))
			return headers
		}},
		{name: "ios counter lost a race", status: http.StatusForbidden, signal: "integrity_failure", weight: 10, reasons: []string{"counter_replay"}, headers: func(t *testing.T, f *gateFixture) map[string]string {
			f.keys.stale = true
			return f.assertionHeaders(t, 1)
		}},
		{name: "broken key header", status: http.StatusBadRequest, headers: func(t *testing.T, f *gateFixture) map[string]string {
			return map[string]string{AttestKeyHeader: "%%"}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newGateFixture(t)
			status := f.call("good", c.headers(t, f))
			if status != c.status || (f.reached == 1) != (status == http.StatusNoContent) {
				t.Fatalf("status %d reached %d", status, f.reached)
			}
			var want []recordedSignal
			if c.signal != "" {
				want = []recordedSignal{{account: "account-1", purpose: "purchase", kind: c.signal, weight: c.weight, reasons: c.reasons}}
			}
			if !slices.EqualFunc(f.signals.signals, want, sameRecordedSignal) {
				t.Fatalf("signals %+v, want %+v", f.signals.signals, want)
			}
		})
	}
}

func sameRecordedSignal(a, b recordedSignal) bool {
	return a.account == b.account && a.purpose == b.purpose && a.kind == b.kind && a.weight == b.weight && slices.Equal(a.reasons, b.reasons)
}

func TestPurchaseGateMovesTheAssertionCounter(t *testing.T) {
	f := newGateFixture(t)
	if status := f.call("good", f.assertionHeaders(t, 4)); status != http.StatusNoContent {
		t.Fatalf("first: %d", status)
	}
	if status := f.call("good", f.assertionHeaders(t, 4)); status != http.StatusForbidden {
		t.Fatalf("same counter again: %d", status)
	}
	if !slices.Equal(f.keys.advanced, []uint32{4}) {
		t.Fatalf("advanced %v", f.keys.advanced)
	}
}

func TestPurchaseGateFailsClosedAndLeavesAuthToTheHandler(t *testing.T) {
	f := newGateFixture(t)
	decoder := stubDecoder{err: errors.New("google down")}
	down, err := NewPlayIntegrity(PlayIntegrityConfig{PackageName: testPackage, CertificateDigests: []string{releaseDigest}, TokenMaxAge: time.Minute}, decoder, f.play.nonces, f.play.clock)
	if err != nil {
		t.Fatal(err)
	}
	gate := NewGate(lenientPolicy(), f.signals, slog.New(slog.NewTextHandler(io.Discard, nil)))
	gate.WithPlay(down)
	f.guard(gate, http.StatusUnauthorized)
	if status := f.call("good", map[string]string{PlayTokenHeader: "token"}); status != http.StatusServiceUnavailable || f.reached != 0 {
		t.Fatalf("outage: %d reached %d", status, f.reached)
	}
	if status := f.call("bad", nil); status != http.StatusUnauthorized || f.reached != 1 {
		t.Fatalf("no token: %d reached %d", status, f.reached)
	}
	if NewGate(Policy{}, f.signals, nil).Enabled() {
		t.Fatal("a gate without verifiers claims to be enabled")
	}
}

func TestSessionStartFollowsThePolicyAndFailsOpen(t *testing.T) {
	f := newGateFixture(t)
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	lenient := NewGate(lenientPolicy(), f.signals, quiet)
	lenient.WithPlay(f.play.verify)
	lenient.WithAppAttest(f.attest.verifier, f.keys)
	strict := NewGate(strictPolicy(), f.signals, quiet)
	strict.WithPlay(f.play.verify)
	nonce := f.attest.issueNonce(t, sessionBinding())
	assertion := signAssertion(t, f.attest.credential, assertionAuthData(testAppID, 1), []byte(nonce))
	cases := []struct {
		name    string
		gate    *Gate
		proof   *pb.IntegrityProof
		allowed bool
		signals int
	}{
		{"genuine android", lenient, &pb.IntegrityProof{Proof: &pb.IntegrityProof_PlayIntegrityToken{PlayIntegrityToken: f.play.token(t, sessionBinding(), nil)}}, true, 0},
		{"genuine ios", lenient, &pb.IntegrityProof{Proof: &pb.IntegrityProof_AppAttest{AppAttest: &pb.AppAttestAssertion{KeyId: keyIDOf(t, f.attest.credential), Nonce: nonce, Assertion: assertion}}}, true, 0},
		{"no proof under a flag policy", lenient, nil, true, 1},
		{"no proof under a block policy", strict, nil, false, 1},
		{"purchase token at session start", strict, &pb.IntegrityProof{Proof: &pb.IntegrityProof_PlayIntegrityToken{PlayIntegrityToken: f.play.token(t, purchaseBinding(), nil)}}, false, 1},
	}
	for _, c := range cases {
		f.signals.signals = nil
		if allowed := c.gate.SessionStart(t.Context(), "account-1", "session-1", c.proof); allowed != c.allowed || len(f.signals.signals) != c.signals {
			t.Fatalf("%s: allowed %v signals %+v", c.name, allowed, f.signals.signals)
		}
	}
	down, err := NewPlayIntegrity(PlayIntegrityConfig{PackageName: testPackage, CertificateDigests: []string{releaseDigest}, TokenMaxAge: time.Minute}, stubDecoder{err: errors.New("google down")}, f.play.nonces, f.play.clock)
	if err != nil {
		t.Fatal(err)
	}
	outage := NewGate(strictPolicy(), f.signals, quiet)
	outage.WithPlay(down)
	if !outage.SessionStart(t.Context(), "account-1", "session-1", &pb.IntegrityProof{Proof: &pb.IntegrityProof_PlayIntegrityToken{PlayIntegrityToken: "token"}}) {
		t.Fatal("an outage refused the session start")
	}
}

func TestMatchStartBlocksRankedAndFailsClosedOnlyForRanked(t *testing.T) {
	f := newGateFixture(t)
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	lenient := NewGate(lenientPolicy(), f.signals, quiet)
	lenient.WithPlay(f.play.verify)
	ranked := Binding{Account: "account-1", Session: "session-1", Purpose: PurposeRankedMatch}
	playToken := func(binding Binding) *pb.IntegrityProof {
		return &pb.IntegrityProof{Proof: &pb.IntegrityProof_PlayIntegrityToken{PlayIntegrityToken: f.play.token(t, binding, nil)}}
	}
	cases := []struct {
		name    string
		queue   pb.QueueKind
		proof   *pb.IntegrityProof
		refusal pb.QueueRefusal
		signals int
	}{
		{"genuine ranked", pb.QueueKind_QUEUE_KIND_RANKED, playToken(ranked), pb.QueueRefusal_QUEUE_REFUSAL_UNSPECIFIED, 0},
		{"no proof ranked", pb.QueueKind_QUEUE_KIND_RANKED, nil, pb.QueueRefusal_QUEUE_REFUSAL_INTEGRITY_FAILED, 1},
		{"session token at ranked", pb.QueueKind_QUEUE_KIND_RANKED, playToken(sessionBinding()), pb.QueueRefusal_QUEUE_REFUSAL_INTEGRITY_FAILED, 1},
		{"no proof casual under a flag policy", pb.QueueKind_QUEUE_KIND_CASUAL, nil, pb.QueueRefusal_QUEUE_REFUSAL_UNSPECIFIED, 1},
	}
	for _, c := range cases {
		f.signals.signals = nil
		if refusal := lenient.MatchStart(t.Context(), "account-1", "session-1", c.queue, c.proof); refusal != c.refusal || len(f.signals.signals) != c.signals {
			t.Fatalf("%s: refusal %v signals %+v", c.name, refusal, f.signals.signals)
		}
	}
	down, err := NewPlayIntegrity(PlayIntegrityConfig{PackageName: testPackage, CertificateDigests: []string{releaseDigest}, TokenMaxAge: time.Minute}, stubDecoder{err: errors.New("google down")}, f.play.nonces, f.play.clock)
	if err != nil {
		t.Fatal(err)
	}
	outage := NewGate(lenientPolicy(), f.signals, quiet)
	outage.WithPlay(down)
	proof := &pb.IntegrityProof{Proof: &pb.IntegrityProof_PlayIntegrityToken{PlayIntegrityToken: "token"}}
	if refusal := outage.MatchStart(t.Context(), "account-1", "session-1", pb.QueueKind_QUEUE_KIND_RANKED, proof); refusal != pb.QueueRefusal_QUEUE_REFUSAL_INTEGRITY_UNAVAILABLE {
		t.Fatalf("an outage at ranked entry answered %v", refusal)
	}
	if refusal := outage.MatchStart(t.Context(), "account-1", "session-1", pb.QueueKind_QUEUE_KIND_CASUAL, proof); refusal != pb.QueueRefusal_QUEUE_REFUSAL_UNSPECIFIED {
		t.Fatalf("an outage at casual entry answered %v", refusal)
	}
}

package integrity

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"testing"
	"time"
)

const testPackage = "com.example.hypedomino"

var (
	releaseDigest = digestOf("release signing certificate")
	repackDigest  = digestOf("someone else's signing certificate")
)

func digestOf(certificate string) string {
	sum := sha256.Sum256([]byte(certificate))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

type stubDecoder struct {
	payloads map[string]PlayIntegrityPayload
	err      error
}

func (d stubDecoder) Decode(_ context.Context, token string) (PlayIntegrityPayload, error) {
	if d.err != nil {
		return PlayIntegrityPayload{}, d.err
	}
	payload, found := d.payloads[token]
	if !found {
		return PlayIntegrityPayload{}, fmt.Errorf("unknown token: %w", ErrTokenRejected)
	}
	return payload, nil
}

func lenientPolicy() Policy {
	return Policy{
		IntegrityFailureAction: ActionFlag,
		IntegrityFailureWeight: 10,
		DeviceRiskAction:       DeviceRiskFlag,
		DeviceRiskWeight:       3,
		MinDeviceLevel:         DeviceLevelBasic,
	}
}

func strictPolicy() Policy {
	policy := lenientPolicy()
	policy.IntegrityFailureAction = ActionBlock
	policy.DeviceRiskAction = DeviceRiskBlockRanked
	policy.MinDeviceLevel = DeviceLevelDevice
	policy.RequireLicensed = true
	return policy
}

func genuinePayload(nonce string) PlayIntegrityPayload {
	return PlayIntegrityPayload{
		RequestDetails: RequestDetails{
			RequestPackageName: testPackage,
			Nonce:              nonce,
			TimestampMillis:    strconv.FormatInt(testStart.Add(-2*time.Second).UnixMilli(), 10),
		},
		AppIntegrity: AppIntegrity{
			AppRecognitionVerdict:   playRecognized,
			PackageName:             testPackage,
			CertificateSha256Digest: []string{releaseDigest},
			VersionCode:             "42",
		},
		DeviceIntegrity: DeviceIntegrity{DeviceRecognitionVerdict: []string{meetsBasicIntegrity, meetsDeviceIntegrity}},
		AccountDetails:  AccountDetails{AppLicensingVerdict: licensed},
	}
}

type playFixture struct {
	clock   *testClock
	nonces  *Nonces
	decoder stubDecoder
	verify  *PlayIntegrity
}

func newPlayFixture(t *testing.T) *playFixture {
	t.Helper()
	clock := newTestClock()
	nonces, _ := newTestNonces(t, clock)
	decoder := stubDecoder{payloads: map[string]PlayIntegrityPayload{}}
	verify, err := NewPlayIntegrity(PlayIntegrityConfig{
		PackageName:        testPackage,
		CertificateDigests: []string{releaseDigest},
		TokenMaxAge:        time.Minute,
	}, decoder, nonces, clock)
	if err != nil {
		t.Fatal(err)
	}
	return &playFixture{clock: clock, nonces: nonces, decoder: decoder, verify: verify}
}

func (f *playFixture) token(t *testing.T, binding Binding, edit func(*PlayIntegrityPayload)) string {
	t.Helper()
	nonce, err := f.nonces.Issue(context.Background(), binding)
	if err != nil {
		t.Fatal(err)
	}
	payload := genuinePayload(nonce.Value)
	if edit != nil {
		edit(&payload)
	}
	token := "token-" + strconv.Itoa(len(f.decoder.payloads))
	f.decoder.payloads[token] = payload
	return token
}

func TestPlayIntegrityCheck(t *testing.T) {
	cases := []struct {
		name    string
		policy  Policy
		purpose Purpose
		edit    func(*PlayIntegrityPayload)
		reasons []Reason
		action  Action
	}{
		{name: "genuine build", policy: strictPolicy(), purpose: PurposeSessionStart, action: ActionAllow},
		{name: "repackaged build blocked", policy: strictPolicy(), purpose: PurposeSessionStart,
			edit:    func(p *PlayIntegrityPayload) { p.AppIntegrity.CertificateSha256Digest = []string{repackDigest} },
			reasons: []Reason{ReasonUnknownSigningCertificate}, action: ActionBlock},
		{name: "repackaged build flagged", policy: lenientPolicy(), purpose: PurposeSessionStart,
			edit:    func(p *PlayIntegrityPayload) { p.AppIntegrity.CertificateSha256Digest = []string{repackDigest} },
			reasons: []Reason{ReasonUnknownSigningCertificate}, action: ActionFlag},
		{name: "extra unknown signer", policy: lenientPolicy(), purpose: PurposeSessionStart,
			edit: func(p *PlayIntegrityPayload) {
				p.AppIntegrity.CertificateSha256Digest = []string{releaseDigest, repackDigest}
			},
			reasons: []Reason{ReasonUnknownSigningCertificate}, action: ActionFlag},
		{name: "no signer digest", policy: lenientPolicy(), purpose: PurposeSessionStart,
			edit:    func(p *PlayIntegrityPayload) { p.AppIntegrity.CertificateSha256Digest = nil },
			reasons: []Reason{ReasonUnknownSigningCertificate}, action: ActionFlag},
		{name: "padded allowlisted digest", policy: strictPolicy(), purpose: PurposeSessionStart,
			edit:   func(p *PlayIntegrityPayload) { p.AppIntegrity.CertificateSha256Digest = []string{releaseDigest + "="} },
			action: ActionAllow},
		{name: "repackaged build at purchase blocked under flag policy", policy: lenientPolicy(), purpose: PurposePurchase,
			edit:    func(p *PlayIntegrityPayload) { p.AppIntegrity.CertificateSha256Digest = []string{repackDigest} },
			reasons: []Reason{ReasonUnknownSigningCertificate}, action: ActionBlock},
		{name: "unrecognised install source", policy: lenientPolicy(), purpose: PurposeSessionStart,
			edit:    func(p *PlayIntegrityPayload) { p.AppIntegrity.AppRecognitionVerdict = "UNRECOGNIZED_VERSION" },
			reasons: []Reason{ReasonUnrecognizedApp}, action: ActionFlag},
		{name: "unevaluated app", policy: lenientPolicy(), purpose: PurposeCasualMatch,
			edit:    func(p *PlayIntegrityPayload) { p.AppIntegrity = AppIntegrity{AppRecognitionVerdict: "UNEVALUATED"} },
			reasons: []Reason{ReasonPackageMismatch, ReasonUnrecognizedApp, ReasonUnknownSigningCertificate}, action: ActionFlag},
		{name: "stale timestamp", policy: lenientPolicy(), purpose: PurposeSessionStart,
			edit: func(p *PlayIntegrityPayload) {
				p.RequestDetails.TimestampMillis = strconv.FormatInt(testStart.Add(-2*time.Minute).UnixMilli(), 10)
			},
			reasons: []Reason{ReasonTokenStale}, action: ActionFlag},
		{name: "future timestamp", policy: lenientPolicy(), purpose: PurposeSessionStart,
			edit: func(p *PlayIntegrityPayload) {
				p.RequestDetails.TimestampMillis = strconv.FormatInt(testStart.Add(2*time.Minute).UnixMilli(), 10)
			},
			reasons: []Reason{ReasonTokenStale}, action: ActionFlag},
		{name: "unparsable timestamp", policy: lenientPolicy(), purpose: PurposeSessionStart,
			edit:    func(p *PlayIntegrityPayload) { p.RequestDetails.TimestampMillis = "yesterday" },
			reasons: []Reason{ReasonTokenMalformed}, action: ActionFlag},
		{name: "nonce never issued", policy: lenientPolicy(), purpose: PurposeSessionStart,
			edit:    func(p *PlayIntegrityPayload) { p.RequestDetails.Nonce = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" },
			reasons: []Reason{ReasonNonceUnknown}, action: ActionFlag},
		{name: "nonce missing", policy: lenientPolicy(), purpose: PurposeSessionStart,
			edit:    func(p *PlayIntegrityPayload) { p.RequestDetails.Nonce = "" },
			reasons: []Reason{ReasonNonceMissing}, action: ActionFlag},
		{name: "standard request carries the nonce as request hash", policy: strictPolicy(), purpose: PurposeSessionStart,
			edit: func(p *PlayIntegrityPayload) {
				p.RequestDetails.RequestHash, p.RequestDetails.Nonce = p.RequestDetails.Nonce, ""
			},
			action: ActionAllow},
		{name: "wrong package in request", policy: lenientPolicy(), purpose: PurposeSessionStart,
			edit:    func(p *PlayIntegrityPayload) { p.RequestDetails.RequestPackageName = "com.example.clone" },
			reasons: []Reason{ReasonPackageMismatch}, action: ActionFlag},
		{name: "wrong package in app integrity", policy: lenientPolicy(), purpose: PurposeSessionStart,
			edit:    func(p *PlayIntegrityPayload) { p.AppIntegrity.PackageName = "com.example.clone" },
			reasons: []Reason{ReasonPackageMismatch}, action: ActionFlag},
		{name: "basic device passes lenient policy", policy: lenientPolicy(), purpose: PurposeRankedMatch,
			edit: func(p *PlayIntegrityPayload) {
				p.DeviceIntegrity.DeviceRecognitionVerdict = []string{meetsBasicIntegrity}
			},
			action: ActionAllow},
		{name: "basic device under strict policy blocks ranked", policy: strictPolicy(), purpose: PurposeRankedMatch,
			edit: func(p *PlayIntegrityPayload) {
				p.DeviceIntegrity.DeviceRecognitionVerdict = []string{meetsBasicIntegrity}
			},
			reasons: []Reason{ReasonWeakDevice}, action: ActionBlock},
		{name: "basic device under strict policy flags session", policy: strictPolicy(), purpose: PurposeSessionStart,
			edit: func(p *PlayIntegrityPayload) {
				p.DeviceIntegrity.DeviceRecognitionVerdict = []string{meetsBasicIntegrity}
			},
			reasons: []Reason{ReasonWeakDevice}, action: ActionFlag},
		{name: "strong label alone meets device level", policy: strictPolicy(), purpose: PurposeRankedMatch,
			edit: func(p *PlayIntegrityPayload) {
				p.DeviceIntegrity.DeviceRecognitionVerdict = []string{meetsStrongIntegrity}
			},
			action: ActionAllow},
		{name: "no device verdict under lenient policy", policy: lenientPolicy(), purpose: PurposeSessionStart,
			edit:    func(p *PlayIntegrityPayload) { p.DeviceIntegrity.DeviceRecognitionVerdict = nil },
			reasons: []Reason{ReasonWeakDevice}, action: ActionFlag},
		{name: "unlicensed with licensing required", policy: strictPolicy(), purpose: PurposeSessionStart,
			edit:    func(p *PlayIntegrityPayload) { p.AccountDetails.AppLicensingVerdict = "UNLICENSED" },
			reasons: []Reason{ReasonUnlicensed}, action: ActionBlock},
		{name: "unlicensed with licensing optional", policy: lenientPolicy(), purpose: PurposeSessionStart,
			edit:   func(p *PlayIntegrityPayload) { p.AccountDetails.AppLicensingVerdict = "UNLICENSED" },
			action: ActionAllow},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newPlayFixture(t)
			binding := sessionBinding()
			binding.Purpose = tc.purpose
			token := fixture.token(t, binding, tc.edit)
			decision, err := fixture.verify.Check(context.Background(), tc.policy, binding, token)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(decision.Reasons, tc.reasons) {
				t.Fatalf("reasons = %v, want %v", decision.Reasons, tc.reasons)
			}
			if decision.Action != tc.action {
				t.Fatalf("action = %v, want %v", decision.Action, tc.action)
			}
			if decision.Passed() != (len(tc.reasons) == 0) {
				t.Fatalf("passed = %v with reasons %v", decision.Passed(), decision.Reasons)
			}
		})
	}
}

func TestPlayIntegrityRejectsNonceOfAnotherBinding(t *testing.T) {
	fixture := newPlayFixture(t)
	token := fixture.token(t, sessionBinding(), nil)
	purchase := sessionBinding()
	purchase.Purpose = PurposePurchase
	decision, err := fixture.verify.Check(context.Background(), lenientPolicy(), purchase, token)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(decision.Reasons, []Reason{ReasonNonceBindingMismatch}) || decision.Allowed() {
		t.Fatalf("decision = %+v, want blocked nonce_binding_mismatch", decision)
	}
}

func TestPlayIntegrityRejectsReplayedToken(t *testing.T) {
	fixture := newPlayFixture(t)
	token := fixture.token(t, sessionBinding(), nil)
	first, err := fixture.verify.Check(context.Background(), strictPolicy(), sessionBinding(), token)
	if err != nil || !first.Passed() {
		t.Fatalf("first check = %+v, %v", first, err)
	}
	replay, err := fixture.verify.Check(context.Background(), strictPolicy(), sessionBinding(), token)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(replay.Reasons, []Reason{ReasonNonceUnknown}) || replay.Allowed() {
		t.Fatalf("replay = %+v, want blocked nonce_unknown", replay)
	}
}

func TestPlayIntegrityRejectsExpiredNonce(t *testing.T) {
	fixture := newPlayFixture(t)
	token := fixture.token(t, sessionBinding(), nil)
	fixture.clock.Advance(5 * time.Minute)
	payload := fixture.decoder.payloads[token]
	payload.RequestDetails.TimestampMillis = strconv.FormatInt(fixture.clock.Now().UnixMilli(), 10)
	fixture.decoder.payloads[token] = payload
	decision, err := fixture.verify.Check(context.Background(), lenientPolicy(), sessionBinding(), token)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(decision.Reasons, []Reason{ReasonNonceExpired}) {
		t.Fatalf("reasons = %v, want nonce_expired", decision.Reasons)
	}
}

func TestPlayIntegrityTokenRejectedByDecoder(t *testing.T) {
	fixture := newPlayFixture(t)
	for _, token := range []string{"", "forged"} {
		decision, err := fixture.verify.Check(context.Background(), strictPolicy(), sessionBinding(), token)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(decision.Reasons, []Reason{ReasonTokenRejected}) || decision.Allowed() {
			t.Fatalf("token %q: decision = %+v", token, decision)
		}
	}
}

func TestPlayIntegrityDecoderOutageIsAnError(t *testing.T) {
	clock := newTestClock()
	nonces, _ := newTestNonces(t, clock)
	outage := errors.New("service unavailable")
	verify, err := NewPlayIntegrity(PlayIntegrityConfig{PackageName: testPackage, CertificateDigests: []string{releaseDigest}, TokenMaxAge: time.Minute}, stubDecoder{err: outage}, nonces, clock)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verify.Check(context.Background(), strictPolicy(), sessionBinding(), "token"); !errors.Is(err, outage) {
		t.Fatalf("err = %v, want the decoder error", err)
	}
}

func TestNewPlayIntegrityRejectsInvalidConfig(t *testing.T) {
	clock := newTestClock()
	nonces, _ := newTestNonces(t, clock)
	valid := PlayIntegrityConfig{PackageName: testPackage, CertificateDigests: []string{releaseDigest}, TokenMaxAge: time.Minute}
	cases := map[string]func(*PlayIntegrityConfig){
		"no package":         func(c *PlayIntegrityConfig) { c.PackageName = "" },
		"empty allowlist":    func(c *PlayIntegrityConfig) { c.CertificateDigests = nil },
		"short digest":       func(c *PlayIntegrityConfig) { c.CertificateDigests = []string{"AAAA"} },
		"undecodable digest": func(c *PlayIntegrityConfig) { c.CertificateDigests = []string{"not base64!"} },
		"no max age":         func(c *PlayIntegrityConfig) { c.TokenMaxAge = 0 },
	}
	for name, edit := range cases {
		config := valid
		edit(&config)
		if _, err := NewPlayIntegrity(config, stubDecoder{}, nonces, clock); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestPlayIntegrityPayloadMatchesTheDocumentedJSON(t *testing.T) {
	documented := `{
	  "requestDetails": {
	    "requestPackageName": "com.package.name",
	    "requestHash": "aGVsbG8gd29scmQgdGhlcmU",
	    "nonce": "aGVsbG8gd29scmQgdGhlcmU",
	    "timestampMillis": "1675655009345"
	  },
	  "appIntegrity": {
	    "appRecognitionVerdict": "PLAY_RECOGNIZED",
	    "packageName": "com.package.name",
	    "certificateSha256Digest": ["6a6a1474b5cbbb2b1aa57e0bc3"],
	    "versionCode": "42"
	  },
	  "deviceIntegrity": {
	    "deviceRecognitionVerdict": ["MEETS_DEVICE_INTEGRITY"]
	  },
	  "accountDetails": {
	    "appLicensingVerdict": "LICENSED"
	  },
	  "environmentDetails": {
	    "playProtectVerdict": "NO_ISSUES"
	  }
	}`
	var payload PlayIntegrityPayload
	if err := json.Unmarshal([]byte(documented), &payload); err != nil {
		t.Fatal(err)
	}
	want := PlayIntegrityPayload{
		RequestDetails:  RequestDetails{RequestPackageName: "com.package.name", Nonce: "aGVsbG8gd29scmQgdGhlcmU", RequestHash: "aGVsbG8gd29scmQgdGhlcmU", TimestampMillis: "1675655009345"},
		AppIntegrity:    AppIntegrity{AppRecognitionVerdict: "PLAY_RECOGNIZED", PackageName: "com.package.name", CertificateSha256Digest: []string{"6a6a1474b5cbbb2b1aa57e0bc3"}, VersionCode: "42"},
		DeviceIntegrity: DeviceIntegrity{DeviceRecognitionVerdict: []string{"MEETS_DEVICE_INTEGRITY"}},
		AccountDetails:  AccountDetails{AppLicensingVerdict: "LICENSED"},
	}
	got, _ := json.Marshal(payload)
	expected, _ := json.Marshal(want)
	if string(got) != string(expected) {
		t.Fatalf("payload = %s, want %s", got, expected)
	}
}

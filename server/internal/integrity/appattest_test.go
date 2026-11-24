package integrity

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/binary"
	"encoding/pem"
	"math/big"
	"slices"
	"testing"
	"time"
)

const (
	testTeamID   = "TEAM123456"
	testBundleID = "com.example.hypedomino"
	testAppID    = testTeamID + "." + testBundleID
)

func cborHead(major byte, argument uint64) []byte {
	switch {
	case argument < 24:
		return []byte{major<<5 | byte(argument)}
	case argument <= 0xff:
		return []byte{major<<5 | 24, byte(argument)}
	case argument <= 0xffff:
		return binary.BigEndian.AppendUint16([]byte{major<<5 | 25}, uint16(argument))
	case argument <= 0xffffffff:
		return binary.BigEndian.AppendUint32([]byte{major<<5 | 26}, uint32(argument))
	}
	return binary.BigEndian.AppendUint64([]byte{major<<5 | 27}, argument)
}

func cborByteString(value []byte) []byte {
	return append(cborHead(cborBytes, uint64(len(value))), value...)
}

func cborTextString(value string) []byte {
	return append(cborHead(cborText, uint64(len(value))), value...)
}

func cborArrayOf(items ...[]byte) []byte {
	return slices.Concat(append([][]byte{cborHead(cborArray, uint64(len(items)))}, items...)...)
}

func cborMapOf(pairs ...[]byte) []byte {
	return slices.Concat(append([][]byte{cborHead(cborMap, uint64(len(pairs)/2))}, pairs...)...)
}

type testAuthority struct {
	certificate *x509.Certificate
	key         *ecdsa.PrivateKey
}

func newTestKey(t *testing.T, curve elliptic.Curve) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func issueCertificate(t *testing.T, template *x509.Certificate, public *ecdsa.PublicKey, issuer testAuthority) *x509.Certificate {
	t.Helper()
	parent := template
	if issuer.certificate != nil {
		parent = issuer.certificate
	}
	der, err := x509.CreateCertificate(rand.Reader, template, parent, public, issuer.key)
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return certificate
}

func newTestRoot(t *testing.T, name string) testAuthority {
	t.Helper()
	key := newTestKey(t, elliptic.P384())
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: name},
		NotBefore:             testStart.Add(-time.Hour),
		NotAfter:              testStart.Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	return testAuthority{certificate: issueCertificate(t, template, &key.PublicKey, testAuthority{key: key}), key: key}
}

func newTestIntermediate(t *testing.T, root testAuthority) testAuthority {
	t.Helper()
	key := newTestKey(t, elliptic.P384())
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(2),
		Subject:               pkix.Name{CommonName: "Test App Attestation CA"},
		NotBefore:             testStart.Add(-time.Hour),
		NotAfter:              testStart.Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	return testAuthority{certificate: issueCertificate(t, template, &key.PublicKey, root), key: key}
}

func credentialCertificate(t *testing.T, issuer testAuthority, credential *ecdsa.PublicKey, nonce []byte) []byte {
	t.Helper()
	extension, err := asn1.Marshal(struct {
		Nonce []byte `asn1:"tag:1,explicit"`
	}{nonce})
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber:    big.NewInt(3),
		Subject:         pkix.Name{CommonName: "credential"},
		NotBefore:       testStart.Add(-time.Minute),
		NotAfter:        testStart.Add(time.Hour),
		KeyUsage:        x509.KeyUsageDigitalSignature,
		ExtraExtensions: []pkix.Extension{{Id: nonceExtensionOID, Value: extension}},
	}
	return issueCertificate(t, template, credential, issuer).Raw
}

func publicKeyBytes(t *testing.T, key *ecdsa.PrivateKey) []byte {
	t.Helper()
	raw, err := key.PublicKey.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func keyIDOf(t *testing.T, key *ecdsa.PrivateKey) []byte {
	t.Helper()
	sum := sha256.Sum256(publicKeyBytes(t, key))
	return sum[:]
}

func attestationAuthData(appID string, counter uint32, aaguid, credentialID []byte) []byte {
	rpIDHash := sha256.Sum256([]byte(appID))
	data := append(rpIDHash[:], attestedCredentialFlag|0x01)
	data = binary.BigEndian.AppendUint32(data, counter)
	data = append(data, aaguid...)
	data = binary.BigEndian.AppendUint16(data, uint16(len(credentialID)))
	data = append(data, credentialID...)
	return append(data, cborMapOf(cborTextString("kty"), cborHead(cborUnsigned, 2))...)
}

func assertionAuthData(appID string, counter uint32) []byte {
	rpIDHash := sha256.Sum256([]byte(appID))
	return binary.BigEndian.AppendUint32(append(rpIDHash[:], 0x01), counter)
}

type attestFixture struct {
	clock        *testClock
	nonces       *Nonces
	verifier     *AppAttest
	root         testAuthority
	intermediate testAuthority
	credential   *ecdsa.PrivateKey
}

func newAttestFixture(t *testing.T, allowDevelopment bool) *attestFixture {
	t.Helper()
	clock := newTestClock()
	nonces, _ := newTestNonces(t, clock)
	root := newTestRoot(t, "Test App Attestation Root CA")
	rootPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: root.certificate.Raw})
	verifier, err := NewAppAttest(AppAttestConfig{TeamID: testTeamID, BundleID: testBundleID, RootCertificatePEM: rootPEM, AllowDevelopment: allowDevelopment}, nonces, clock)
	if err != nil {
		t.Fatal(err)
	}
	return &attestFixture{
		clock:        clock,
		nonces:       nonces,
		verifier:     verifier,
		root:         root,
		intermediate: newTestIntermediate(t, root),
		credential:   newTestKey(t, elliptic.P256()),
	}
}

type attestationParts struct {
	format       string
	appID        string
	counter      uint32
	aaguid       []byte
	credentialID []byte
	clientData   []byte
	issuer       testAuthority
	chainTail    [][]byte
}

func (f *attestFixture) issueNonce(t *testing.T, binding Binding) string {
	t.Helper()
	nonce, err := f.nonces.Issue(context.Background(), binding)
	if err != nil {
		t.Fatal(err)
	}
	return nonce.Value
}

func (f *attestFixture) attestation(t *testing.T, nonce string, edit func(*attestationParts)) []byte {
	t.Helper()
	parts := attestationParts{
		format:       appAttestFormat,
		appID:        testAppID,
		aaguid:       productionAAGUID,
		credentialID: keyIDOf(t, f.credential),
		clientData:   []byte(nonce),
		issuer:       f.intermediate,
		chainTail:    [][]byte{f.intermediate.certificate.Raw},
	}
	if edit != nil {
		edit(&parts)
	}
	authData := attestationAuthData(parts.appID, parts.counter, parts.aaguid, parts.credentialID)
	leaf := credentialCertificate(t, parts.issuer, &f.credential.PublicKey, challengeDigest(authData, parts.clientData))
	chain := [][]byte{cborByteString(leaf)}
	for _, der := range parts.chainTail {
		chain = append(chain, cborByteString(der))
	}
	return cborMapOf(
		cborTextString("fmt"), cborTextString(parts.format),
		cborTextString("attStmt"), cborMapOf(
			cborTextString("x5c"), cborArrayOf(chain...),
			cborTextString("receipt"), cborByteString([]byte("receipt")),
		),
		cborTextString("authData"), cborByteString(authData),
	)
}

func TestAppAttestAttestation(t *testing.T) {
	cases := []struct {
		name             string
		allowDevelopment bool
		edit             func(t *testing.T, p *attestationParts)
		keyID            func(f *attestFixture) []byte
		reasons          []Reason
	}{
		{name: "genuine app"},
		{name: "wrong rp id", edit: func(_ *testing.T, p *attestationParts) { p.appID = testTeamID + ".com.example.clone" },
			reasons: []Reason{ReasonAppIDMismatch}},
		{name: "develop environment in production", edit: func(_ *testing.T, p *attestationParts) { p.aaguid = developmentAAGUID },
			reasons: []Reason{ReasonEnvironmentMismatch}},
		{name: "develop environment where allowed", allowDevelopment: true, edit: func(_ *testing.T, p *attestationParts) { p.aaguid = developmentAAGUID }},
		{name: "unknown aaguid", allowDevelopment: true, edit: func(_ *testing.T, p *attestationParts) { p.aaguid = []byte("0123456789abcdef") },
			reasons: []Reason{ReasonEnvironmentMismatch}},
		{name: "attested for another challenge", edit: func(_ *testing.T, p *attestationParts) { p.clientData = []byte("another challenge") },
			reasons: []Reason{ReasonChallengeMismatch}},
		{name: "nonzero counter", edit: func(_ *testing.T, p *attestationParts) { p.counter = 1 },
			reasons: []Reason{ReasonCounterInvalid}},
		{name: "credential id of another key", edit: func(_ *testing.T, p *attestationParts) { p.credentialID = make([]byte, 32) },
			reasons: []Reason{ReasonKeyIDMismatch}},
		{name: "key id of another key", keyID: func(_ *attestFixture) []byte { return make([]byte, 32) },
			reasons: []Reason{ReasonKeyIDMismatch}},
		{name: "chain from an untrusted root", edit: func(t *testing.T, p *attestationParts) {
			other := newTestIntermediate(t, newTestRoot(t, "Untrusted Root"))
			p.issuer, p.chainTail = other, [][]byte{other.certificate.Raw}
		}, reasons: []Reason{ReasonCertificateChain}},
		{name: "missing intermediate", edit: func(_ *testing.T, p *attestationParts) { p.chainTail = nil },
			reasons: []Reason{ReasonCertificateChain}},
		{name: "other format", edit: func(_ *testing.T, p *attestationParts) { p.format = "packed" },
			reasons: []Reason{ReasonAttestationMalformed}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newAttestFixture(t, tc.allowDevelopment)
			nonce := fixture.issueNonce(t, sessionBinding())
			var edit func(*attestationParts)
			if tc.edit != nil {
				edit = func(p *attestationParts) { tc.edit(t, p) }
			}
			attestation := fixture.attestation(t, nonce, edit)
			keyID := keyIDOf(t, fixture.credential)
			if tc.keyID != nil {
				keyID = tc.keyID(fixture)
			}
			decision, key, err := fixture.verifier.Attest(context.Background(), strictPolicy(), sessionBinding(), nonce, keyID, attestation)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(decision.Reasons, tc.reasons) {
				t.Fatalf("reasons = %v, want %v", decision.Reasons, tc.reasons)
			}
			if len(tc.reasons) > 0 {
				if decision.Allowed() || key.PublicKey != nil {
					t.Fatalf("failed attestation allowed = %v, key = %+v", decision.Allowed(), key)
				}
				return
			}
			if !decision.Allowed() || !slices.Equal(key.PublicKey, publicKeyBytes(t, fixture.credential)) || !slices.Equal(key.KeyID, keyID) || key.Counter != 0 || string(key.Receipt) != "receipt" || key.Development != tc.allowDevelopment {
				t.Fatalf("decision = %+v, key = %+v", decision, key)
			}
		})
	}
}

func TestAppAttestAttestationRejectsBadNonces(t *testing.T) {
	fixture := newAttestFixture(t, false)
	keyID := keyIDOf(t, fixture.credential)
	issued := fixture.issueNonce(t, sessionBinding())
	attestation := fixture.attestation(t, issued, nil)
	first, _, err := fixture.verifier.Attest(context.Background(), strictPolicy(), sessionBinding(), issued, keyID, attestation)
	if err != nil || !first.Passed() {
		t.Fatalf("first = %+v, %v", first, err)
	}
	cases := map[string]struct {
		nonce   string
		binding Binding
		want    Reason
	}{
		"replayed":       {issued, sessionBinding(), ReasonNonceUnknown},
		"never issued":   {"self-chosen challenge", sessionBinding(), ReasonNonceUnknown},
		"missing":        {"", sessionBinding(), ReasonNonceMissing},
		"other purpose":  {fixture.issueNonce(t, sessionBinding()), Binding{Account: "account-1", Session: "session-1", Purpose: PurposePurchase}, ReasonNonceBindingMismatch},
		"other accounts": {fixture.issueNonce(t, sessionBinding()), Binding{Account: "account-2", Session: "session-1", Purpose: PurposeSessionStart}, ReasonNonceBindingMismatch},
	}
	for name, tc := range cases {
		decision, _, err := fixture.verifier.Attest(context.Background(), strictPolicy(), tc.binding, tc.nonce, keyID, fixture.attestation(t, tc.nonce, nil))
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(decision.Reasons, []Reason{tc.want}) || decision.Allowed() {
			t.Errorf("%s: decision = %+v, want blocked %v", name, decision, tc.want)
		}
	}
}

func TestAppAttestAttestationRejectsGarbage(t *testing.T) {
	fixture := newAttestFixture(t, false)
	keyID := keyIDOf(t, fixture.credential)
	for name, attestation := range map[string][]byte{
		"empty":         nil,
		"not cbor":      []byte("attestation"),
		"not a map":     cborArrayOf(),
		"no statement":  cborMapOf(cborTextString("fmt"), cborTextString(appAttestFormat), cborTextString("authData"), cborByteString(make([]byte, 64))),
		"short authdat": cborMapOf(cborTextString("fmt"), cborTextString(appAttestFormat), cborTextString("attStmt"), cborMapOf(cborTextString("x5c"), cborArrayOf(), cborTextString("receipt"), cborByteString(nil)), cborTextString("authData"), cborByteString(make([]byte, 10))),
	} {
		nonce := fixture.issueNonce(t, sessionBinding())
		decision, _, err := fixture.verifier.Attest(context.Background(), strictPolicy(), sessionBinding(), nonce, keyID, attestation)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(decision.Reasons, []Reason{ReasonAttestationMalformed}) {
			t.Errorf("%s: reasons = %v", name, decision.Reasons)
		}
	}
}

func (f *attestFixture) attestedKey(t *testing.T, counter uint32) AttestedKey {
	t.Helper()
	return AttestedKey{KeyID: keyIDOf(t, f.credential), PublicKey: publicKeyBytes(t, f.credential), Counter: counter}
}

func signAssertion(t *testing.T, signer *ecdsa.PrivateKey, authData, clientData []byte) []byte {
	t.Helper()
	digest := sha256.Sum256(challengeDigest(authData, clientData))
	signature, err := ecdsa.SignASN1(rand.Reader, signer, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return cborMapOf(cborTextString("signature"), cborByteString(signature), cborTextString("authenticatorData"), cborByteString(authData))
}

func TestAppAttestAssertion(t *testing.T) {
	cases := []struct {
		name        string
		stored      uint32
		counter     uint32
		appID       string
		otherSigner bool
		otherData   bool
		reasons     []Reason
	}{
		{name: "first assertion", stored: 0, counter: 1, appID: testAppID},
		{name: "later assertion", stored: 7, counter: 9, appID: testAppID},
		{name: "counter replay", stored: 7, counter: 7, appID: testAppID, reasons: []Reason{ReasonCounterReplay}},
		{name: "counter rollback", stored: 7, counter: 3, appID: testAppID, reasons: []Reason{ReasonCounterReplay}},
		{name: "wrong rp id", stored: 0, counter: 1, appID: testTeamID + ".com.example.clone", reasons: []Reason{ReasonAppIDMismatch}},
		{name: "signed by another key", stored: 0, counter: 1, appID: testAppID, otherSigner: true, reasons: []Reason{ReasonSignatureInvalid}},
		{name: "signed over other client data", stored: 0, counter: 1, appID: testAppID, otherData: true, reasons: []Reason{ReasonSignatureInvalid}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newAttestFixture(t, false)
			binding := Binding{Account: "account-1", Session: "session-1", Purpose: PurposePurchase}
			nonce := fixture.issueNonce(t, binding)
			signer, clientData := fixture.credential, []byte(nonce)
			if tc.otherSigner {
				signer = newTestKey(t, elliptic.P256())
			}
			if tc.otherData {
				clientData = []byte("other request")
			}
			assertion := signAssertion(t, signer, assertionAuthData(tc.appID, tc.counter), clientData)
			decision, counter, err := fixture.verifier.Assert(context.Background(), strictPolicy(), binding, nonce, fixture.attestedKey(t, tc.stored), assertion)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(decision.Reasons, tc.reasons) {
				t.Fatalf("reasons = %v, want %v", decision.Reasons, tc.reasons)
			}
			wantCounter := tc.counter
			if len(tc.reasons) > 0 {
				wantCounter = tc.stored
			}
			if counter != wantCounter || decision.Allowed() != (len(tc.reasons) == 0) {
				t.Fatalf("counter = %d, allowed = %v", counter, decision.Allowed())
			}
		})
	}
}

func TestAppAttestAssertionRejectsReplayedNonce(t *testing.T) {
	fixture := newAttestFixture(t, false)
	nonce := fixture.issueNonce(t, sessionBinding())
	assertion := signAssertion(t, fixture.credential, assertionAuthData(testAppID, 1), []byte(nonce))
	first, counter, err := fixture.verifier.Assert(context.Background(), strictPolicy(), sessionBinding(), nonce, fixture.attestedKey(t, 0), assertion)
	if err != nil || !first.Passed() || counter != 1 {
		t.Fatalf("first = %+v, counter %d, %v", first, counter, err)
	}
	replay, _, err := fixture.verifier.Assert(context.Background(), strictPolicy(), sessionBinding(), nonce, fixture.attestedKey(t, 0), assertion)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(replay.Reasons, []Reason{ReasonNonceUnknown}) || replay.Allowed() {
		t.Fatalf("replay = %+v", replay)
	}
}

func TestAppAttestAssertionRejectsDevelopmentKeyInProduction(t *testing.T) {
	fixture := newAttestFixture(t, false)
	nonce := fixture.issueNonce(t, sessionBinding())
	key := fixture.attestedKey(t, 0)
	key.Development = true
	assertion := signAssertion(t, fixture.credential, assertionAuthData(testAppID, 1), []byte(nonce))
	decision, _, err := fixture.verifier.Assert(context.Background(), strictPolicy(), sessionBinding(), nonce, key, assertion)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(decision.Reasons, []Reason{ReasonEnvironmentMismatch}) {
		t.Fatalf("reasons = %v", decision.Reasons)
	}
}

func TestAppAttestAssertionRejectsMalformedInput(t *testing.T) {
	fixture := newAttestFixture(t, false)
	for name, assertion := range map[string][]byte{
		"not cbor":       []byte("assertion"),
		"no signature":   cborMapOf(cborTextString("authenticatorData"), cborByteString(assertionAuthData(testAppID, 1))),
		"short authdata": cborMapOf(cborTextString("signature"), cborByteString([]byte{1}), cborTextString("authenticatorData"), cborByteString([]byte{1, 2})),
	} {
		nonce := fixture.issueNonce(t, sessionBinding())
		decision, _, err := fixture.verifier.Assert(context.Background(), strictPolicy(), sessionBinding(), nonce, fixture.attestedKey(t, 0), assertion)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(decision.Reasons, []Reason{ReasonAttestationMalformed}) {
			t.Errorf("%s: reasons = %v", name, decision.Reasons)
		}
	}
	if _, _, err := fixture.verifier.Assert(context.Background(), strictPolicy(), sessionBinding(), "n", AttestedKey{PublicKey: []byte{4, 1}}, nil); err == nil {
		t.Fatal("invalid stored key accepted")
	}
}

func TestNewAppAttestRejectsInvalidConfig(t *testing.T) {
	clock := newTestClock()
	nonces, _ := newTestNonces(t, clock)
	root := newTestRoot(t, "root")
	rootPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: root.certificate.Raw})
	for name, config := range map[string]AppAttestConfig{
		"no team":   {BundleID: testBundleID, RootCertificatePEM: rootPEM},
		"no bundle": {TeamID: testTeamID, RootCertificatePEM: rootPEM},
		"no root":   {TeamID: testTeamID, BundleID: testBundleID},
	} {
		if _, err := NewAppAttest(config, nonces, clock); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestDecodeCBORRejectsMalformedInput(t *testing.T) {
	nested := cborArrayOf(cborArrayOf(cborArrayOf(cborArrayOf(cborArrayOf(cborArrayOf())))))
	for name, data := range map[string][]byte{
		"empty":             nil,
		"truncated bytes":   {0x45, 1, 2},
		"truncated head":    {0x59, 0x01},
		"indefinite length": {0x5f, 0x41, 0x00, 0xff},
		"reserved info":     {0x1c},
		"trailing data":     {0x01, 0x02},
		"negative integer":  {0x20},
		"float":             {0xf9, 0x00, 0x00},
		"integer map key":   cborMapOf(cborHead(cborUnsigned, 1), cborHead(cborUnsigned, 2)),
		"duplicate key":     cborMapOf(cborTextString("a"), cborHead(cborUnsigned, 1), cborTextString("a"), cborHead(cborUnsigned, 2)),
		"invalid utf8":      {0x61, 0xff},
		"huge array":        cborHead(cborArray, 1<<40),
		"huge map":          cborHead(cborMap, 1<<40),
		"too deep":          nested,
	} {
		if _, err := decodeCBOR(data); err == nil {
			t.Errorf("%s: decoded", name)
		}
	}
	value, err := decodeCBOR(cborMapOf(cborTextString("n"), cborHead(cborUnsigned, 70000), cborTextString("b"), cborByteString([]byte{9})))
	if err != nil {
		t.Fatal(err)
	}
	entries := value.(map[string]any)
	if entries["n"] != uint64(70000) || string(entries["b"].([]byte)) != "\x09" {
		t.Fatalf("decoded %v", entries)
	}
}

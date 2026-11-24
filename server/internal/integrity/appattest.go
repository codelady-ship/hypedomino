package integrity

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	appAttestFormat        = "apple-appattest"
	rpIDHashLength         = 32
	flagsOffset            = 32
	counterOffset          = 33
	aaguidOffset           = 37
	credentialLengthOffset = 53
	credentialIDOffset     = 55
	attestedCredentialFlag = 0x40
)

var (
	nonceExtensionOID     = asn1.ObjectIdentifier{1, 2, 840, 113635, 100, 8, 2}
	productionAAGUID      = []byte("appattest\x00\x00\x00\x00\x00\x00\x00")
	developmentAAGUID     = []byte("appattestdevelop")
	errMalformedAuthData  = errors.New("integrity: malformed authenticator data")
	errMalformedAttestKey = errors.New("integrity: stored attested key is invalid")
)

type AppAttestConfig struct {
	TeamID             string
	BundleID           string
	RootCertificatePEM []byte
	AllowDevelopment   bool
}

type AttestedKey struct {
	KeyID       []byte
	PublicKey   []byte
	Counter     uint32
	Receipt     []byte
	Development bool
}

type AppAttest struct {
	appIDHash        [sha256.Size]byte
	roots            *x509.CertPool
	allowDevelopment bool
	nonces           *Nonces
	clock            Clock
}

func NewAppAttest(config AppAttestConfig, nonces *Nonces, clock Clock) (*AppAttest, error) {
	switch {
	case config.TeamID == "" || config.BundleID == "":
		return nil, errors.New("integrity: team id and bundle id are required")
	case nonces == nil || clock == nil:
		return nil, errors.New("integrity: nonces and clock are required")
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(config.RootCertificatePEM) {
		return nil, errors.New("integrity: no App Attest root certificate in PEM")
	}
	return &AppAttest{
		appIDHash:        sha256.Sum256([]byte(config.TeamID + "." + config.BundleID)),
		roots:            roots,
		allowDevelopment: config.AllowDevelopment,
		nonces:           nonces,
		clock:            clock,
	}, nil
}

func (a *AppAttest) Attest(ctx context.Context, policy Policy, binding Binding, nonce string, keyID, attestation []byte) (Decision, AttestedKey, error) {
	nonceReason, err := a.nonces.redeem(ctx, nonce, binding)
	if err != nil {
		return Decision{}, AttestedKey{}, err
	}
	if nonceReason != 0 {
		return decide(policy, binding.Purpose, []Reason{nonceReason}), AttestedKey{}, nil
	}
	key, reasons := a.verifyAttestation(attestation, []byte(nonce), keyID)
	if len(reasons) > 0 {
		return decide(policy, binding.Purpose, reasons), AttestedKey{}, nil
	}
	return decide(policy, binding.Purpose, nil), key, nil
}

func (a *AppAttest) Assert(ctx context.Context, policy Policy, binding Binding, nonce string, key AttestedKey, assertion []byte) (Decision, uint32, error) {
	publicKey, err := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), key.PublicKey)
	if err != nil {
		return Decision{}, key.Counter, fmt.Errorf("%w: %v", errMalformedAttestKey, err)
	}
	nonceReason, err := a.nonces.redeem(ctx, nonce, binding)
	if err != nil {
		return Decision{}, key.Counter, err
	}
	if nonceReason != 0 {
		return decide(policy, binding.Purpose, []Reason{nonceReason}), key.Counter, nil
	}
	counter, reasons := a.verifyAssertion(assertion, []byte(nonce), publicKey, key)
	if len(reasons) > 0 {
		return decide(policy, binding.Purpose, reasons), key.Counter, nil
	}
	return decide(policy, binding.Purpose, nil), counter, nil
}

type attestationObject struct {
	certificates [][]byte
	receipt      []byte
	authData     []byte
}

type authenticatorData struct {
	rpIDHash     []byte
	counter      uint32
	aaguid       []byte
	credentialID []byte
}

func (a *AppAttest) verifyAttestation(encoded, clientData, keyID []byte) (AttestedKey, []Reason) {
	object, parsed := parseAttestationObject(encoded)
	if !parsed {
		return AttestedKey{}, []Reason{ReasonAttestationMalformed}
	}
	authData, err := parseAuthenticatorData(object.authData, true)
	if err != nil {
		return AttestedKey{}, []Reason{ReasonAttestationMalformed}
	}
	credential, err := a.verifyChain(object.certificates)
	if err != nil {
		return AttestedKey{}, []Reason{ReasonCertificateChain}
	}
	publicKey, isECDSA := credential.PublicKey.(*ecdsa.PublicKey)
	if !isECDSA || publicKey.Curve != elliptic.P256() {
		return AttestedKey{}, []Reason{ReasonAttestationMalformed}
	}
	publicKeyBytes, err := publicKey.Bytes()
	if err != nil {
		return AttestedKey{}, []Reason{ReasonAttestationMalformed}
	}
	var reasons []Reason
	if !bytes.Equal(certificateNonce(credential), challengeDigest(object.authData, clientData)) {
		reasons = append(reasons, ReasonChallengeMismatch)
	}
	publicKeyHash := sha256.Sum256(publicKeyBytes)
	if !bytes.Equal(publicKeyHash[:], keyID) || !bytes.Equal(authData.credentialID, keyID) {
		reasons = append(reasons, ReasonKeyIDMismatch)
	}
	if !bytes.Equal(authData.rpIDHash, a.appIDHash[:]) {
		reasons = append(reasons, ReasonAppIDMismatch)
	}
	if authData.counter != 0 {
		reasons = append(reasons, ReasonCounterInvalid)
	}
	development, environmentAccepted := a.environment(authData.aaguid)
	if !environmentAccepted {
		reasons = append(reasons, ReasonEnvironmentMismatch)
	}
	key := AttestedKey{KeyID: keyID, PublicKey: publicKeyBytes, Receipt: object.receipt, Development: development}
	return key, reasons
}

func (a *AppAttest) verifyChain(certificates [][]byte) (*x509.Certificate, error) {
	if len(certificates) == 0 {
		return nil, errors.New("integrity: empty certificate chain")
	}
	parsed := make([]*x509.Certificate, 0, len(certificates))
	for _, der := range certificates {
		certificate, err := x509.ParseCertificate(der)
		if err != nil {
			return nil, err
		}
		parsed = append(parsed, certificate)
	}
	intermediates := x509.NewCertPool()
	for _, certificate := range parsed[1:] {
		intermediates.AddCert(certificate)
	}
	_, err := parsed[0].Verify(x509.VerifyOptions{
		Roots:         a.roots,
		Intermediates: intermediates,
		CurrentTime:   a.clock.Now(),
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	})
	return parsed[0], err
}

func (a *AppAttest) environment(aaguid []byte) (development, accepted bool) {
	switch {
	case bytes.Equal(aaguid, productionAAGUID):
		return false, true
	case bytes.Equal(aaguid, developmentAAGUID):
		return true, a.allowDevelopment
	}
	return false, false
}

func (a *AppAttest) verifyAssertion(encoded, clientData []byte, publicKey *ecdsa.PublicKey, key AttestedKey) (uint32, []Reason) {
	signature, authDataBytes, parsed := parseAssertion(encoded)
	if !parsed {
		return 0, []Reason{ReasonAttestationMalformed}
	}
	authData, err := parseAuthenticatorData(authDataBytes, false)
	if err != nil {
		return 0, []Reason{ReasonAttestationMalformed}
	}
	var reasons []Reason
	signedDigest := sha256.Sum256(challengeDigest(authDataBytes, clientData))
	if !ecdsa.VerifyASN1(publicKey, signedDigest[:], signature) {
		reasons = append(reasons, ReasonSignatureInvalid)
	}
	if !bytes.Equal(authData.rpIDHash, a.appIDHash[:]) {
		reasons = append(reasons, ReasonAppIDMismatch)
	}
	if authData.counter <= key.Counter {
		reasons = append(reasons, ReasonCounterReplay)
	}
	if key.Development && !a.allowDevelopment {
		reasons = append(reasons, ReasonEnvironmentMismatch)
	}
	return authData.counter, reasons
}

func challengeDigest(authData, clientData []byte) []byte {
	clientDataHash := sha256.Sum256(clientData)
	digest := sha256.Sum256(append(append([]byte{}, authData...), clientDataHash[:]...))
	return digest[:]
}

func certificateNonce(certificate *x509.Certificate) []byte {
	for _, extension := range certificate.Extensions {
		if !extension.Id.Equal(nonceExtensionOID) {
			continue
		}
		var value struct {
			Nonce []byte `asn1:"tag:1,explicit"`
		}
		rest, err := asn1.Unmarshal(extension.Value, &value)
		if err != nil || len(rest) > 0 {
			return nil
		}
		return value.Nonce
	}
	return nil
}

func parseAttestationObject(encoded []byte) (attestationObject, bool) {
	decoded, err := decodeCBOR(encoded)
	if err != nil {
		return attestationObject{}, false
	}
	root, isMap := decoded.(map[string]any)
	if !isMap || root["fmt"] != appAttestFormat {
		return attestationObject{}, false
	}
	statement, isMap := root["attStmt"].(map[string]any)
	authData, hasAuthData := root["authData"].([]byte)
	if !isMap || !hasAuthData {
		return attestationObject{}, false
	}
	receipt, hasReceipt := statement["receipt"].([]byte)
	chain, hasChain := statement["x5c"].([]any)
	if !hasReceipt || !hasChain {
		return attestationObject{}, false
	}
	certificates := make([][]byte, 0, len(chain))
	for _, entry := range chain {
		der, isBytes := entry.([]byte)
		if !isBytes {
			return attestationObject{}, false
		}
		certificates = append(certificates, der)
	}
	return attestationObject{certificates: certificates, receipt: receipt, authData: authData}, true
}

func parseAssertion(encoded []byte) (signature, authData []byte, parsed bool) {
	decoded, err := decodeCBOR(encoded)
	if err != nil {
		return nil, nil, false
	}
	root, isMap := decoded.(map[string]any)
	if !isMap {
		return nil, nil, false
	}
	signature, hasSignature := root["signature"].([]byte)
	authData, hasAuthData := root["authenticatorData"].([]byte)
	return signature, authData, hasSignature && hasAuthData
}

func parseAuthenticatorData(data []byte, withCredential bool) (authenticatorData, error) {
	if len(data) < aaguidOffset {
		return authenticatorData{}, errMalformedAuthData
	}
	parsed := authenticatorData{
		rpIDHash: data[:rpIDHashLength],
		counter:  binary.BigEndian.Uint32(data[counterOffset:aaguidOffset]),
	}
	if !withCredential {
		return parsed, nil
	}
	if data[flagsOffset]&attestedCredentialFlag == 0 || len(data) < credentialIDOffset {
		return authenticatorData{}, errMalformedAuthData
	}
	credentialEnd := credentialIDOffset + int(binary.BigEndian.Uint16(data[credentialLengthOffset:credentialIDOffset]))
	if len(data) < credentialEnd {
		return authenticatorData{}, errMalformedAuthData
	}
	parsed.aaguid = data[aaguidOffset:credentialLengthOffset]
	parsed.credentialID = data[credentialIDOffset:credentialEnd]
	return parsed, nil
}

package integrity

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/googleauth/googleauthtest"
)

func newPlayDecoder(t *testing.T) (*PlayDecoder, *googleauthtest.Server) {
	t.Helper()
	play := googleauthtest.New(t, playIntegrityScope)
	play.Mux.HandleFunc("POST /v1/com.example.hypedomino:decodeIntegrityToken", func(w http.ResponseWriter, r *http.Request) {
		var request map[string]string
		if !googleauthtest.Authorized(r) || r.Header.Get("Content-Type") != "application/json" || json.NewDecoder(r.Body).Decode(&request) != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch request["integrity_token"] {
		case "tok-ok":
			w.Write([]byte(`{"tokenPayloadExternal":{"requestDetails":{"requestPackageName":"com.example.hypedomino","nonce":"n-1","timestampMillis":"1793448000000"},"appIntegrity":{"appRecognitionVerdict":"PLAY_RECOGNIZED","packageName":"com.example.hypedomino","certificateSha256Digest":["abc"],"versionCode":"42"},"deviceIntegrity":{"deviceRecognitionVerdict":["MEETS_DEVICE_INTEGRITY"]},"accountDetails":{"appLicensingVerdict":"LICENSED"}}}`))
		case "tok-forged":
			http.Error(w, `{"error":{"code":400,"status":"INVALID_ARGUMENT"}}`, http.StatusBadRequest)
		case "tok-empty":
			w.Write([]byte(`{}`))
		default:
			http.Error(w, "boom", http.StatusInternalServerError)
		}
	})
	decoder, err := NewPlayDecoder(PlayDecoderConfig{PackageName: "com.example.hypedomino", ServiceAccountFile: play.File, Endpoint: play.URL, Now: newTestClock().Now})
	if err != nil {
		t.Fatal(err)
	}
	return decoder, play
}

func TestPlayDecoderReturnsTheServerDecodedVerdict(t *testing.T) {
	decoder, play := newPlayDecoder(t)
	for range 2 {
		payload, err := decoder.Decode(t.Context(), "tok-ok")
		if err != nil {
			t.Fatal(err)
		}
		if payload.RequestDetails.Nonce != "n-1" || payload.AppIntegrity.AppRecognitionVerdict != playRecognized || payload.AppIntegrity.CertificateSha256Digest[0] != "abc" || payload.DeviceIntegrity.DeviceRecognitionVerdict[0] != meetsDeviceIntegrity || payload.AccountDetails.AppLicensingVerdict != licensed {
			t.Fatalf("payload %+v", payload)
		}
	}
	if play.Tokens() != 1 {
		t.Fatalf("%d service account tokens for two calls", play.Tokens())
	}
}

func TestPlayDecoderSeparatesRejectedTokensFromOutages(t *testing.T) {
	decoder, play := newPlayDecoder(t)
	cases := map[string]error{"tok-forged": ErrTokenRejected, "tok-empty": ErrDecoderUnavailable, "tok-broken": ErrDecoderUnavailable}
	for token, want := range cases {
		if _, err := decoder.Decode(t.Context(), token); !errors.Is(err, want) {
			t.Fatalf("%s: %v", token, err)
		}
	}
	play.Close()
	if _, err := decoder.Decode(t.Context(), "tok-ok"); !errors.Is(err, ErrDecoderUnavailable) {
		t.Fatalf("service down: %v", err)
	}
}

func TestPlayDecoderNeedsAUsableServiceAccount(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.json")
	for name, config := range map[string]PlayDecoderConfig{
		"no package": {ServiceAccountFile: missing},
		"no file":    {PackageName: "com.example.hypedomino"},
		"missing":    {PackageName: "com.example.hypedomino", ServiceAccountFile: missing},
	} {
		if _, err := NewPlayDecoder(config); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
}

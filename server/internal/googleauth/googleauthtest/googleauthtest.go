package googleauthtest

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/googleauth"
)

const (
	Email       = "service@example.iam.gserviceaccount.com"
	AccessToken = "at-1"
)

type Server struct {
	*httptest.Server
	Mux   *http.ServeMux
	File  string
	key   *rsa.PrivateKey
	scope string
	mu    sync.Mutex
	count int
}

func New(t *testing.T, scope string) *Server {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{Mux: http.NewServeMux(), key: key, scope: scope}
	s.Mux.HandleFunc("POST /token", s.token)
	s.Server = httptest.NewServer(s.Mux)
	t.Cleanup(s.Server.Close)
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	account, _ := json.Marshal(map[string]string{
		"client_email": Email,
		"private_key":  string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})),
		"token_uri":    s.URL + "/token",
	})
	s.File = filepath.Join(t.TempDir(), "service-account.json")
	if err := os.WriteFile(s.File, account, 0o600); err != nil {
		t.Fatal(err)
	}
	return s
}

func (s *Server) Tokens() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.count
}

func Authorized(r *http.Request) bool {
	return r.Header.Get("Authorization") == "Bearer "+AccessToken
}

func (s *Server) token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil || r.Form.Get("grant_type") != googleauth.Grant {
		http.Error(w, "grant", http.StatusBadRequest)
		return
	}
	segments := strings.Split(r.Form.Get("assertion"), ".")
	if len(segments) != 3 {
		http.Error(w, "assertion", http.StatusBadRequest)
		return
	}
	signature, _ := base64.RawURLEncoding.DecodeString(segments[2])
	digest := sha256.Sum256([]byte(segments[0] + "." + segments[1]))
	if rsa.VerifyPKCS1v15(&s.key.PublicKey, crypto.SHA256, digest[:], signature) != nil {
		http.Error(w, "signature", http.StatusUnauthorized)
		return
	}
	payload, _ := base64.RawURLEncoding.DecodeString(segments[1])
	var claims map[string]any
	if json.Unmarshal(payload, &claims) != nil || claims["iss"] != Email || claims["scope"] != s.scope || claims["aud"] != s.URL+"/token" {
		http.Error(w, "claims", http.StatusUnauthorized)
		return
	}
	s.mu.Lock()
	s.count++
	s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"access_token":"` + AccessToken + `","expires_in":3600,"token_type":"Bearer"}`))
}

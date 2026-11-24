package token

import "net/http"

type Verifier interface {
	Verify(access string) (Claims, error)
}

func Authorize(verifier Verifier, r *http.Request) (Claims, error) {
	return verifier.Verify(bearer(r))
}

type AppealVerifier interface {
	VerifyAppeal(token string) (AppealClaims, error)
}

func AuthorizeAppeal(verifier AppealVerifier, r *http.Request) (AppealClaims, error) {
	return verifier.VerifyAppeal(bearer(r))
}

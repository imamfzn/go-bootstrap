package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/imamfzn/go-bootstrap/internal/domain"
	"github.com/imamfzn/go-bootstrap/internal/http/response"
)

// Verifier checks a bearer token and returns the user it belongs to.
// Implement it for your identity provider (Firebase Auth, an OIDC issuer,
// your own session store) and pass it to router.New.
type Verifier interface {
	Verify(ctx context.Context, token string) (userID string, err error)
}

// VerifierFunc adapts a function to a Verifier.
type VerifierFunc func(ctx context.Context, token string) (string, error)

// Verify calls f.
func (f VerifierFunc) Verify(ctx context.Context, token string) (string, error) {
	return f(ctx, token)
}

// DevInsecureVerifier accepts any token and uses it as the user ID. It
// exists so the API can be tried locally with no identity provider:
//
//	curl -H 'Authorization: Bearer user-1' ...
//
// It performs no verification at all. Never enable it outside a developer
// machine.
func DevInsecureVerifier() Verifier {
	return VerifierFunc(func(_ context.Context, token string) (string, error) {
		return token, nil
	})
}

type userIDKey struct{}

// UserID returns the authenticated user of the request, or an empty string
// outside routes protected by Auth.
func UserID(ctx context.Context) string {
	id, _ := ctx.Value(userIDKey{}).(string)
	return id
}

// Auth requires a valid bearer token and stores its user in the context.
func Auth(verifier Verifier, errs *response.ErrorWriter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || token == "" {
				errs.Write(w, r, domain.ErrUnauthorized)
				return
			}

			userID, err := verifier.Verify(r.Context(), token)
			if err != nil || userID == "" {
				errs.Write(w, r, fmt.Errorf("%w: verify token: %v", domain.ErrUnauthorized, err))
				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userIDKey{}, userID)))
		})
	}
}

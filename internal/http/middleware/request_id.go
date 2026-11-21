// Package middleware holds the HTTP middleware of the REST API.
package middleware

import (
	"net/http"

	"github.com/imamfzn/go-bootstrap/internal/platform/ident"
	"github.com/imamfzn/go-bootstrap/internal/platform/requestid"
)

const headerRequestID = "X-Request-ID"

// maxRequestIDLength bounds an identifier supplied by a client, since it
// is copied into every log line of the request.
const maxRequestIDLength = 64

// RequestID gives every request an identifier: the one the client sent, or
// a new one. It is stored in the context and echoed in the response.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(headerRequestID)
		if id == "" || len(id) > maxRequestIDLength {
			id = ident.New()
		}

		w.Header().Set(headerRequestID, id)
		next.ServeHTTP(w, r.WithContext(requestid.With(r.Context(), id)))
	})
}

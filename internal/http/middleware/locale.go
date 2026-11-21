package middleware

import (
	"net/http"

	"github.com/imamfzn/go-bootstrap/internal/platform/i18n"
)

// Locale picks the response language from Accept-Language and stores it in
// the context.
func Locale(catalog *i18n.Catalog) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			locale := catalog.Match(r.Header.Get("Accept-Language"))

			w.Header().Set("Content-Language", locale)
			next.ServeHTTP(w, r.WithContext(i18n.WithLocale(r.Context(), locale)))
		})
	}
}

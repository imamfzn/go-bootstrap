package middleware

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/imamfzn/go-bootstrap/internal/http/response"
)

// Recover turns a panic in a handler into a 500 response, so one bad
// request cannot take the process down.
func Recover(logger *slog.Logger, errs *response.ErrorWriter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				rec := recover()
				if rec == nil {
					return
				}
				// net/http uses this value to abort a response on purpose.
				if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(rec)
				}

				logger.ErrorContext(r.Context(), "handler panicked",
					"panic", fmt.Sprint(rec),
					"stack", string(debug.Stack()),
				)
				errs.Write(w, r, fmt.Errorf("handler panicked: %v", rec))
			}()

			next.ServeHTTP(w, r)
		})
	}
}

package response

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/imamfzn/go-bootstrap/internal/domain"
	"github.com/imamfzn/go-bootstrap/internal/platform/i18n"
)

// ErrorBody is the body of every error response. Clients switch on Code;
// Message is for people and is translated.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// defaultDomainStatus is the status of a domain error with no entry in
// statusByError: the request was understood but is not acceptable.
const defaultDomainStatus = http.StatusUnprocessableEntity

// statusByError lists the domain errors that are not a 422.
var statusByError = map[*domain.Error]int{
	domain.ErrInternal:         http.StatusInternalServerError,
	domain.ErrRequestMalformed: http.StatusBadRequest,
	domain.ErrUnauthorized:     http.StatusUnauthorized,
	domain.ErrForbidden:        http.StatusForbidden,
	domain.ErrNotFound:         http.StatusNotFound,
	domain.ErrMethodNotAllowed: http.StatusMethodNotAllowed,

	domain.ErrOrderNotFound:         http.StatusNotFound,
	domain.ErrOrderAlreadyExists:    http.StatusConflict,
	domain.ErrOrderCancelNotAllowed: http.StatusConflict,
	domain.ErrOrderPayNotAllowed:    http.StatusConflict,
}

// StatusFor returns the HTTP status of a domain error.
func StatusFor(e *domain.Error) int {
	if status, ok := statusByError[e]; ok {
		return status
	}
	return defaultDomainStatus
}

// ErrorWriter turns an error into a response. It is the only place that
// knows HTTP statuses and the only place that translates messages.
type ErrorWriter struct {
	catalog *i18n.Catalog
	logger  *slog.Logger
}

// NewErrorWriter returns an ErrorWriter.
func NewErrorWriter(catalog *i18n.Catalog, logger *slog.Logger) *ErrorWriter {
	return &ErrorWriter{catalog: catalog, logger: logger}
}

// Write responds with the domain error found in err. Any other error is
// logged in full and answered with a generic 500, so internal detail never
// reaches a client.
func (e *ErrorWriter) Write(w http.ResponseWriter, r *http.Request, err error) {
	ctx := r.Context()

	domainErr, ok := errors.AsType[*domain.Error](err)
	if !ok {
		e.logger.ErrorContext(ctx, "unhandled error", "err", err)
		domainErr = domain.ErrInternal
	}

	JSON(w, StatusFor(domainErr), ErrorBody{
		Code:    domainErr.Code,
		Message: e.catalog.Lookup(i18n.LocaleFrom(ctx), domainErr.Code, domainErr.Message),
	})
}

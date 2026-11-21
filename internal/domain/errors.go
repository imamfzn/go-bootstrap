package domain

import "slices"

// Error is a business error that is safe to show to a client. It carries no
// transport detail: each inbound adapter decides how to render it.
//
// Errors are sentinels. Return the declared value itself and compare with
// errors.Is; add detail by wrapping with %w, never by changing a field.
type Error struct {
	// Code is the stable identifier clients switch on and the key used to
	// translate Message. Format: {DOMAIN}_{USECASE}_{ERROR-CAUSE}.
	Code string
	// Message is the default, English text.
	Message string
}

func (e *Error) Error() string { return e.Code }

// registry is written only while package variables are initialised.
var registry []*Error

func newError(code, message string) *Error {
	e := &Error{Code: code, Message: message}
	registry = append(registry, e)
	return e
}

// Errors returns every declared domain error. Tests use it to check that
// each code is translated and mapped.
func Errors() []*Error { return slices.Clone(registry) }

// Errors that do not belong to one domain.
var (
	ErrInternal         = newError("INTERNAL", "Internal error")
	ErrRequestMalformed = newError("REQUEST_MALFORMED", "The request could not be read")
	ErrUnauthorized     = newError("UNAUTHORIZED", "Authentication is required")
	ErrForbidden        = newError("FORBIDDEN", "You do not have access to this resource")
	ErrNotFound         = newError("NOT_FOUND", "Resource not found")
	ErrMethodNotAllowed = newError("METHOD_NOT_ALLOWED", "Method not allowed")
)

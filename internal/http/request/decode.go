// Package request holds the request bodies of the REST API and reads them.
// It only parses: every rule about what a value may be lives in the domain
// and the use cases, so it also applies to the worker and the jobs.
package request

import (
	"encoding/json/v2"
	"fmt"
	"net/http"

	"github.com/imamfzn/go-bootstrap/internal/domain"
)

// maxBodyBytes bounds a request body.
const maxBodyBytes = 1 << 20

// Decode reads the JSON body into T. The body must be one JSON value with
// no unknown fields and nothing after it. Any failure is reported as
// domain.ErrRequestMalformed.
func Decode[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var v T

	body := http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.UnmarshalRead(body, &v, json.RejectUnknownMembers(true)); err != nil {
		var zero T
		return zero, fmt.Errorf("%w: %w", domain.ErrRequestMalformed, err)
	}
	return v, nil
}

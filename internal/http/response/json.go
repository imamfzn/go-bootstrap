// Package response holds the response bodies of the REST API and writes
// them. A domain value is never serialised directly: each one is mapped to
// a type declared here, so the API shape changes only on purpose.
package response

import (
	"encoding/json/v2"
	"net/http"
)

// JSON writes v as the response body with the given status.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	// The status line is already sent, so a failure here cannot be reported
	// to the client. It means the client went away or v cannot be encoded;
	// the second is caught by the tests of each response type.
	_ = json.MarshalWrite(w, v)
}

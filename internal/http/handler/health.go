// Package handler holds the HTTP handlers of the REST API. A handler reads
// the request, calls one use case and writes the response. It returns an
// error instead of writing one: the router turns it into a response.
package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/imamfzn/go-bootstrap/internal/http/response"
)

// Health answers the platform's probes.
type Health struct {
	ready func(ctx context.Context) error
}

// NewHealth returns the health handler. ready reports whether the service
// can reach what it depends on.
func NewHealth(ready func(ctx context.Context) error) *Health {
	return &Health{ready: ready}
}

type healthBody struct {
	Status string `json:"status"`
}

// Live reports that the process is running. It checks nothing else, so a
// database outage does not make the platform restart healthy instances.
func (h *Health) Live(w http.ResponseWriter, _ *http.Request) error {
	response.JSON(w, http.StatusOK, &healthBody{Status: "ok"})
	return nil
}

// Ready reports whether the service can serve traffic.
func (h *Health) Ready(w http.ResponseWriter, r *http.Request) error {
	if err := h.ready(r.Context()); err != nil {
		return fmt.Errorf("readiness check: %w", err)
	}
	response.JSON(w, http.StatusOK, &healthBody{Status: "ok"})
	return nil
}

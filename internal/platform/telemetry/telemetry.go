// Package telemetry is where tracing and metrics are set up.
//
// The skeleton ships without an exporter so it has no opinion about where
// telemetry goes. To add OpenTelemetry, build the tracer and meter
// providers in Setup, register them as the globals, and return a shutdown
// function that flushes them. Then wrap the HTTP handler and the outbound
// HTTP transport with otelhttp in internal/app.
package telemetry

import "context"

// Shutdown flushes and stops telemetry. Call it before the process exits.
type Shutdown func(ctx context.Context) error

// Setup starts telemetry for the service. It currently does nothing.
func Setup(_ context.Context, _ string) (Shutdown, error) {
	return func(context.Context) error { return nil }, nil
}

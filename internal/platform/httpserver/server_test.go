package httpserver_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/imamfzn/go-bootstrap/internal/platform/httpserver"
)

func TestServe_DrainsInFlightRequestOnShutdown(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	started := make(chan struct{})
	release := make(chan struct{})
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		_, _ = io.WriteString(w, "done")
	})

	ctx, cancel := context.WithCancel(t.Context())
	served := make(chan error, 1)
	go func() {
		served <- httpserver.Serve(ctx, ln, httpserver.Config{ShutdownTimeout: 5 * time.Second}, h)
	}()

	type result struct {
		body string
		err  error
	}
	got := make(chan result, 1)
	go func() {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+ln.Addr().String(), nil)
		if err != nil {
			got <- result{err: err}
			return
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			got <- result{err: err}
			return
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		got <- result{body: string(body), err: err}
	}()

	<-started
	cancel()
	close(release)

	if res := <-got; res.err != nil || res.body != "done" {
		t.Fatalf("in-flight request: body %q, err %v", res.body, res.err)
	}
	if err := <-served; err != nil {
		t.Fatalf("Serve returned %v, want nil after a clean shutdown", err)
	}
}

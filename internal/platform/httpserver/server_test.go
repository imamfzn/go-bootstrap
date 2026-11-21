package httpserver_test

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/imamfzn/go-bootstrap/internal/platform/httpserver"
)

// pipeListener is a net.Listener that never touches the network: every
// connection is an in-memory pipe.
type pipeListener struct {
	conns  chan net.Conn
	closed chan struct{}
}

func newPipeListener() *pipeListener {
	return &pipeListener{conns: make(chan net.Conn), closed: make(chan struct{})}
}

func (l *pipeListener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.conns:
		return conn, nil
	case <-l.closed:
		return nil, net.ErrClosed
	}
}

func (l *pipeListener) Close() error {
	select {
	case <-l.closed:
	default:
		close(l.closed)
	}
	return nil
}

func (l *pipeListener) Addr() net.Addr { return pipeAddr{} }

// dial hands the server one end of a pipe and returns the other.
func (l *pipeListener) dial(ctx context.Context) (net.Conn, error) {
	client, server := net.Pipe()
	select {
	case l.conns <- server:
		return client, nil
	case <-l.closed:
		return nil, net.ErrClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// client returns an HTTP client whose connections go through the listener.
func (l *pipeListener) client() *http.Client {
	return &http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) { return l.dial(ctx) },
	}}
}

type pipeAddr struct{}

func (pipeAddr) Network() string { return "pipe" }
func (pipeAddr) String() string  { return "pipe" }

func get(ctx context.Context, client *http.Client) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://service.test/", nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	return string(body), err
}

func TestServe_DrainsInFlightRequestOnShutdown(t *testing.T) {
	ln := newPipeListener()
	client := ln.client()
	t.Cleanup(client.CloseIdleConnections)

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
		served <- httpserver.Serve(ctx, ln, httpserver.Config{ShutdownTimeout: time.Minute}, h)
	}()

	type result struct {
		body string
		err  error
	}
	got := make(chan result, 1)
	go func() {
		body, err := get(t.Context(), client)
		got <- result{body, err}
	}()

	// Shut down while the request is still being handled, then let it finish.
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

func TestServe_RequestContextSurvivesShutdownSignal(t *testing.T) {
	ln := newPipeListener()
	client := ln.client()
	t.Cleanup(client.CloseIdleConnections)

	started := make(chan struct{})
	release := make(chan struct{})
	cancelled := make(chan bool, 1)
	h := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		cancelled <- r.Context().Err() != nil
	})

	ctx, cancel := context.WithCancel(t.Context())
	served := make(chan error, 1)
	go func() {
		served <- httpserver.Serve(ctx, ln, httpserver.Config{ShutdownTimeout: time.Minute}, h)
	}()
	go func() { _, _ = get(t.Context(), client) }()

	<-started
	cancel()
	close(release)

	if <-cancelled {
		t.Error("the shutdown signal cancelled an in-flight request, so its work would be abandoned")
	}
	if err := <-served; err != nil {
		t.Fatalf("Serve returned %v", err)
	}
}

func TestServe_ReportsListenerFailure(t *testing.T) {
	ln := newPipeListener()
	_ = ln.Close()

	err := httpserver.Serve(t.Context(), ln, httpserver.Config{ShutdownTimeout: time.Minute}, http.NotFoundHandler())

	if !errors.Is(err, net.ErrClosed) {
		t.Fatalf("Serve = %v, want it to report the closed listener", err)
	}
}

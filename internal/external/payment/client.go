// Package payment talks to the payment provider. It implements
// usecase.PaymentGateway.
//
// The HTTP API here is a placeholder: replace the paths and wire types with
// the real provider's, and keep the provider's types inside this package.
package payment

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/imamfzn/go-bootstrap/internal/domain"
)

// Config describes how to reach the provider.
type Config struct {
	BaseURL string
	APIKey  string
}

// Client charges through the provider's HTTP API.
type Client struct {
	http    *http.Client
	baseURL *url.URL
	apiKey  string
}

// NewClient returns a client. httpClient must have a timeout.
func NewClient(httpClient *http.Client, cfg Config) (*Client, error) {
	baseURL, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse payment base url: %w", err)
	}
	return &Client{http: httpClient, baseURL: baseURL, apiKey: cfg.APIKey}, nil
}

// Charge takes payment for an order. The order ID is sent as the
// idempotency key, so the provider charges once however often this runs.
func (c *Client) Charge(ctx context.Context, charge domain.Charge) error {
	body, err := json.Marshal(chargeRequest{Reference: charge.OrderID, Amount: charge.Amount})
	if err != nil {
		return fmt.Errorf("encode charge: %w", err)
	}

	endpoint := c.baseURL.JoinPath("v1", "charges").String()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build charge request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Idempotency-Key", charge.OrderID)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("charge order %s: %w", charge.OrderID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return translate(resp)
}

// Approving is a PaymentGateway that approves every charge without calling
// anything. It exists so the service runs locally with no provider; never
// wire it in production.
type Approving struct {
	Logger *slog.Logger
}

// Charge approves the charge.
func (a Approving) Charge(ctx context.Context, charge domain.Charge) error {
	a.Logger.WarnContext(ctx, "payment approved by the local stand-in, no money moved", "order_id", charge.OrderID)
	return nil
}

// maxErrorBody bounds how much of a failed response is read.
const maxErrorBody = 4 << 10

func readError(r io.Reader) errorResponse {
	var body errorResponse
	// The body is only used to describe the failure, so a response that
	// cannot be decoded is not itself an error.
	_ = json.UnmarshalRead(io.LimitReader(r, maxErrorBody), &body)
	return body
}

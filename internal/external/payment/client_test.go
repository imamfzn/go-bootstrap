package payment_test

import (
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/imamfzn/go-bootstrap/internal/domain"
	"github.com/imamfzn/go-bootstrap/internal/external/payment"
)

// roundTripFunc answers a request in memory, so no connection is opened.
type roundTripFunc func(r *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func respond(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func newClient(t *testing.T, transport roundTripFunc) *payment.Client {
	t.Helper()

	client, err := payment.NewClient(&http.Client{Transport: transport}, &payment.Config{
		BaseURL: "https://payments.test/api",
		APIKey:  "secret",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

func TestCharge_SendsRequest(t *testing.T) {
	type seen struct {
		method, url, auth, idempotencyKey, contentType string
		body                                           map[string]any
	}
	var got seen

	client := newClient(t, func(r *http.Request) (*http.Response, error) {
		got = seen{
			method:         r.Method,
			url:            r.URL.String(),
			auth:           r.Header.Get("Authorization"),
			idempotencyKey: r.Header.Get("Idempotency-Key"),
			contentType:    r.Header.Get("Content-Type"),
		}
		if err := json.UnmarshalRead(r.Body, &got.body); err != nil {
			t.Errorf("request body is not JSON: %v", err)
		}
		return respond(http.StatusCreated, `{}`), nil
	})

	if err := client.Charge(t.Context(), &domain.Charge{OrderID: "o1", Amount: 5000}); err != nil {
		t.Fatalf("Charge: %v", err)
	}

	if got.method != http.MethodPost || got.url != "https://payments.test/api/v1/charges" {
		t.Errorf("request = %s %s, want POST https://payments.test/api/v1/charges", got.method, got.url)
	}
	if got.auth != "Bearer secret" {
		t.Errorf("Authorization = %q", got.auth)
	}
	if got.idempotencyKey != "o1" {
		t.Errorf("Idempotency-Key = %q, want o1", got.idempotencyKey)
	}
	if got.contentType != "application/json" {
		t.Errorf("Content-Type = %q", got.contentType)
	}
	if got.body["reference"] != "o1" || got.body["amount"] != float64(5000) {
		t.Errorf("body = %v", got.body)
	}
}

func TestCharge_Responses(t *testing.T) {
	tests := []struct {
		name         string
		status       int
		body         string
		wantDeclined bool
		wantErr      bool
	}{
		{"accepted", http.StatusOK, `{}`, false, false},
		{"declined", http.StatusPaymentRequired, `{"code":"card_declined"}`, true, true},
		{"rejected", http.StatusUnprocessableEntity, `{"code":"limit"}`, true, true},
		{"provider failure", http.StatusBadGateway, `upstream timeout`, false, true},
		{"bad credentials", http.StatusUnauthorized, `{"code":"bad_key"}`, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newClient(t, func(*http.Request) (*http.Response, error) {
				return respond(tt.status, tt.body), nil
			})

			err := client.Charge(t.Context(), &domain.Charge{OrderID: "o1", Amount: 5000})

			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, want an error: %v", err, tt.wantErr)
			}
			if got := errors.Is(err, domain.ErrOrderPaymentDeclined); got != tt.wantDeclined {
				t.Errorf("is a declined-payment error = %v, want %v (err: %v)", got, tt.wantDeclined, err)
			}
		})
	}
}

func TestCharge_TransportFailureIsNotADomainError(t *testing.T) {
	boom := errors.New("connection refused")
	client := newClient(t, func(*http.Request) (*http.Response, error) { return nil, boom })

	err := client.Charge(t.Context(), &domain.Charge{OrderID: "o1", Amount: 5000})

	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want it to wrap %v", err, boom)
	}
	if _, ok := errors.AsType[*domain.Error](err); ok {
		t.Errorf("a transport failure became a domain error, so the worker would drop the message: %v", err)
	}
}

func TestNewClient_InvalidBaseURL(t *testing.T) {
	if _, err := payment.NewClient(http.DefaultClient, &payment.Config{BaseURL: "://nope"}); err == nil {
		t.Fatal("NewClient accepted an invalid base URL")
	}
}

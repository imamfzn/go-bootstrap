package payment_test

import (
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/imamfzn/go-bootstrap/internal/domain"
	"github.com/imamfzn/go-bootstrap/internal/external/payment"
)

func newClient(t *testing.T, h http.HandlerFunc) *payment.Client {
	t.Helper()

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	client, err := payment.NewClient(&http.Client{Timeout: 5 * time.Second}, payment.Config{BaseURL: srv.URL, APIKey: "secret"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

func TestCharge_SendsRequest(t *testing.T) {
	type seen struct {
		method, path, auth, idempotencyKey string
		body                               map[string]any
	}
	var got seen

	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = seen{
			method:         r.Method,
			path:           r.URL.Path,
			auth:           r.Header.Get("Authorization"),
			idempotencyKey: r.Header.Get("Idempotency-Key"),
		}
		if err := json.UnmarshalRead(r.Body, &got.body); err != nil {
			t.Errorf("request body is not JSON: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
	})

	if err := client.Charge(t.Context(), domain.Charge{OrderID: "o1", Amount: 5000}); err != nil {
		t.Fatalf("Charge: %v", err)
	}

	if got.method != http.MethodPost || got.path != "/v1/charges" {
		t.Errorf("request = %s %s, want POST /v1/charges", got.method, got.path)
	}
	if got.auth != "Bearer secret" {
		t.Errorf("Authorization = %q", got.auth)
	}
	if got.idempotencyKey != "o1" {
		t.Errorf("Idempotency-Key = %q, want o1", got.idempotencyKey)
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
			client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, tt.body)
			})

			err := client.Charge(t.Context(), domain.Charge{OrderID: "o1", Amount: 5000})

			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, want an error: %v", err, tt.wantErr)
			}
			if got := errors.Is(err, domain.ErrOrderPaymentDeclined); got != tt.wantDeclined {
				t.Errorf("is a declined-payment error = %v, want %v (err: %v)", got, tt.wantDeclined, err)
			}
		})
	}
}

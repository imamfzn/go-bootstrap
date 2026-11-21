package request_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/imamfzn/go-bootstrap/internal/domain"
	"github.com/imamfzn/go-bootstrap/internal/http/request"
)

func decode(t *testing.T, body string) (request.CreateOrder, error) {
	t.Helper()

	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/v1/orders", strings.NewReader(body))
	return request.Decode[request.CreateOrder](httptest.NewRecorder(), r)
}

func TestDecode_Valid(t *testing.T) {
	got, err := decode(t, `{"product_id":"p1","quantity":2,"amount":5000,"note":"hi"}`)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	want := request.CreateOrder{ProductID: "p1", Quantity: 2, Amount: 5000, Note: "hi"}
	if got != want {
		t.Fatalf("Decode = %+v, want %+v", got, want)
	}
}

func TestDecode_Malformed(t *testing.T) {
	tests := []struct {
		name, body string
	}{
		{"empty body", ``},
		{"truncated", `{"product_id":"p1"`},
		{"wrong type", `{"quantity":"two"}`},
		{"unknown field", `{"product_id":"p1","user_id":"someone-else"}`},
		{"trailing data", `{"product_id":"p1"} {"product_id":"p2"}`},
		{"duplicate field", `{"product_id":"p1","product_id":"p2"}`},
		{"too large", `{"note":"` + strings.Repeat("a", 2<<20) + `"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := decode(t, tt.body); !errors.Is(err, domain.ErrRequestMalformed) {
				t.Fatalf("Decode = %v, want ErrRequestMalformed", err)
			}
		})
	}
}

func TestCreateOrder_ToDomain_TakesUserFromArgument(t *testing.T) {
	got := request.CreateOrder{ProductID: "p1", Quantity: 2, Amount: 5000}.ToDomain("u1")

	want := domain.CreateOrder{UserID: "u1", ProductID: "p1", Quantity: 2, Amount: 5000}
	if got != want {
		t.Fatalf("ToDomain = %+v, want %+v", got, want)
	}
}

package router_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/imamfzn/go-bootstrap/internal/domain"
	"github.com/imamfzn/go-bootstrap/internal/http/middleware"
	"github.com/imamfzn/go-bootstrap/internal/http/response"
	"github.com/imamfzn/go-bootstrap/internal/http/router"
	"github.com/imamfzn/go-bootstrap/internal/platform/i18n"
	"github.com/imamfzn/go-bootstrap/internal/usecase"
	"github.com/imamfzn/go-bootstrap/locales"
)

var createdAt = time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC)

// fakeOrders is the use case as the HTTP layer sees it.
type fakeOrders struct {
	usecase.OrderUsecase

	created []domain.CreateOrder
	err     error
	panics  bool
}

func (f *fakeOrders) Create(_ context.Context, input domain.CreateOrder) (*domain.Order, error) {
	if f.panics {
		panic("boom")
	}
	if f.err != nil {
		return nil, f.err
	}
	f.created = append(f.created, input)
	return domain.NewOrder("o1", input, createdAt), nil
}

func (f *fakeOrders) Get(_ context.Context, userID, id string) (*domain.Order, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &domain.Order{ID: id, UserID: userID, ProductID: "p1", Quantity: 1, Amount: 100, Status: domain.OrderPending, CreatedAt: createdAt, UpdatedAt: createdAt}, nil
}

func (f *fakeOrders) Cancel(_ context.Context, userID, id string) (*domain.Order, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &domain.Order{ID: id, UserID: userID, Status: domain.OrderCancelled, CreatedAt: createdAt, UpdatedAt: createdAt}, nil
}

type options struct {
	orders *fakeOrders
	ready  error
}

func newAPI(t *testing.T, opts options) http.Handler {
	t.Helper()

	catalog, err := i18n.Load(locales.FS, "en")
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	if opts.orders == nil {
		opts.orders = &fakeOrders{}
	}

	return router.New(router.Deps{
		Orders:   opts.orders,
		Catalog:  catalog,
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Verifier: middleware.DevInsecureVerifier(),
		Ready:    func(context.Context) error { return opts.ready },
	})
}

type call struct {
	method, path, body string
	header             map[string]string
}

func do(t *testing.T, api http.Handler, c call) *httptest.ResponseRecorder {
	t.Helper()

	var body io.Reader
	if c.body != "" {
		body = strings.NewReader(c.body)
	}
	r := httptest.NewRequestWithContext(t.Context(), c.method, c.path, body)
	for k, v := range c.header {
		r.Header.Set(k, v)
	}

	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, r)
	return rec
}

func decodeBody[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()

	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v, json.RejectUnknownMembers(true)); err != nil {
		t.Fatalf("decode %d response: %v\n%s", rec.Code, err, rec.Body.String())
	}
	return v
}

func asUser(id string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + id}
}

func TestCreateOrder(t *testing.T) {
	orders := &fakeOrders{}
	api := newAPI(t, options{orders: orders})

	rec := do(t, api, call{
		method: http.MethodPost, path: "/v1/orders", header: asUser("u1"),
		body: `{"product_id":"p1","quantity":2,"amount":5000,"note":"hi"}`,
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201\n%s", rec.Code, rec.Body.String())
	}
	got := decodeBody[response.Order](t, rec)
	want := response.Order{
		ID: "o1", ProductID: "p1", Quantity: 2, Amount: 5000, Note: "hi",
		Status: "pending", CreatedAt: createdAt, UpdatedAt: createdAt,
	}
	if got != want {
		t.Errorf("body = %+v, want %+v", got, want)
	}

	wantInput := domain.CreateOrder{UserID: "u1", ProductID: "p1", Quantity: 2, Amount: 5000, Note: "hi"}
	if len(orders.created) != 1 || orders.created[0] != wantInput {
		t.Errorf("use case input = %+v, want %+v", orders.created, wantInput)
	}
	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("response has no X-Request-ID")
	}
}

func TestCreateOrder_MalformedBody(t *testing.T) {
	orders := &fakeOrders{}
	api := newAPI(t, options{orders: orders})

	rec := do(t, api, call{method: http.MethodPost, path: "/v1/orders", header: asUser("u1"), body: `{"quantity":`})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := decodeBody[response.ErrorBody](t, rec); got.Code != "REQUEST_MALFORMED" {
		t.Errorf("code = %q, want REQUEST_MALFORMED", got.Code)
	}
	if len(orders.created) != 0 {
		t.Error("the use case ran for a malformed request")
	}
}

func TestErrorResponses(t *testing.T) {
	tests := []struct {
		name       string
		call       call
		orders     *fakeOrders
		wantStatus int
		wantCode   string
	}{
		{
			name:       "no token",
			call:       call{method: http.MethodGet, path: "/v1/orders/o1"},
			wantStatus: http.StatusUnauthorized, wantCode: "UNAUTHORIZED",
		},
		{
			name:       "wrong scheme",
			call:       call{method: http.MethodGet, path: "/v1/orders/o1", header: map[string]string{"Authorization": "Basic abc"}},
			wantStatus: http.StatusUnauthorized, wantCode: "UNAUTHORIZED",
		},
		{
			name:       "business rule from the use case",
			call:       call{method: http.MethodPost, path: "/v1/orders/o1/cancel", header: asUser("u1")},
			orders:     &fakeOrders{err: domain.ErrOrderCancelNotAllowed},
			wantStatus: http.StatusConflict, wantCode: "ORDER_CANCEL_NOT-ALLOWED",
		},
		{
			name:       "validation error defaults to 422",
			call:       call{method: http.MethodPost, path: "/v1/orders", header: asUser("u1"), body: `{}`},
			orders:     &fakeOrders{err: domain.ErrOrderProductRequired},
			wantStatus: http.StatusUnprocessableEntity, wantCode: "ORDER_CREATE_PRODUCT-REQUIRED",
		},
		{
			name:       "unexpected error is a generic 500",
			call:       call{method: http.MethodGet, path: "/v1/orders/o1", header: asUser("u1")},
			orders:     &fakeOrders{err: errors.New("connection refused")},
			wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL",
		},
		{
			name:       "panic is a generic 500",
			call:       call{method: http.MethodPost, path: "/v1/orders", header: asUser("u1"), body: `{}`},
			orders:     &fakeOrders{panics: true},
			wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL",
		},
		{
			name:       "unknown route",
			call:       call{method: http.MethodGet, path: "/nope"},
			wantStatus: http.StatusNotFound, wantCode: "NOT_FOUND",
		},
		{
			name:       "wrong method",
			call:       call{method: http.MethodDelete, path: "/healthz"},
			wantStatus: http.StatusMethodNotAllowed, wantCode: "METHOD_NOT_ALLOWED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(t, newAPI(t, options{orders: tt.orders}), tt.call)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			got := decodeBody[response.ErrorBody](t, rec)
			if got.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", got.Code, tt.wantCode)
			}
			if got.Message == "" || strings.Contains(got.Message, "connection refused") {
				t.Errorf("message = %q", got.Message)
			}
		})
	}
}

func TestErrorMessageFollowsAcceptLanguage(t *testing.T) {
	api := newAPI(t, options{orders: &fakeOrders{err: domain.ErrOrderNotFound}})
	get := func(lang string) (response.ErrorBody, string) {
		header := asUser("u1")
		header["Accept-Language"] = lang
		rec := do(t, api, call{method: http.MethodGet, path: "/v1/orders/o1", header: header})
		return decodeBody[response.ErrorBody](t, rec), rec.Header().Get("Content-Language")
	}

	en, enLang := get("en")
	id, idLang := get("id-ID,id;q=0.9")
	fr, frLang := get("fr")

	if en.Code != id.Code {
		t.Errorf("code changed with the language: %q vs %q", en.Code, id.Code)
	}
	if id.Message == en.Message {
		t.Errorf("Indonesian message %q was not translated", id.Message)
	}
	if fr.Message != en.Message {
		t.Errorf("unsupported language got %q, want the English fallback %q", fr.Message, en.Message)
	}
	if enLang != "en" || idLang != "id" || frLang != "en" {
		t.Errorf("Content-Language = %q, %q, %q; want en, id, en", enLang, idLang, frLang)
	}
}

func TestGetOrder_UsesAuthenticatedUser(t *testing.T) {
	api := newAPI(t, options{})

	rec := do(t, api, call{method: http.MethodGet, path: "/v1/orders/o7", header: asUser("u1")})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200\n%s", rec.Code, rec.Body.String())
	}
	if got := decodeBody[response.Order](t, rec); got.ID != "o7" {
		t.Errorf("order id = %q, want o7", got.ID)
	}
}

func TestHealth(t *testing.T) {
	tests := []struct {
		name, path string
		ready      error
		wantStatus int
	}{
		{"live", "/healthz", nil, http.StatusOK},
		{"live ignores dependencies", "/healthz", errors.New("database down"), http.StatusOK},
		{"ready", "/readyz", nil, http.StatusOK},
		{"not ready", "/readyz", errors.New("database down"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(t, newAPI(t, options{ready: tt.ready}), call{method: http.MethodGet, path: tt.path})

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestRequestID_EchoesClientValue(t *testing.T) {
	api := newAPI(t, options{})

	rec := do(t, api, call{method: http.MethodGet, path: "/healthz", header: map[string]string{"X-Request-ID": "abc-123"}})

	if got := rec.Header().Get("X-Request-ID"); got != "abc-123" {
		t.Fatalf("X-Request-ID = %q, want abc-123", got)
	}
}

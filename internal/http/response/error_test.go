package response_test

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/imamfzn/go-bootstrap/internal/domain"
	"github.com/imamfzn/go-bootstrap/internal/http/response"
	"github.com/imamfzn/go-bootstrap/internal/platform/i18n"
	"github.com/imamfzn/go-bootstrap/locales"
)

func write(t *testing.T, locale string, err error) (status int, body response.ErrorBody, logs string) {
	t.Helper()

	catalog, loadErr := i18n.Load(locales.FS, "en")
	if loadErr != nil {
		t.Fatalf("load catalog: %v", loadErr)
	}
	var logBuf bytes.Buffer
	errs := response.NewErrorWriter(catalog, slog.New(slog.NewTextHandler(&logBuf, nil)))

	r := httptest.NewRequestWithContext(i18n.WithLocale(t.Context(), locale), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	errs.Write(rec, r, err)

	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Errorf("Content-Type = %q, want JSON", got)
	}
	if decodeErr := json.Unmarshal(rec.Body.Bytes(), &body, json.RejectUnknownMembers(true)); decodeErr != nil {
		t.Fatalf("body is not exactly {code, message}: %v\n%s", decodeErr, rec.Body.String())
	}
	return rec.Code, body, logBuf.String()
}

func TestErrorWriter_DomainErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"mapped", domain.ErrOrderNotFound, http.StatusNotFound, "ORDER_GET_NOT-FOUND"},
		{"wrapped", fmt.Errorf("get order o1: %w", domain.ErrOrderNotFound), http.StatusNotFound, "ORDER_GET_NOT-FOUND"},
		{"conflict", domain.ErrOrderCancelNotAllowed, http.StatusConflict, "ORDER_CANCEL_NOT-ALLOWED"},
		{"malformed request", domain.ErrRequestMalformed, http.StatusBadRequest, "REQUEST_MALFORMED"},
		{"unmapped defaults to 422", domain.ErrOrderQuantityInvalid, http.StatusUnprocessableEntity, "ORDER_CREATE_QUANTITY-INVALID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body, logs := write(t, "en", tt.err)

			if status != tt.wantStatus {
				t.Errorf("status = %d, want %d", status, tt.wantStatus)
			}
			if body.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", body.Code, tt.wantCode)
			}
			if body.Message == "" {
				t.Error("message is empty")
			}
			if logs != "" {
				t.Errorf("a domain error was logged as unhandled: %s", logs)
			}
		})
	}
}

func TestErrorWriter_TranslatesMessage(t *testing.T) {
	_, en, _ := write(t, "en", domain.ErrOrderNotFound)
	_, id, _ := write(t, "id", domain.ErrOrderNotFound)

	if en.Message != domain.ErrOrderNotFound.Message {
		t.Errorf("en message = %q, want the default %q", en.Message, domain.ErrOrderNotFound.Message)
	}
	if id.Message == en.Message {
		t.Errorf("id message %q was not translated", id.Message)
	}
	if id.Code != en.Code {
		t.Errorf("code changed with the locale: %q vs %q", id.Code, en.Code)
	}
}

func TestErrorWriter_UnknownErrorIsGeneric500(t *testing.T) {
	secret := errors.New(`pq: relation "orders" does not exist`)

	status, body, logs := write(t, "en", secret)

	if status != http.StatusInternalServerError || body.Code != "INTERNAL" {
		t.Errorf("response = %d %q, want 500 INTERNAL", status, body.Code)
	}
	if strings.Contains(body.Message, "orders") {
		t.Errorf("internal detail leaked to the client: %q", body.Message)
	}
	if !strings.Contains(logs, "does not exist") {
		t.Errorf("the real error was not logged: %q", logs)
	}
}

func TestStatusFor_EveryMappedErrorIsDeclared(t *testing.T) {
	// Every domain error has a status: its own, or the 422 default.
	for _, e := range domain.Errors() {
		if status := response.StatusFor(e); status < 400 || status > 599 {
			t.Errorf("%s maps to %d, which is not an error status", e.Code, status)
		}
	}
}

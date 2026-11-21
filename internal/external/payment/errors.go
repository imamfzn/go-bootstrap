package payment

import (
	"fmt"
	"net/http"

	"github.com/imamfzn/go-bootstrap/internal/domain"
)

// translate turns a failed provider response into an error the core
// understands. A refusal the customer can act on becomes a domain error;
// anything else stays a plain error, which callers treat as retryable.
func translate(resp *http.Response) error {
	body := readError(resp.Body)

	switch resp.StatusCode {
	case http.StatusPaymentRequired, http.StatusUnprocessableEntity:
		return fmt.Errorf("%w: provider code %q", domain.ErrOrderPaymentDeclined, body.Code)
	default:
		return fmt.Errorf("payment provider returned %d: code %q: %s", resp.StatusCode, body.Code, body.Message)
	}
}

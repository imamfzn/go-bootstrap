package usecase

import (
	"context"

	"github.com/imamfzn/go-bootstrap/internal/domain"
)

// This file declares every outside system the use cases depend on, named
// for what is needed rather than for the vendor.
// Implementations live in internal/external/<system>.

// PaymentGateway takes payments.
type PaymentGateway interface {
	// Charge takes payment once per charge.OrderID, however often it is
	// called. It returns domain.ErrOrderPaymentDeclined when the payment
	// is refused.
	Charge(ctx context.Context, charge *domain.Charge) error
}

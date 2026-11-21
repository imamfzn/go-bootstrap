// Package domain holds the business types, their behaviour and their errors.
// It imports nothing from this module and performs no I/O.
package domain

import (
	"time"
	"unicode/utf8"
)

// OrderStatus is the lifecycle state of an order.
type OrderStatus string

const (
	OrderPending   OrderStatus = "pending"
	OrderPaid      OrderStatus = "paid"
	OrderCancelled OrderStatus = "cancelled"
	OrderExpired   OrderStatus = "expired"
)

const (
	MaxOrderQuantity   = 100
	MaxOrderNoteLength = 500
)

// Order is a customer's order. Amount is in minor currency units.
type Order struct {
	ID        string
	UserID    string
	ProductID string
	Quantity  int
	Amount    int64
	Note      string
	Status    OrderStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CreateOrder is the input for creating an order.
type CreateOrder struct {
	UserID    string
	ProductID string
	Quantity  int
	Amount    int64
	Note      string
}

// Validate checks the rules that need no I/O. Each rule has its own error.
func (p CreateOrder) Validate() error {
	switch {
	case p.UserID == "":
		return ErrUnauthorized
	case p.ProductID == "":
		return ErrOrderProductRequired
	case p.Quantity < 1 || p.Quantity > MaxOrderQuantity:
		return ErrOrderQuantityInvalid
	case p.Amount < 1:
		return ErrOrderAmountInvalid
	case utf8.RuneCountInString(p.Note) > MaxOrderNoteLength:
		return ErrOrderNoteTooLong
	}
	return nil
}

// NewOrder builds a pending order from validated input.
func NewOrder(id string, p CreateOrder, now time.Time) *Order {
	return &Order{
		ID:        id,
		UserID:    p.UserID,
		ProductID: p.ProductID,
		Quantity:  p.Quantity,
		Amount:    p.Amount,
		Note:      p.Note,
		Status:    OrderPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// OwnedBy reports whether the order belongs to the user.
func (o *Order) OwnedBy(userID string) bool { return o.UserID == userID }

// Cancel moves a pending order to cancelled.
func (o *Order) Cancel(now time.Time) error {
	if o.Status != OrderPending {
		return ErrOrderCancelNotAllowed
	}
	o.Status = OrderCancelled
	o.UpdatedAt = now
	return nil
}

// MarkPaid moves a pending order to paid.
func (o *Order) MarkPaid(now time.Time) error {
	if o.Status != OrderPending {
		return ErrOrderPayNotAllowed
	}
	o.Status = OrderPaid
	o.UpdatedAt = now
	return nil
}

// Charge is a request to take payment for an order. OrderID doubles as the
// idempotency key, so a retried charge is never taken twice.
type Charge struct {
	OrderID string
	Amount  int64
}

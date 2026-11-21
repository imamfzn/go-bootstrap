// Package usecase holds the business logic. It owns every contract of the
// core: what it offers to inbound adapters (the *Usecase interfaces) and
// what it needs from outbound ones (repository.go, publisher.go,
// external.go, transactor.go). It imports only the domain and pure helpers.
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/imamfzn/go-bootstrap/internal/domain"
	"github.com/imamfzn/go-bootstrap/internal/platform/clock"
)

// OrderUsecase is everything that can be done with orders.
type OrderUsecase interface {
	// Create validates the input, stores a pending order and announces it.
	Create(ctx context.Context, input *domain.CreateOrder) (*domain.Order, error)
	// Get returns the user's own order.
	Get(ctx context.Context, userID, id string) (*domain.Order, error)
	// Cancel cancels the user's own pending order.
	Cancel(ctx context.Context, userID, id string) (*domain.Order, error)
	// Pay charges a pending order and marks it paid. Calling it again for
	// a paid order does nothing.
	Pay(ctx context.Context, id string) error
	// ExpirePending expires orders left pending for too long and returns
	// how many it expired.
	ExpirePending(ctx context.Context) (int, error)
}

// OrderDeps is what the order use case needs.
type OrderDeps struct {
	Orders   OrderRepository
	Events   OrderPublisher
	Payments PaymentGateway
	Tx       Transactor
	Clock    clock.Clock
	NewID    func() string
	// PendingTTL is how long an order may stay pending before it expires.
	PendingTTL time.Duration
}

type orderUsecase struct {
	orders     OrderRepository
	events     OrderPublisher
	payments   PaymentGateway
	tx         Transactor
	clock      clock.Clock
	newID      func() string
	pendingTTL time.Duration
}

// NewOrder returns the order use case.
func NewOrder(d *OrderDeps) OrderUsecase {
	return &orderUsecase{
		orders:     d.Orders,
		events:     d.Events,
		payments:   d.Payments,
		tx:         d.Tx,
		clock:      d.Clock,
		newID:      d.NewID,
		pendingTTL: d.PendingTTL,
	}
}

func (u *orderUsecase) Create(ctx context.Context, input *domain.CreateOrder) (*domain.Order, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	order := domain.NewOrder(u.newID(), input, u.clock.Now())
	if err := u.orders.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	// Published after the insert has committed, never inside a transaction.
	// If this fails the caller sees an error and the order stays pending
	// until ExpirePending removes it. Use an outbox table instead when an
	// event must never be lost.
	if err := u.events.OrderCreated(ctx, order); err != nil {
		return nil, fmt.Errorf("announce order %s: %w", order.ID, err)
	}

	return order, nil
}

func (u *orderUsecase) Get(ctx context.Context, userID, id string) (*domain.Order, error) {
	order, err := u.orders.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get order %s: %w", id, err)
	}
	// Another user's order is reported as missing, so IDs cannot be probed.
	if !order.OwnedBy(userID) {
		return nil, domain.ErrOrderNotFound
	}
	return order, nil
}

func (u *orderUsecase) Cancel(ctx context.Context, userID, id string) (*domain.Order, error) {
	var order *domain.Order

	err := u.tx.InTx(ctx, func(ctx context.Context) error {
		o, err := u.orders.GetForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if !o.OwnedBy(userID) {
			return domain.ErrOrderNotFound
		}
		if err := o.Cancel(u.clock.Now()); err != nil {
			return err
		}
		if err := u.orders.Update(ctx, o); err != nil {
			return err
		}
		order = o
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("cancel order %s: %w", id, err)
	}

	return order, nil
}

func (u *orderUsecase) Pay(ctx context.Context, id string) error {
	order, err := u.orders.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("get order %s: %w", id, err)
	}
	if order.Status == domain.OrderPaid {
		return nil
	}
	if order.Status != domain.OrderPending {
		return domain.ErrOrderPayNotAllowed
	}

	// The charge is an outside call, so it stays out of the transaction.
	// The gateway deduplicates by order ID, which makes a redelivered
	// message safe.
	if err := u.payments.Charge(ctx, &domain.Charge{OrderID: order.ID, Amount: order.Amount}); err != nil {
		return fmt.Errorf("charge order %s: %w", id, err)
	}

	err = u.tx.InTx(ctx, func(ctx context.Context) error {
		o, err := u.orders.GetForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if o.Status == domain.OrderPaid {
			return nil
		}
		// An order cancelled between the charge and this point fails here.
		// A real system would refund it; the skeleton only reports it.
		if err := o.MarkPaid(u.clock.Now()); err != nil {
			return err
		}
		return u.orders.Update(ctx, o)
	})
	if err != nil {
		return fmt.Errorf("mark order %s paid: %w", id, err)
	}

	return nil
}

func (u *orderUsecase) ExpirePending(ctx context.Context) (int, error) {
	now := u.clock.Now()

	expired, err := u.orders.ExpirePending(ctx, now.Add(-u.pendingTTL), now)
	if err != nil {
		return 0, fmt.Errorf("expire pending orders: %w", err)
	}
	return expired, nil
}

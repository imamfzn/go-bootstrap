package usecase

import (
	"context"
	"time"

	"github.com/imamfzn/go-bootstrap/internal/domain"
)

// This file declares every repository the use cases depend on.
// Implementations live in internal/repository.

// OrderRepository stores orders.
type OrderRepository interface {
	// Create inserts the order. It returns domain.ErrOrderAlreadyExists
	// when the ID is taken.
	Create(ctx context.Context, order *domain.Order) error
	// Get returns domain.ErrOrderNotFound when there is no such order.
	Get(ctx context.Context, id string) (*domain.Order, error)
	// GetForUpdate is Get with a row lock held until the transaction ends.
	GetForUpdate(ctx context.Context, id string) (*domain.Order, error)
	// Update saves the mutable fields of the order.
	Update(ctx context.Context, order *domain.Order) error
	// ExpirePending marks pending orders created before the cutoff as
	// expired and returns how many it changed.
	ExpirePending(ctx context.Context, createdBefore, now time.Time) (int, error)
}

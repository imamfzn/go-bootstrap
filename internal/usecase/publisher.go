package usecase

import (
	"context"

	"github.com/imamfzn/go-bootstrap/internal/domain"
)

// This file declares every event publisher the use cases depend on.
// Methods are named for the business fact, never for a topic.
// Implementations live in internal/publisher.

// OrderPublisher announces what happened to orders.
type OrderPublisher interface {
	OrderCreated(ctx context.Context, order *domain.Order) error
}

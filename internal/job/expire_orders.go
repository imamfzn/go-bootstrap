// Package job holds the scheduled tasks. A job calls a use case and exits.
// Jobs must be safe to run twice and to resume after being interrupted.
package job

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/imamfzn/go-bootstrap/internal/usecase"
)

// Func is one runnable job.
type Func func(ctx context.Context) error

// ExpireOrders expires orders that stayed pending for too long.
type ExpireOrders struct {
	orders usecase.OrderUsecase
	logger *slog.Logger
}

// NewExpireOrders returns the job.
func NewExpireOrders(orders usecase.OrderUsecase, logger *slog.Logger) *ExpireOrders {
	return &ExpireOrders{orders: orders, logger: logger}
}

// Run executes the job once.
func (j *ExpireOrders) Run(ctx context.Context) error {
	expired, err := j.orders.ExpirePending(ctx)
	if err != nil {
		return fmt.Errorf("expire orders: %w", err)
	}

	j.logger.InfoContext(ctx, "expired pending orders", "count", expired)
	return nil
}

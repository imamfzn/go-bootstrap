package app

import (
	"context"

	"github.com/imamfzn/go-bootstrap/internal/config"
	eventmw "github.com/imamfzn/go-bootstrap/internal/event/middleware"
	eventrouter "github.com/imamfzn/go-bootstrap/internal/event/router"
	"github.com/imamfzn/go-bootstrap/internal/platform/queue"
)

// RunWorker consumes subscriptions until ctx is cancelled, then waits for
// in-flight handlers.
func RunWorker(ctx context.Context, cfg *config.Worker) error {
	c, err := newCore(ctx, &cfg.Base)
	if err != nil {
		return err
	}
	defer c.close(ctx)

	// The first middleware is outermost. Logging sees the final outcome of
	// a message; Recover sits next to the handler so a panic passes through
	// the error policy like any other failure.
	consumer := c.pubsub.NewConsumer(cfg.Concurrency,
		queue.RequestID(),
		queue.Logging(c.logger),
		eventmw.ErrorPolicy(c.logger),
		queue.Timeout(cfg.HandlerTimeout),
		queue.Recover(c.logger),
	)

	eventrouter.RegisterOrder(consumer, &eventrouter.OrderSubscriptions{Created: cfg.SubscriptionOrderCreated}, c.orders)

	c.logger.InfoContext(ctx, "worker consuming", "concurrency", cfg.Concurrency)

	err = consumer.Run(ctx)
	logExit(ctx, c.logger, "worker", err)
	return err
}

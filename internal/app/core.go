package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/imamfzn/go-bootstrap/internal/config"
	"github.com/imamfzn/go-bootstrap/internal/external/payment"
	"github.com/imamfzn/go-bootstrap/internal/platform/clock"
	"github.com/imamfzn/go-bootstrap/internal/platform/httpclient"
	"github.com/imamfzn/go-bootstrap/internal/platform/ident"
	"github.com/imamfzn/go-bootstrap/internal/platform/queue/gcppubsub"
	"github.com/imamfzn/go-bootstrap/internal/publisher"
	"github.com/imamfzn/go-bootstrap/internal/repository"
	"github.com/imamfzn/go-bootstrap/internal/usecase"
)

// core is the business layer with its outbound adapters attached. The API,
// the worker and the jobs all drive the same use cases.
type core struct {
	*infra

	pubsub *gcppubsub.Client
	orders usecase.OrderUsecase
}

func newCore(ctx context.Context, cfg config.Base) (*core, error) {
	inf, err := newInfra(ctx, cfg)
	if err != nil {
		return nil, err
	}

	c, err := buildCore(ctx, cfg, inf)
	if err != nil {
		inf.close(ctx)
		return nil, err
	}
	return c, nil
}

func buildCore(ctx context.Context, cfg config.Base, inf *infra) (*core, error) {
	if cfg.PubSubProjectID == "" {
		return nil, errors.New("PUBSUB_PROJECT_ID is required")
	}
	pubsubClient, err := gcppubsub.New(ctx, cfg.PubSubProjectID)
	if err != nil {
		return nil, fmt.Errorf("connect to pubsub: %w", err)
	}

	payments, err := newPaymentGateway(ctx, cfg, inf)
	if err != nil {
		_ = pubsubClient.Close()
		return nil, err
	}

	// Outbound adapters.
	orderRepo := repository.NewOrder(inf.db)
	orderEvents := publisher.NewOrder(pubsubClient, publisher.OrderTopics{Created: cfg.TopicOrderCreated})

	// Use cases.
	orders := usecase.NewOrder(usecase.OrderDeps{
		Orders:     orderRepo,
		Events:     orderEvents,
		Payments:   payments,
		Tx:         inf.db,
		Clock:      clock.System(),
		NewID:      ident.New,
		PendingTTL: cfg.OrderPendingTTL,
	})

	return &core{infra: inf, pubsub: pubsubClient, orders: orders}, nil
}

func newPaymentGateway(ctx context.Context, cfg config.Base, inf *infra) (usecase.PaymentGateway, error) {
	if cfg.PaymentBaseURL == "" {
		inf.logger.WarnContext(ctx, "PAYMENT_BASE_URL is not set: every payment is approved without a provider")
		return payment.Approving{Logger: inf.logger}, nil
	}

	client, err := payment.NewClient(httpclient.New(cfg.PaymentTimeout), payment.Config{
		BaseURL: cfg.PaymentBaseURL,
		APIKey:  cfg.PaymentAPIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("create payment client: %w", err)
	}
	return client, nil
}

func (c *core) close(ctx context.Context) {
	// Flush pending publishes before the database goes away.
	if err := c.pubsub.Close(); err != nil {
		c.logger.ErrorContext(ctx, "close pubsub", "err", err)
	}
	c.infra.close(ctx)
}

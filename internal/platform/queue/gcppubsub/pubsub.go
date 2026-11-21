// Package gcppubsub connects the queue package to Google Cloud Pub/Sub.
// It is the only package that imports the Pub/Sub SDK.
package gcppubsub

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"cloud.google.com/go/pubsub/v2"
	"golang.org/x/sync/errgroup"

	"github.com/imamfzn/go-bootstrap/internal/platform/queue"
)

// Client publishes to topics and creates consumers. It honours
// PUBSUB_EMULATOR_HOST, so local development needs no code change.
type Client struct {
	client *pubsub.Client

	mu         sync.Mutex
	publishers map[string]*pubsub.Publisher
}

// New connects to Pub/Sub in the given project.
func New(ctx context.Context, projectID string) (*Client, error) {
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("create pubsub client: %w", err)
	}
	return &Client{client: client, publishers: make(map[string]*pubsub.Publisher)}, nil
}

// Send publishes msg to topic and waits for the broker to accept it.
// It implements queue.Sender.
func (c *Client) Send(ctx context.Context, topic string, msg *queue.Message) error {
	result := c.publisher(topic).Publish(ctx, &pubsub.Message{
		Data:       msg.Data,
		Attributes: msg.Attributes,
	})
	if _, err := result.Get(ctx); err != nil {
		return fmt.Errorf("publish to %s: %w", topic, err)
	}
	return nil
}

func (c *Client) publisher(topic string) *pubsub.Publisher {
	c.mu.Lock()
	defer c.mu.Unlock()

	p, ok := c.publishers[topic]
	if !ok {
		p = c.client.Publisher(topic)
		c.publishers[topic] = p
	}
	return p
}

// Close flushes pending publishes and releases the connection.
func (c *Client) Close() error {
	c.mu.Lock()
	for _, p := range c.publishers {
		p.Stop()
	}
	clear(c.publishers)
	c.mu.Unlock()

	if err := c.client.Close(); err != nil {
		return fmt.Errorf("close pubsub client: %w", err)
	}
	return nil
}

// Consumer receives from one or more subscriptions.
type Consumer struct {
	client      *pubsub.Client
	concurrency int
	middlewares []queue.Middleware
	handlers    map[string]queue.Handler
}

// NewConsumer returns a consumer that runs at most concurrency handlers per
// subscription at a time. The middlewares wrap every registered handler,
// the first one outermost.
func (c *Client) NewConsumer(concurrency int, mws ...queue.Middleware) *Consumer {
	return &Consumer{
		client:      c.client,
		concurrency: concurrency,
		middlewares: mws,
		handlers:    make(map[string]queue.Handler),
	}
}

// Register binds a handler to a subscription. Registering the same
// subscription twice replaces the earlier handler.
func (c *Consumer) Register(subscription string, h queue.Handler) {
	c.handlers[subscription] = queue.Chain(h, c.middlewares...)
}

// Run receives until ctx is cancelled, then waits for in-flight handlers.
// It returns the first receive error, which stops every subscription.
func (c *Consumer) Run(ctx context.Context) error {
	if len(c.handlers) == 0 {
		return errors.New("no subscriptions registered")
	}

	g, ctx := errgroup.WithContext(ctx)
	for subscription, handle := range c.handlers {
		sub := c.client.Subscriber(subscription)
		sub.ReceiveSettings.MaxOutstandingMessages = c.concurrency

		g.Go(func() error {
			err := sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
				msg := &queue.Message{
					ID:           m.ID,
					Subscription: subscription,
					Data:         m.Data,
					Attributes:   m.Attributes,
				}
				if m.DeliveryAttempt != nil {
					msg.Attempt = *m.DeliveryAttempt
				}

				if err := handle(ctx, msg); err != nil {
					m.Nack()
					return
				}
				m.Ack()
			})
			if err != nil {
				return fmt.Errorf("receive from %s: %w", subscription, err)
			}
			return nil
		})
	}
	return g.Wait()
}

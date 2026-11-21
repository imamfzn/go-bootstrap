// Package publisher implements the publishers declared in internal/usecase.
// Each maps a domain value to an event payload and sends it.
package publisher

import (
	"context"

	"github.com/imamfzn/go-bootstrap/internal/domain"
	"github.com/imamfzn/go-bootstrap/internal/event"
	"github.com/imamfzn/go-bootstrap/internal/platform/queue"
)

// OrderTopics names the topics order events go to. Values come from config.
type OrderTopics struct {
	Created string
}

// Order publishes order events.
type Order struct {
	sender queue.Sender
	topics OrderTopics
}

// NewOrder returns the order publisher.
func NewOrder(sender queue.Sender, topics OrderTopics) *Order {
	return &Order{sender: sender, topics: topics}
}

func (p *Order) OrderCreated(ctx context.Context, order *domain.Order) error {
	return queue.Publish(ctx, p.sender, p.topics.Created, event.TypeOrderCreated, event.OrderCreated{
		OrderID:   order.ID,
		UserID:    order.UserID,
		Amount:    order.Amount,
		CreatedAt: order.CreatedAt,
	})
}

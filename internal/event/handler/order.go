// Package handler holds the message handlers of the worker. A handler maps
// an event payload to a use case call and nothing more.
//
// Pub/Sub delivers at least once, so every handler must be safe to run
// twice for the same event.
package handler

import (
	"context"

	"github.com/imamfzn/go-bootstrap/internal/event"
	"github.com/imamfzn/go-bootstrap/internal/platform/queue"
	"github.com/imamfzn/go-bootstrap/internal/usecase"
)

// Order handles order events.
type Order struct {
	orders usecase.OrderUsecase
}

// NewOrder returns the order event handler.
func NewOrder(orders usecase.OrderUsecase) *Order {
	return &Order{orders: orders}
}

// Created takes payment for a newly created order.
func (h *Order) Created(ctx context.Context, env *queue.Envelope[event.OrderCreated]) error {
	return h.orders.Pay(ctx, env.Payload.OrderID)
}

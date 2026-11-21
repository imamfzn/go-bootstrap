// Package router binds subscriptions to message handlers, the way
// internal/http/router binds routes.
package router

import (
	"github.com/imamfzn/go-bootstrap/internal/event/handler"
	"github.com/imamfzn/go-bootstrap/internal/platform/queue"
	"github.com/imamfzn/go-bootstrap/internal/usecase"
)

// Registrar is the part of a consumer that routers need.
type Registrar interface {
	Register(subscription string, h queue.Handler)
}

// OrderSubscriptions names the subscriptions carrying order events.
// Values come from config.
type OrderSubscriptions struct {
	Created string
}

// RegisterOrder binds the order event handlers.
func RegisterOrder(r Registrar, subs *OrderSubscriptions, orders usecase.OrderUsecase) {
	h := handler.NewOrder(orders)

	r.Register(subs.Created, queue.JSON(h.Created))
}

package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/imamfzn/go-bootstrap/internal/http/handler"
)

func registerOrder(r chi.Router, a adapter, h *handler.Order) {
	r.Post("/orders", a.handle(h.Create))
	r.Get("/orders/{id}", a.handle(h.Get))
	r.Post("/orders/{id}/cancel", a.handle(h.Cancel))
}

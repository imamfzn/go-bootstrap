package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/imamfzn/go-bootstrap/internal/http/middleware"
	"github.com/imamfzn/go-bootstrap/internal/http/request"
	"github.com/imamfzn/go-bootstrap/internal/http/response"
	"github.com/imamfzn/go-bootstrap/internal/usecase"
)

// Order serves the order endpoints.
type Order struct {
	orders usecase.OrderUsecase
}

// NewOrder returns the order handler.
func NewOrder(orders usecase.OrderUsecase) *Order {
	return &Order{orders: orders}
}

// Create handles POST /v1/orders.
func (h *Order) Create(w http.ResponseWriter, r *http.Request) error {
	req, err := request.Decode[request.CreateOrder](w, r)
	if err != nil {
		return err
	}

	order, err := h.orders.Create(r.Context(), req.ToDomain(middleware.UserID(r.Context())))
	if err != nil {
		return err
	}

	response.JSON(w, http.StatusCreated, response.FromOrder(order))
	return nil
}

// Get handles GET /v1/orders/{id}.
func (h *Order) Get(w http.ResponseWriter, r *http.Request) error {
	order, err := h.orders.Get(r.Context(), middleware.UserID(r.Context()), chi.URLParam(r, "id"))
	if err != nil {
		return err
	}

	response.JSON(w, http.StatusOK, response.FromOrder(order))
	return nil
}

// Cancel handles POST /v1/orders/{id}/cancel.
func (h *Order) Cancel(w http.ResponseWriter, r *http.Request) error {
	order, err := h.orders.Cancel(r.Context(), middleware.UserID(r.Context()), chi.URLParam(r, "id"))
	if err != nil {
		return err
	}

	response.JSON(w, http.StatusOK, response.FromOrder(order))
	return nil
}

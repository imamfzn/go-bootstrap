package handler_test

import (
	"context"
	"errors"
	"testing"

	"github.com/imamfzn/go-bootstrap/internal/event"
	"github.com/imamfzn/go-bootstrap/internal/event/handler"
	"github.com/imamfzn/go-bootstrap/internal/platform/queue"
	"github.com/imamfzn/go-bootstrap/internal/usecase"
)

type fakeOrders struct {
	usecase.OrderUsecase

	paid []string
	err  error
}

func (f *fakeOrders) Pay(_ context.Context, id string) error {
	f.paid = append(f.paid, id)
	return f.err
}

func TestOrder_Created_PaysTheOrder(t *testing.T) {
	orders := &fakeOrders{}
	h := handler.NewOrder(orders)

	err := h.Created(t.Context(), &queue.Envelope[event.OrderCreated]{Payload: event.OrderCreated{OrderID: "o1"}})
	if err != nil {
		t.Fatalf("Created: %v", err)
	}

	if len(orders.paid) != 1 || orders.paid[0] != "o1" {
		t.Fatalf("paid orders = %v, want [o1]", orders.paid)
	}
}

func TestOrder_Created_ReturnsUsecaseError(t *testing.T) {
	boom := errors.New("database down")
	h := handler.NewOrder(&fakeOrders{err: boom})

	err := h.Created(t.Context(), &queue.Envelope[event.OrderCreated]{Payload: event.OrderCreated{OrderID: "o1"}})

	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want %v so the message is redelivered", err, boom)
	}
}

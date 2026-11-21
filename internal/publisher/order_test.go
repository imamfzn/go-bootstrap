package publisher_test

import (
	"context"
	"testing"
	"time"

	"github.com/imamfzn/go-bootstrap/internal/domain"
	"github.com/imamfzn/go-bootstrap/internal/event"
	"github.com/imamfzn/go-bootstrap/internal/platform/queue"
	"github.com/imamfzn/go-bootstrap/internal/publisher"
)

type fakeSender struct {
	topic string
	msg   *queue.Message
}

func (s *fakeSender) Send(_ context.Context, topic string, msg *queue.Message) error {
	s.topic, s.msg = topic, msg
	return nil
}

func TestOrder_OrderCreated(t *testing.T) {
	sender := &fakeSender{}
	pub := publisher.NewOrder(sender, &publisher.OrderTopics{Created: "order-created"})
	createdAt := time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC)
	order := &domain.Order{ID: "o1", UserID: "u1", Amount: 5000, CreatedAt: createdAt}

	if err := pub.OrderCreated(t.Context(), order); err != nil {
		t.Fatalf("OrderCreated: %v", err)
	}

	if sender.topic != "order-created" {
		t.Errorf("topic = %q, want order-created", sender.topic)
	}

	var got *queue.Envelope[event.OrderCreated]
	decode := queue.JSON(func(_ context.Context, env *queue.Envelope[event.OrderCreated]) error {
		got = env
		return nil
	})
	if err := decode(t.Context(), sender.msg); err != nil {
		t.Fatalf("decode published message: %v", err)
	}

	want := event.OrderCreated{OrderID: "o1", UserID: "u1", Amount: 5000, CreatedAt: createdAt}
	if got.Payload != want {
		t.Errorf("payload = %+v, want %+v", got.Payload, want)
	}
	if got.Meta.Type != event.TypeOrderCreated {
		t.Errorf("event type = %q, want %q", got.Meta.Type, event.TypeOrderCreated)
	}
}

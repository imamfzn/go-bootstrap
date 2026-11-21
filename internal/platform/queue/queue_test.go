package queue_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"io"
	"log/slog"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/imamfzn/go-bootstrap/internal/platform/queue"
	"github.com/imamfzn/go-bootstrap/internal/platform/requestid"
)

type orderPaid struct {
	OrderID string `json:"order_id"`
}

type fakeSender struct {
	topic string
	msg   *queue.Message
	err   error
}

func (s *fakeSender) Send(_ context.Context, topic string, msg *queue.Message) error {
	s.topic, s.msg = topic, msg
	return s.err
}

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestChain_FirstMiddlewareIsOutermost(t *testing.T) {
	var calls []string
	mw := func(name string) queue.Middleware {
		return func(next queue.Handler) queue.Handler {
			return func(ctx context.Context, msg *queue.Message) error {
				calls = append(calls, name)
				return next(ctx, msg)
			}
		}
	}
	h := queue.Chain(func(context.Context, *queue.Message) error {
		calls = append(calls, "handler")
		return nil
	}, mw("first"), mw("second"))

	if err := h(t.Context(), &queue.Message{}); err != nil {
		t.Fatalf("handler: %v", err)
	}

	if want := []string{"first", "second", "handler"}; !slices.Equal(calls, want) {
		t.Fatalf("call order = %v, want %v", calls, want)
	}
}

func TestPublish_RoundTripsThroughJSON(t *testing.T) {
	sender := &fakeSender{}
	ctx := requestid.With(t.Context(), "req-9")

	if err := queue.Publish(ctx, sender, "orders", "order.paid", &orderPaid{OrderID: "o1"}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	if sender.topic != "orders" {
		t.Errorf("topic = %q, want orders", sender.topic)
	}
	if got := sender.msg.Attributes[queue.AttrEventType]; got != "order.paid" {
		t.Errorf("event_type attribute = %q", got)
	}
	if got := sender.msg.Attributes[queue.AttrRequestID]; got != "req-9" {
		t.Errorf("request_id attribute = %q", got)
	}

	var got *queue.Envelope[orderPaid]
	h := queue.JSON(func(_ context.Context, env *queue.Envelope[orderPaid]) error {
		got = env
		return nil
	})
	if err := h(t.Context(), sender.msg); err != nil {
		t.Fatalf("decode published message: %v", err)
	}
	if got.Payload.OrderID != "o1" || got.Meta.Type != "order.paid" || got.Meta.RequestID != "req-9" {
		t.Errorf("decoded envelope = %+v", got)
	}
	if got.Meta.ID == "" || got.Meta.OccurredAt.IsZero() {
		t.Errorf("envelope metadata is incomplete: %+v", got.Meta)
	}
}

func TestPublish_OmitsEmptyRequestID(t *testing.T) {
	sender := &fakeSender{}

	if err := queue.Publish(t.Context(), sender, "orders", "order.paid", &orderPaid{OrderID: "o1"}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	if _, ok := sender.msg.Attributes[queue.AttrRequestID]; ok {
		t.Error("request_id attribute set without a request identifier")
	}
	var raw struct {
		Meta map[string]any `json:"meta"`
	}
	if err := json.Unmarshal(sender.msg.Data, &raw); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if _, ok := raw.Meta["request_id"]; ok {
		t.Error("request_id present in the body without a request identifier")
	}
}

func TestPublish_SendFailure(t *testing.T) {
	boom := errors.New("broker down")

	err := queue.Publish(t.Context(), &fakeSender{err: boom}, "orders", "order.paid", &orderPaid{})

	if !errors.Is(err, boom) {
		t.Fatalf("Publish = %v, want it to wrap %v", err, boom)
	}
}

func TestJSON_MalformedBody(t *testing.T) {
	called := false
	h := queue.JSON(func(context.Context, *queue.Envelope[orderPaid]) error {
		called = true
		return nil
	})

	err := h(t.Context(), &queue.Message{Data: []byte(`{"payload": `)})

	if !errors.Is(err, queue.ErrMalformed) {
		t.Fatalf("err = %v, want ErrMalformed", err)
	}
	if called {
		t.Error("handler ran for a malformed message")
	}
}

func TestRequestID_FromAttributes(t *testing.T) {
	var got string
	h := queue.RequestID()(func(ctx context.Context, _ *queue.Message) error {
		got = requestid.From(ctx)
		return nil
	})

	msg := &queue.Message{Attributes: map[string]string{queue.AttrRequestID: "req-3"}}
	if err := h(t.Context(), msg); err != nil {
		t.Fatalf("handler: %v", err)
	}

	if got != "req-3" {
		t.Fatalf("request id in context = %q, want req-3", got)
	}
}

func TestRecover_PanicBecomesError(t *testing.T) {
	h := queue.Recover(discardLogger())(func(context.Context, *queue.Message) error {
		panic("boom")
	})

	if err := h(t.Context(), &queue.Message{ID: "m1"}); err == nil {
		t.Fatal("a panicking handler returned nil, so the message would be acknowledged")
	}
}

func TestTimeout_CancelsSlowHandler(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := queue.Timeout(time.Second)(func(ctx context.Context, _ *queue.Message) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Minute):
				return nil
			}
		})

		err := h(t.Context(), &queue.Message{})

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("err = %v, want context.DeadlineExceeded", err)
		}
	})
}

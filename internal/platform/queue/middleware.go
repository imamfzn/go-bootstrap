package queue

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/imamfzn/go-bootstrap/internal/platform/requestid"
)

// RequestID copies the request identifier from the message attributes into
// the context, so logs of the consumer join those of the publisher.
func RequestID() Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, msg *Message) error {
			if id := msg.Attributes[AttrRequestID]; id != "" {
				ctx = requestid.With(ctx, id)
			}
			return next(ctx, msg)
		}
	}
}

// Recover turns a panic in a handler into an error, so one bad message
// cannot stop the consumer. The message is redelivered.
func Recover(logger *slog.Logger) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, msg *Message) (err error) {
			defer func() {
				if r := recover(); r != nil {
					logger.ErrorContext(ctx, "handler panicked",
						"panic", fmt.Sprint(r),
						"stack", string(debug.Stack()),
						"message_id", msg.ID,
					)
					err = fmt.Errorf("handler panicked: %v", r)
				}
			}()
			return next(ctx, msg)
		}
	}
}

// Logging records one line per handled message. It never logs the body.
func Logging(logger *slog.Logger) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, msg *Message) error {
			start := time.Now()
			err := next(ctx, msg)

			attrs := []any{
				"subscription", msg.Subscription,
				"message_id", msg.ID,
				"event_type", msg.Attributes[AttrEventType],
				"attempt", msg.Attempt,
				"duration_ms", time.Since(start).Milliseconds(),
			}
			if err != nil {
				logger.ErrorContext(ctx, "message failed", append(attrs, "err", err)...)
				return err
			}
			logger.InfoContext(ctx, "message handled", attrs...)
			return nil
		}
	}
}

// Timeout bounds how long a handler may run.
func Timeout(d time.Duration) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, msg *Message) error {
			ctx, cancel := context.WithTimeout(ctx, d)
			defer cancel()
			return next(ctx, msg)
		}
	}
}

// Package middleware holds the consumer middleware that knows about this
// service's errors. Broker-independent middleware lives in platform/queue.
package middleware

import (
	"context"
	"errors"
	"log/slog"

	"github.com/imamfzn/go-bootstrap/internal/domain"
	"github.com/imamfzn/go-bootstrap/internal/platform/queue"
)

// ErrorPolicy decides whether a failed message is worth retrying.
//
//   - A malformed body or a domain error will fail the same way every time,
//     so the message is acknowledged and the failure logged.
//   - Any other error is treated as temporary: it is returned, the message
//     is redelivered, and the subscription's dead-letter policy bounds the
//     retries.
func ErrorPolicy(logger *slog.Logger) queue.Middleware {
	return func(next queue.Handler) queue.Handler {
		return func(ctx context.Context, msg *queue.Message) error {
			err := next(ctx, msg)
			if err == nil {
				return nil
			}

			if errors.Is(err, queue.ErrMalformed) {
				logger.WarnContext(ctx, "dropping malformed message", "message_id", msg.ID, "err", err)
				return nil
			}
			if domainErr, ok := errors.AsType[*domain.Error](err); ok {
				logger.WarnContext(ctx, "dropping message rejected by a business rule",
					"message_id", msg.ID, "code", domainErr.Code, "err", err)
				return nil
			}

			return err
		}
	}
}

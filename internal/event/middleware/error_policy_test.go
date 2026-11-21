package middleware_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/imamfzn/go-bootstrap/internal/domain"
	"github.com/imamfzn/go-bootstrap/internal/event/middleware"
	"github.com/imamfzn/go-bootstrap/internal/platform/queue"
)

func TestErrorPolicy(t *testing.T) {
	temporary := errors.New("connection reset")

	tests := []struct {
		name string
		err  error
		want error
	}{
		{"success is acknowledged", nil, nil},
		{"malformed message is acknowledged", fmt.Errorf("%w: unexpected end", queue.ErrMalformed), nil},
		{"domain error is acknowledged", domain.ErrOrderPayNotAllowed, nil},
		{"wrapped domain error is acknowledged", fmt.Errorf("pay order o1: %w", domain.ErrOrderNotFound), nil},
		{"other error is redelivered", temporary, temporary},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := middleware.ErrorPolicy(logger)(func(context.Context, *queue.Message) error {
				return tt.err
			})

			if got := h(t.Context(), &queue.Message{ID: "m1"}); !errors.Is(got, tt.want) {
				t.Fatalf("handler returned %v, want %v", got, tt.want)
			}
		})
	}
}

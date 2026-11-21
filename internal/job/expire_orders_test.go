package job_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/imamfzn/go-bootstrap/internal/job"
	"github.com/imamfzn/go-bootstrap/internal/usecase"
)

type fakeOrders struct {
	usecase.OrderUsecase

	calls int
	err   error
}

func (f *fakeOrders) ExpirePending(context.Context) (int, error) {
	f.calls++
	return 2, f.err
}

func TestExpireOrders_Run(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	boom := errors.New("database down")

	tests := []struct {
		name string
		err  error
	}{
		{"success", nil},
		{"failure is returned so the job exits non-zero", boom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orders := &fakeOrders{err: tt.err}

			err := job.NewExpireOrders(orders, logger).Run(t.Context())

			if !errors.Is(err, tt.err) {
				t.Fatalf("Run = %v, want %v", err, tt.err)
			}
			if orders.calls != 1 {
				t.Errorf("use case called %d times, want 1", orders.calls)
			}
		})
	}
}

package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/imamfzn/go-bootstrap/internal/domain"
)

func validCreateOrder() *domain.CreateOrder {
	return &domain.CreateOrder{UserID: "u1", ProductID: "p1", Quantity: 2, Amount: 5000}
}

func TestCreateOrder_Validate(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*domain.CreateOrder)
		want   error
	}{
		{"valid", func(*domain.CreateOrder) {}, nil},
		{"missing user", func(p *domain.CreateOrder) { p.UserID = "" }, domain.ErrUnauthorized},
		{"missing product", func(p *domain.CreateOrder) { p.ProductID = "" }, domain.ErrOrderProductRequired},
		{"zero quantity", func(p *domain.CreateOrder) { p.Quantity = 0 }, domain.ErrOrderQuantityInvalid},
		{"quantity at limit", func(p *domain.CreateOrder) { p.Quantity = domain.MaxOrderQuantity }, nil},
		{"quantity over limit", func(p *domain.CreateOrder) { p.Quantity = domain.MaxOrderQuantity + 1 }, domain.ErrOrderQuantityInvalid},
		{"zero amount", func(p *domain.CreateOrder) { p.Amount = 0 }, domain.ErrOrderAmountInvalid},
		{"note at limit", func(p *domain.CreateOrder) { p.Note = strings.Repeat("é", domain.MaxOrderNoteLength) }, nil},
		{"note over limit", func(p *domain.CreateOrder) { p.Note = strings.Repeat("é", domain.MaxOrderNoteLength+1) }, domain.ErrOrderNoteTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := validCreateOrder()
			tt.mutate(p)

			if err := p.Validate(); !errors.Is(err, tt.want) {
				t.Fatalf("Validate() = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestOrder_Cancel(t *testing.T) {
	created := time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC)
	later := created.Add(time.Hour)

	tests := []struct {
		status     domain.OrderStatus
		wantErr    error
		wantStatus domain.OrderStatus
	}{
		{domain.OrderPending, nil, domain.OrderCancelled},
		{domain.OrderPaid, domain.ErrOrderCancelNotAllowed, domain.OrderPaid},
		{domain.OrderCancelled, domain.ErrOrderCancelNotAllowed, domain.OrderCancelled},
		{domain.OrderExpired, domain.ErrOrderCancelNotAllowed, domain.OrderExpired},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			o := domain.NewOrder("o1", validCreateOrder(), created)
			o.Status = tt.status

			err := o.Cancel(later)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Cancel() = %v, want %v", err, tt.wantErr)
			}
			if o.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", o.Status, tt.wantStatus)
			}
			if wantUpdated := tt.wantErr == nil; o.UpdatedAt.Equal(later) != wantUpdated {
				t.Errorf("UpdatedAt = %v, updated should be %v", o.UpdatedAt, wantUpdated)
			}
		})
	}
}

func TestOrder_MarkPaid(t *testing.T) {
	now := time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC)

	pending := domain.NewOrder("o1", validCreateOrder(), now)
	if err := pending.MarkPaid(now); err != nil {
		t.Fatalf("MarkPaid on pending: %v", err)
	}
	if pending.Status != domain.OrderPaid {
		t.Errorf("Status = %q, want paid", pending.Status)
	}

	cancelled := domain.NewOrder("o2", validCreateOrder(), now)
	cancelled.Status = domain.OrderCancelled
	if err := cancelled.MarkPaid(now); !errors.Is(err, domain.ErrOrderPayNotAllowed) {
		t.Fatalf("MarkPaid on cancelled = %v, want ErrOrderPayNotAllowed", err)
	}
}

func TestErrors_UniqueCodes(t *testing.T) {
	seen := map[string]bool{}
	for _, e := range domain.Errors() {
		if e.Code == "" || e.Message == "" {
			t.Errorf("error %+v has an empty code or message", e)
		}
		if seen[e.Code] {
			t.Errorf("duplicate error code %q", e.Code)
		}
		seen[e.Code] = true
	}
	if len(seen) == 0 {
		t.Fatal("no domain errors registered")
	}
}

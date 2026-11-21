package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/imamfzn/go-bootstrap/internal/domain"
	"github.com/imamfzn/go-bootstrap/internal/platform/clock"
	"github.com/imamfzn/go-bootstrap/internal/usecase"
)

var now = time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC)

type fixture struct {
	orders   *fakeOrders
	events   *fakeEvents
	payments *fakePayments
	uc       usecase.OrderUsecase
}

func newFixture(orders ...domain.Order) *fixture {
	f := &fixture{
		orders:   newFakeOrders(orders...),
		events:   &fakeEvents{},
		payments: &fakePayments{},
	}
	f.uc = usecase.NewOrder(usecase.OrderDeps{
		Orders:     f.orders,
		Events:     f.events,
		Payments:   f.payments,
		Tx:         fakeTx{},
		Clock:      clock.Fixed(now),
		NewID:      func() string { return "new-order" },
		PendingTTL: 24 * time.Hour,
	})
	return f
}

func order(id, userID string, status domain.OrderStatus) domain.Order {
	return domain.Order{
		ID: id, UserID: userID, ProductID: "p1", Quantity: 1, Amount: 5000,
		Status: status, CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour),
	}
}

func TestCreate_StoresPendingOrderAndAnnouncesIt(t *testing.T) {
	f := newFixture()

	got, err := f.uc.Create(t.Context(), domain.CreateOrder{UserID: "u1", ProductID: "p1", Quantity: 2, Amount: 9000})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if got.ID != "new-order" || got.Status != domain.OrderPending || !got.CreatedAt.Equal(now) {
		t.Errorf("returned order = %+v", got)
	}
	stored, ok := f.orders.rows["new-order"]
	if !ok {
		t.Fatal("order was not stored")
	}
	if stored.UserID != "u1" || stored.Quantity != 2 || stored.Amount != 9000 {
		t.Errorf("stored order = %+v", stored)
	}
	if len(f.events.created) != 1 || f.events.created[0] != "new-order" {
		t.Errorf("announced orders = %v, want [new-order]", f.events.created)
	}
}

func TestCreate_InvalidInputStoresNothing(t *testing.T) {
	f := newFixture()

	_, err := f.uc.Create(t.Context(), domain.CreateOrder{UserID: "u1", ProductID: "p1", Quantity: 0, Amount: 9000})

	if !errors.Is(err, domain.ErrOrderQuantityInvalid) {
		t.Fatalf("err = %v, want ErrOrderQuantityInvalid", err)
	}
	if len(f.orders.rows) != 0 {
		t.Errorf("an invalid order was stored: %v", f.orders.rows)
	}
	if len(f.events.created) != 0 {
		t.Errorf("an invalid order was announced: %v", f.events.created)
	}
}

func TestCreate_AnnounceFailureIsReported(t *testing.T) {
	f := newFixture()
	f.events.err = errors.New("broker down")

	_, err := f.uc.Create(t.Context(), domain.CreateOrder{UserID: "u1", ProductID: "p1", Quantity: 1, Amount: 100})

	if !errors.Is(err, f.events.err) {
		t.Fatalf("err = %v, want it to wrap the publish error", err)
	}
}

func TestGet(t *testing.T) {
	tests := []struct {
		name, userID, id string
		want             error
	}{
		{"own order", "u1", "o1", nil},
		{"another user's order looks missing", "u2", "o1", domain.ErrOrderNotFound},
		{"unknown order", "u1", "nope", domain.ErrOrderNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(order("o1", "u1", domain.OrderPending))

			got, err := f.uc.Get(t.Context(), tt.userID, tt.id)

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
			if tt.want == nil && got.ID != tt.id {
				t.Errorf("got order %q, want %q", got.ID, tt.id)
			}
		})
	}
}

func TestCancel(t *testing.T) {
	tests := []struct {
		name       string
		status     domain.OrderStatus
		userID     string
		want       error
		wantStatus domain.OrderStatus
	}{
		{"pending order", domain.OrderPending, "u1", nil, domain.OrderCancelled},
		{"paid order", domain.OrderPaid, "u1", domain.ErrOrderCancelNotAllowed, domain.OrderPaid},
		{"already cancelled", domain.OrderCancelled, "u1", domain.ErrOrderCancelNotAllowed, domain.OrderCancelled},
		{"another user's order", domain.OrderPending, "u2", domain.ErrOrderNotFound, domain.OrderPending},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(order("o1", "u1", tt.status))

			got, err := f.uc.Cancel(t.Context(), tt.userID, "o1")

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
			if stored := f.orders.rows["o1"]; stored.Status != tt.wantStatus {
				t.Errorf("stored status = %q, want %q", stored.Status, tt.wantStatus)
			}
			if tt.want == nil {
				if got.Status != domain.OrderCancelled || !got.UpdatedAt.Equal(now) {
					t.Errorf("returned order = %+v", got)
				}
			} else if f.orders.updates != 0 {
				t.Errorf("a rejected cancel wrote to the repository %d times", f.orders.updates)
			}
		})
	}
}

func TestPay_ChargesAndMarksPaid(t *testing.T) {
	f := newFixture(order("o1", "u1", domain.OrderPending))

	if err := f.uc.Pay(t.Context(), "o1"); err != nil {
		t.Fatalf("Pay: %v", err)
	}

	if want := (domain.Charge{OrderID: "o1", Amount: 5000}); len(f.payments.charges) != 1 || f.payments.charges[0] != want {
		t.Errorf("charges = %+v, want one of %+v", f.payments.charges, want)
	}
	if got := f.orders.rows["o1"].Status; got != domain.OrderPaid {
		t.Errorf("stored status = %q, want paid", got)
	}
}

func TestPay_AlreadyPaidDoesNotChargeAgain(t *testing.T) {
	f := newFixture(order("o1", "u1", domain.OrderPaid))

	if err := f.uc.Pay(t.Context(), "o1"); err != nil {
		t.Fatalf("Pay on a paid order: %v", err)
	}

	if len(f.payments.charges) != 0 {
		t.Errorf("a paid order was charged again: %+v", f.payments.charges)
	}
}

func TestPay_CancelledOrderIsRejectedWithoutCharge(t *testing.T) {
	f := newFixture(order("o1", "u1", domain.OrderCancelled))

	err := f.uc.Pay(t.Context(), "o1")

	if !errors.Is(err, domain.ErrOrderPayNotAllowed) {
		t.Fatalf("err = %v, want ErrOrderPayNotAllowed", err)
	}
	if len(f.payments.charges) != 0 {
		t.Errorf("a cancelled order was charged: %+v", f.payments.charges)
	}
}

func TestPay_DeclinedPaymentLeavesOrderPending(t *testing.T) {
	f := newFixture(order("o1", "u1", domain.OrderPending))
	f.payments.err = domain.ErrOrderPaymentDeclined

	err := f.uc.Pay(t.Context(), "o1")

	if !errors.Is(err, domain.ErrOrderPaymentDeclined) {
		t.Fatalf("err = %v, want ErrOrderPaymentDeclined", err)
	}
	if got := f.orders.rows["o1"].Status; got != domain.OrderPending {
		t.Errorf("stored status = %q, want pending", got)
	}
}

func TestPay_UnknownOrder(t *testing.T) {
	f := newFixture()

	if err := f.uc.Pay(t.Context(), "nope"); !errors.Is(err, domain.ErrOrderNotFound) {
		t.Fatalf("err = %v, want ErrOrderNotFound", err)
	}
}

func TestExpirePending_UsesTTLAsCutoff(t *testing.T) {
	f := newFixture()
	f.orders.expireResult = 3

	got, err := f.uc.ExpirePending(t.Context())
	if err != nil {
		t.Fatalf("ExpirePending: %v", err)
	}

	if got != 3 {
		t.Errorf("expired = %d, want 3", got)
	}
	if want := now.Add(-24 * time.Hour); !f.orders.expireCutoff.Equal(want) {
		t.Errorf("cutoff = %v, want %v", f.orders.expireCutoff, want)
	}
}

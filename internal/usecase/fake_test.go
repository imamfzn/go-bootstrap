package usecase_test

import (
	"context"
	"time"

	"github.com/imamfzn/go-bootstrap/internal/domain"
	"github.com/imamfzn/go-bootstrap/internal/usecase"
)

// The fakes embed the interface they stand in for, so adding a method to an
// interface does not break them. A method a test did not expect panics.

// fakeOrders is an in-memory OrderRepository.
type fakeOrders struct {
	usecase.OrderRepository

	rows    map[string]domain.Order
	updates int

	expireCutoff time.Time
	expireResult int
}

func newFakeOrders(orders ...domain.Order) *fakeOrders {
	f := &fakeOrders{rows: make(map[string]domain.Order)}
	for _, o := range orders {
		f.rows[o.ID] = o
	}
	return f
}

func (f *fakeOrders) Create(_ context.Context, order *domain.Order) error {
	if _, ok := f.rows[order.ID]; ok {
		return domain.ErrOrderAlreadyExists
	}
	f.rows[order.ID] = *order
	return nil
}

func (f *fakeOrders) Get(_ context.Context, id string) (*domain.Order, error) {
	row, ok := f.rows[id]
	if !ok {
		return nil, domain.ErrOrderNotFound
	}
	return &row, nil
}

func (f *fakeOrders) GetForUpdate(ctx context.Context, id string) (*domain.Order, error) {
	return f.Get(ctx, id)
}

func (f *fakeOrders) Update(_ context.Context, order *domain.Order) error {
	if _, ok := f.rows[order.ID]; !ok {
		return domain.ErrOrderNotFound
	}
	f.rows[order.ID] = *order
	f.updates++
	return nil
}

func (f *fakeOrders) ExpirePending(_ context.Context, createdBefore, _ time.Time) (int, error) {
	f.expireCutoff = createdBefore
	return f.expireResult, nil
}

// fakeEvents records published events.
type fakeEvents struct {
	usecase.OrderPublisher

	created []string
	err     error
}

func (f *fakeEvents) OrderCreated(_ context.Context, order *domain.Order) error {
	if f.err != nil {
		return f.err
	}
	f.created = append(f.created, order.ID)
	return nil
}

// fakePayments records charges.
type fakePayments struct {
	usecase.PaymentGateway

	charges []domain.Charge
	err     error
}

func (f *fakePayments) Charge(_ context.Context, charge domain.Charge) error {
	f.charges = append(f.charges, charge)
	return f.err
}

// fakeTx runs the function directly, with no real transaction.
type fakeTx struct{}

func (fakeTx) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

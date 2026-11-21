package repository_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/imamfzn/go-bootstrap/internal/domain"
	"github.com/imamfzn/go-bootstrap/internal/platform/ident"
	"github.com/imamfzn/go-bootstrap/internal/platform/postgres"
	"github.com/imamfzn/go-bootstrap/internal/repository"
)

// openDB connects to the database named by TEST_DATABASE_URL, which must
// already be migrated (make migrate-up). Without it the test is skipped,
// so `go test ./...` works on a machine with no database.
func openDB(t *testing.T) *postgres.DB {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	db, err := postgres.Open(t.Context(), postgres.Config{URL: url, MaxConns: 4})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(db.Close)
	return db
}

// newOrder returns an order with a fresh ID, so tests do not collide and
// need no cleanup between runs. Postgres keeps microseconds, so times are
// truncated to make them compare equal after a round trip.
func newOrder(status domain.OrderStatus, createdAt time.Time) *domain.Order {
	createdAt = createdAt.UTC().Truncate(time.Microsecond)
	return &domain.Order{
		ID:        ident.New(),
		UserID:    "user-" + ident.New(),
		ProductID: "product-1",
		Quantity:  2,
		Amount:    5000,
		Note:      "leave at the door",
		Status:    status,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
}

func TestOrder_CreateAndGet(t *testing.T) {
	db := openDB(t)
	repo := repository.NewOrder(db)
	want := newOrder(domain.OrderPending, time.Now())

	if err := repo.Create(t.Context(), want); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.Get(t.Context(), want.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) || !got.UpdatedAt.Equal(want.UpdatedAt) {
		t.Errorf("times = %v / %v, want %v / %v", got.CreatedAt, got.UpdatedAt, want.CreatedAt, want.UpdatedAt)
	}
	// Times were compared as instants above; align them so the remaining
	// fields can be compared as a whole.
	got.CreatedAt, got.UpdatedAt = want.CreatedAt, want.UpdatedAt
	if *got != *want {
		t.Errorf("Get returned\n %+v\nwant\n %+v", *got, *want)
	}
}

func TestOrder_Create_DuplicateID(t *testing.T) {
	db := openDB(t)
	repo := repository.NewOrder(db)
	order := newOrder(domain.OrderPending, time.Now())

	if err := repo.Create(t.Context(), order); err != nil {
		t.Fatalf("first Create: %v", err)
	}

	if err := repo.Create(t.Context(), order); !errors.Is(err, domain.ErrOrderAlreadyExists) {
		t.Fatalf("second Create = %v, want ErrOrderAlreadyExists", err)
	}
}

func TestOrder_Get_NotFound(t *testing.T) {
	db := openDB(t)
	repo := repository.NewOrder(db)

	if _, err := repo.Get(t.Context(), ident.New()); !errors.Is(err, domain.ErrOrderNotFound) {
		t.Fatalf("Get = %v, want ErrOrderNotFound", err)
	}
}

func TestOrder_Update(t *testing.T) {
	db := openDB(t)
	repo := repository.NewOrder(db)
	order := newOrder(domain.OrderPending, time.Now())
	if err := repo.Create(t.Context(), order); err != nil {
		t.Fatalf("Create: %v", err)
	}

	later := order.CreatedAt.Add(time.Minute)
	if err := order.Cancel(later); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if err := repo.Update(t.Context(), order); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := repo.Get(t.Context(), order.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != domain.OrderCancelled || !got.UpdatedAt.Equal(later) {
		t.Errorf("after Update: status %q, updated_at %v", got.Status, got.UpdatedAt)
	}
}

func TestOrder_Update_NotFound(t *testing.T) {
	db := openDB(t)
	repo := repository.NewOrder(db)

	err := repo.Update(t.Context(), newOrder(domain.OrderPending, time.Now()))

	if !errors.Is(err, domain.ErrOrderNotFound) {
		t.Fatalf("Update = %v, want ErrOrderNotFound", err)
	}
}

func TestOrder_ExpirePending(t *testing.T) {
	db := openDB(t)
	repo := repository.NewOrder(db)

	// A cutoff far in the past keeps this test from touching rows that
	// other tests created at the current time.
	cutoff := time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
	now := time.Now().UTC().Truncate(time.Microsecond)

	stale := newOrder(domain.OrderPending, cutoff.Add(-time.Hour))
	fresh := newOrder(domain.OrderPending, cutoff.Add(time.Hour))
	stalePaid := newOrder(domain.OrderPaid, cutoff.Add(-time.Hour))
	for _, o := range []*domain.Order{stale, fresh, stalePaid} {
		if err := repo.Create(t.Context(), o); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	expired, err := repo.ExpirePending(t.Context(), cutoff, now)
	if err != nil {
		t.Fatalf("ExpirePending: %v", err)
	}
	if expired < 1 {
		t.Errorf("expired = %d, want at least the stale order", expired)
	}

	wantStatus := map[string]domain.OrderStatus{
		stale.ID:     domain.OrderExpired,
		fresh.ID:     domain.OrderPending,
		stalePaid.ID: domain.OrderPaid,
	}
	for id, want := range wantStatus {
		got, err := repo.Get(t.Context(), id)
		if err != nil {
			t.Fatalf("Get %s: %v", id, err)
		}
		if got.Status != want {
			t.Errorf("order %s status = %q, want %q", id, got.Status, want)
		}
	}
}

func TestInTx_RollsBackOnError(t *testing.T) {
	db := openDB(t)
	repo := repository.NewOrder(db)
	order := newOrder(domain.OrderPending, time.Now())
	boom := errors.New("boom")

	err := db.InTx(t.Context(), func(ctx context.Context) error {
		if err := repo.Create(ctx, order); err != nil {
			return err
		}
		return boom
	})

	if !errors.Is(err, boom) {
		t.Fatalf("InTx = %v, want %v", err, boom)
	}
	if _, err := repo.Get(t.Context(), order.ID); !errors.Is(err, domain.ErrOrderNotFound) {
		t.Fatalf("Get after rollback = %v, want ErrOrderNotFound", err)
	}
}

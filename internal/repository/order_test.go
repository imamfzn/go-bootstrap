package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v5"

	"github.com/imamfzn/go-bootstrap/internal/domain"
	"github.com/imamfzn/go-bootstrap/internal/platform/postgres"
	"github.com/imamfzn/go-bootstrap/internal/repository"
)

var (
	createdAt = time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC)
	updatedAt = createdAt.Add(time.Minute)

	orderColumns = []string{"id", "user_id", "product_id", "quantity", "amount", "note", "status", "created_at", "updated_at"}
)

// mockDB hands the repository a pgxmock pool, so no PostgreSQL is needed.
type mockDB struct {
	pool pgxmock.PgxPoolIface
}

func (m *mockDB) Conn(context.Context) postgres.Querier { return m.pool }

// newRepo returns the repository on a mock pool. Every expectation set on
// the mock must be met by the end of the test.
func newRepo(t *testing.T) (*repository.Order, pgxmock.PgxPoolIface) {
	t.Helper()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("create mock pool: %v", err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("database expectations not met: %v", err)
		}
	})

	return repository.NewOrder(&mockDB{pool: mock}), mock
}

func sampleOrder() *domain.Order {
	return &domain.Order{
		ID: "o1", UserID: "u1", ProductID: "p1", Quantity: 2, Amount: 5000, Note: "leave at the door",
		Status: domain.OrderPending, CreatedAt: createdAt, UpdatedAt: updatedAt,
	}
}

// orderRow is the sample order as the database returns it.
func orderRow() []any {
	return []any{"o1", "u1", "p1", 2, int64(5000), "leave at the door", "pending", createdAt, updatedAt}
}

func isDomainError(err error) bool {
	_, ok := errors.AsType[*domain.Error](err)
	return ok
}

func TestOrder_Create(t *testing.T) {
	repo, mock := newRepo(t)
	mock.ExpectExec("INSERT INTO orders").
		WithArgs("o1", "u1", "p1", 2, int64(5000), "leave at the door", "pending", createdAt, updatedAt).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	if err := repo.Create(t.Context(), sampleOrder()); err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestOrder_Create_Errors(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name       string
		execErr    error
		want       error
		wantDomain bool
	}{
		{"duplicate id becomes a domain error", &pgconn.PgError{Code: "23505"}, domain.ErrOrderAlreadyExists, true},
		{"another constraint stays a plain error", &pgconn.PgError{Code: "23514"}, nil, false},
		{"driver failure stays a plain error", boom, boom, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newRepo(t)
			mock.ExpectExec("INSERT INTO orders").WillReturnError(tt.execErr)

			err := repo.Create(t.Context(), sampleOrder())

			if err == nil {
				t.Fatal("Create succeeded, want an error")
			}
			if tt.want != nil && !errors.Is(err, tt.want) {
				t.Errorf("err = %v, want %v", err, tt.want)
			}
			if isDomainError(err) != tt.wantDomain {
				t.Errorf("is a domain error = %v, want %v (err: %v)", isDomainError(err), tt.wantDomain, err)
			}
		})
	}
}

func TestOrder_Get(t *testing.T) {
	repo, mock := newRepo(t)
	mock.ExpectQuery("SELECT .+ FROM orders WHERE id = ").
		WithArgs("o1").
		WillReturnRows(pgxmock.NewRows(orderColumns).AddRow(orderRow()...))

	got, err := repo.Get(t.Context(), "o1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if *got != *sampleOrder() {
		t.Errorf("Get returned\n %+v\nwant\n %+v", *got, *sampleOrder())
	}
}

func TestOrder_Get_ReturnsTimesInUTC(t *testing.T) {
	jakarta := time.FixedZone("WIB", 7*60*60)
	row := orderRow()
	row[7], row[8] = createdAt.In(jakarta), updatedAt.In(jakarta)

	repo, mock := newRepo(t)
	mock.ExpectQuery("SELECT .+ FROM orders").
		WillReturnRows(pgxmock.NewRows(orderColumns).AddRow(row...))

	got, err := repo.Get(t.Context(), "o1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.CreatedAt.Location() != time.UTC || !got.CreatedAt.Equal(createdAt) {
		t.Errorf("CreatedAt = %v, want %v in UTC", got.CreatedAt, createdAt)
	}
	if got.UpdatedAt.Location() != time.UTC || !got.UpdatedAt.Equal(updatedAt) {
		t.Errorf("UpdatedAt = %v, want %v in UTC", got.UpdatedAt, updatedAt)
	}
}

func TestOrder_Get_Errors(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name       string
		expect     func(mock pgxmock.PgxPoolIface)
		want       error
		wantDomain bool
	}{
		{
			name: "no row becomes a domain error",
			expect: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT .+ FROM orders").WillReturnRows(pgxmock.NewRows(orderColumns))
			},
			want: domain.ErrOrderNotFound, wantDomain: true,
		},
		{
			name: "query failure stays a plain error",
			expect: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT .+ FROM orders").WillReturnError(boom)
			},
			want: boom,
		},
		{
			name: "failure while reading stays a plain error",
			expect: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT .+ FROM orders").
					WillReturnRows(pgxmock.NewRows(orderColumns).AddRow(orderRow()...).RowError(0, boom))
			},
			want: boom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newRepo(t)
			tt.expect(mock)

			_, err := repo.Get(t.Context(), "o1")

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
			if isDomainError(err) != tt.wantDomain {
				t.Errorf("is a domain error = %v, want %v (err: %v)", isDomainError(err), tt.wantDomain, err)
			}
		})
	}
}

func TestOrder_GetForUpdate_LocksTheRow(t *testing.T) {
	repo, mock := newRepo(t)
	mock.ExpectQuery("SELECT .+ FROM orders WHERE id = .+ FOR UPDATE").
		WithArgs("o1").
		WillReturnRows(pgxmock.NewRows(orderColumns).AddRow(orderRow()...))

	got, err := repo.GetForUpdate(t.Context(), "o1")
	if err != nil {
		t.Fatalf("GetForUpdate: %v", err)
	}

	if got.ID != "o1" {
		t.Errorf("order id = %q, want o1", got.ID)
	}
}

func TestOrder_Update(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name   string
		expect func(exec *pgxmock.ExpectedExec)
		want   error
	}{
		{
			name:   "row updated",
			expect: func(exec *pgxmock.ExpectedExec) { exec.WillReturnResult(pgxmock.NewResult("UPDATE", 1)) },
		},
		{
			name:   "no such row becomes a domain error",
			expect: func(exec *pgxmock.ExpectedExec) { exec.WillReturnResult(pgxmock.NewResult("UPDATE", 0)) },
			want:   domain.ErrOrderNotFound,
		},
		{
			name:   "driver failure stays a plain error",
			expect: func(exec *pgxmock.ExpectedExec) { exec.WillReturnError(boom) },
			want:   boom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := sampleOrder()
			order.Status = domain.OrderCancelled

			repo, mock := newRepo(t)
			tt.expect(mock.ExpectExec("UPDATE orders SET").
				WithArgs("cancelled", "leave at the door", updatedAt, "o1"))

			if err := repo.Update(t.Context(), order); !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestOrder_ExpirePending(t *testing.T) {
	cutoff := createdAt.Add(-24 * time.Hour)

	repo, mock := newRepo(t)
	mock.ExpectExec("UPDATE orders SET").
		WithArgs("expired", createdAt, "pending", cutoff).
		WillReturnResult(pgxmock.NewResult("UPDATE", 3))

	expired, err := repo.ExpirePending(t.Context(), cutoff, createdAt)
	if err != nil {
		t.Fatalf("ExpirePending: %v", err)
	}

	if expired != 3 {
		t.Errorf("expired = %d, want 3", expired)
	}
}

func TestOrder_ExpirePending_Error(t *testing.T) {
	boom := errors.New("connection reset")

	repo, mock := newRepo(t)
	mock.ExpectExec("UPDATE orders SET").WillReturnError(boom)

	if _, err := repo.ExpirePending(t.Context(), createdAt, createdAt); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want %v", err, boom)
	}
}

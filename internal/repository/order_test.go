package repository_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/imamfzn/go-bootstrap/internal/domain"
	"github.com/imamfzn/go-bootstrap/internal/repository"
)

var (
	createdAt = time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC)
	updatedAt = createdAt.Add(time.Minute)

	orderColumns = []string{"id", "user_id", "product_id", "quantity", "amount", "note", "status", "created_at", "updated_at"}
)

func sampleOrder() *domain.Order {
	return &domain.Order{
		ID: "o1", UserID: "u1", ProductID: "p1", Quantity: 2, Amount: 5000, Note: "leave at the door",
		Status: domain.OrderPending, CreatedAt: createdAt, UpdatedAt: updatedAt,
	}
}

// orderRow is the sample order as the database would return it.
func orderRow() []any {
	return []any{"o1", "u1", "p1", 2, int64(5000), "leave at the door", "pending", createdAt, updatedAt}
}

func isDomainError(err error) bool {
	_, ok := errors.AsType[*domain.Error](err)
	return ok
}

func TestOrder_Create(t *testing.T) {
	db := &fakeDB{execTag: pgconn.NewCommandTag("INSERT 0 1")}

	if err := repository.NewOrder(db).Create(t.Context(), sampleOrder()); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if !strings.Contains(db.sql, "INSERT INTO orders") {
		t.Errorf("statement = %q, want an insert into orders", db.sql)
	}
	want := map[string]any{
		"id": "o1", "user_id": "u1", "product_id": "p1", "quantity": 2, "amount": int64(5000),
		"note": "leave at the door", "status": "pending", "created_at": createdAt, "updated_at": updatedAt,
	}
	got := db.namedArgs()
	for name, value := range want {
		if got[name] != value {
			t.Errorf("argument %s = %v, want %v", name, got[name], value)
		}
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
			db := &fakeDB{execErr: tt.execErr}

			err := repository.NewOrder(db).Create(t.Context(), sampleOrder())

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
	db := &fakeDB{rows: newFakeRows(orderColumns, orderRow())}

	got, err := repository.NewOrder(db).Get(t.Context(), "o1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if *got != *sampleOrder() {
		t.Errorf("Get returned\n %+v\nwant\n %+v", *got, *sampleOrder())
	}
	if db.namedArgs()["id"] != "o1" {
		t.Errorf("queried id = %v, want o1", db.namedArgs()["id"])
	}
	if strings.Contains(db.sql, "FOR UPDATE") {
		t.Errorf("Get locked the row: %q", db.sql)
	}
}

func TestOrder_Get_ReturnsTimesInUTC(t *testing.T) {
	jakarta := time.FixedZone("WIB", 7*60*60)
	row := orderRow()
	row[7], row[8] = createdAt.In(jakarta), updatedAt.In(jakarta)
	db := &fakeDB{rows: newFakeRows(orderColumns, row)}

	got, err := repository.NewOrder(db).Get(t.Context(), "o1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.CreatedAt.Location() != time.UTC || !got.CreatedAt.Equal(createdAt) {
		t.Errorf("CreatedAt = %v, want %v in UTC", got.CreatedAt, createdAt)
	}
}

func TestOrder_Get_Errors(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name       string
		db         *fakeDB
		want       error
		wantDomain bool
	}{
		{"no row becomes a domain error", &fakeDB{rows: newFakeRows(orderColumns)}, domain.ErrOrderNotFound, true},
		{"query failure stays a plain error", &fakeDB{queryErr: boom}, boom, false},
		{"failure while reading stays a plain error", &fakeDB{rows: &fakeRows{columns: orderColumns, err: boom}}, boom, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := repository.NewOrder(tt.db).Get(t.Context(), "o1")

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
	db := &fakeDB{rows: newFakeRows(orderColumns, orderRow())}

	got, err := repository.NewOrder(db).GetForUpdate(t.Context(), "o1")
	if err != nil {
		t.Fatalf("GetForUpdate: %v", err)
	}

	if got.ID != "o1" {
		t.Errorf("order id = %q, want o1", got.ID)
	}
	if !strings.Contains(db.sql, "FOR UPDATE") {
		t.Errorf("statement does not lock the row: %q", db.sql)
	}
}

func TestOrder_Update(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name string
		db   *fakeDB
		want error
	}{
		{"row updated", &fakeDB{execTag: pgconn.NewCommandTag("UPDATE 1")}, nil},
		{"no such row becomes a domain error", &fakeDB{execTag: pgconn.NewCommandTag("UPDATE 0")}, domain.ErrOrderNotFound},
		{"driver failure stays a plain error", &fakeDB{execErr: boom}, boom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := sampleOrder()
			order.Status = domain.OrderCancelled

			err := repository.NewOrder(tt.db).Update(t.Context(), order)

			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
			if got := tt.db.namedArgs()["status"]; got != "cancelled" {
				t.Errorf("status argument = %v, want cancelled", got)
			}
		})
	}
}

func TestOrder_ExpirePending(t *testing.T) {
	db := &fakeDB{execTag: pgconn.NewCommandTag("UPDATE 3")}
	cutoff := createdAt.Add(-24 * time.Hour)

	expired, err := repository.NewOrder(db).ExpirePending(t.Context(), cutoff, createdAt)
	if err != nil {
		t.Fatalf("ExpirePending: %v", err)
	}

	if expired != 3 {
		t.Errorf("expired = %d, want 3", expired)
	}
	args := db.namedArgs()
	if args["created_before"] != cutoff || args["now"] != createdAt {
		t.Errorf("cutoff = %v, now = %v; want %v, %v", args["created_before"], args["now"], cutoff, createdAt)
	}
	if args["pending"] != "pending" || args["expired"] != "expired" {
		t.Errorf("statuses = %v -> %v, want pending -> expired", args["pending"], args["expired"])
	}
}

func TestOrder_ExpirePending_Error(t *testing.T) {
	boom := errors.New("connection reset")
	db := &fakeDB{execErr: boom}

	if _, err := repository.NewOrder(db).ExpirePending(t.Context(), createdAt, createdAt); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want %v", err, boom)
	}
}

// Package repository implements the repositories declared in
// internal/usecase on PostgreSQL, with hand-written SQL.
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/imamfzn/go-bootstrap/internal/domain"
	"github.com/imamfzn/go-bootstrap/internal/platform/postgres"
)

const (
	orderColumns = `id, user_id, product_id, quantity, amount, note, status, created_at, updated_at`

	orderInsert = `
INSERT INTO orders (` + orderColumns + `)
VALUES (@id, @user_id, @product_id, @quantity, @amount, @note, @status, @created_at, @updated_at)`

	orderGet = `
SELECT ` + orderColumns + `
FROM orders
WHERE id = @id`

	orderGetForUpdate = orderGet + `
FOR UPDATE`

	orderUpdate = `
UPDATE orders
SET status = @status, note = @note, updated_at = @updated_at
WHERE id = @id`

	orderExpirePending = `
UPDATE orders
SET status = @expired, updated_at = @now
WHERE status = @pending AND created_at < @created_before`
)

// Order stores orders in PostgreSQL.
type Order struct {
	db *postgres.DB
}

// NewOrder returns the order repository.
func NewOrder(db *postgres.DB) *Order {
	return &Order{db: db}
}

func (r *Order) Create(ctx context.Context, order *domain.Order) error {
	_, err := r.db.Conn(ctx).Exec(ctx, orderInsert, orderArgs(order))
	switch {
	case postgres.IsUniqueViolation(err):
		return domain.ErrOrderAlreadyExists
	case err != nil:
		return fmt.Errorf("insert order %s: %w", order.ID, err)
	}
	return nil
}

func (r *Order) Get(ctx context.Context, id string) (*domain.Order, error) {
	return r.queryOne(ctx, orderGet, id)
}

func (r *Order) GetForUpdate(ctx context.Context, id string) (*domain.Order, error) {
	return r.queryOne(ctx, orderGetForUpdate, id)
}

func (r *Order) queryOne(ctx context.Context, query, id string) (*domain.Order, error) {
	rows, err := r.db.Conn(ctx).Query(ctx, query, pgx.NamedArgs{"id": id})
	if err != nil {
		return nil, fmt.Errorf("query order %s: %w", id, err)
	}

	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[orderRow])
	switch {
	case postgres.IsNoRows(err):
		return nil, domain.ErrOrderNotFound
	case err != nil:
		return nil, fmt.Errorf("scan order %s: %w", id, err)
	}
	return row.toDomain(), nil
}

func (r *Order) Update(ctx context.Context, order *domain.Order) error {
	tag, err := r.db.Conn(ctx).Exec(ctx, orderUpdate, orderArgs(order))
	if err != nil {
		return fmt.Errorf("update order %s: %w", order.ID, err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrOrderNotFound
	}
	return nil
}

func (r *Order) ExpirePending(ctx context.Context, createdBefore, now time.Time) (int, error) {
	tag, err := r.db.Conn(ctx).Exec(ctx, orderExpirePending, pgx.NamedArgs{
		"expired":        string(domain.OrderExpired),
		"pending":        string(domain.OrderPending),
		"now":            now,
		"created_before": createdBefore,
	})
	if err != nil {
		return 0, fmt.Errorf("expire pending orders: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

package repository

import (
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/imamfzn/go-bootstrap/internal/domain"
)

// orderRow mirrors one row of the orders table. It never leaves this
// package: callers only see domain.Order.
type orderRow struct {
	ID        string    `db:"id"`
	UserID    string    `db:"user_id"`
	ProductID string    `db:"product_id"`
	Quantity  int       `db:"quantity"`
	Amount    int64     `db:"amount"`
	Note      string    `db:"note"`
	Status    string    `db:"status"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (r *orderRow) toDomain() *domain.Order {
	return &domain.Order{
		ID:        r.ID,
		UserID:    r.UserID,
		ProductID: r.ProductID,
		Quantity:  r.Quantity,
		Amount:    r.Amount,
		Note:      r.Note,
		Status:    domain.OrderStatus(r.Status),
		CreatedAt: r.CreatedAt.UTC(),
		UpdatedAt: r.UpdatedAt.UTC(),
	}
}

func orderArgs(o *domain.Order) pgx.NamedArgs {
	return pgx.NamedArgs{
		"id":         o.ID,
		"user_id":    o.UserID,
		"product_id": o.ProductID,
		"quantity":   o.Quantity,
		"amount":     o.Amount,
		"note":       o.Note,
		"status":     string(o.Status),
		"created_at": o.CreatedAt,
		"updated_at": o.UpdatedAt,
	}
}

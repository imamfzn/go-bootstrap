package response

import (
	"time"

	"github.com/imamfzn/go-bootstrap/internal/domain"
)

// Order is the JSON form of an order.
type Order struct {
	ID        string    `json:"id"`
	ProductID string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Amount    int64     `json:"amount"`
	Note      string    `json:"note"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FromOrder maps a domain order to its JSON form.
func FromOrder(o *domain.Order) Order {
	return Order{
		ID:        o.ID,
		ProductID: o.ProductID,
		Quantity:  o.Quantity,
		Amount:    o.Amount,
		Note:      o.Note,
		Status:    string(o.Status),
		CreatedAt: o.CreatedAt,
		UpdatedAt: o.UpdatedAt,
	}
}

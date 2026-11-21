package request

import "github.com/imamfzn/go-bootstrap/internal/domain"

// CreateOrder is the body of POST /v1/orders.
type CreateOrder struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
	Amount    int64  `json:"amount"`
	Note      string `json:"note"`
}

// ToDomain builds the use case input. The user comes from authentication,
// never from the body.
func (r CreateOrder) ToDomain(userID string) domain.CreateOrder {
	return domain.CreateOrder{
		UserID:    userID,
		ProductID: r.ProductID,
		Quantity:  r.Quantity,
		Amount:    r.Amount,
		Note:      r.Note,
	}
}

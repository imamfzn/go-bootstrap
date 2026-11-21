// Package event defines the message contracts: the payload of every event
// this service publishes or consumes. It is shared by internal/publisher
// and internal/event/handler and imports nothing from this module.
//
// Payloads are a public contract. Changes must be additive: never rename
// or remove a field a consumer may read.
package event

import "time"

// Event type names, sent in the envelope and as a message attribute.
const (
	TypeOrderCreated = "order.created"
)

// OrderCreated is published when an order has been stored.
type OrderCreated struct {
	OrderID   string    `json:"order_id"`
	UserID    string    `json:"user_id"`
	Amount    int64     `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
}

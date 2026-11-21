// Package queue is the broker-independent messaging core: the message
// shape, handler and middleware types, the event envelope, and publishing.
// The Pub/Sub SDK is used only in the gcppubsub subpackage.
package queue

import (
	"context"
	"slices"
)

// Attribute names set on every published message.
const (
	AttrEventType = "event_type"
	AttrRequestID = "request_id"
)

// Message is one message as seen by handlers and middleware.
type Message struct {
	ID           string
	Subscription string
	Data         []byte
	Attributes   map[string]string
	// Attempt is the delivery attempt, starting at 1. It is 0 when the
	// subscription has no dead-letter policy, because the broker then does
	// not count attempts.
	Attempt int
}

// Handler processes one message. Returning nil acknowledges it. Returning
// an error asks the broker to redeliver it.
type Handler func(ctx context.Context, msg *Message) error

// Middleware wraps a Handler.
type Middleware func(next Handler) Handler

// Chain wraps h so that the first middleware runs outermost.
func Chain(h Handler, mws ...Middleware) Handler {
	for _, mw := range slices.Backward(mws) {
		h = mw(h)
	}
	return h
}

// Sender delivers a message to a topic and returns once the broker has
// accepted it.
type Sender interface {
	Send(ctx context.Context, topic string, msg *Message) error
}

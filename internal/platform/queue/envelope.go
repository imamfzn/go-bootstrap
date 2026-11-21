package queue

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"time"

	"github.com/imamfzn/go-bootstrap/internal/platform/ident"
	"github.com/imamfzn/go-bootstrap/internal/platform/requestid"
)

// ErrMalformed marks a message whose body could not be decoded. Retrying
// cannot fix it, so consumers acknowledge it.
var ErrMalformed = errors.New("malformed message")

// Envelope is the JSON body of every message.
type Envelope[T any] struct {
	Meta    Metadata `json:"meta"`
	Payload T        `json:"payload"`
}

// Metadata describes the event itself. ID is unique per event, which lets
// consumers detect a redelivery.
type Metadata struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	RequestID  string    `json:"request_id,omitzero"`
	OccurredAt time.Time `json:"occurred_at"`
}

// Publish wraps payload in an Envelope and sends it to topic.
func Publish[T any](ctx context.Context, s Sender, topic, eventType string, payload T) error {
	env := Envelope[T]{
		Meta: Metadata{
			ID:         ident.New(),
			Type:       eventType,
			RequestID:  requestid.From(ctx),
			OccurredAt: time.Now().UTC(),
		},
		Payload: payload,
	}

	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("encode %s event: %w", eventType, err)
	}

	attrs := map[string]string{AttrEventType: eventType}
	if env.Meta.RequestID != "" {
		attrs[AttrRequestID] = env.Meta.RequestID
	}

	if err := s.Send(ctx, topic, &Message{Data: data, Attributes: attrs}); err != nil {
		return fmt.Errorf("send %s event: %w", eventType, err)
	}
	return nil
}

// JSON adapts a typed handler to a Handler by decoding the envelope.
func JSON[T any](fn func(ctx context.Context, env *Envelope[T]) error) Handler {
	return func(ctx context.Context, msg *Message) error {
		var env Envelope[T]
		if err := json.Unmarshal(msg.Data, &env); err != nil {
			return fmt.Errorf("%w: %w", ErrMalformed, err)
		}
		return fn(ctx, &env)
	}
}

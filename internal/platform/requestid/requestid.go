// Package requestid carries a request identifier through a context, so one
// request can be followed across HTTP, logs and published messages.
package requestid

import "context"

type key struct{}

// With returns a context carrying the identifier.
func With(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, key{}, id)
}

// From returns the identifier in the context, or an empty string.
func From(ctx context.Context) string {
	id, _ := ctx.Value(key{}).(string)
	return id
}

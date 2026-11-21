package usecase

import "context"

// Transactor runs a function in one database transaction. Repository calls
// made with the context passed to fn join that transaction. Returning an
// error rolls everything back.
type Transactor interface {
	InTx(ctx context.Context, fn func(ctx context.Context) error) error
}

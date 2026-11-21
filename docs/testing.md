# Testing

## Running

```sh
make test          # go test -race ./...
```

Repository tests need a migrated PostgreSQL:

```sh
make up
make migrate-up
make test          # TEST_DATABASE_URL comes from .env
```

Without `TEST_DATABASE_URL` the repository tests skip, so `go test ./...` passes on a machine with no database. CI always runs them.

## End to end

`make smoke` builds the three binaries and runs them against the local Postgres and Pub/Sub emulator (`make up`). It creates an order through the API, waits for the worker to pay it, checks the error responses, runs a job and checks that both long-running binaries stop cleanly on SIGTERM. CI runs it on every pull request.

## Rules

- `*_test.go` beside the code, in package `xxx_test`: tests use only the exported API.
- Fakes are written by hand and embed the interface they replace. No mock generators.
- A fake shared by several test files of a package goes in `fake_test.go`.
- Assert on results and state. Assert that a call happened only when the call is the behaviour ("published once", "not charged").
- One behaviour per test, named for it: `TestCancel_PaidOrder`.
- Table-driven where cases share a shape.
- Use `t.Context()`, a fixed clock (`clock.Fixed`) and `testing/synctest`; never `time.Sleep`.
- `app` and `cmd` are not unit-tested: they only wire.

A test that still passes after the logic it covers is deleted is a bug in the test.

## What each layer tests against

| Layer | Depends on in tests | Example |
|---|---|---|
| `domain` | nothing | `internal/domain/order_test.go` |
| `usecase` | in-memory fakes of its interfaces | `internal/usecase/order_test.go`, `fake_test.go` |
| `repository` | real PostgreSQL | `internal/repository/order_test.go` |
| `publisher` | a fake `queue.Sender` | `internal/publisher/order_test.go` |
| `external/*` | `httptest.Server` | `internal/external/payment/client_test.go` |
| `http` | the whole router with a fake use case | `internal/http/router/router_test.go` |
| `event` | a fake use case | `internal/event/handler/order_test.go` |
| `platform/*` | nothing outside the standard library | `internal/platform/queue/queue_test.go` |

## Writing a fake

Embed the interface, then implement only what the test needs. A new method on the interface then breaks no fake, and a call the test did not expect panics instead of passing silently.

```go
type fakeOrders struct {
	usecase.OrderRepository

	rows map[string]domain.Order
}

func (f *fakeOrders) Get(_ context.Context, id string) (*domain.Order, error) {
	row, ok := f.rows[id]
	if !ok {
		return nil, domain.ErrOrderNotFound
	}
	return &row, nil
}
```

## Tests that guard the structure

- `locales/locales_test.go`: every domain error code is translated in every locale.
- `internal/domain`: error codes are unique.
- `db/embed_test.go`: migrations are embedded and correctly named.
- `internal/http/response/error_test.go`: an unknown error never leaks its text to a client.

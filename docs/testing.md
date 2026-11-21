# Testing

## Running

```sh
make test          # go test -race ./...
```

Nothing else has to be running. Every `*_test.go` in this repository is a unit test: it opens no network connection, needs no database and no Docker, and does not wait on the real clock.

## End to end

`make smoke` builds the three binaries and runs them against the local Postgres and Pub/Sub emulator (`make up`). It creates an order through the API, waits for the worker to pay it, checks the error responses, runs a job and checks that both long-running binaries stop cleanly on SIGTERM.

It is a script, not a Go test, and it is never run by `make test`. It is the only check that executes the real SQL and the real Pub/Sub client, so run it after changing a repository, a migration, a publisher or the wiring in `internal/app`.

## Rules

- `*_test.go` beside the code, in package `xxx_test`: tests use only the exported API.
- A test touches nothing outside its own process. No database, no Docker, no listening port, no outbound request, no file outside the embedded ones.
- Fakes for our own interfaces are written by hand and embed the interface they replace. No mock generators.
- For a third-party system, an established mock library may be used instead of writing one, provided it is maintained and builds against the versions in `go.mod`. See "Faking the database" below.
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
| `repository` | an in-memory fake of the database handle | `internal/repository/order_test.go`, `fake_test.go` |
| `publisher` | a fake `queue.Sender` | `internal/publisher/order_test.go` |
| `external/*` | a fake `http.RoundTripper` | `internal/external/payment/client_test.go` |
| `http` | the whole router with a fake use case | `internal/http/router/router_test.go` |
| `event` | a fake use case | `internal/event/handler/order_test.go` |
| `platform/*` | nothing outside the process; the HTTP server runs on an in-memory pipe | `internal/platform/queue/queue_test.go`, `internal/platform/httpserver/server_test.go` |

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

## Faking the database

A repository takes a `repository.DB`, a one-method interface that `*postgres.DB` implements. Tests pass `fakeDB` from `internal/repository/fake_test.go` instead: it records the statement and arguments the repository ran and answers with the result, rows or error the test prepared. That is enough to test what a repository is responsible for: mapping a row to a domain value, and translating driver errors into domain errors.

What these tests cannot tell you is whether the SQL is valid. `make smoke` does that.

[pgxmock](https://github.com/pashagolub/pgxmock) would be the natural library to use here. At the time of writing its latest release, v5.2.0, does not compile against pgx v5.13.0 (`pgx.BatchResults` gained a `Tx` method), so using it would mean holding the production driver back for the sake of a test dependency. When a release that builds with the current pgx exists, `fakeDB` can be replaced by `pgxmock.NewPool()` without touching the repositories.

## Faking an HTTP provider

Give the client an `http.Client` whose `Transport` is a function:

```go
type roundTripFunc func(r *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
```

The test inspects the request and returns a response, and no connection is opened.

## Tests that guard the structure

- `locales/locales_test.go`: every domain error code is translated in every locale.
- `internal/domain`: error codes are unique.
- `db/embed_test.go`: migrations are embedded and correctly named.
- `internal/http/response/error_test.go`: an unknown error never leaks its text to a client.

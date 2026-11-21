# Conventions

The rules of this codebase: decisions a compiler or linter will not catch. [architecture.md](architecture.md) explains the reasoning behind them.

## Layers

```
cmd/*                  one main per binary; calls app and nothing else
internal/app           composition root; the only place things are constructed
internal/config        the only place environment variables are read
internal/domain        types, behaviour, errors; pure
internal/usecase       business logic and every contract of the core
internal/repository    outbound: PostgreSQL
internal/publisher     outbound: Pub/Sub
internal/external/*    outbound: other services, one package per system
internal/http          inbound: REST
internal/event         message contracts; inbound: Pub/Sub
internal/job           inbound: scheduled tasks
internal/platform/*    generic plumbing with no business knowledge
db/                    migrations, embedded
locales/               translations, embedded
```

## Imports

- `domain`, `platform/*` and the root `event` package import nothing else from this module.
- `usecase` imports `domain`, and pure helpers such as `platform/clock`.
- `repository` and `external/*` import `domain` and `platform/*`.
- `publisher` imports `domain`, `event` and `platform/queue`.
- `http`, `event/handler`, `event/router` and `job` import `usecase`, `domain` and `platform/*`.
- Only `app` constructs things. `cmd/*` only calls `app`.
- pgx appears only in `platform/postgres` and `repository`. The Pub/Sub SDK appears only in `platform/queue/gcppubsub`.
- `depguard` in `.golangci.yaml` enforces the core of this.

## Interfaces

- `usecase` owns every contract. What it offers (`OrderUsecase`) sits beside its implementation. What it needs lives in `repository.go`, `publisher.go`, `external.go` and `transactor.go`.
- Use case structs are unexported; constructors return the interface.
- Adapters return concrete types. `app` passes them where an interface is expected.

## Parameters and results

- A struct is passed and returned as a pointer: `Create(ctx, input *domain.CreateOrder) (*domain.Order, error)`, `NewOrder(d *OrderDeps)`, `postgres.Open(ctx, cfg *Config)`.
- Methods on structs use pointer receivers.
- Exceptions: slices, arrays and maps are passed as they are, and so are small standard value types such as `time.Time` and `time.Duration`.
- A pointer parameter is expected to be non-nil. Callers are inside this module, so functions do not check for nil.

## Domain

- Types, behaviour as methods, and errors. No struct tags, no I/O.
- Input types have a hand-written `Validate()`. Each rule returns its own error.
- Errors are sentinels declared with `newError(code, message)`: a `Code` and a default English `Message`, with no HTTP status.
- Codes follow `{DOMAIN}_{USECASE}_{ERROR-CAUSE}`, for example `ORDER_CANCEL_NOT-ALLOWED`.
- Never modify a sentinel. Add detail by wrapping it with `%w`.
- A rule with business meaning belongs here even when it is about time or text.

## Use cases

- All validation happens here: call `Validate()` first, then check the rules that need I/O. Never assume an inbound adapter validated anything.
- Own the transaction boundary through `Transactor`.
- Take and return domain types only. Never see a topic name, a payload struct, a request or a response.
- Read the time from the injected `clock.Clock`, never `time.Now()`.
- Do not log.

## Repositories

- SQL is hand-written, in `const` at the top of the file, with explicit columns. No ORM, no query generator, no `SELECT *`.
- The row struct is private, in `*_model.go`, with `toDomain()`.
- Translate expected driver errors (no row, unique violation) into `domain.Err*`. Wrap everything else with `%w`.
- Use `db.Conn(ctx)` for every statement, so the call joins a transaction when there is one.
- No business rules. No calls to other repositories.

## Publishers and events

- One publisher type per domain, one method per event, named for the business fact (`OrderCreated`).
- Payload structs live in the root `event` package and are shared by the publisher and the handler. Changes are additive only.
- Topic and subscription names come from config.
- Publishing waits for the broker. Never publish inside an open transaction; publish after commit, and use an outbox table when an event must not be lost.
- Message handlers are idempotent: Pub/Sub delivers at least once.
- The worker acknowledges on success, on a domain error and on a malformed body. Anything else is redelivered. Retry and dead-letter limits are set on the subscription, in Terraform.

## HTTP

- A handler decodes, calls one use case and responds. It returns `error` and never writes an error itself.
- `request` only parses. A body that cannot be read is `400 REQUEST_MALFORMED`.
- `json` tags exist only in `request`, `response` and `event`. A domain value is never serialised directly.
- Every error response is exactly `{"code": "...", "message": "..."}`.
- `response.ErrorWriter` is the only place that maps errors to statuses and the only place that translates. A domain error gets its status from `statusByError`, or 422. Any other error is logged and answered with a generic 500.

## Jobs

- A job calls a use case and exits non-zero on failure.
- Jobs are safe to run twice and to resume.

## External systems

- One package per system, named for the system. The interface in `usecase/external.go` is named for the need (`PaymentGateway`).
- The provider's DTOs and SDK types stay inside the package.
- Start from `httpclient.New`, so every call has a timeout. Retry only idempotent calls.
- Translate the provider's refusals into `domain.Err*`.

## Platform and helpers

- Check the standard library first: `slices`, `maps`, `strings`, `cmp`, `min`, `max`, `new(expr)`, `uuid`.
- Name a package for what it provides (`clock`, `ident`). Use an `x` suffix when extending a standard package's theme (`timex`, `slicex`).
- No `util`, `common`, `helper` or `lib`. No package named after a standard one.
- Do not extract early: a helper stays unexported in its package until a third package needs it.

## Errors, context and logging

- Wrap with `fmt.Errorf("doing x: %w", err)`. Compare with `errors.Is` and `errors.AsType`.
- Log once, at the edge, and only for errors that are not domain errors. Never log and return.
- `ctx` is the first parameter and is never stored in a struct. `context.Background()` appears only in `main`.
- Use `slog.*Context`. Never log secrets, tokens or message and request bodies.
- No `panic` outside `main`.

## Locales

- `locales/<locale>.json` maps an error code to its message.
- Every domain error code exists in every locale; a test enforces it. `en.json` matches the default messages in `domain`.

## Configuration and runtime

- Environment variables are read only in `config`, per binary, and checked at startup.
- No package-level mutable state and no `init()`.
- Every goroutine has an owner that waits for it. Nothing runs after an HTTP response is sent.
- Shutdown completes within 10 seconds.

## Go version and style

- The module targets the Go version in `go.mod`. Prefer current standard library and language features over older idioms and over third-party packages: `encoding/json/v2`, `uuid`, `errors.AsType`, `slices`, `maps`, `cmp.Or`, `strings.SplitSeq`, `sync.WaitGroup.Go`, `testing/synctest`.
- Run `make fix` to apply the `go fix` modernizers.
- Files are `snake_case.go` with the domain as prefix (`order_model.go`). Types are `OrderUsecase`, `OrderRepository`, `OrderHandler`.

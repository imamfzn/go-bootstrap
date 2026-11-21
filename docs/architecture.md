# Architecture

One codebase, three binaries, one image.

| Binary | Runs as | Driven by |
|---|---|---|
| `cmd/api` | Cloud Run service | HTTP requests |
| `cmd/job <name>` | Cloud Run Job | a schedule, or the deploy pipeline (`migrate`) |
| `cmd/worker` | long-running container | Pub/Sub subscriptions |

All three drive the same use cases. They differ only in what calls them.

## Shape

```
        inbound                     core                      outbound

  http    ─┐                                              ┌─  repository  → PostgreSQL
  event   ─┼──▶  usecase  ──▶  domain                     ├─  publisher   → Pub/Sub
  job     ─┘        │                                     └─  external/*  → other services
                    └── declares the interfaces ──────────▶  implemented by
```

- **`domain`** holds the business types, their behaviour and their errors. It is pure Go with no I/O.
- **`usecase`** holds the logic and owns every contract: the `*Usecase` interfaces it offers, and the repository, publisher and gateway interfaces it needs.
- **Outbound adapters** implement those interfaces. Each translates the outside world's types and errors into domain ones, so nothing foreign crosses into the core.
- **Inbound adapters** turn a request, a message or a schedule into one use case call.
- **`platform`** is plumbing that knows nothing about the business and could be copied into another service unchanged.
- **`app`** wires everything together. It is the only package that constructs anything, so the dependency graph of each binary reads top to bottom in one file.

The rule that keeps this honest: imports point inward. `usecase` never imports an adapter, a driver or a framework. `.golangci.yaml` enforces it with `depguard`.

## Why the core owns the interfaces

The use cases are the stable centre. What changes over the life of a service is what surrounds them: REST gains a gRPC sibling, a queue is swapped, a table moves. Declaring both sides of the core's contracts in `usecase` means such a change touches one adapter package and the wiring in `app`.

## How an error travels

```
PostgreSQL / provider error
   │  repository, external:  expected case → domain.Err*;  anything else → wrapped with %w
   ▼
usecase:  passes errors up, adds business-rule errors
   ▼
http/response.ErrorWriter   domain error → its status (default 422), code, translated message
                            anything else → logged, generic 500
event/middleware.ErrorPolicy  domain error or malformed body → acknowledge
                              anything else → redeliver
```

A `domain.Error` carries a stable `Code` and a default message, and nothing about transport. HTTP statuses live in one table in `http/response`; the worker's acknowledge-or-retry decision lives in one middleware. Each edge maps the same errors its own way.

Every error response has the same body:

```json
{ "code": "ORDER_CANCEL_NOT-ALLOWED", "message": "This order can no longer be cancelled" }
```

`code` is for programs and never changes with language. `message` is for people and follows `Accept-Language`, looked up by code in `locales/`.

## Validation

All of it happens in the core, because the worker and the jobs reach the use cases without passing through HTTP.

- Rules that need no I/O are a `Validate()` method on the domain input type.
- Rules that need the database are checked in the use case.
- `http/request` only parses. "We could not read this" is a 400; "we read it and it is not acceptable" is a 422 with a specific code.

## Transactions

A use case opens a transaction with `Transactor.InTx`. The transaction travels in the `context`, and `postgres.DB.Conn(ctx)` returns it when present, so a repository never knows whether it is inside one. Calls to outside systems stay outside transactions.

## Events

The publisher maps a domain value to a payload in `internal/event` and sends it in an envelope with an event ID, type, request ID and timestamp. The worker decodes the same payload type and calls a use case.

Publishing and writing to the database are two systems, so they cannot be atomic by default. The example publishes after the insert commits and reports a failure to the caller. When an event must never be lost, write it to an outbox table in the same transaction and relay it from a job.

Pub/Sub delivers at least once. Handlers are written to be safe to run twice: `Pay` does nothing for an order that is already paid, and the payment provider deduplicates on the order ID.

## Migrations

Plain SQL files run by goose. Locally and in CI the goose CLI applies them. In production the deploy pipeline runs `job migrate` as a Cloud Run Job, inside the project, before rolling out the API; the files are embedded in the binary, so an image always carries the schema that matches its code. The API never migrates at startup.

## What the skeleton leaves to you

- **Authentication.** `middleware.Verifier` is the seam. The only implementation shipped trusts the token as the user ID and is refused unless `AUTH_DEV_INSECURE=true`.
- **The payment provider.** `external/payment` is a placeholder HTTP client that shows the pattern. With no `PAYMENT_BASE_URL` every payment is approved locally.
- **Telemetry.** `platform/telemetry.Setup` is an empty seam for OpenTelemetry.
- **Infrastructure.** The Cloud Run service and jobs, Cloud SQL, topics, subscriptions with their retry and dead-letter policies, and IAM belong in Terraform.

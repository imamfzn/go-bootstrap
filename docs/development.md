# Development

## Prerequisites

- Go, at the version in `go.mod`
- Docker, for PostgreSQL and the Pub/Sub emulator
- [golangci-lint](https://golangci-lint.run/welcome/install), for `make lint`

## First run

```sh
cp .env.example .env
make up            # Postgres and the Pub/Sub emulator, with the topic and subscription
make migrate-up
make run-api       # in one terminal
make run-worker    # in another
```

`.env` is loaded by the Makefile. The binaries themselves read only real environment variables.

## Commands

| Task | Command |
|---|---|
| List every command | `make help` |
| Start / stop local dependencies | `make up`, `make down` |
| Run the API / worker | `make run-api`, `make run-worker` |
| Run a job | `make run-job name=expire-orders` |
| Tests | `make test` |
| End-to-end check of the running binaries | `make smoke` |
| Vet, lint | `make vet`, `make lint` |
| Apply the Go modernizers and format | `make fix` |
| All checks: tidy, vet, lint, test | `make check` |
| Build the binaries / the image | `make build`, `make image` |

## Configuration

Every variable is declared in `internal/config/config.go` and documented in `.env.example`. Each binary loads only its own struct (`API`, `Worker`, `Job`), all of which embed `Base`.

Three settings exist only for local development and must stay off elsewhere:

| Variable | Effect |
|---|---|
| `AUTH_DEV_INSECURE=true` | trusts the bearer token as the user ID, with no verification |
| `PAYMENT_BASE_URL` empty | approves every payment without calling a provider |
| `PUBSUB_EMULATOR_HOST` | sends Pub/Sub traffic to the emulator |

## Adding a feature

1. `db/migrations`: the table or column. See [migrations.md](migrations.md).
2. `domain`: the type, its behaviour, its errors. Add each new code to every file in `locales/`.
3. `usecase`: the logic, and any new method on the interfaces it needs.
4. `repository`, and `publisher` or `external` if needed: the implementation.
5. `http`: request, response, handler, route. Add a line to `statusByError` when the error is not a 422.
6. `app`: the wiring, usually two or three lines in `core.go`.

Each step has a test. See [testing.md](testing.md).

## Adding a job

1. Add the use case method.
2. Add a type with a `Run(ctx) error` method in `internal/job`.
3. Register it by name in the map in `internal/app/job.go`.

## Adding a subscription

1. Declare the payload in `internal/event`.
2. Add the handler method in `internal/event/handler` and bind it in `internal/event/router`.
3. Add the subscription name to `config.Worker` and pass it in `internal/app/worker.go`.
4. Create the subscription, with its retry and dead-letter policy, in Terraform, and in `make pubsub-setup` for local use.

## Using this repository as a template

```sh
make rename module=github.com/acme/orders
```

Then replace the `order` example with your own domain. Three seams are left open on purpose; [architecture.md](architecture.md) says where each plugs in:

- authentication (`middleware.Verifier`)
- the payment provider (`internal/external/payment`)
- telemetry (`internal/platform/telemetry`)

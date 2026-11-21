# go-bootstrap

A starting point for a Go service on Google Cloud: a REST API on Cloud Run, scheduled work on Cloud Run Jobs and a Pub/Sub consumer, sharing one codebase and one image, with PostgreSQL on Cloud SQL.

It is a layered design (`domain`, `usecase`, `repository`, adapters) with the rules written down in [AGENTS.md](AGENTS.md) and explained in [docs/architecture.md](docs/architecture.md). A small `order` domain runs through every layer as a worked example.

## Stack

| Concern | Choice |
|---|---|
| Language | Go 1.27 |
| HTTP | `net/http` with [chi](https://github.com/go-chi/chi) |
| Database | PostgreSQL through [pgx](https://github.com/jackc/pgx), hand-written SQL |
| Migrations | [goose](https://github.com/pressly/goose), plain SQL, embedded |
| Messaging | Google Cloud Pub/Sub |
| Config | environment variables through [envconfig](https://github.com/kelseyhightower/envconfig) |
| JSON, UUIDs, logging | standard library: `encoding/json/v2`, `uuid`, `log/slog` |

## Quick start

Requires Go 1.27 and Docker.

```sh
cp .env.example .env
make up            # Postgres and the Pub/Sub emulator
make migrate-up
make run-api       # in one terminal
make run-worker    # in another
```

Then:

```sh
# Create an order. Locally the bearer token is used as the user ID.
curl -s localhost:8080/v1/orders \
  -H 'Authorization: Bearer user-1' \
  -d '{"product_id":"p1","quantity":2,"amount":5000}'

# The worker receives order.created and marks the order paid.
curl -s localhost:8080/v1/orders/<id> -H 'Authorization: Bearer user-1'

# Errors are always {code, message}; the message follows Accept-Language.
curl -s localhost:8080/v1/orders/<id>/cancel -X POST \
  -H 'Authorization: Bearer user-1' -H 'Accept-Language: id'
```

`make help` lists every command.

## Using it as a template

```sh
make rename module=github.com/acme/orders
```

Then replace the `order` example with your own domain, following "Adding a feature" in [AGENTS.md](AGENTS.md). Three seams are deliberately left open: authentication, the payment provider and telemetry. [docs/architecture.md](docs/architecture.md) says where each one plugs in.

## Layout

```
cmd/            api, job, worker
internal/
  app/          wiring
  config/       environment variables
  domain/       types, behaviour, errors
  usecase/      business logic and its contracts
  repository/   PostgreSQL
  publisher/    Pub/Sub, outbound
  external/     other services
  http/         REST: handler, middleware, request, response, router
  event/        message contracts and the worker's handlers
  job/          scheduled tasks
  platform/     plumbing: postgres, queue, httpserver, logger, i18n, ...
db/             migrations
locales/        translations
deploy/         Dockerfile
```

## Deploying

`deploy/Dockerfile` builds one distroless image holding `/bin/api`, `/bin/job` and `/bin/worker`. `.github/workflows/deploy.yaml` is a template that builds the image, runs the `migrate` job and then rolls out the API; it expects the Cloud Run resources to exist already.

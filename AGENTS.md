# AGENTS.md

Entry point for coding agents and new contributors. The guidelines themselves live in `docs/`; read the one that matches your task before changing code.

## Read first

| Document | Read it when |
|---|---|
| [docs/architecture.md](docs/architecture.md) | you need to understand how the pieces fit: layers, the error flow, transactions, events |
| [docs/conventions.md](docs/conventions.md) | **before any code change.** The rules for every layer: imports, interfaces, errors, naming |
| [docs/testing.md](docs/testing.md) | before writing or changing a test |
| [docs/migrations.md](docs/migrations.md) | before touching `db/migrations` or anything about the schema |
| [docs/development.md](docs/development.md) | to set up, run things, or follow the steps for adding a feature, a job or a subscription |

## Done means

1. The change follows [docs/conventions.md](docs/conventions.md).
2. The change has a test, written as [docs/testing.md](docs/testing.md) describes.
3. `make check` passes: tidy, vet, lint and tests.

## The rules most often broken

These are in the documents above; they are repeated here because a mistake in any of them is easy to make and hard to notice.

- `usecase` imports only `domain` and pure helpers. Never a repository, a driver, a framework or an SDK.
- All validation lives in `domain` and `usecase`. `http/request` only parses.
- A failure a client may see is a `domain.Err*` with its own code. Add every new code to every file in `locales/`.
- Every error response is exactly `{"code", "message"}`, written only by `response.ErrorWriter`.
- SQL is hand-written in the repository. No ORM, no query generator.
- Tests live in package `xxx_test` and use hand-written fakes. No mock generators.
- Never edit an applied migration.
- Only `internal/app` constructs things; only `internal/config` reads the environment.

## Keeping the docs true

When a change makes a document wrong, update the document in the same change. When a new convention is agreed, add it to [docs/conventions.md](docs/conventions.md), not here.

# Migrations

Schema changes are plain SQL files in `db/migrations`, run by [goose](https://github.com/pressly/goose).

## Commands

```sh
make migrate-new name=add_orders_index   # create a file
make migrate-up                          # apply pending migrations
make migrate-status                      # show what is applied
make migrate-down                        # roll back the last one (development only)
```

A file is named `<YYYYMMDDHHMMSS>_<snake_case>.sql` and has two sections:

```sql
-- +goose Up
ALTER TABLE orders ADD COLUMN shipped_at timestamptz;

-- +goose Down
ALTER TABLE orders DROP COLUMN shipped_at;
```

goose runs each migration in a transaction, so a failure rolls back cleanly.

## Where migrations run

| Where | How |
|---|---|
| Local | `make migrate-up`, using the goose CLI against `DATABASE_URL` |
| Production | the `migrate` job, run by the deploy pipeline before the API rolls out |

The files are embedded in the binary (`db/embed.go`), so `/bin/job migrate` needs nothing but the image and always carries the schema that matches its code. The pipeline starts the job with `gcloud run jobs execute --wait`: the runner never connects to the database, and a failed migration stops the rollout.

The API never migrates at startup. Several instances start at once, and a slow migration would fail the startup probe.

The goose CLI is a development tool pinned in `tools.mod`, separate from `go.mod`, so its database drivers do not become dependencies of the service. Do not run `go mod tidy` on `tools.mod`.

## Rules

- Never edit a migration that has been applied. Write a new one.
- Expand, then contract. Old and new code run side by side during a rollout, so every migration must work with the release before it: add a nullable column, ship code that writes it, backfill, and only then add the constraint or drop the old column.
- On a busy table, create indexes with `CREATE INDEX CONCURRENTLY`, which needs `-- +goose NO TRANSACTION` at the top of the file, and set a `lock_timeout` so a blocked statement fails fast instead of queueing every query behind it.
- Backfills are jobs, batched and resumable. They are not migrations.
- `Down` is for development. In production, roll forward with a new migration.
- Give the `migrate` job its own database user with DDL rights. The API and worker need only DML.

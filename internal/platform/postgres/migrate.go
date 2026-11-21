package postgres

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// Migrate applies every pending migration found at the root of migrations.
// It is safe to run repeatedly: applied migrations are skipped.
func (db *DB) Migrate(ctx context.Context, migrations fs.FS) error {
	sqlDB := stdlib.OpenDBFromPool(db.pool)
	defer sqlDB.Close()

	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations)
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}

	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	slog.InfoContext(ctx, "migrations applied", "count", len(results))
	return nil
}

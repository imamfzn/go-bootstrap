package app

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/imamfzn/go-bootstrap/db"
	"github.com/imamfzn/go-bootstrap/internal/config"
	"github.com/imamfzn/go-bootstrap/internal/job"
)

// jobMigrate needs only the database, so it runs without the rest of the
// core. That lets it run before topics or the payment provider exist.
const jobMigrate = "migrate"

// RunJob runs the named job once.
func RunJob(ctx context.Context, cfg config.Job, name string) error {
	if name == jobMigrate {
		return runMigrate(ctx, cfg)
	}

	c, err := newCore(ctx, cfg.Base)
	if err != nil {
		return err
	}
	defer c.close(ctx)

	jobs := map[string]job.Func{
		"expire-orders": job.NewExpireOrders(c.orders, c.logger).Run,
	}

	run, ok := jobs[name]
	if !ok {
		known := append(slices.Sorted(maps.Keys(jobs)), jobMigrate)
		return fmt.Errorf("unknown job %q (known: %s)", name, strings.Join(known, ", "))
	}

	c.logger.InfoContext(ctx, "job started", "job", name)
	if err := run(ctx); err != nil {
		return fmt.Errorf("job %s: %w", name, err)
	}
	c.logger.InfoContext(ctx, "job finished", "job", name)
	return nil
}

func runMigrate(ctx context.Context, cfg config.Job) error {
	inf, err := newInfra(ctx, cfg.Base)
	if err != nil {
		return err
	}
	defer inf.close(ctx)

	migrations, err := db.Migrations()
	if err != nil {
		return err
	}
	return inf.db.Migrate(ctx, migrations)
}

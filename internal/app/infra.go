// Package app is the composition root: the only package that constructs
// things. Each Run function builds what its binary needs, top to bottom,
// and blocks until the binary should exit. Nothing here decides anything.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/imamfzn/go-bootstrap/internal/config"
	"github.com/imamfzn/go-bootstrap/internal/platform/logger"
	"github.com/imamfzn/go-bootstrap/internal/platform/postgres"
	"github.com/imamfzn/go-bootstrap/internal/platform/telemetry"
)

// infra is what every binary needs before any business code runs.
type infra struct {
	logger    *slog.Logger
	db        *postgres.DB
	telemetry telemetry.Shutdown
}

func newInfra(ctx context.Context, cfg *config.Base) (*infra, error) {
	log := logger.New(os.Stdout, cfg.LogLevel).With("service", cfg.ServiceName)
	// The default logger is the one process-wide value this module sets,
	// so that libraries logging through slog use the same output.
	slog.SetDefault(log)

	shutdown, err := telemetry.Setup(ctx, cfg.ServiceName)
	if err != nil {
		return nil, fmt.Errorf("set up telemetry: %w", err)
	}

	db, err := postgres.Open(ctx, &postgres.Config{
		URL:                    cfg.DatabaseURL,
		InstanceConnectionName: cfg.DBInstanceConnectionName,
		IAMAuth:                cfg.DBIAMAuth,
		MaxConns:               cfg.DBMaxConns,
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	return &infra{logger: log, db: db, telemetry: shutdown}, nil
}

// close releases everything. ctx is the run context, which is usually
// already cancelled by the time this runs, so cleanup detaches from it.
func (i *infra) close(ctx context.Context) {
	i.db.Close()
	if err := i.telemetry(context.WithoutCancel(ctx)); err != nil {
		i.logger.ErrorContext(ctx, "shut down telemetry", "err", err)
	}
}

// logExit records why a binary is stopping. After a signal the cause of the
// context names the signal.
func logExit(ctx context.Context, log *slog.Logger, name string, err error) {
	if err != nil {
		log.ErrorContext(ctx, name+" stopped with an error", "err", err)
		return
	}
	log.InfoContext(ctx, name+" stopped", "reason", fmt.Sprint(context.Cause(ctx)))
}

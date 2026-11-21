// Command api serves the REST API. It runs as a Cloud Run service.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/imamfzn/go-bootstrap/internal/app"
	"github.com/imamfzn/go-bootstrap/internal/config"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "api:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	cfg, err := config.LoadAPI()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	return app.RunAPI(ctx, cfg)
}

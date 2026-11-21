// Command job runs one named task and exits. It runs as a Cloud Run Job:
//
//	job migrate
//	job expire-orders
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/imamfzn/go-bootstrap/internal/app"
	"github.com/imamfzn/go-bootstrap/internal/config"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "job:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: job <name>")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	cfg, err := config.LoadJob()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	return app.RunJob(ctx, cfg, args[0])
}

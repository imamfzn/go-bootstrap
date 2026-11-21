package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/imamfzn/go-bootstrap/internal/config"
	"github.com/imamfzn/go-bootstrap/internal/http/middleware"
	"github.com/imamfzn/go-bootstrap/internal/http/router"
	"github.com/imamfzn/go-bootstrap/internal/platform/httpserver"
	"github.com/imamfzn/go-bootstrap/internal/platform/i18n"
	"github.com/imamfzn/go-bootstrap/locales"
)

// RunAPI serves the REST API until ctx is cancelled.
func RunAPI(ctx context.Context, cfg config.API) error {
	c, err := newCore(ctx, cfg.Base)
	if err != nil {
		return err
	}
	defer c.close(ctx)

	catalog, err := i18n.Load(locales.FS, "en")
	if err != nil {
		return fmt.Errorf("load locales: %w", err)
	}

	verifier, err := newVerifier(ctx, cfg, c)
	if err != nil {
		return err
	}

	api := router.New(router.Deps{
		Orders:   c.orders,
		Catalog:  catalog,
		Logger:   c.logger,
		Verifier: verifier,
		Ready:    c.db.Ping,
	})

	addr := net.JoinHostPort("", strconv.Itoa(cfg.Port))
	c.logger.InfoContext(ctx, "api listening", "addr", addr)

	err = httpserver.Run(ctx, httpserver.Config{Addr: addr, ShutdownTimeout: cfg.ShutdownTimeout}, api)
	logExit(ctx, c.logger, "api", err)
	return err
}

// newVerifier returns how bearer tokens are checked. The skeleton has no
// identity provider: plug yours in here by implementing middleware.Verifier.
func newVerifier(ctx context.Context, cfg config.API, c *core) (middleware.Verifier, error) {
	if cfg.AuthDevInsecure {
		c.logger.WarnContext(ctx, "AUTH_DEV_INSECURE is on: bearer tokens are trusted as user IDs without verification")
		return middleware.DevInsecureVerifier(), nil
	}
	return nil, errors.New("no token verifier is configured: implement middleware.Verifier for your identity provider, or set AUTH_DEV_INSECURE=true on a developer machine")
}

// Package router assembles the REST API: middleware, routes and the
// adapter that lets handlers return errors.
package router

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/imamfzn/go-bootstrap/internal/domain"
	"github.com/imamfzn/go-bootstrap/internal/http/handler"
	"github.com/imamfzn/go-bootstrap/internal/http/middleware"
	"github.com/imamfzn/go-bootstrap/internal/http/response"
	"github.com/imamfzn/go-bootstrap/internal/platform/i18n"
	"github.com/imamfzn/go-bootstrap/internal/usecase"
)

// Deps is what the API needs.
type Deps struct {
	Orders usecase.OrderUsecase

	Catalog  *i18n.Catalog
	Logger   *slog.Logger
	Verifier middleware.Verifier
	// Ready reports whether the service can reach what it depends on.
	Ready func(ctx context.Context) error
}

// handlerFunc is an http.HandlerFunc that reports failure by returning it.
type handlerFunc func(w http.ResponseWriter, r *http.Request) error

// adapter turns a handlerFunc into an http.HandlerFunc. Every error from
// every handler goes through the one ErrorWriter.
type adapter struct {
	errs *response.ErrorWriter
}

func (a adapter) handle(h handlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			a.errs.Write(w, r, err)
		}
	}
}

func fail(err error) handlerFunc {
	return func(http.ResponseWriter, *http.Request) error { return err }
}

// New returns the API as an http.Handler.
func New(d Deps) http.Handler {
	errs := response.NewErrorWriter(d.Catalog, d.Logger)
	a := adapter{errs: errs}

	r := chi.NewRouter()
	r.Use(
		middleware.RequestID,
		middleware.Logging(d.Logger),
		middleware.Locale(d.Catalog),
		middleware.Recover(d.Logger, errs),
	)
	r.NotFound(a.handle(fail(domain.ErrNotFound)))
	r.MethodNotAllowed(a.handle(fail(domain.ErrMethodNotAllowed)))

	health := handler.NewHealth(d.Ready)
	r.Get("/healthz", a.handle(health.Live))
	r.Get("/readyz", a.handle(health.Ready))

	r.Route("/v1", func(r chi.Router) {
		r.Use(middleware.Auth(d.Verifier, errs))

		registerOrder(r, a, handler.NewOrder(d.Orders))
	})

	return r
}

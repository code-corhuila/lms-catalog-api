package httpserver

import (
	"crypto/rsa"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/code-corhuila/lms-catalog-api/internal/adapter/in/httpapi/handler"
	"github.com/code-corhuila/lms-catalog-api/internal/adapter/in/httpapi/middleware"
)

// RouterConfig carries what the router needs to wire itself.
type RouterConfig struct {
	DB                          *pgxpool.Pool
	JWTPublicKey                *rsa.PublicKey
	InternalJWTSecret           string
	CORSOrigin                  string
	Books                       *handler.BookHandler
	FeatureCatalogSearchEnabled bool
	// Authorizer calls lms-iam's /authorize (ADR-012). nil disables the
	// check entirely (local dev without lms-iam running), never silently
	// passing a request through where a check was expected.
	Authorizer middleware.Authorizer
}

// NewRouter builds the chi router with the base middleware stack, health
// endpoints, and the Catalog bounded context's /books routes.
func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.CorrelationID)
	r.Use(middleware.CORS(cfg.CORSOrigin))

	health := handler.NewHealthHandler(cfg.DB)
	r.Get("/health", health.Liveness)
	r.Get("/health/ready", health.Readiness)

	// requirePermission is a no-op when Authorizer is nil (lms-iam not
	// configured for this run) rather than a silent allow-everything.
	requirePermission := func(permission string) func(http.Handler) http.Handler {
		if cfg.Authorizer == nil {
			return func(next http.Handler) http.Handler { return next }
		}
		return middleware.RequirePermission(cfg.Authorizer, permission)
	}

	r.Route("/api/v1", func(api chi.Router) {
		api.Group(func(protected chi.Router) {
			protected.Use(middleware.RequireAuth(cfg.JWTPublicKey, cfg.InternalJWTSecret))

			protected.Route("/books", func(books chi.Router) {
				books.With(requirePermission("books:write")).Post("/", cfg.Books.Create) // HU-04
				// FEATURE_CATALOG_SEARCH_ENABLED (week 9, config hardening):
				// default on; exists so search can be turned off per-
				// environment without a deploy. When off, GET isn't
				// registered on this path at all — verified directly:
				// chi answers 405 (the path exists for POST), not a 5xx
				// from a half-wired handler.
				if cfg.FeatureCatalogSearchEnabled {
					books.With(requirePermission("books:read")).Get("/", cfg.Books.List) // HU-04, search half
				}
				books.With(requirePermission("books:write")).Post("/{id}/loan-copy", cfg.Books.LoanCopy)     // needed by circulation-service
				books.With(requirePermission("books:write")).Post("/{id}/return-copy", cfg.Books.ReturnCopy) // needed by circulation-service
				books.With(requirePermission("books:write")).Patch("/{id}", cfg.Books.Update)                // HU-09
				books.With(requirePermission("books:write")).Delete("/{id}", cfg.Books.Delete)
			})
		})
	})

	return r
}

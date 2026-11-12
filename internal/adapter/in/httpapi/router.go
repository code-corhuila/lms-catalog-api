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
	// IAMAuthorizer backs RequirePermission on the destructive /books
	// routes (ADR-012) — closes part of the risk-9 finding in
	// library-docs/05-architecture/security/stride-MENESESLUIS-bot.md §4
	// row E: a leaked internal token could previously rewrite the
	// catalog with no role check at all.
	IAMAuthorizer middleware.Authorizer
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

	r.Route("/api/v1", func(api chi.Router) {
		api.Group(func(protected chi.Router) {
			protected.Use(middleware.RequireAuth(cfg.JWTPublicKey, cfg.InternalJWTSecret))

			protected.Route("/books", func(books chi.Router) {
				books.With(middleware.RequirePermission("books:write", cfg.IAMAuthorizer)).
					Post("/", cfg.Books.Create) // HU-04, ADR-012
				// FEATURE_CATALOG_SEARCH_ENABLED (week 9, config hardening):
				// default on; exists so search can be turned off per-
				// environment without a deploy. When off, GET isn't
				// registered on this path at all — verified directly:
				// chi answers 405 (the path exists for POST), not a 5xx
				// from a half-wired handler.
				if cfg.FeatureCatalogSearchEnabled {
					books.Get("/", cfg.Books.List) // HU-04, search half
				}
				books.With(middleware.RequireCaller("circulation-service")).Post("/{id}/loan-copy", cfg.Books.LoanCopy)     // ADR-013: internal-only
				books.With(middleware.RequireCaller("circulation-service")).Post("/{id}/return-copy", cfg.Books.ReturnCopy) // ADR-013: internal-only
				books.With(middleware.RequireCaller("lms-worker")).Post("/reindex", cfg.Books.Reindex)                      // ADR-013: internal-only
				books.With(middleware.RequirePermission("books:write", cfg.IAMAuthorizer)).
					Patch("/{id}", cfg.Books.Update) // HU-09, ADR-012
				books.With(middleware.RequirePermission("books:delete", cfg.IAMAuthorizer)).
					Delete("/{id}", cfg.Books.Delete) // ADR-012
			})
		})
	})

	return r
}

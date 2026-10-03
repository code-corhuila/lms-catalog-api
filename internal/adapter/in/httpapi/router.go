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
	DB                *pgxpool.Pool
	JWTPublicKey      *rsa.PublicKey
	InternalJWTSecret string
	CORSOrigin        string
	Books             *handler.BookHandler
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
				books.Post("/", cfg.Books.Create)                     // HU-04
				books.Get("/", cfg.Books.List)                        // HU-04, search half
				books.Post("/{id}/loan-copy", cfg.Books.LoanCopy)     // needed by circulation-service
				books.Post("/{id}/return-copy", cfg.Books.ReturnCopy) // needed by circulation-service
			})
		})
	})

	return r
}

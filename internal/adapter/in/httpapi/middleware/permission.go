package middleware

import (
	"context"
	"net/http"
)

// Authorizer is the driven port RequirePermission depends on — satisfied
// by iamclient.Client, kept here as a minimal interface so this package
// never imports a concrete HTTP client (rules/2-anexos/C-api-hexagonal.md).
type Authorizer interface {
	Authorize(ctx context.Context, administratorID, permission string) (bool, error)
}

// RequirePermission calls lms-iam's /authorize (ADR-012) for the caller's
// administratorId before letting a destructive /books write through. Runs
// after RequireAuth, which already put the caller's sub into context —
// for a real Administrator session that's their administratorId; this
// middleware is never applied to an internal-only route, so it never sees
// a service-token sub (ADR-013's RequireCaller covers those instead).
//
// Fails closed: if lms-iam is unreachable or errors, the request is
// rejected, not allowed through. ADR-012 already names this as a known,
// accepted risk ("a down lms-iam could block every protected operation
// system-wide") — not something this middleware silently works around.
func RequirePermission(permission string, authz Authorizer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			administratorID := AdministratorID(r.Context())
			allowed, err := authz.Authorize(r.Context(), administratorID, permission)
			if err != nil || !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"FORBIDDEN","message":"This administrator cannot perform this action"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

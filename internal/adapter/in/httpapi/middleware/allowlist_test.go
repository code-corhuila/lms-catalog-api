package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func withCaller(sub string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), administratorIDKey{}, sub)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestRequireCaller_Allowed(t *testing.T) {
	handler := withCaller("circulation-service")(RequireCaller("circulation-service")(okHandler()))

	req := httptest.NewRequest(http.MethodPost, "/books/1/loan-copy", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRequireCaller_Denied(t *testing.T) {
	handler := withCaller("membership-service")(RequireCaller("circulation-service")(okHandler()))

	req := httptest.NewRequest(http.MethodPost, "/books/1/loan-copy", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestRequireCaller_EmptySubDenied(t *testing.T) {
	handler := RequireCaller("circulation-service")(okHandler())

	req := httptest.NewRequest(http.MethodPost, "/books/1/loan-copy", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

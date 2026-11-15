package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

var errBoom = errors.New("boom")

type fakeAuthorizer struct {
	allowed bool
	err     error
}

func (f fakeAuthorizer) Authorize(_ context.Context, _, _ string) (bool, error) {
	return f.allowed, f.err
}

func TestRequirePermission_Allowed(t *testing.T) {
	handler := withCaller("admin-1")(RequirePermission("books:write", fakeAuthorizer{allowed: true})(okHandler()))

	req := httptest.NewRequest(http.MethodPost, "/books", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRequirePermission_Denied(t *testing.T) {
	handler := withCaller("admin-1")(RequirePermission("books:write", fakeAuthorizer{allowed: false})(okHandler()))

	req := httptest.NewRequest(http.MethodPost, "/books", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestRequirePermission_FailsClosedOnIAMError(t *testing.T) {
	handler := withCaller("admin-1")(RequirePermission("books:write", fakeAuthorizer{err: errBoom})(okHandler()))

	req := httptest.NewRequest(http.MethodPost, "/books", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 (fail closed), got %d", rec.Code)
	}
}

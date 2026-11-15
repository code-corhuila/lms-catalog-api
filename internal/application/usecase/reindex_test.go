package usecase_test

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/code-corhuila/lms-catalog-api/internal/application/usecase"
)

func TestReindex_IsANoOpStub(t *testing.T) {
	uc := usecase.NewReindex(zap.NewNop().Sugar())

	if err := uc.Execute(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

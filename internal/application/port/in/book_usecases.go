// Package in holds the driving (inbound) ports — what this service offers,
// as interfaces the HTTP adapter depends on instead of the concrete use case
// structs directly (rules/2-anexos/C-api-hexagonal.md, "Puertos de entrada").
package in

import (
	"context"

	"github.com/code-corhuila/lms-catalog-api/internal/domain/catalog"
)

// CreateBookUseCase registers a new Book (HU-04).
//
// replayed reports whether idempotencyKey had already been used: when true,
// the returned book is the one originally created by that key, and the HTTP
// adapter must answer 200, not 201 (rules/2-anexos/C-api-hexagonal.md,
// numeral 5.3.8).
type CreateBookUseCase interface {
	Execute(ctx context.Context, title, author, isbn, category string, year, totalCopies int, idempotencyKey string) (book *catalog.Book, replayed bool, err error)
}

// LoanBookCopyUseCase decrements a Book's availability — called by
// circulation-service when registering a loan (HU-06).
type LoanBookCopyUseCase interface {
	Execute(ctx context.Context, id string) (*catalog.Book, error)
}

// ReturnBookCopyUseCase increments a Book's availability — called by
// circulation-service when registering a return (HU-07).
type ReturnBookCopyUseCase interface {
	Execute(ctx context.Context, id string) (*catalog.Book, error)
}

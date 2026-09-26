// Package out holds the driven (outbound) ports every use case depends on —
// what the application needs from the outside world, never how it's implemented
// (rules/2-anexos/C-api-hexagonal.md, "Puertos de salida").
package out

import (
	"context"

	"github.com/code-corhuila/lms-catalog-api/internal/domain/catalog"
)

// BookRepository is the driven port for Book persistence.
type BookRepository interface {
	FindByID(ctx context.Context, id string) (*catalog.Book, error)
	FindByISBN(ctx context.Context, isbn string) (*catalog.Book, error)
	Search(ctx context.Context, query, category string, page, limit int) (books []*catalog.Book, total int, err error)
	Save(ctx context.Context, b *catalog.Book) error
}

// IdempotencyStore is the driven port for the idempotent-creation check
// (rules/2-anexos/C-api-hexagonal.md, numeral 5.3.8): a repeated POST /books
// with the same Idempotency-Key must return the original book, not create a
// second one.
//
// Provisional: the only adapter today (internal/adapter/out/idempotency) is an
// in-memory map — it does not survive a restart, and does not coordinate
// across more than one running instance. A durable adapter belongs in
// lms-catalog-db's own idempotency_key table (rules/2-anexos/A-db-postgres.md),
// which does not exist yet.
type IdempotencyStore interface {
	// Get returns the bookID previously saved under key, and found=true, or
	// found=false if key has never been used.
	Get(ctx context.Context, key string) (bookID string, found bool, err error)
	// Save records that key produced bookID.
	Save(ctx context.Context, key, bookID string) error
}

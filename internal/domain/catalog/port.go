package catalog

import "context"

// BookRepository is the driven port for Book persistence.
type BookRepository interface {
	FindByID(ctx context.Context, id string) (*Book, error)
	FindByISBN(ctx context.Context, isbn string) (*Book, error)
	Search(ctx context.Context, query, category string, page, limit int) (books []*Book, total int, err error)
	Save(ctx context.Context, b *Book) error
}

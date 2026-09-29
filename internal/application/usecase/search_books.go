package usecase

import (
	"context"

	out "github.com/code-corhuila/lms-catalog-api/internal/application/port/out"
	"github.com/code-corhuila/lms-catalog-api/internal/domain/catalog"
)

// SearchBooks implements the list/search half of HU-04.
type SearchBooks struct {
	Books out.BookRepository
}

func NewSearchBooks(books out.BookRepository) *SearchBooks {
	return &SearchBooks{Books: books}
}

func (uc *SearchBooks) Execute(ctx context.Context, query, category string, page, limit int) ([]*catalog.Book, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return uc.Books.Search(ctx, query, category, page, limit)
}

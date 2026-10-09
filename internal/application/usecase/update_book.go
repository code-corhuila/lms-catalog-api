package usecase

import (
	"context"

	out "github.com/code-corhuila/lms-catalog-api/internal/application/port/out"
	"github.com/code-corhuila/lms-catalog-api/internal/domain/catalog"
)

// UpdateBook implements HU-09's acceptance criteria. ISBN is intentionally not
// editable here — it is Book's stable business key
// (library-docs/02-domain/entities-and-rules.md, modeling note on Book) — so
// there is no "duplicate ISBN on edit" case to reject. Copy counts are not
// touched either, so existing loans referencing the book are unaffected.
type UpdateBook struct {
	Books out.BookRepository
}

func NewUpdateBook(books out.BookRepository) *UpdateBook {
	return &UpdateBook{Books: books}
}

func (uc *UpdateBook) Execute(ctx context.Context, id, title, author, category string, year int) (*catalog.Book, error) {
	book, err := uc.Books.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := book.Update(title, author, category, year); err != nil {
		return nil, err
	}

	if err := uc.Books.Save(ctx, book); err != nil {
		return nil, err
	}
	return book, nil
}

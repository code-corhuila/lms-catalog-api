package usecase

import (
	"context"

	out "github.com/code-corhuila/lms-catalog-api/internal/application/port/out"
	"github.com/code-corhuila/lms-catalog-api/internal/domain/catalog"
)

// UpdateBook implements HU-09's edit acceptance criteria — title, author,
// category, year, and totalCopies are editable; ISBN is not (catalog.Book.Update).
type UpdateBook struct {
	Books out.BookRepository
}

func NewUpdateBook(books out.BookRepository) *UpdateBook {
	return &UpdateBook{Books: books}
}

func (uc *UpdateBook) Execute(ctx context.Context, id, title, author, category string, year, totalCopies int) (*catalog.Book, error) {
	book, err := uc.Books.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := book.Update(title, author, category, year); err != nil {
		return nil, err
	}
	if err := book.UpdateCopies(totalCopies); err != nil {
		return nil, err
	}
	if err := uc.Books.Save(ctx, book); err != nil {
		return nil, err
	}
	return book, nil
}

package usecase

import (
	"context"
	"errors"

	"github.com/code-corhuila/lms-catalog-api/internal/domain/catalog"
)

// ErrISBNAlreadyExists — FR-008: rejects registration when the ISBN is already
// registered.
var ErrISBNAlreadyExists = errors.New("isbn already exists")

// CreateBook implements HU-04's acceptance criteria.
type CreateBook struct {
	Books catalog.BookRepository
}

func NewCreateBook(books catalog.BookRepository) *CreateBook {
	return &CreateBook{Books: books}
}

func (uc *CreateBook) Execute(ctx context.Context, title, author, isbn, category string, year, totalCopies int) (*catalog.Book, error) {
	if _, err := uc.Books.FindByISBN(ctx, isbn); err == nil {
		return nil, ErrISBNAlreadyExists
	} else if !errors.Is(err, catalog.ErrBookNotFound) {
		return nil, err
	}

	book, err := catalog.NewBook(title, author, isbn, category, year, totalCopies)
	if err != nil {
		return nil, err
	}

	if err := uc.Books.Save(ctx, book); err != nil {
		return nil, err
	}
	return book, nil
}

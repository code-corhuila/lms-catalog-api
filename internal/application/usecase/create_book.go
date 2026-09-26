package usecase

import (
	"context"
	"errors"

	out "github.com/code-corhuila/lms-catalog-api/internal/application/port/out"
	"github.com/code-corhuila/lms-catalog-api/internal/domain/catalog"
)

// ErrISBNAlreadyExists — FR-008: rejects registration when the ISBN is already
// registered.
var ErrISBNAlreadyExists = errors.New("isbn already exists")

// CreateBook implements HU-04's acceptance criteria, plus the idempotent
// creation rules/2-anexos/C-api-hexagonal.md (numeral 5.3.8) requires: a
// repeated request carrying the same Idempotency-Key must not create a
// second Book.
type CreateBook struct {
	Books       out.BookRepository
	Idempotency out.IdempotencyStore
}

func NewCreateBook(books out.BookRepository, idempotency out.IdempotencyStore) *CreateBook {
	return &CreateBook{Books: books, Idempotency: idempotency}
}

// Execute registers a Book. replayed=true means idempotencyKey had already
// been used — the returned book is the original one, and the caller (the
// HTTP adapter) must answer 200, not 201.
func (uc *CreateBook) Execute(ctx context.Context, title, author, isbn, category string, year, totalCopies int, idempotencyKey string) (book *catalog.Book, replayed bool, err error) {
	if idempotencyKey != "" {
		if existingID, found, err := uc.Idempotency.Get(ctx, idempotencyKey); err != nil {
			return nil, false, err
		} else if found {
			existing, err := uc.Books.FindByID(ctx, existingID)
			if err != nil {
				return nil, false, err
			}
			return existing, true, nil
		}
	}

	if _, err := uc.Books.FindByISBN(ctx, isbn); err == nil {
		return nil, false, ErrISBNAlreadyExists
	} else if !errors.Is(err, catalog.ErrBookNotFound) {
		return nil, false, err
	}

	newBook, err := catalog.NewBook(title, author, isbn, category, year, totalCopies)
	if err != nil {
		return nil, false, err
	}

	if err := uc.Books.Save(ctx, newBook); err != nil {
		return nil, false, err
	}

	if idempotencyKey != "" {
		if err := uc.Idempotency.Save(ctx, idempotencyKey, newBook.ID); err != nil {
			return nil, false, err
		}
	}

	return newBook, false, nil
}

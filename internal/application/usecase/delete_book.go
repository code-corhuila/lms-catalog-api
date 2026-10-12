package usecase

import (
	"context"
	"errors"

	out "github.com/code-corhuila/lms-catalog-api/internal/application/port/out"
)

// ErrBookHasActiveLoans rejects a delete when some copies are still on loan —
// deleting the book out from under an open Loan would orphan circulation-service's
// record of what was borrowed.
var ErrBookHasActiveLoans = errors.New("book has copies currently on loan")

// DeleteBook removes a Book from the catalog. Not tied to any HU — the
// project's other domains never hard-delete (students are deactivated, loans
// are returned), but the catalog portal's UI needs a way to remove a bad
// registration.
type DeleteBook struct {
	Books out.BookRepository
}

func NewDeleteBook(books out.BookRepository) *DeleteBook {
	return &DeleteBook{Books: books}
}

func (uc *DeleteBook) Execute(ctx context.Context, id string) error {
	book, err := uc.Books.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if book.AvailableCopies < book.TotalCopies {
		return ErrBookHasActiveLoans
	}
	return uc.Books.Delete(ctx, id)
}

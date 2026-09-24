package usecase

import (
	"context"

	"github.com/code-corhuila/lms-catalog-api/internal/domain/catalog"
)

// LoanBookCopy and ReturnBookCopy expose catalog.Book's copy-count changes
// over HTTP — needed by circulation-service (HU-06/HU-07), which can no
// longer call BookRepository directly now that Book lives in a different
// database (library-docs/09-microservices/service-boundary-rules.md).
type LoanBookCopy struct {
	Books catalog.BookRepository
}

func NewLoanBookCopy(books catalog.BookRepository) *LoanBookCopy {
	return &LoanBookCopy{Books: books}
}

func (uc *LoanBookCopy) Execute(ctx context.Context, id string) (*catalog.Book, error) {
	book, err := uc.Books.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := book.LoanOneCopy(); err != nil {
		return nil, err
	}
	if err := uc.Books.Save(ctx, book); err != nil {
		return nil, err
	}
	return book, nil
}

type ReturnBookCopy struct {
	Books catalog.BookRepository
}

func NewReturnBookCopy(books catalog.BookRepository) *ReturnBookCopy {
	return &ReturnBookCopy{Books: books}
}

func (uc *ReturnBookCopy) Execute(ctx context.Context, id string) (*catalog.Book, error) {
	book, err := uc.Books.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := book.ReturnOneCopy(); err != nil {
		return nil, err
	}
	if err := uc.Books.Save(ctx, book); err != nil {
		return nil, err
	}
	return book, nil
}

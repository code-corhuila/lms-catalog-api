package usecase_test

import (
	"context"
	"testing"

	"github.com/code-corhuila/lms-catalog-api/internal/application/usecase"
	"github.com/code-corhuila/lms-catalog-api/internal/domain/catalog"
)

// fakeBookRepository — Fake test double (11-quality/tdd-guide.md).
type fakeBookRepository struct {
	byISBN map[string]*catalog.Book
	byID   map[string]*catalog.Book
}

func newFakeBookRepository() *fakeBookRepository {
	return &fakeBookRepository{byISBN: map[string]*catalog.Book{}, byID: map[string]*catalog.Book{}}
}

func (f *fakeBookRepository) FindByID(_ context.Context, id string) (*catalog.Book, error) {
	b, ok := f.byID[id]
	if !ok {
		return nil, catalog.ErrBookNotFound
	}
	return b, nil
}

func (f *fakeBookRepository) FindByISBN(_ context.Context, isbn string) (*catalog.Book, error) {
	b, ok := f.byISBN[isbn]
	if !ok {
		return nil, catalog.ErrBookNotFound
	}
	return b, nil
}

func (f *fakeBookRepository) Search(_ context.Context, _, _ string, _, _ int) ([]*catalog.Book, int, error) {
	return nil, 0, nil
}

func (f *fakeBookRepository) Save(_ context.Context, b *catalog.Book) error {
	f.byISBN[b.ISBN] = b
	f.byID[b.ID] = b
	return nil
}

func TestCreateBook_InitializesAvailabilityToTotalCopies(t *testing.T) {
	repo := newFakeBookRepository()
	uc := usecase.NewCreateBook(repo)

	book, err := uc.Execute(context.Background(), "Clean Code", "Robert C. Martin", "978-0132350884", "Software", 2008, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if book.AvailableCopies != 3 {
		t.Fatalf("expected availableCopies=3, got %d", book.AvailableCopies)
	}
}

func TestCreateBook_RejectsDuplicateISBN(t *testing.T) {
	// FR-008: reject registration when the ISBN is already registered.
	repo := newFakeBookRepository()
	uc := usecase.NewCreateBook(repo)

	_, err := uc.Execute(context.Background(), "Book A", "Author A", "978-0132350884", "Cat", 2020, 1)
	if err != nil {
		t.Fatalf("unexpected error on first registration: %v", err)
	}

	_, err = uc.Execute(context.Background(), "Book B", "Author B", "978-0132350884", "Cat", 2021, 2)
	if err != usecase.ErrISBNAlreadyExists {
		t.Fatalf("expected ErrISBNAlreadyExists, got %v", err)
	}
}

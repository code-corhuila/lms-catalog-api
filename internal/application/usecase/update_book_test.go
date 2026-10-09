package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/code-corhuila/lms-catalog-api/internal/application/usecase"
	"github.com/code-corhuila/lms-catalog-api/internal/domain/catalog"
)

// fakeBookRepository is declared once in create_book_test.go (same package) —
// reused here rather than redeclared.

func seedFakeBook(t *testing.T, repo *fakeBookRepository) *catalog.Book {
	t.Helper()
	book, err := catalog.NewBook("Clean Code", "Robert C. Martin", "978-0132350884", "Software", 2008, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := repo.Save(context.Background(), book); err != nil {
		t.Fatalf("unexpected error saving fixture: %v", err)
	}
	return book
}

func TestUpdateBook_Succeeds(t *testing.T) {
	repo := newFakeBookRepository()
	book := seedFakeBook(t, repo)
	uc := usecase.NewUpdateBook(repo)

	updated, err := uc.Execute(context.Background(), book.ID, "Clean Code (2nd ed.)", "Robert C. Martin", "Software Engineering", 2009)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Title != "Clean Code (2nd ed.)" || updated.Category != "Software Engineering" || updated.Year != 2009 {
		t.Fatalf("expected the edited values, got %+v", updated)
	}
	if stored, _ := repo.FindByID(context.Background(), book.ID); stored.Title != "Clean Code (2nd ed.)" {
		t.Fatalf("expected the edit to be saved, got title %q", stored.Title)
	}
}

func TestUpdateBook_KeepsISBNAndCopiesUnchanged(t *testing.T) {
	// FR-012: ISBN is immutable through this action — it isn't even a parameter.
	// FR-011: copy counts are untouched, so existing loans are unaffected.
	repo := newFakeBookRepository()
	book := seedFakeBook(t, repo)
	if err := book.LoanOneCopy(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	uc := usecase.NewUpdateBook(repo)

	updated, err := uc.Execute(context.Background(), book.ID, "New Title", "New Author", "New Category", 2020)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.ISBN != "978-0132350884" {
		t.Fatalf("expected ISBN to remain unchanged, got %q", updated.ISBN)
	}
	if updated.TotalCopies != 3 || updated.AvailableCopies != 2 {
		t.Fatalf("expected copies 2/3 to be kept, got %d/%d", updated.AvailableCopies, updated.TotalCopies)
	}
}

func TestUpdateBook_FailsWhenNotFound(t *testing.T) {
	uc := usecase.NewUpdateBook(newFakeBookRepository())

	_, err := uc.Execute(context.Background(), "does-not-exist", "Title", "Author", "Category", 2020)
	if !errors.Is(err, catalog.ErrBookNotFound) {
		t.Fatalf("expected ErrBookNotFound, got %v", err)
	}
}

func TestUpdateBook_RejectsEmptyTitle(t *testing.T) {
	repo := newFakeBookRepository()
	book := seedFakeBook(t, repo)
	uc := usecase.NewUpdateBook(repo)

	if _, err := uc.Execute(context.Background(), book.ID, "", "Author", "Category", 2020); err == nil {
		t.Fatal("expected an error for an empty title")
	}
	if stored, _ := repo.FindByID(context.Background(), book.ID); stored.Title != "Clean Code" {
		t.Fatalf("expected the stored title to be untouched, got %q", stored.Title)
	}
}

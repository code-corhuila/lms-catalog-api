package catalog_test

import (
	"testing"

	"github.com/code-corhuila/lms-catalog-api/internal/domain/catalog"
)

func TestNewBook_InitializesAvailabilityToTotalCopies(t *testing.T) {
	b, err := catalog.NewBook("Clean Code", "Robert C. Martin", "978-0132350884", "Software", 2008, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.AvailableCopies != 3 {
		t.Fatalf("expected availableCopies=3, got %d", b.AvailableCopies)
	}
}

func TestNewBook_RejectsZeroCopies(t *testing.T) {
	// INV-003: a book must be registered with at least one copy.
	_, err := catalog.NewBook("Title", "Author", "isbn", "Category", 2020, 0)
	if err == nil {
		t.Fatal("expected error for totalCopies=0")
	}
}

func TestLoanOneCopy_DecrementsAvailability(t *testing.T) {
	b, _ := catalog.NewBook("Title", "Author", "isbn", "Category", 2020, 2)
	if err := b.LoanOneCopy(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.AvailableCopies != 1 {
		t.Fatalf("expected availableCopies=1, got %d", b.AvailableCopies)
	}
}

func TestLoanOneCopy_RejectsWhenNoCopiesAvailable(t *testing.T) {
	// INV-001: availability never goes below zero.
	b, _ := catalog.NewBook("Title", "Author", "isbn", "Category", 2020, 1)
	_ = b.LoanOneCopy()
	if err := b.LoanOneCopy(); err != catalog.ErrNoCopiesAvailable {
		t.Fatalf("expected ErrNoCopiesAvailable, got %v", err)
	}
}

func TestReturnOneCopy_RejectsWhenWouldExceedTotalCopies(t *testing.T) {
	// INV-001: availability never exceeds total stock.
	b, _ := catalog.NewBook("Title", "Author", "isbn", "Category", 2020, 1)
	if err := b.ReturnOneCopy(); err != catalog.ErrWouldExceedTotalCopies {
		t.Fatalf("expected ErrWouldExceedTotalCopies, got %v", err)
	}
}

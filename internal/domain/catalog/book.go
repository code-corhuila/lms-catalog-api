// Package catalog implements the Catalog bounded context
// (library-docs/02-domain/domain-map.md): the book catalog and copy-count availability.
package catalog

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrNoCopiesAvailable — INV-001: a loan is rejected if it would make availableCopies negative.
var ErrNoCopiesAvailable = errors.New("no copies available")

// ErrWouldExceedTotalCopies — INV-001: a return is rejected if it would make
// availableCopies exceed totalCopies.
var ErrWouldExceedTotalCopies = errors.New("return would exceed total copies")

// ErrBookNotFound is the port-level "not found" error.
var ErrBookNotFound = errors.New("book not found")

// Book is the Catalog bounded context's Aggregate Root.
// There is no separate Copy entity in v1 — see
// library-docs/02-domain/entities-and-rules.md, Entity: Book, modeling note.
type Book struct {
	ID              string
	Title           string
	Author          string
	ISBN            string
	Category        string
	Year            int
	TotalCopies     int
	AvailableCopies int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NewBook registers a new book — INV-003: must be registered with at least one copy.
// Availability is initialized equal to totalCopies (HU-04).
func NewBook(title, author, isbn, category string, year, totalCopies int) (*Book, error) {
	if title == "" {
		return nil, errors.New("title must not be empty")
	}
	if author == "" {
		return nil, errors.New("author must not be empty")
	}
	if totalCopies < 1 {
		return nil, errors.New("totalCopies must be at least 1")
	}

	now := time.Now().UTC()
	return &Book{
		ID:              uuid.NewString(),
		Title:           title,
		Author:          author,
		ISBN:            isbn,
		Category:        category,
		Year:            year,
		TotalCopies:     totalCopies,
		AvailableCopies: totalCopies,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

// LoanOneCopy decrements availability by one — called by the Circulation domain
// service when a loan is registered (HU-06).
func (b *Book) LoanOneCopy() error {
	if b.AvailableCopies <= 0 {
		return ErrNoCopiesAvailable
	}
	b.AvailableCopies--
	b.UpdatedAt = time.Now().UTC()
	return nil
}

// ReturnOneCopy increments availability by one — called when a return is registered (HU-07).
func (b *Book) ReturnOneCopy() error {
	if b.AvailableCopies >= b.TotalCopies {
		return ErrWouldExceedTotalCopies
	}
	b.AvailableCopies++
	b.UpdatedAt = time.Now().UTC()
	return nil
}

// Update applies an edit (HU-09) — ISBN is intentionally not editable through this
// method; see library-docs/07-api/contracts/openapi/library-api.yaml (UpdateBookRequest).
func (b *Book) Update(title, author, category string, year int) error {
	if title == "" {
		return errors.New("title must not be empty")
	}
	if author == "" {
		return errors.New("author must not be empty")
	}
	b.Title = title
	b.Author = author
	b.Category = category
	b.Year = year
	b.UpdatedAt = time.Now().UTC()
	return nil
}

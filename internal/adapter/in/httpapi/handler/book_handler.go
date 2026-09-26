package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	in "github.com/code-corhuila/lms-catalog-api/internal/application/port/in"
	"github.com/code-corhuila/lms-catalog-api/internal/application/usecase"
	"github.com/code-corhuila/lms-catalog-api/internal/domain/catalog"
	"github.com/code-corhuila/lms-catalog-api/internal/adapter/in/httpapi/middleware"
	"github.com/code-corhuila/lms-catalog-api/internal/adapter/in/httpapi/response"
)

// BookHandler implements the /books endpoints (HU-04). It depends on
// application/port/in interfaces, not the concrete usecase.X structs
// (rules/2-anexos/C-api-hexagonal.md).
type BookHandler struct {
	createBook     in.CreateBookUseCase
	loanBookCopy   in.LoanBookCopyUseCase
	returnBookCopy in.ReturnBookCopyUseCase
}

func NewBookHandler(createBook in.CreateBookUseCase, loanBookCopy in.LoanBookCopyUseCase, returnBookCopy in.ReturnBookCopyUseCase) *BookHandler {
	return &BookHandler{createBook: createBook, loanBookCopy: loanBookCopy, returnBookCopy: returnBookCopy}
}

type createBookRequest struct {
	Title       string `json:"title"`
	Author      string `json:"author"`
	ISBN        string `json:"isbn"`
	Category    string `json:"category"`
	Year        int    `json:"year"`
	TotalCopies int    `json:"totalCopies"`
}

const timeFormat = "2006-01-02T15:04:05Z07:00"

type bookResponse struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Author          string `json:"author"`
	ISBN            string `json:"isbn"`
	Category        string `json:"category"`
	Year            int    `json:"year"`
	TotalCopies     int    `json:"totalCopies"`
	AvailableCopies int    `json:"availableCopies"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

func toBookResponse(b *catalog.Book) bookResponse {
	return bookResponse{
		ID:              b.ID,
		Title:           b.Title,
		Author:          b.Author,
		ISBN:            b.ISBN,
		Category:        b.Category,
		Year:            b.Year,
		TotalCopies:     b.TotalCopies,
		AvailableCopies: b.AvailableCopies,
		CreatedAt:       b.CreatedAt.Format(timeFormat),
		UpdatedAt:       b.UpdatedAt.Format(timeFormat),
	}
}

// Create — POST /books (HU-04, FR-007, FR-008). Idempotent by the
// Idempotency-Key header (rules/2-anexos/C-api-hexagonal.md, numeral 5.3.8): a
// retried request with the same key returns the original book and 200, not a
// second book and 201.
func (h *BookHandler) Create(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.FromContext(r.Context())
	idempotencyKey := r.Header.Get("Idempotency-Key")

	var req createBookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", traceID)
		return
	}

	book, replayed, err := h.createBook.Execute(r.Context(), req.Title, req.Author, req.ISBN, req.Category, req.Year, req.TotalCopies, idempotencyKey)
	switch {
	case errors.Is(err, usecase.ErrISBNAlreadyExists):
		response.Error(w, http.StatusConflict, "ISBN_ALREADY_EXISTS", "This ISBN is already registered", traceID)
		return
	case err != nil:
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), traceID)
		return
	}

	if replayed {
		response.JSON(w, http.StatusOK, toBookResponse(book))
		return
	}

	w.Header().Set("Location", fmt.Sprintf("/api/v1/books/%s", book.ID))
	response.JSON(w, http.StatusCreated, toBookResponse(book))
}

// LoanCopy — POST /books/{id}/loan-copy. Called by circulation-service when
// registering a loan (HU-06); not part of the public UI-facing contract.
func (h *BookHandler) LoanCopy(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.FromContext(r.Context())
	id := chi.URLParam(r, "id")

	book, err := h.loanBookCopy.Execute(r.Context(), id)
	switch {
	case errors.Is(err, catalog.ErrBookNotFound):
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Book not found", traceID)
		return
	case errors.Is(err, catalog.ErrNoCopiesAvailable):
		response.Error(w, http.StatusConflict, "NO_COPIES_AVAILABLE", "There are no available copies of this book", traceID)
		return
	case err != nil:
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", traceID)
		return
	}

	response.JSON(w, http.StatusOK, toBookResponse(book))
}

// ReturnCopy — POST /books/{id}/return-copy. Called by circulation-service
// when registering a return (HU-07).
func (h *BookHandler) ReturnCopy(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.FromContext(r.Context())
	id := chi.URLParam(r, "id")

	book, err := h.returnBookCopy.Execute(r.Context(), id)
	switch {
	case errors.Is(err, catalog.ErrBookNotFound):
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Book not found", traceID)
		return
	case err != nil:
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", traceID)
		return
	}

	response.JSON(w, http.StatusOK, toBookResponse(book))
}

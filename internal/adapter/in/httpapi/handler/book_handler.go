package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/code-corhuila/lms-catalog-api/internal/adapter/in/httpapi/middleware"
	"github.com/code-corhuila/lms-catalog-api/internal/adapter/in/httpapi/response"
	in "github.com/code-corhuila/lms-catalog-api/internal/application/port/in"
	"github.com/code-corhuila/lms-catalog-api/internal/application/usecase"
	"github.com/code-corhuila/lms-catalog-api/internal/domain/catalog"
)

// BookHandler implements the /books endpoints (HU-04). It depends on
// application/port/in interfaces, not the concrete usecase.X structs
// (rules/2-anexos/C-api-hexagonal.md).
type BookHandler struct {
	createBook     in.CreateBookUseCase
	updateBook     in.UpdateBookUseCase
	loanBookCopy   in.LoanBookCopyUseCase
	returnBookCopy in.ReturnBookCopyUseCase
	searchBooks    in.SearchBooksUseCase
}

func NewBookHandler(createBook in.CreateBookUseCase, updateBook in.UpdateBookUseCase, loanBookCopy in.LoanBookCopyUseCase, returnBookCopy in.ReturnBookCopyUseCase, searchBooks in.SearchBooksUseCase) *BookHandler {
	return &BookHandler{createBook: createBook, updateBook: updateBook, loanBookCopy: loanBookCopy, returnBookCopy: returnBookCopy, searchBooks: searchBooks}
}

func clampPage(page int) int {
	if page < 1 {
		return 1
	}
	return page
}

func clampLimit(limit int) int {
	if limit < 1 || limit > 100 {
		return 20
	}
	return limit
}

type createBookRequest struct {
	Title       string `json:"title"`
	Author      string `json:"author"`
	ISBN        string `json:"isbn"`
	Category    string `json:"category"`
	Year        int    `json:"year"`
	TotalCopies int    `json:"totalCopies"`
}

// updateBookRequest has no isbn on purpose — see usecase.UpdateBook's doc comment.
type updateBookRequest struct {
	Title    string `json:"title"`
	Author   string `json:"author"`
	Category string `json:"category"`
	Year     int    `json:"year"`
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

// Update — PATCH /books/{id} (HU-09, FR-011, FR-012). Title and author are
// checked here first so lms-catalog-portal can flag each field from details
// (rules/2-anexos/C-api-hexagonal.md, "Validación en la frontera").
func (h *BookHandler) Update(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.FromContext(r.Context())
	id := chi.URLParam(r, "id")

	var req updateBookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", traceID)
		return
	}

	var details []response.FieldDetail
	if strings.TrimSpace(req.Title) == "" {
		details = append(details, response.FieldDetail{Field: "title", Message: "title must not be empty"})
	}
	if strings.TrimSpace(req.Author) == "" {
		details = append(details, response.FieldDetail{Field: "author", Message: "author must not be empty"})
	}
	if len(details) > 0 {
		response.ValidationError(w, traceID, details...)
		return
	}

	book, err := h.updateBook.Execute(r.Context(), id, req.Title, req.Author, req.Category, req.Year)
	switch {
	case errors.Is(err, catalog.ErrBookNotFound):
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Book not found", traceID)
		return
	case err != nil:
		// Book.Update's only rules (title/author) were already checked above,
		// so whatever is left is infrastructure, not the request.
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", traceID)
		return
	}

	response.JSON(w, http.StatusOK, toBookResponse(book))
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
		// 422, not 409 — a domain rule (INV-001) forbids the loan, the
		// request itself doesn't collide with existing state
		// (rules/2-anexos/C-api-hexagonal.md, numeral 5.3.11 / D-G08).
		response.Error(w, http.StatusUnprocessableEntity, "NO_COPIES_AVAILABLE", "There are no available copies of this book", traceID)
		return
	case err != nil:
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", traceID)
		return
	}

	response.JSON(w, http.StatusOK, toBookResponse(book))
}

// List — GET /books (HU-04, search half). meta carries the full
// {total, page, limit, totalPages} envelope, echoing the page/limit actually
// served after clamping (rules/2-anexos/C-api-hexagonal.md, "Listados").
func (h *BookHandler) List(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.FromContext(r.Context())

	rawPage, _ := strconv.Atoi(r.URL.Query().Get("page"))
	rawLimit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	search := r.URL.Query().Get("search")
	category := r.URL.Query().Get("category")

	page := clampPage(rawPage)
	limit := clampLimit(rawLimit)

	books, total, err := h.searchBooks.Execute(r.Context(), search, category, page, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", traceID)
		return
	}

	items := make([]bookResponse, 0, len(books))
	for _, b := range books {
		items = append(items, toBookResponse(b))
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": map[string]any{
			"total":      total,
			"page":       page,
			"limit":      limit,
			"totalPages": totalPages,
		},
	})
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

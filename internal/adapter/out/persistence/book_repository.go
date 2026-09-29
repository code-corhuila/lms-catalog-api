package persistence

import (
	"context"
	"errors"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/code-corhuila/lms-catalog-api/internal/domain/catalog"
)

// BookRepository implements catalog.BookRepository against PostgreSQL — the
// only code allowed to touch the `catalog.book` table
// (library-docs/09-microservices/service-boundary-rules.md).
type BookRepository struct {
	db *pgxpool.Pool
}

func NewBookRepository(db *pgxpool.Pool) *BookRepository {
	return &BookRepository{db: db}
}

const bookColumns = `id, title, author, isbn, category, year, total_copies, available_copies, created_at, updated_at`

func scanBook(row pgx.Row) (*catalog.Book, error) {
	var b catalog.Book
	if err := row.Scan(&b.ID, &b.Title, &b.Author, &b.ISBN, &b.Category, &b.Year, &b.TotalCopies, &b.AvailableCopies, &b.CreatedAt, &b.UpdatedAt); err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *BookRepository) FindByID(ctx context.Context, id string) (*catalog.Book, error) {
	// A malformed id (not a UUID at all) is "not found", not a server error —
	// without this, postgres's own "invalid input syntax for type uuid"
	// surfaces as a raw driver error the handler can't distinguish from a
	// real failure, and the caller gets a 500 for what's really a 404.
	if _, err := uuid.Parse(id); err != nil {
		return nil, catalog.ErrBookNotFound
	}

	query := `SELECT ` + bookColumns + ` FROM catalog.book WHERE id = $1`
	b, err := scanBook(r.db.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, catalog.ErrBookNotFound
	}
	return b, err
}

func (r *BookRepository) FindByISBN(ctx context.Context, isbn string) (*catalog.Book, error) {
	query := `SELECT ` + bookColumns + ` FROM catalog.book WHERE isbn = $1`
	b, err := scanBook(r.db.QueryRow(ctx, query, isbn))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, catalog.ErrBookNotFound
	}
	return b, err
}

func (r *BookRepository) Search(ctx context.Context, q, category string, page, limit int) ([]*catalog.Book, int, error) {
	offset := (page - 1) * limit
	query := `SELECT ` + bookColumns + ` FROM catalog.book WHERE 1=1`
	countQuery := `SELECT count(*) FROM catalog.book WHERE 1=1`
	var args []any

	if q != "" {
		args = append(args, "%"+q+"%")
		clause := " AND (title ILIKE $" + strconv.Itoa(len(args)) + " OR author ILIKE $" + strconv.Itoa(len(args)) + " OR isbn ILIKE $" + strconv.Itoa(len(args)) + ")"
		query += clause
		countQuery += clause
	}
	if category != "" {
		args = append(args, category)
		clause := " AND category = $" + strconv.Itoa(len(args))
		query += clause
		countQuery += clause
	}

	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limitArgs := append(append([]any{}, args...), limit, offset)
	query += " ORDER BY title LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)

	rows, err := r.db.Query(ctx, query, limitArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var books []*catalog.Book
	for rows.Next() {
		b, err := scanBook(rows)
		if err != nil {
			return nil, 0, err
		}
		books = append(books, b)
	}

	return books, total, nil
}

func (r *BookRepository) Save(ctx context.Context, b *catalog.Book) error {
	const query = `
		INSERT INTO catalog.book (id, title, author, isbn, category, year, total_copies, available_copies, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title,
			author = EXCLUDED.author,
			isbn = EXCLUDED.isbn,
			category = EXCLUDED.category,
			year = EXCLUDED.year,
			total_copies = EXCLUDED.total_copies,
			available_copies = EXCLUDED.available_copies,
			updated_at = EXCLUDED.updated_at`

	_, err := r.db.Exec(ctx, query, b.ID, b.Title, b.Author, b.ISBN, b.Category, b.Year, b.TotalCopies, b.AvailableCopies, b.CreatedAt, b.UpdatedAt)
	return err
}

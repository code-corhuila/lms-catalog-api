package persistence

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// IdempotencyStore implements out.IdempotencyStore against PostgreSQL's
// catalog.idempotency_key table (lms-catalog-db). Durable — replaces the
// provisional in-memory adapter now that the table exists.
type IdempotencyStore struct {
	db *pgxpool.Pool
}

func NewIdempotencyStore(db *pgxpool.Pool) *IdempotencyStore {
	return &IdempotencyStore{db: db}
}

func (s *IdempotencyStore) Get(ctx context.Context, key string) (bookID string, found bool, err error) {
	const query = `SELECT book_id FROM catalog.idempotency_key WHERE key = $1`
	err = s.db.QueryRow(ctx, query, key).Scan(&bookID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return bookID, true, nil
}

func (s *IdempotencyStore) Save(ctx context.Context, key, bookID string) error {
	const query = `
		INSERT INTO catalog.idempotency_key (key, book_id)
		VALUES ($1, $2)
		ON CONFLICT (key) DO NOTHING`
	_, err := s.db.Exec(ctx, query, key, bookID)
	return err
}

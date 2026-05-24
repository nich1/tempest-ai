// Package users is a thin wrapper around the generated sqlc.Queries
// that adds error translation (no user-enumeration leak).
package users

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nich1/tempest-ai/internal/db/sqlc"
	"github.com/nich1/tempest-ai/internal/models"
)

// ErrNotFound is returned when a user lookup yields no row. Callers
// translating this into auth flows should still return generic
// "invalid credentials" errors externally.
var ErrNotFound = errors.New("user not found")

// Repository wraps the generated query type with a connection pool.
type Repository struct {
	q *sqlc.Queries
}

// New builds a Repository.
func New(pool *pgxpool.Pool) *Repository {
	return &Repository{q: sqlc.New(pool)}
}

// Create inserts a new user. The password should already be hashed.
func (r *Repository) Create(ctx context.Context, email, passwordHash string) (sqlc.User, error) {
	return r.q.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        email,
		PasswordHash: passwordHash,
	})
}

// GetByEmail returns ErrNotFound if no user matches.
func (r *Repository) GetByEmail(ctx context.Context, email string) (sqlc.User, error) {
	u, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.User{}, ErrNotFound
		}
		return sqlc.User{}, err
	}
	return u, nil
}

// GetByID returns ErrNotFound if no user matches.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (sqlc.User, error) {
	u, err := r.q.GetUserByID(ctx, models.PgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.User{}, ErrNotFound
		}
		return sqlc.User{}, err
	}
	return u, nil
}

package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nich1/tempest-ai/internal/db/sqlc"
	"github.com/nich1/tempest-ai/internal/models"
)

// Session is the in-memory shape returned from Lookup. It joins user data
// onto the session so callers don't need a second query.
type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ExpiresAt time.Time
	UserEmail string
}

// Sessions is the service that manages session lifecycle.
type Sessions struct {
	q   *sqlc.Queries
	ttl time.Duration
}

// NewSessions builds the service. ttl is the lifetime of a new session.
func NewSessions(pool *pgxpool.Pool, ttl time.Duration) *Sessions {
	return &Sessions{q: sqlc.New(pool), ttl: ttl}
}

// Create issues a new session for the given user and returns the raw
// token (which goes into the cookie) plus the persisted session.
func (s *Sessions) Create(ctx context.Context, userID uuid.UUID) (rawToken string, sess Session, err error) {
	raw, hash, err := GenerateToken()
	if err != nil {
		return "", Session{}, err
	}
	expires := time.Now().Add(s.ttl)
	row, err := s.q.CreateSession(ctx, sqlc.CreateSessionParams{
		UserID:    models.PgUUID(userID),
		TokenHash: hash,
		ExpiresAt: pgtype.Timestamptz{Time: expires, Valid: true},
	})
	if err != nil {
		return "", Session{}, err
	}
	return raw, Session{
		ID:        uuid.UUID(row.ID.Bytes),
		UserID:    userID,
		ExpiresAt: row.ExpiresAt.Time,
	}, nil
}

// Lookup resolves a raw cookie token to its Session. Returns
// ErrSessionNotFound or ErrSessionExpired on miss.
func (s *Sessions) Lookup(ctx context.Context, rawToken string) (Session, error) {
	if rawToken == "" {
		return Session{}, ErrSessionNotFound
	}
	hash := HashToken(rawToken)
	row, err := s.q.GetSessionByTokenHash(ctx, hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Session{}, ErrSessionNotFound
		}
		return Session{}, err
	}
	// Best-effort touch; ignore errors so login doesn't fail on a transient
	// write hiccup against an otherwise-valid session.
	_ = s.q.TouchSession(ctx, hash)
	return Session{
		ID:        uuid.UUID(row.ID.Bytes),
		UserID:    uuid.UUID(row.UserID.Bytes),
		ExpiresAt: row.ExpiresAt.Time,
		UserEmail: row.UserEmail,
	}, nil
}

// Revoke deletes the session row that matches the raw token. No-op if
// the token doesn't match anything.
func (s *Sessions) Revoke(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}
	return s.q.DeleteSessionByTokenHash(ctx, HashToken(rawToken))
}

// RevokeAllForUser deletes every session belonging to the user. Useful
// for "log out everywhere" or post-password-change cleanup.
func (s *Sessions) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	return s.q.DeleteSessionsForUser(ctx, models.PgUUID(userID))
}

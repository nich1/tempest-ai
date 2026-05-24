// Package jobs is a thin repository over the generated sqlc.Queries that
// adds business shape (DTO mapping, ErrNotFound translation).
package jobs

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nich1/tempest-ai/internal/db/sqlc"
	"github.com/nich1/tempest-ai/internal/models"
)

// ErrNotFound is returned when a job lookup yields no row.
var ErrNotFound = errors.New("job not found")

// CreateParams bundles the inputs to Create. We use a typed struct
// instead of positional args because there are a lot of fields.
type CreateParams struct {
	UserID              uuid.UUID
	InputSchema         json.RawMessage
	OutputSchema        json.RawMessage
	Inputs              json.RawMessage
	Prompt              string
	SystemPrompt        string
	FileBlobKey         string
	FileBlobSize        int64
	FileBlobContentType string
	Provider            string
}

// Repository persists jobs.
type Repository struct {
	q *sqlc.Queries
}

// New builds a Repository.
func New(pool *pgxpool.Pool) *Repository {
	return &Repository{q: sqlc.New(pool)}
}

// Create inserts a PENDING job.
func (r *Repository) Create(ctx context.Context, p CreateParams) (sqlc.Job, error) {
	params := sqlc.CreateJobParams{
		UserID:       models.PgUUID(p.UserID),
		InputSchema:  p.InputSchema,
		OutputSchema: p.OutputSchema,
		Inputs:       p.Inputs,
		Prompt:       p.Prompt,
		Provider:     p.Provider,
	}
	if p.SystemPrompt != "" {
		params.SystemPrompt = &p.SystemPrompt
	}
	if p.FileBlobKey != "" {
		params.FileBlobKey = &p.FileBlobKey
	}
	if p.FileBlobSize > 0 {
		params.FileBlobSize = &p.FileBlobSize
	}
	if p.FileBlobContentType != "" {
		params.FileBlobContentType = &p.FileBlobContentType
	}
	return r.q.CreateJob(ctx, params)
}

// GetByID returns ErrNotFound if no job matches.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (sqlc.Job, error) {
	j, err := r.q.GetJobByID(ctx, models.PgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.Job{}, ErrNotFound
		}
		return sqlc.Job{}, err
	}
	return j, nil
}

// GetByIDForUser scopes the lookup to a specific user.
func (r *Repository) GetByIDForUser(ctx context.Context, id, userID uuid.UUID) (sqlc.Job, error) {
	j, err := r.q.GetJobByIDForUser(ctx, sqlc.GetJobByIDForUserParams{
		ID:     models.PgUUID(id),
		UserID: models.PgUUID(userID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.Job{}, ErrNotFound
		}
		return sqlc.Job{}, err
	}
	return j, nil
}

// ListForUser returns the user's recent jobs.
func (r *Repository) ListForUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]sqlc.Job, error) {
	return r.q.ListJobsForUser(ctx, sqlc.ListJobsForUserParams{
		UserID: models.PgUUID(userID),
		Limit:  int32(limit),
		Offset: int32(offset),
	})
}

// CountForUser returns the user's total job count.
func (r *Repository) CountForUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	return r.q.CountJobsForUser(ctx, models.PgUUID(userID))
}

// MarkProcessing transitions a job to PROCESSING and increments attempt.
func (r *Repository) MarkProcessing(ctx context.Context, id uuid.UUID) (sqlc.Job, error) {
	return r.q.MarkJobProcessing(ctx, models.PgUUID(id))
}

// MarkCompleted stores the validated output and transitions to COMPLETED.
func (r *Repository) MarkCompleted(ctx context.Context, id uuid.UUID, output json.RawMessage) (sqlc.Job, error) {
	return r.q.MarkJobCompleted(ctx, sqlc.MarkJobCompletedParams{
		ID:     models.PgUUID(id),
		Output: &output,
	})
}

// MarkFailed transitions to FAILED and records the user-visible error.
func (r *Repository) MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) (sqlc.Job, error) {
	return r.q.MarkJobFailed(ctx, sqlc.MarkJobFailedParams{
		ID:           models.PgUUID(id),
		ErrorMessage: &errMsg,
	})
}

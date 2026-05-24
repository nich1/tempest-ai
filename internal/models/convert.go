package models

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/nich1/tempest-ai/internal/db/sqlc"
)

// UserFromRow converts a sqlc User row to its API DTO.
func UserFromRow(u sqlc.User) UserDTO {
	return UserDTO{
		ID:        uuidFromPg(u.ID),
		Email:     u.Email,
		CreatedAt: u.CreatedAt.Time,
	}
}

// JobFromRow converts a sqlc Job row to its API DTO.
func JobFromRow(j sqlc.Job) JobDTO {
	d := JobDTO{
		ID:           uuidFromPg(j.ID),
		Status:       JobStatus(j.Status),
		InputSchema:  j.InputSchema,
		OutputSchema: j.OutputSchema,
		Inputs:       j.Inputs,
		Prompt:       j.Prompt,
		Provider:     j.Provider,
		Attempt:      j.Attempt,
		CreatedAt:    j.CreatedAt.Time,
		UpdatedAt:    j.UpdatedAt.Time,
	}
	if j.SystemPrompt != nil {
		d.SystemPrompt = *j.SystemPrompt
	}
	if j.FileBlobKey != nil {
		d.FileBlobKey = *j.FileBlobKey
	}
	if j.FileBlobSize != nil {
		d.FileBlobSize = *j.FileBlobSize
	}
	if j.FileBlobContentType != nil {
		d.FileBlobContentType = *j.FileBlobContentType
	}
	if j.Output != nil {
		d.Output = *j.Output
	}
	if j.ErrorMessage != nil {
		d.ErrorMessage = *j.ErrorMessage
	}
	if j.StartedAt.Valid {
		t := j.StartedAt.Time
		d.StartedAt = &t
	}
	if j.CompletedAt.Valid {
		t := j.CompletedAt.Time
		d.CompletedAt = &t
	}
	return d
}

// PgUUID wraps a uuid.UUID into a pgtype.UUID.
func PgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func uuidFromPg(p pgtype.UUID) uuid.UUID {
	if !p.Valid {
		return uuid.Nil
	}
	return uuid.UUID(p.Bytes)
}

package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// JobStatus mirrors the DB CHECK constraint.
type JobStatus string

const (
	JobStatusPending    JobStatus = "PENDING"
	JobStatusProcessing JobStatus = "PROCESSING"
	JobStatusCompleted  JobStatus = "COMPLETED"
	JobStatusFailed     JobStatus = "FAILED"
)

// IsTerminal reports whether the status will never change again.
func (s JobStatus) IsTerminal() bool {
	return s == JobStatusCompleted || s == JobStatusFailed
}

// JobDTO is the wire shape for jobs returned to clients.
type JobDTO struct {
	ID                  uuid.UUID       `json:"id"`
	Status              JobStatus       `json:"status"`
	InputSchema         json.RawMessage `json:"input_schema"`
	OutputSchema        json.RawMessage `json:"output_schema"`
	Inputs              json.RawMessage `json:"inputs"`
	Prompt              string          `json:"prompt"`
	SystemPrompt        string          `json:"system_prompt,omitempty"`
	FileBlobKey         string          `json:"file_blob_key,omitempty"`
	FileBlobSize        int64           `json:"file_blob_size,omitempty"`
	FileBlobContentType string          `json:"file_blob_content_type,omitempty"`
	Output              json.RawMessage `json:"output,omitempty"`
	ErrorMessage        string          `json:"error_message,omitempty"`
	Provider            string          `json:"provider"`
	Attempt             int32           `json:"attempt"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
	StartedAt           *time.Time      `json:"started_at,omitempty"`
	CompletedAt         *time.Time      `json:"completed_at,omitempty"`
}

// CreateJobRequest is the body of POST /jobs.
type CreateJobRequest struct {
	InputSchema  json.RawMessage `json:"input_schema" binding:"required"`
	OutputSchema json.RawMessage `json:"output_schema" binding:"required"`
	Inputs       json.RawMessage `json:"inputs" binding:"required"`
	Prompt       string          `json:"prompt" binding:"required"`
	SystemPrompt string          `json:"system_prompt,omitempty"`
	FileBlobKey  string          `json:"file_blob_key,omitempty"`
	// Provider format: "<provider>:<model>". E.g. "ollama:llama3:8b" or
	// "anthropic:claude-3-5-sonnet-20241022". If empty, server substitutes
	// LLM_PROVIDER_DEFAULT:LLM_MODEL_DEFAULT.
	Provider string `json:"provider,omitempty"`
}

// JobListResponse is GET /jobs.
type JobListResponse struct {
	Jobs       []JobDTO `json:"jobs"`
	TotalCount int64    `json:"total_count"`
	Limit      int      `json:"limit"`
	Offset     int      `json:"offset"`
}

// FileUploadURLRequest asks the API to mint a presigned PUT URL.
type FileUploadURLRequest struct {
	ContentType string `json:"content_type" binding:"required"`
	SizeBytes   int64  `json:"size_bytes" binding:"required,min=1"`
}

// FileUploadURLResponse carries the URL plus the key the client should
// reference in its subsequent POST /jobs call.
type FileUploadURLResponse struct {
	UploadURL string `json:"upload_url"`
	BlobKey   string `json:"blob_key"`
	MaxSize   int64  `json:"max_size_bytes"`
	ExpiresIn int    `json:"expires_in_seconds"`
}

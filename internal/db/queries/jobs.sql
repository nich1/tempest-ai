-- name: CreateJob :one
INSERT INTO jobs (
    user_id,
    status,
    input_schema,
    output_schema,
    inputs,
    prompt,
    system_prompt,
    file_blob_key,
    file_blob_size,
    file_blob_content_type,
    provider
)
VALUES ($1, 'PENDING', $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetJobByID :one
SELECT * FROM jobs
WHERE id = $1
LIMIT 1;

-- name: GetJobByIDForUser :one
SELECT * FROM jobs
WHERE id = $1 AND user_id = $2
LIMIT 1;

-- name: ListJobsForUser :many
SELECT * FROM jobs
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: MarkJobProcessing :one
UPDATE jobs
SET status = 'PROCESSING',
    started_at = now(),
    updated_at = now(),
    attempt = attempt + 1
WHERE id = $1
RETURNING *;

-- name: MarkJobCompleted :one
UPDATE jobs
SET status = 'COMPLETED',
    output = $2,
    completed_at = now(),
    updated_at = now(),
    error_message = NULL
WHERE id = $1
RETURNING *;

-- name: MarkJobFailed :one
UPDATE jobs
SET status = 'FAILED',
    error_message = $2,
    completed_at = now(),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: CountJobsForUser :one
SELECT COUNT(*) FROM jobs WHERE user_id = $1;

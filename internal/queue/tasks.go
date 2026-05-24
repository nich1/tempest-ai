// Package queue is the Asynq plumbing layer. Both apps/api (enqueue side)
// and apps/consumers (consume side) import it.
//
// Task names are constants here so producer and consumer can never disagree
// on the queue key. ProcessJobPayload carries the request_id across the
// queue boundary so consumer logs can be correlated back to the API request.
package queue

import (
	"github.com/google/uuid"
)

// Task type constants. Keep these in sync with the asynq.Server queue
// registration in server.go.
const (
	// TypeProcessJob runs a single LLM-extraction job (the only task type
	// in v1). The payload references a row in the jobs table by ID; the
	// consumer fetches all the actual data from Postgres on receive.
	TypeProcessJob = "job:process"
)

// Queue names used for asynq.Config.Queues priority weighting.
const (
	QueueCritical = "critical"
	QueueDefault  = "default"
	QueueBulk     = "bulk"
)

// ProcessJobPayload is the body of a TypeProcessJob task.
//
// We intentionally don't ship the schemas / inputs / prompts in the
// payload - they live in Postgres and the consumer fetches them by ID.
// Keeping the payload tiny keeps Redis memory usage tiny and avoids
// drift between the queued data and the latest DB state.
type ProcessJobPayload struct {
	JobID     uuid.UUID `json:"job_id"`
	RequestID string    `json:"request_id"`
}

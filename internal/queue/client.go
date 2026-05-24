package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/nich1/tempest-ai/internal/config"
)

// Client is the enqueue-side wrapper. The API holds one of these.
type Client struct {
	c *asynq.Client
}

// NewClient connects to Redis with the given config.
func NewClient(cfg config.Redis) *Client {
	return &Client{c: asynq.NewClient(asynq.RedisClientOpt{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})}
}

// Close releases the underlying Redis connection pool.
func (c *Client) Close() error { return c.c.Close() }

// EnqueueProcessJob schedules a TypeProcessJob task. The request_id rides
// with the task into Redis so the consumer can correlate its logs back
// to the originating HTTP request.
//
// Uses asynq.Unique to dedupe accidental double-submits within a short
// window; no harm done if the task expires from the unique-set.
func (c *Client) EnqueueProcessJob(ctx context.Context, jobID uuid.UUID, requestID string) (taskID string, err error) {
	payload, err := json.Marshal(ProcessJobPayload{JobID: jobID, RequestID: requestID})
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}
	task := asynq.NewTask(TypeProcessJob, payload)
	info, err := c.c.EnqueueContext(ctx, task,
		asynq.Queue(QueueDefault),
		asynq.MaxRetry(5),
		asynq.Unique(60*time.Second), // dedupe accidental double-submit within 60s
	)
	if err != nil {
		return "", fmt.Errorf("enqueue: %w", err)
	}
	return info.ID, nil
}

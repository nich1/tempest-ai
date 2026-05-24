package queue

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"

	"github.com/nich1/tempest-ai/internal/logging"
)

// LoggingMiddleware extracts the request_id and job_id from the task
// payload and attaches them (plus the asynq task_id) to a scoped logger
// stored on ctx. Downstream handlers should use logging.FromContext(ctx).
//
// It also emits start/end log lines so a single grep on request_id
// returns: HTTP request -> enqueue -> task.received -> task.completed.
func LoggingMiddleware(base *slog.Logger) asynq.MiddlewareFunc {
	return func(next asynq.Handler) asynq.Handler {
		return asynq.HandlerFunc(func(ctx context.Context, t *asynq.Task) error {
			start := time.Now()
			logger := base
			ctx = logging.WithLogger(ctx, logger)

			taskID, _ := asynq.GetTaskID(ctx)
			ctx = logging.WithTaskID(ctx, taskID)

			var payload ProcessJobPayload
			if err := json.Unmarshal(t.Payload(), &payload); err == nil {
				if payload.RequestID != "" {
					ctx = logging.WithRequestID(ctx, payload.RequestID)
				}
				if payload.JobID.String() != "" {
					ctx = logging.WithJobID(ctx, payload.JobID.String())
				}
			}

			retryCount, _ := asynq.GetRetryCount(ctx)
			maxRetry, _ := asynq.GetMaxRetry(ctx)

			logging.FromContext(ctx).Info("task.received",
				slog.String("task_type", t.Type()),
				slog.Int("retry", retryCount),
				slog.Int("max_retry", maxRetry),
			)

			err := next.ProcessTask(ctx, t)

			lvl := slog.LevelInfo
			event := "task.completed"
			if err != nil {
				lvl = slog.LevelError
				event = "task.failed"
			}
			logging.FromContext(ctx).LogAttrs(ctx, lvl, event,
				slog.String("task_type", t.Type()),
				slog.Duration("latency", time.Since(start)),
				slog.Any("error", err),
			)
			return err
		})
	}
}

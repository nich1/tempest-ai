package queue

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/hibiken/asynq"

	"github.com/nich1/tempest-ai/internal/config"
)

// NewServer builds an asynq.Server with queue priorities and retry policy
// from config.
func NewServer(cfg config.Redis, w config.Worker) *asynq.Server {
	return asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     cfg.Addr,
			Password: cfg.Password,
			DB:       cfg.DB,
		},
		asynq.Config{
			Concurrency:     w.Concurrency,
			Queues:          parseQueues(w.QueuePriorities),
			StrictPriority:  w.StrictPriority,
			ShutdownTimeout: w.ShutdownTimeout,
			RetryDelayFunc:  exponentialBackoff,
		},
	)
}

// NewInspector exposes Asynq's read-only inspection API for the API's
// /health endpoint and any operational tooling.
func NewInspector(cfg config.Redis) *asynq.Inspector {
	return asynq.NewInspector(asynq.RedisClientOpt{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
}

// parseQueues turns "critical:6,default:3,bulk:1" into Asynq's queue map.
func parseQueues(spec string) map[string]int {
	out := map[string]int{}
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := strings.Index(part, ":")
		if idx <= 0 {
			continue
		}
		name := strings.TrimSpace(part[:idx])
		weight, err := strconv.Atoi(strings.TrimSpace(part[idx+1:]))
		if err != nil || weight < 1 {
			continue
		}
		out[name] = weight
	}
	if len(out) == 0 {
		out[QueueDefault] = 1
	}
	return out
}

// exponentialBackoff: 30s, 60s, 2m, 4m, 8m, 16m, ... capped at 1h.
func exponentialBackoff(n int, _ error, _ *asynq.Task) time.Duration {
	d := 30 * time.Second
	for i := 0; i < n; i++ {
		d *= 2
		if d > time.Hour {
			return time.Hour
		}
	}
	return d
}

// ServerHealth reports a transient liveness probe against the broker.
func ServerHealth(ctx context.Context, ins *asynq.Inspector) error {
	_, err := ins.Queues()
	return err
}

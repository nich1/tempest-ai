// Package main is the consumer entrypoint.
//
// Connects to Redis, Postgres, and MinIO, registers the LLMJobProcessor
// for queue.TypeProcessJob, and runs the Asynq server until SIGTERM.
//
// Horizontal scaling: just run more instances. Asynq's broker is Redis
// so consumers compete for work cleanly without coordination.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"

	"github.com/nich1/tempest-ai/internal/config"
	"github.com/nich1/tempest-ai/internal/db"
	"github.com/nich1/tempest-ai/internal/jobs"
	"github.com/nich1/tempest-ai/internal/llm"
	"github.com/nich1/tempest-ai/internal/logging"
	"github.com/nich1/tempest-ai/internal/processors"
	"github.com/nich1/tempest-ai/internal/queue"
	"github.com/nich1/tempest-ai/internal/storage"
)

func main() {
	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadConsumer()
	if err != nil {
		panic(fmt.Errorf("load config: %w", err))
	}

	logger := logging.FromConfig("consumers", cfg.Logging)
	slog.SetDefault(logger)
	logger.Info("consumers.starting",
		slog.String("env", string(cfg.Env)),
		slog.Int("concurrency", cfg.Worker.Concurrency),
	)

	pool, err := db.New(rootCtx, cfg.Postgres)
	if err != nil {
		logger.Error("consumers.db_connect_failed", slog.Any("error", err))
		return
	}
	defer pool.Close()

	store, err := storage.New(rootCtx, cfg.MinIO)
	if err != nil {
		logger.Error("consumers.storage_connect_failed", slog.Any("error", err))
		return
	}

	processor := processors.NewLLMJobProcessor(
		cfg,
		jobs.New(pool),
		store,
		llm.NewFactory(cfg.LLM),
	)

	server := queue.NewServer(cfg.Redis, cfg.Worker)

	mux := asynq.NewServeMux()
	mux.Use(queue.LoggingMiddleware(logger))
	mux.HandleFunc(queue.TypeProcessJob, processor.Handle)

	go func() {
		if err := server.Run(mux); err != nil {
			logger.Error("consumers.run_failed", slog.Any("error", err))
			stop()
		}
	}()
	logger.Info("consumers.ready")

	<-rootCtx.Done()
	logger.Info("consumers.shutting_down")
	server.Shutdown()
	logger.Info("consumers.stopped")
}

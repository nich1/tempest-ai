// Package main is the API server entrypoint.
//
// Wires every dependency, runs migrations, mounts Gin routes (with auth
// middleware where needed), and gracefully shuts down on SIGTERM.
//
// @title           Tempest AI API
// @version         0.1
// @description     LLM schema manager - submit jobs that run user-defined input/output schemas through any of the configured LLM providers.
// @BasePath        /
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/nich1/tempest-ai/docs"

	"github.com/nich1/tempest-ai/internal/auth"
	"github.com/nich1/tempest-ai/internal/config"
	"github.com/nich1/tempest-ai/internal/db"
	"github.com/nich1/tempest-ai/internal/handlers"
	"github.com/nich1/tempest-ai/internal/jobs"
	"github.com/nich1/tempest-ai/internal/llm"
	"github.com/nich1/tempest-ai/internal/logging"
	"github.com/nich1/tempest-ai/internal/middleware"
	"github.com/nich1/tempest-ai/internal/queue"
	"github.com/nich1/tempest-ai/internal/storage"
	"github.com/nich1/tempest-ai/internal/users"
)

func main() {
	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadAPI()
	if err != nil {
		panic(fmt.Errorf("load config: %w", err))
	}

	logger := logging.FromConfig("api", cfg.Logging)
	slog.SetDefault(logger)
	logger.Info("api.starting",
		slog.String("env", string(cfg.Env)),
		slog.Int("port", cfg.API.Port),
	)

	if err := db.RunMigrations(cfg.Postgres.DSN()); err != nil {
		logger.Error("api.migrations_failed", slog.Any("error", err))
		return
	}
	logger.Info("api.migrations_applied")

	pool, err := db.New(rootCtx, cfg.Postgres)
	if err != nil {
		logger.Error("api.db_connect_failed", slog.Any("error", err))
		return
	}
	defer pool.Close()

	store, err := storage.New(rootCtx, cfg.MinIO)
	if err != nil {
		logger.Error("api.storage_connect_failed", slog.Any("error", err))
		return
	}

	qClient := queue.NewClient(cfg.Redis)
	defer qClient.Close()
	inspector := queue.NewInspector(cfg.Redis)
	defer inspector.Close()

	deps := &handlers.Deps{
		Cfg:       cfg,
		DB:        pool,
		Users:     users.New(pool),
		Jobs:      jobs.New(pool),
		Sessions:  auth.NewSessions(pool, cfg.API.SessionTTL),
		Storage:   store,
		Queue:     qClient,
		Inspector: inspector,
		Factory:   llm.NewFactory(cfg.LLM),
	}

	if cfg.Env.IsProd() {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(
		middleware.RequestID(),
		middleware.Logger(logger),
		middleware.Recoverer(),
		middleware.CORS(cfg.API.CORSAllowedOrigins),
	)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/health", deps.Health)

	authGroup := r.Group("/auth")
	{
		authGroup.POST("/signup", deps.Signup)
		authGroup.POST("/login", deps.Login)
		authGroup.POST("/logout", deps.Logout)
		authGroup.GET("/me",
			middleware.RequireAuth(deps.Sessions, cfg.API.CookieName),
			deps.Me,
		)
	}

	jobsGroup := r.Group("/jobs")
	jobsGroup.Use(middleware.RequireAuth(deps.Sessions, cfg.API.CookieName))
	{
		jobsGroup.POST("", deps.CreateJob)
		jobsGroup.GET("", deps.ListJobs)
		jobsGroup.GET("/:id", deps.GetJob)
		jobsGroup.POST("/file-upload-url", deps.FileUploadURL)
	}

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.API.Port),
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("api.serve_failed", slog.Any("error", err))
			stop()
		}
	}()
	logger.Info("api.ready")

	<-rootCtx.Done()
	logger.Info("api.shutting_down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("api.shutdown_failed", slog.Any("error", err))
	}
	logger.Info("api.stopped")
}

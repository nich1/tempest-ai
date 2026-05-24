// Package handlers contains the Gin HTTP handlers. Routing happens in
// apps/api/main.go; these are small per-resource files.
package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nich1/tempest-ai/internal/auth"
	"github.com/nich1/tempest-ai/internal/config"
	"github.com/nich1/tempest-ai/internal/jobs"
	"github.com/nich1/tempest-ai/internal/llm"
	"github.com/nich1/tempest-ai/internal/logging"
	"github.com/nich1/tempest-ai/internal/middleware"
	"github.com/nich1/tempest-ai/internal/models"
	"github.com/nich1/tempest-ai/internal/queue"
	"github.com/nich1/tempest-ai/internal/storage"
	"github.com/nich1/tempest-ai/internal/users"

	"github.com/hibiken/asynq"
)

// Deps bundles the dependencies wired through every handler.
type Deps struct {
	Cfg       config.APIConfig
	DB        *pgxpool.Pool
	Users     *users.Repository
	Jobs      *jobs.Repository
	Sessions  *auth.Sessions
	Storage   *storage.Client
	Queue     *queue.Client
	Inspector *asynq.Inspector
	Factory   *llm.Factory
}

// errResp is a helper for emitting errors with the request_id baked in.
func errResp(c *gin.Context, status int, msg string) {
	c.AbortWithStatusJSON(status, models.ErrorResponse{
		Error:     msg,
		RequestID: middleware.RequestIDFrom(c),
	})
}

// internalErr logs a 500 with full error detail but returns a generic
// message to the client.
func internalErr(c *gin.Context, event string, err error) {
	logging.FromContext(c.Request.Context()).Error(event, slog.Any("error", err))
	errResp(c, http.StatusInternalServerError, "internal server error")
}

// errIs reports whether err matches any of the provided sentinels.
func errIs(err error, sentinels ...error) bool {
	for _, s := range sentinels {
		if errors.Is(err, s) {
			return true
		}
	}
	return false
}

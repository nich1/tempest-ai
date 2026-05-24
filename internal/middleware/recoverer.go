package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"github.com/nich1/tempest-ai/internal/logging"
	"github.com/nich1/tempest-ai/internal/models"
)

// Recoverer turns panics into structured 500s so the process keeps serving.
func Recoverer() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logging.FromContext(c.Request.Context()).LogAttrs(
					c.Request.Context(),
					slog.LevelError,
					"http.panic",
					slog.Any("panic", r),
					slog.String("stack", string(debug.Stack())),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, models.ErrorResponse{
					Error:     "internal server error",
					RequestID: RequestIDFrom(c),
				})
			}
		}()
		c.Next()
	}
}

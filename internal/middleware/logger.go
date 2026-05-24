package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nich1/tempest-ai/internal/logging"
)

// Logger middleware attaches the base logger to the request context and
// emits one structured access log per HTTP request.
func Logger(base *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := logging.WithLogger(c.Request.Context(), base)
		c.Request = c.Request.WithContext(ctx)

		start := time.Now()
		c.Next()
		latency := time.Since(start)

		logger := logging.FromContext(c.Request.Context())
		event := "http.request"
		lvl := slog.LevelInfo
		if c.Writer.Status() >= 500 {
			lvl = slog.LevelError
		} else if c.Writer.Status() >= 400 {
			lvl = slog.LevelWarn
		}
		logger.LogAttrs(c.Request.Context(), lvl, event,
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", c.Writer.Status()),
			slog.Int("size", c.Writer.Size()),
			slog.Duration("latency", latency),
			slog.String("client_ip", c.ClientIP()),
			slog.String("user_agent", c.Request.UserAgent()),
		)
	}
}

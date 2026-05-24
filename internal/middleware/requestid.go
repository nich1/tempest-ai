// Package middleware holds Gin middleware: request ID, structured logger,
// CORS, panic recovery, session-based auth.
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/nich1/tempest-ai/internal/logging"
)

// HeaderRequestID is the header name we use for cross-service tracing.
const HeaderRequestID = "X-Request-ID"

const ctxRequestIDKey = "request_id"

// RequestID middleware: generate (or pass through) a request_id, set it
// on the response header and the context, and attach it to the
// context-carried logger so every downstream slog call includes it.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(HeaderRequestID)
		if id == "" {
			id = uuid.NewString()
		}
		c.Writer.Header().Set(HeaderRequestID, id)
		c.Set(ctxRequestIDKey, id)

		ctx := logging.WithRequestID(c.Request.Context(), id)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// RequestIDFrom returns the request_id stored on the gin context, if any.
func RequestIDFrom(c *gin.Context) string {
	if v, ok := c.Get(ctxRequestIDKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

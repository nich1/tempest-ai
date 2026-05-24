package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/nich1/tempest-ai/internal/auth"
	"github.com/nich1/tempest-ai/internal/logging"
	"github.com/nich1/tempest-ai/internal/models"
)

const (
	ctxSessionKey = "session"
)

// RequireAuth resolves the session cookie. Aborts with 401 on miss.
func RequireAuth(sessions *auth.Sessions, cookieName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, err := c.Cookie(cookieName)
		if err != nil || raw == "" {
			abort401(c, "authentication required")
			return
		}
		sess, err := sessions.Lookup(c.Request.Context(), raw)
		if err != nil {
			if errors.Is(err, auth.ErrSessionNotFound) || errors.Is(err, auth.ErrSessionExpired) {
				abort401(c, "session invalid")
				return
			}
			logging.FromContext(c.Request.Context()).Error("auth.lookup_failed",
				"error", err,
			)
			abort401(c, "authentication required")
			return
		}
		c.Set(ctxSessionKey, sess)
		ctx := logging.WithUserID(c.Request.Context(), sess.UserID.String())
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// SessionFrom returns the authenticated session attached by RequireAuth.
func SessionFrom(c *gin.Context) (auth.Session, bool) {
	v, ok := c.Get(ctxSessionKey)
	if !ok {
		return auth.Session{}, false
	}
	sess, ok := v.(auth.Session)
	return sess, ok
}

func abort401(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, models.ErrorResponse{
		Error:     msg,
		RequestID: RequestIDFrom(c),
	})
}

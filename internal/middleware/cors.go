package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS configures cross-origin requests for the Next.js client. We must
// AllowCredentials so the browser sends the session cookie.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", HeaderRequestID},
		ExposeHeaders:    []string{HeaderRequestID},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}

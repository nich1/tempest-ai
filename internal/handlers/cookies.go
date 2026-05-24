package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func (d *Deps) setSessionCookie(c *gin.Context, raw string) {
	maxAge := int(d.Cfg.API.SessionTTL.Seconds())
	secure := d.Cfg.Env.IsProd()
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		d.Cfg.API.CookieName,
		raw,
		maxAge,
		"/",
		d.Cfg.API.CookieDomain,
		secure,
		true, // httpOnly
	)
}

func (d *Deps) clearSessionCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		d.Cfg.API.CookieName,
		"",
		-1,
		"/",
		d.Cfg.API.CookieDomain,
		d.Cfg.Env.IsProd(),
		true,
	)
}

// isUniqueViolation peeks at the underlying error chain. We don't want to
// pull in pgconn just to detect this in handlers, but the user-create
// path is the only place we care, so it's a small import.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	if pgErr, ok := err.(*pgconn.PgError); ok {
		return pgErr.Code == "23505"
	}
	return strings.Contains(err.Error(), "23505")
}

// bytesToUUID converts a pgtype.UUID into a uuid.UUID. Centralized so we
// don't sprinkle this conversion across handlers.
func bytesToUUID(p pgtype.UUID) uuid.UUID {
	return uuid.UUID(p.Bytes)
}

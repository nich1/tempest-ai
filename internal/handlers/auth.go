package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/nich1/tempest-ai/internal/auth"
	"github.com/nich1/tempest-ai/internal/logging"
	"github.com/nich1/tempest-ai/internal/middleware"
	"github.com/nich1/tempest-ai/internal/models"
	"github.com/nich1/tempest-ai/internal/users"
)

// Signup creates a new user account.
//
// @Summary      Sign up
// @Description  Create a new user account and start a session.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body models.SignupRequest true "Signup payload"
// @Success      200 {object} models.AuthResponse
// @Failure      400 {object} models.ErrorResponse
// @Failure      409 {object} models.ErrorResponse
// @Router       /auth/signup [post]
func (d *Deps) Signup(c *gin.Context) {
	var req models.SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errResp(c, http.StatusBadRequest, "invalid request body")
		return
	}

	hash, err := auth.HashPassword(req.Password, d.Cfg.API.BcryptCost)
	if err != nil {
		internalErr(c, "auth.hash_failed", err)
		return
	}

	user, err := d.Users.Create(c.Request.Context(), req.Email, hash)
	if err != nil {
		// Postgres unique violation on (email) -> 409
		if isUniqueViolation(err) {
			errResp(c, http.StatusConflict, "email already registered")
			return
		}
		internalErr(c, "auth.create_user_failed", err)
		return
	}

	rawToken, _, err := d.Sessions.Create(c.Request.Context(), bytesToUUID(user.ID))
	if err != nil {
		internalErr(c, "auth.create_session_failed", err)
		return
	}
	d.setSessionCookie(c, rawToken)

	logging.FromContext(c.Request.Context()).Info("auth.signup",
		slog.String("user_id", bytesToUUID(user.ID).String()),
	)

	c.JSON(http.StatusOK, models.AuthResponse{User: models.UserFromRow(user)})
}

// Login authenticates an existing user.
//
// @Summary      Log in
// @Description  Authenticate and start a new session.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body models.LoginRequest true "Login payload"
// @Success      200 {object} models.AuthResponse
// @Failure      400 {object} models.ErrorResponse
// @Failure      401 {object} models.ErrorResponse
// @Router       /auth/login [post]
func (d *Deps) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errResp(c, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := d.Users.GetByEmail(c.Request.Context(), req.Email)
	if err != nil {
		if errors.Is(err, users.ErrNotFound) {
			// Run a no-op verify to keep timing roughly equal to the real path.
			_ = auth.VerifyPassword(req.Password, "")
			errResp(c, http.StatusUnauthorized, "invalid credentials")
			return
		}
		internalErr(c, "auth.get_user_failed", err)
		return
	}

	if err := auth.VerifyPassword(req.Password, user.PasswordHash); err != nil {
		errResp(c, http.StatusUnauthorized, "invalid credentials")
		return
	}

	rawToken, _, err := d.Sessions.Create(c.Request.Context(), bytesToUUID(user.ID))
	if err != nil {
		internalErr(c, "auth.create_session_failed", err)
		return
	}
	d.setSessionCookie(c, rawToken)

	logging.FromContext(c.Request.Context()).Info("auth.login",
		slog.String("user_id", bytesToUUID(user.ID).String()),
	)

	c.JSON(http.StatusOK, models.AuthResponse{User: models.UserFromRow(user)})
}

// Logout revokes the current session.
//
// @Summary      Log out
// @Description  Revoke the current session and clear the cookie.
// @Tags         auth
// @Produce      json
// @Success      204
// @Router       /auth/logout [post]
func (d *Deps) Logout(c *gin.Context) {
	raw, _ := c.Cookie(d.Cfg.API.CookieName)
	if raw != "" {
		if err := d.Sessions.Revoke(c.Request.Context(), raw); err != nil {
			logging.FromContext(c.Request.Context()).Warn("auth.revoke_failed", slog.Any("error", err))
		}
	}
	d.clearSessionCookie(c)
	c.Status(http.StatusNoContent)
}

// Me returns the currently-authenticated user.
//
// @Summary      Current user
// @Description  Return the authenticated user.
// @Tags         auth
// @Produce      json
// @Success      200 {object} models.UserDTO
// @Failure      401 {object} models.ErrorResponse
// @Router       /auth/me [get]
func (d *Deps) Me(c *gin.Context) {
	sess, ok := middleware.SessionFrom(c)
	if !ok {
		errResp(c, http.StatusUnauthorized, "authentication required")
		return
	}
	user, err := d.Users.GetByID(c.Request.Context(), sess.UserID)
	if err != nil {
		internalErr(c, "auth.get_user_failed", err)
		return
	}
	c.JSON(http.StatusOK, models.UserFromRow(user))
}

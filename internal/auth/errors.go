// Package auth handles password hashing, opaque session tokens, and
// session lifecycle (create / lookup / revoke).
//
// Tokens are stored hashed (sha256) at rest; the cookie carries the raw
// token. This means a database leak doesn't immediately compromise live
// sessions.
package auth

import "errors"

// ErrInvalidCredentials is returned for both "no such user" and "wrong
// password" so we don't leak which emails are registered.
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrSessionExpired is returned when a token's session row exists but
// has passed expires_at.
var ErrSessionExpired = errors.New("session expired")

// ErrSessionNotFound is returned when no session row matches the token.
var ErrSessionNotFound = errors.New("session not found")

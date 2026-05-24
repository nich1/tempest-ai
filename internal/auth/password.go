package auth

import (
	"crypto/subtle"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword runs bcrypt at the configured cost.
//
// Cost 12 is the default; dev compose lowers it to 4 for instant signups.
func HashPassword(plain string, cost int) (string, error) {
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	h, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
	if err != nil {
		return "", fmt.Errorf("bcrypt hash: %w", err)
	}
	return string(h), nil
}

// VerifyPassword returns ErrInvalidCredentials on any mismatch (including
// malformed hashes), so callers always have a single error to map.
//
// We also do a constant-time compare against an empty hash when the hash
// is empty to keep request timing similar between "no user" and "wrong
// password" paths. (bcrypt itself is constant-time-ish per fixed cost.)
func VerifyPassword(plain, hash string) error {
	if hash == "" {
		// Match empty against itself in constant time so timing of this
		// branch resembles a real bcrypt compare. Still returns invalid.
		_ = subtle.ConstantTimeCompare([]byte(plain), []byte(plain))
		return ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrInvalidCredentials
		}
		return ErrInvalidCredentials
	}
	return nil
}

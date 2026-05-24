package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// TokenBytes is the raw entropy size of a session token (256 bits).
const TokenBytes = 32

// GenerateToken returns (rawToken, tokenHash). The raw token is what
// goes into the cookie; the hash is what gets stored in Postgres.
func GenerateToken() (raw, hash string, err error) {
	buf := make([]byte, TokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("read random: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	hash = HashToken(raw)
	return raw, hash, nil
}

// HashToken hashes a raw session token for DB lookup. sha256 is plenty
// for opaque random tokens with 256 bits of entropy; we don't need
// adaptive hashing here.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

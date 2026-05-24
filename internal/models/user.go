// Package models holds API DTOs (request/response shapes).
//
// Row structs come from internal/db/sqlc; DTOs strip secret material
// (password_hash) and present a stable wire shape independent of storage.
package models

import (
	"time"

	"github.com/google/uuid"
)

// UserDTO is the user shape returned to API clients. PasswordHash never
// leaves this package.
type UserDTO struct {
	ID        uuid.UUID `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email     string    `json:"email" example:"alice@example.com"`
	CreatedAt time.Time `json:"created_at"`
}

// SignupRequest is the body of POST /auth/signup.
type SignupRequest struct {
	Email    string `json:"email" binding:"required,email" example:"alice@example.com"`
	Password string `json:"password" binding:"required,min=8" example:"correct horse battery staple"`
}

// LoginRequest is the body of POST /auth/login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"alice@example.com"`
	Password string `json:"password" binding:"required" example:"correct horse battery staple"`
}

// AuthResponse is returned from successful signup and login.
type AuthResponse struct {
	User UserDTO `json:"user"`
}

package models

// ErrorResponse is the standard error wire shape.
type ErrorResponse struct {
	Error     string `json:"error" example:"invalid credentials"`
	RequestID string `json:"request_id,omitempty" example:"01h..."`
}

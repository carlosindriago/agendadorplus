package domain

import "fmt"

// DomainError is the base type for all domain-level errors.
// These errors are mapped to HTTP status codes by the HTTP adapter.
type DomainError struct {
	Code    string // Machine-readable error code (e.g., "slot_locked")
	Message string // Human-readable message
}

func (e *DomainError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// --- Sentinel domain errors ---

// ErrSlotLocked is returned when a time slot is already locked by another transaction.
// Maps to HTTP 409 Conflict.
var ErrSlotLocked = &DomainError{
	Code:    "slot_locked",
	Message: "this time slot is currently being booked by another user",
}

// ErrSlotUnavailable is returned when a time slot has already been booked.
// Maps to HTTP 409 Conflict.
var ErrSlotUnavailable = &DomainError{
	Code:    "slot_unavailable",
	Message: "this time slot is no longer available",
}

// ErrNotFound is returned when a requested resource doesn't exist.
// Maps to HTTP 404 Not Found.
var ErrNotFound = &DomainError{
	Code:    "not_found",
	Message: "the requested resource was not found",
}

// ErrInvalidCredentials is returned when login credentials are wrong.
// Maps to HTTP 401 Unauthorized.
var ErrInvalidCredentials = &DomainError{
	Code:    "invalid_credentials",
	Message: "email or password is incorrect",
}

// --- Error constructors ---

// NewValidationError creates a new validation error.
// Maps to HTTP 400 Bad Request.
func NewValidationError(message string) *DomainError {
	return &DomainError{
		Code:    "validation_error",
		Message: message,
	}
}

// NewConflictError creates a new conflict error with a custom message.
// Maps to HTTP 409 Conflict.
func NewConflictError(message string) *DomainError {
	return &DomainError{
		Code:    "conflict",
		Message: message,
	}
}

// IsDomainError checks if an error is a DomainError and returns it.
func IsDomainError(err error) (*DomainError, bool) {
	de, ok := err.(*DomainError)
	return de, ok
}

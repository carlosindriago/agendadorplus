package domain

import (
	"time"

	"github.com/google/uuid"
)

// User represents a professional who owns an agenda.
// In the MVP, there's a single admin user.
type User struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	Email        string
	PasswordHash string
	Name         string
	CreatedAt    time.Time
}

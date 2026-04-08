package domain

import (
	"time"

	"github.com/google/uuid"
)

// Service represents a service offered by a professional.
// Examples: "Consulta General (30 min)", "Consulta Extendida (60 min)".
type Service struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	Name         string
	DurationMins int
	IsActive     bool
	CreatedAt    time.Time
}

// Validate checks that the service has valid data.
func (s Service) Validate() error {
	if s.Name == "" {
		return NewValidationError("service name is required")
	}
	if s.DurationMins <= 0 {
		return NewValidationError("service duration must be positive")
	}
	return nil
}

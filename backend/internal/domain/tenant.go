// Package domain contains the core business entities and rules.
// This is the innermost layer of the hexagonal architecture.
// It has ZERO dependencies on external packages (no frameworks, no DB drivers).
package domain

import (
	"time"

	"github.com/google/uuid"
)

// Tenant represents an organization. In the MVP, there's a single default tenant.
// In the EE, this enables multi-tenancy.
type Tenant struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
}

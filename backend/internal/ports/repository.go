// Package ports defines the interfaces (contracts) for the hexagonal architecture.
// Ports are the boundary between the application core (domain + use cases)
// and the outside world (adapters).
//
// In hexagonal architecture:
//   - DRIVEN ports (repositories, notifiers) are implemented by adapters
//     and called BY use cases.
//   - DRIVING ports (services) are implemented by use cases
//     and called BY adapters (HTTP handlers).
package ports

import (
	"context"
	"time"

	"github.com/carlosindriago/agendadorplus/internal/domain"
	"github.com/google/uuid"
)

// --- DRIVEN PORTS (implemented by adapters, called by use cases) ---

// AppointmentRepository defines the contract for appointment persistence.
type AppointmentRepository interface {
	// BookSlot atomically books a time slot within a transaction.
	// It uses SELECT FOR UPDATE NOWAIT to acquire an exclusive row lock.
	//
	// Returns:
	//   - domain.ErrSlotLocked if the slot is locked by another transaction
	//   - domain.ErrSlotUnavailable if the slot is already booked
	//   - The created Appointment on success
	BookSlot(ctx context.Context, req domain.BookingRequest) (*domain.Appointment, error)

	// ListByUser returns all appointments for a given user, ordered by date.
	ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.Appointment, error)
}

// SlotRepository defines the contract for time slot persistence.
type SlotRepository interface {
	// GetAvailableByUserAndDate returns available time slots for a user on a specific date (UTC).
	GetAvailableByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) ([]domain.TimeSlot, error)

	// BulkCreate inserts multiple time slots at once.
	// Skips slots that would violate the unique constraint (idempotent).
	BulkCreate(ctx context.Context, slots []domain.TimeSlot) (int, error)
}

// ServiceRepository defines the contract for service persistence.
type ServiceRepository interface {
	// List returns all active services for a tenant.
	List(ctx context.Context, tenantID uuid.UUID) ([]domain.Service, error)

	// GetServiceByID returns a service by its ID.
	GetServiceByID(ctx context.Context, id uuid.UUID) (*domain.Service, error)

	// Create creates a new service.
	Create(ctx context.Context, service *domain.Service) error

	// Update updates an existing service.
	Update(ctx context.Context, service *domain.Service) error
}

// UserRepository defines the contract for user persistence.
type UserRepository interface {
	// FindByEmail returns a user by email. Returns domain.ErrNotFound if not found.
	FindByEmail(ctx context.Context, email string) (*domain.User, error)

	// GetUserByID returns a user by ID. Returns domain.ErrNotFound if not found.
	GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

// Notifier defines the contract for sending notifications.
// In the MVP, this is implemented by a simple logger.
type Notifier interface {
	// SendBookingConfirmation sends a confirmation notification for a booking.
	// This is called asynchronously (in a goroutine) and must be safe to fail.
	SendBookingConfirmation(ctx context.Context, appointment *domain.Appointment) error
}

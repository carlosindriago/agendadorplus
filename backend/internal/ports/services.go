package ports

import (
	"context"
	"time"

	"github.com/carlosindriago/agendadorplus/internal/domain"
	"github.com/google/uuid"
)

// --- DRIVING PORTS (implemented by use cases, called by HTTP handlers) ---

// BookingService defines the contract for booking operations.
type BookingService interface {
	// Book creates a new appointment for the given booking request.
	// Handles the full transactional flow: lock → validate → insert → notify.
	Book(ctx context.Context, req domain.BookingRequest) (*domain.Appointment, error)

	// ListAppointments returns all appointments for a user.
	ListAppointments(ctx context.Context, userID uuid.UUID) ([]domain.Appointment, error)
}

// AvailabilityService defines the contract for availability queries.
type AvailabilityService interface {
	// GetAvailableSlots returns available time slots for a user on a specific date.
	// The date is expected in UTC; the frontend handles timezone conversion.
	GetAvailableSlots(ctx context.Context, userID uuid.UUID, date time.Time) ([]domain.TimeSlot, error)
}

// SlotGeneratorService defines the contract for generating time slots.
type SlotGeneratorService interface {
	// GenerateSlots creates time slots based on the generation request.
	// Returns the number of slots created.
	GenerateSlots(ctx context.Context, req domain.SlotGenerationRequest) (int, error)
}

// AuthService defines the contract for authentication operations.
type AuthService interface {
	// Login validates credentials and returns a JWT token.
	Login(ctx context.Context, email, password string) (token string, user *domain.User, err error)
}

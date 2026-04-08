package domain

import (
	"time"

	"github.com/google/uuid"
)

// AppointmentStatus represents the status of an appointment.
type AppointmentStatus string

const (
	StatusConfirmed AppointmentStatus = "confirmed"
	StatusCancelled AppointmentStatus = "cancelled"
)

// Appointment represents a confirmed booking.
//
// DESIGN DECISION (MVP): Each appointment is linked to exactly one TimeSlot
// via a UNIQUE constraint on time_slot_id. This enables simple Row-Level
// Lock with SELECT FOR UPDATE NOWAIT to prevent double-booking.
type Appointment struct {
	ID             uuid.UUID
	TimeSlotID     uuid.UUID
	ServiceID      uuid.UUID
	ClientName     string
	ClientEmail    string
	ClientTimezone string // IANA timezone (e.g., "America/Lima")
	Status         AppointmentStatus
	CreatedAt      time.Time
}

// BookingRequest represents the data needed to book an appointment.
type BookingRequest struct {
	TimeSlotID     uuid.UUID
	ServiceID      uuid.UUID
	ClientName     string
	ClientEmail    string
	ClientTimezone string
}

// Validate checks that the booking request has valid data.
func (r BookingRequest) Validate() error {
	if r.TimeSlotID == uuid.Nil {
		return NewValidationError("time slot ID is required")
	}
	if r.ServiceID == uuid.Nil {
		return NewValidationError("service ID is required")
	}
	if r.ClientName == "" {
		return NewValidationError("client name is required")
	}
	if r.ClientEmail == "" {
		return NewValidationError("client email is required")
	}
	if r.ClientTimezone == "" {
		return NewValidationError("client timezone is required")
	}
	// Validate timezone is a valid IANA timezone
	if _, err := time.LoadLocation(r.ClientTimezone); err != nil {
		return NewValidationError("invalid timezone: " + r.ClientTimezone)
	}
	return nil
}

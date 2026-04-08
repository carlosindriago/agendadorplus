package domain

import (
	"time"

	"github.com/google/uuid"
)

// TimeSlot represents a pre-calculated availability block.
//
// DESIGN DECISION (MVP): We use pre-calculated fixed slots instead of
// dynamic range overlap queries. This simplifies the booking logic
// (1 slot = 1 potential appointment) and makes Row-Level Lock trivial.
//
// TRADE-OFF: Does not support variable durations or dynamic buffers.
// The EE will migrate to a temporal range overlap model.
//
// All times are stored in UTC. The frontend is responsible for
// converting to/from the client's local timezone.
type TimeSlot struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	UserID       uuid.UUID
	StartTimeUTC time.Time
	EndTimeUTC   time.Time
	IsAvailable  bool
	CreatedAt    time.Time
}

// Validate checks that the time slot has valid data.
func (ts TimeSlot) Validate() error {
	if ts.StartTimeUTC.IsZero() {
		return NewValidationError("start time is required")
	}
	if ts.EndTimeUTC.IsZero() {
		return NewValidationError("end time is required")
	}
	if !ts.EndTimeUTC.After(ts.StartTimeUTC) {
		return NewValidationError("end time must be after start time")
	}
	return nil
}

// Duration returns the duration of the time slot.
func (ts TimeSlot) Duration() time.Duration {
	return ts.EndTimeUTC.Sub(ts.StartTimeUTC)
}

// SlotGenerationRequest represents a request to generate time slots.
type SlotGenerationRequest struct {
	TenantID     uuid.UUID
	UserID       uuid.UUID
	StartDate    time.Time     // First date to generate slots for (UTC)
	EndDate      time.Time     // Last date to generate slots for (UTC)
	DayStartHour int           // Hour of day to start (0-23, in UTC)
	DayEndHour   int           // Hour of day to end (0-23, in UTC)
	SlotDuration time.Duration // Duration of each slot
	Weekdays     []time.Weekday // Which days of the week to generate for
}

// Validate checks that the slot generation request has valid data.
func (r SlotGenerationRequest) Validate() error {
	if r.StartDate.IsZero() {
		return NewValidationError("start date is required")
	}
	if r.EndDate.IsZero() {
		return NewValidationError("end date is required")
	}
	if r.EndDate.Before(r.StartDate) {
		return NewValidationError("end date must not be before start date")
	}
	if r.DayStartHour < 0 || r.DayStartHour > 23 {
		return NewValidationError("day start hour must be between 0 and 23")
	}
	if r.DayEndHour < 0 || r.DayEndHour > 23 {
		return NewValidationError("day end hour must be between 0 and 23")
	}
	if r.DayEndHour <= r.DayStartHour {
		return NewValidationError("day end hour must be after day start hour")
	}
	if r.SlotDuration <= 0 {
		return NewValidationError("slot duration must be positive")
	}
	if len(r.Weekdays) == 0 {
		return NewValidationError("at least one weekday is required")
	}
	return nil
}

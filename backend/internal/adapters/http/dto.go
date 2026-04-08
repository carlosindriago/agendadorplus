package http

import (
	"time"

	"github.com/google/uuid"
)

// --- Request DTOs ---

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type BookAppointmentRequest struct {
	TimeSlotID     uuid.UUID `json:"time_slot_id" binding:"required"`
	ServiceID      uuid.UUID `json:"service_id" binding:"required"`
	ClientName     string    `json:"client_name" binding:"required"`
	ClientEmail    string    `json:"client_email" binding:"required,email"`
	// ClientTimezone is handled by middleware
}

type GenerateSlotsRequest struct {
	StartDate    string `json:"start_date" binding:"required"` // format: YYYY-MM-DD
	EndDate      string `json:"end_date" binding:"required"`   // format: YYYY-MM-DD
	DayStartHour int    `json:"day_start_hour" binding:"required,min=0,max=23"`
	DayEndHour   int    `json:"day_end_hour" binding:"required,min=0,max=23"`
	SlotDuration int    `json:"slot_duration_mins" binding:"required,min=5"`
	Weekdays     []int  `json:"weekdays" binding:"required,min=1"` // 0=Sun, 1=Mon, ..., 6=Sat
}

// --- Response DTOs ---

type LoginResponse struct {
	Token string     `json:"token"`
	User  UserDetail `json:"user"`
}

type UserDetail struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
}

type TimeSlotResponse struct {
	ID        uuid.UUID `json:"id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

type AppointmentResponse struct {
	ID          uuid.UUID `json:"id"`
	TimeSlotID  uuid.UUID `json:"time_slot_id"`
	StartTime   time.Time `json:"start_time"`   // Populated from joins
	EndTime     time.Time `json:"end_time"`     // Populated from joins
	ServiceName string    `json:"service_name"` // Populated from joins
	ClientName  string    `json:"client_name"`
	Status      string    `json:"status"`
}

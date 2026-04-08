package usecases

import (
	"context"
	"log/slog"

	"github.com/carlosindriago/agendadorplus/internal/domain"
	"github.com/carlosindriago/agendadorplus/internal/ports"
	"github.com/google/uuid"
)

// BookingUseCase orchestrates the booking flow.
// It implements ports.BookingService.
type BookingUseCase struct {
	appointmentRepo ports.AppointmentRepository
	notifier        ports.Notifier
	logger          *slog.Logger
}

// Compile-time check that BookingUseCase implements ports.BookingService.
var _ ports.BookingService = (*BookingUseCase)(nil)

// NewBookingUseCase creates a new BookingUseCase.
func NewBookingUseCase(
	appointmentRepo ports.AppointmentRepository,
	notifier ports.Notifier,
	logger *slog.Logger,
) *BookingUseCase {
	return &BookingUseCase{
		appointmentRepo: appointmentRepo,
		notifier:        notifier,
		logger:          logger,
	}
}

// Book creates a new appointment.
//
// Flow:
//  1. Validate the booking request (domain rules)
//  2. Delegate to repository which handles the transactional lock
//     (SELECT FOR UPDATE NOWAIT → insert appointment → commit)
//  3. On success, fire async notification (with recover)
//  4. Return the created appointment
func (uc *BookingUseCase) Book(ctx context.Context, req domain.BookingRequest) (*domain.Appointment, error) {
	// Step 1: Domain validation
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Step 2: Atomic booking (transaction handled by repository)
	appointment, err := uc.appointmentRepo.BookSlot(ctx, req)
	if err != nil {
		uc.logger.WarnContext(ctx, "booking failed",
			"slot_id", req.TimeSlotID,
			"error", err,
		)
		return nil, err
	}

	uc.logger.InfoContext(ctx, "appointment booked successfully",
		"appointment_id", appointment.ID,
		"slot_id", req.TimeSlotID,
		"client_email", req.ClientEmail,
	)

	// Step 3: Async notification (safe goroutine with recover)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				uc.logger.Error("notification panic recovered",
					"appointment_id", appointment.ID,
					"panic", r,
				)
			}
		}()

		// Use a detached context so the notification isn't cancelled
		// when the HTTP request completes
		if err := uc.notifier.SendBookingConfirmation(context.Background(), appointment); err != nil {
			uc.logger.Error("failed to send booking confirmation",
				"appointment_id", appointment.ID,
				"error", err,
			)
		}
	}()

	return appointment, nil
}

// ListAppointments returns all appointments for a user.
func (uc *BookingUseCase) ListAppointments(ctx context.Context, userID uuid.UUID) ([]domain.Appointment, error) {
	return uc.appointmentRepo.ListByUser(ctx, userID)
}

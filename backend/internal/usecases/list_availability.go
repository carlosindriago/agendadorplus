package usecases

import (
	"context"
	"log/slog"
	"time"

	"github.com/carlosindriago/agendadorplus/internal/domain"
	"github.com/carlosindriago/agendadorplus/internal/ports"
	"github.com/google/uuid"
)

// AvailabilityUseCase handles availability queries.
// It implements ports.AvailabilityService.
type AvailabilityUseCase struct {
	slotRepo ports.SlotRepository
	logger   *slog.Logger
}

// Compile-time check that AvailabilityUseCase implements ports.AvailabilityService.
var _ ports.AvailabilityService = (*AvailabilityUseCase)(nil)

// NewAvailabilityUseCase creates a new AvailabilityUseCase.
func NewAvailabilityUseCase(slotRepo ports.SlotRepository, logger *slog.Logger) *AvailabilityUseCase {
	return &AvailabilityUseCase{
		slotRepo: slotRepo,
		logger:   logger,
	}
}

// GetAvailableSlots returns available time slots for a user on a specific date.
//
// The returned slots have their times in UTC. The frontend is responsible
// for converting to the client's local timezone using the X-Timezone header.
func (uc *AvailabilityUseCase) GetAvailableSlots(ctx context.Context, userID uuid.UUID, date time.Time) ([]domain.TimeSlot, error) {
	if userID == uuid.Nil {
		return nil, domain.NewValidationError("user ID is required")
	}

	slots, err := uc.slotRepo.GetAvailableByUserAndDate(ctx, userID, date)
	if err != nil {
		uc.logger.ErrorContext(ctx, "failed to get available slots",
			"user_id", userID,
			"date", date,
			"error", err,
		)
		return nil, err
	}

	uc.logger.InfoContext(ctx, "availability queried",
		"user_id", userID,
		"date", date.Format("2006-01-02"),
		"available_slots", len(slots),
	)

	return slots, nil
}

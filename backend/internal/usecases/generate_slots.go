package usecases

import (
	"context"
	"log/slog"
	"time"

	"github.com/carlosindriago/agendadorplus/internal/domain"
	"github.com/carlosindriago/agendadorplus/internal/ports"
	"github.com/google/uuid"
)

// SlotGeneratorUseCase handles generating pre-calculated time slots.
// It implements ports.SlotGeneratorService.
type SlotGeneratorUseCase struct {
	slotRepo ports.SlotRepository
	logger   *slog.Logger
}

// Compile-time check that SlotGeneratorUseCase implements ports.SlotGeneratorService.
var _ ports.SlotGeneratorService = (*SlotGeneratorUseCase)(nil)

// NewSlotGeneratorUseCase creates a new SlotGeneratorUseCase.
func NewSlotGeneratorUseCase(slotRepo ports.SlotRepository, logger *slog.Logger) *SlotGeneratorUseCase {
	return &SlotGeneratorUseCase{
		slotRepo: slotRepo,
		logger:   logger,
	}
}

// GenerateSlots creates time slots based on the generation request.
//
// Algorithm:
//  1. Validate the request
//  2. Iterate through each day in the date range
//  3. For each day that matches the requested weekdays:
//     - Generate slots from DayStartHour to DayEndHour with the given duration
//  4. Bulk insert all generated slots (idempotent — skips duplicates)
//
// All generated times are in UTC.
func (uc *SlotGeneratorUseCase) GenerateSlots(ctx context.Context, req domain.SlotGenerationRequest) (int, error) {
	if err := req.Validate(); err != nil {
		return 0, err
	}

	weekdaySet := make(map[time.Weekday]bool)
	for _, wd := range req.Weekdays {
		weekdaySet[wd] = true
	}

	var slots []domain.TimeSlot

	// Iterate through each day in the range
	for d := req.StartDate; !d.After(req.EndDate); d = d.AddDate(0, 0, 1) {
		// Skip days not in the requested weekdays
		if !weekdaySet[d.Weekday()] {
			continue
		}

		// Generate slots for this day
		dayStart := time.Date(d.Year(), d.Month(), d.Day(), req.DayStartHour, 0, 0, 0, time.UTC)
		dayEnd := time.Date(d.Year(), d.Month(), d.Day(), req.DayEndHour, 0, 0, 0, time.UTC)

		for slotStart := dayStart; slotStart.Add(req.SlotDuration).Before(dayEnd) || slotStart.Add(req.SlotDuration).Equal(dayEnd); slotStart = slotStart.Add(req.SlotDuration) {
			slots = append(slots, domain.TimeSlot{
				ID:           uuid.New(),
				TenantID:     req.TenantID,
				UserID:       req.UserID,
				StartTimeUTC: slotStart,
				EndTimeUTC:   slotStart.Add(req.SlotDuration),
				IsAvailable:  true,
			})
		}
	}

	if len(slots) == 0 {
		uc.logger.WarnContext(ctx, "no slots generated for the given parameters",
			"start_date", req.StartDate,
			"end_date", req.EndDate,
			"weekdays", req.Weekdays,
		)
		return 0, nil
	}

	created, err := uc.slotRepo.BulkCreate(ctx, slots)
	if err != nil {
		uc.logger.ErrorContext(ctx, "failed to bulk create slots",
			"total_slots", len(slots),
			"error", err,
		)
		return 0, err
	}

	uc.logger.InfoContext(ctx, "slots generated successfully",
		"requested", len(slots),
		"created", created,
		"start_date", req.StartDate.Format("2006-01-02"),
		"end_date", req.EndDate.Format("2006-01-02"),
	)

	return created, nil
}

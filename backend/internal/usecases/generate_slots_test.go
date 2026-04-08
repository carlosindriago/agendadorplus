package usecases_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/carlosindriago/agendadorplus/internal/domain"
	"github.com/carlosindriago/agendadorplus/internal/usecases"
	"github.com/google/uuid"
)

func Test_GenerateSlots(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("GENERATES_CORRECT_SLOTS_FOR_RANGE", func(t *testing.T) {
		repo := &mockSlotRepo{}
		uc := usecases.NewSlotGeneratorUseCase(repo, slog.Default())

		// Generate 30-min slots from 9am to 5pm (16 slots per day)
		// for Monday to Friday, for 1 week
		startDate := time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC) // Monday
		endDate := time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC)   // Friday

		created, err := uc.GenerateSlots(context.Background(), domain.SlotGenerationRequest{
			TenantID:     tenantID,
			UserID:       userID,
			StartDate:    startDate,
			EndDate:      endDate,
			DayStartHour: 9,
			DayEndHour:   17,
			SlotDuration: 30 * time.Minute,
			Weekdays:     []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday},
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// 5 days × 16 slots (9:00-9:30, 9:30-10:00, ..., 16:30-17:00) = 80 slots
		expectedSlots := 5 * 16
		if created != expectedSlots {
			t.Errorf("expected %d slots, got %d", expectedSlots, created)
		}
	})

	t.Run("RESPECTS_SLOT_DURATION", func(t *testing.T) {
		repo := &mockSlotRepo{}
		uc := usecases.NewSlotGeneratorUseCase(repo, slog.Default())

		startDate := time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC) // Monday

		created, err := uc.GenerateSlots(context.Background(), domain.SlotGenerationRequest{
			TenantID:     tenantID,
			UserID:       userID,
			StartDate:    startDate,
			EndDate:      startDate, // Same day
			DayStartHour: 9,
			DayEndHour:   12,
			SlotDuration: 60 * time.Minute, // 1 hour slots
			Weekdays:     []time.Weekday{time.Monday},
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// 3 hours / 1 hour = 3 slots (9-10, 10-11, 11-12)
		if created != 3 {
			t.Errorf("expected 3 slots for 1h duration in 3h window, got %d", created)
		}

		// Verify each slot has the correct duration
		for _, slot := range repo.slots {
			dur := slot.EndTimeUTC.Sub(slot.StartTimeUTC)
			if dur != 60*time.Minute {
				t.Errorf("expected 60min slot, got %v", dur)
			}
		}
	})

	t.Run("SKIPS_NON_REQUESTED_WEEKDAYS", func(t *testing.T) {
		repo := &mockSlotRepo{}
		uc := usecases.NewSlotGeneratorUseCase(repo, slog.Default())

		// Mon 13 to Sun 19 (full week), but only request Monday
		startDate := time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC)

		created, err := uc.GenerateSlots(context.Background(), domain.SlotGenerationRequest{
			TenantID:     tenantID,
			UserID:       userID,
			StartDate:    startDate,
			EndDate:      endDate,
			DayStartHour: 9,
			DayEndHour:   11,
			SlotDuration: 30 * time.Minute,
			Weekdays:     []time.Weekday{time.Monday}, // Only Monday
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Only 1 Monday in range, 2 hours / 30 min = 4 slots
		if created != 4 {
			t.Errorf("expected 4 slots (only Monday), got %d", created)
		}
	})

	t.Run("ALL_SLOTS_ARE_UTC", func(t *testing.T) {
		repo := &mockSlotRepo{}
		uc := usecases.NewSlotGeneratorUseCase(repo, slog.Default())

		startDate := time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC)

		_, err := uc.GenerateSlots(context.Background(), domain.SlotGenerationRequest{
			TenantID:     tenantID,
			UserID:       userID,
			StartDate:    startDate,
			EndDate:      startDate,
			DayStartHour: 9,
			DayEndHour:   10,
			SlotDuration: 30 * time.Minute,
			Weekdays:     []time.Weekday{time.Monday},
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for i, slot := range repo.slots {
			if slot.StartTimeUTC.Location() != time.UTC {
				t.Errorf("slot %d start time not in UTC: %v", i, slot.StartTimeUTC.Location())
			}
			if slot.EndTimeUTC.Location() != time.UTC {
				t.Errorf("slot %d end time not in UTC: %v", i, slot.EndTimeUTC.Location())
			}
		}
	})

	t.Run("VALIDATION_ERROR_on_invalid_request", func(t *testing.T) {
		repo := &mockSlotRepo{}
		uc := usecases.NewSlotGeneratorUseCase(repo, slog.Default())

		// No weekdays
		_, err := uc.GenerateSlots(context.Background(), domain.SlotGenerationRequest{
			TenantID:     tenantID,
			UserID:       userID,
			StartDate:    time.Now(),
			EndDate:      time.Now().Add(24 * time.Hour),
			DayStartHour: 9,
			DayEndHour:   17,
			SlotDuration: 30 * time.Minute,
			Weekdays:     []time.Weekday{}, // Empty!
		})

		if err == nil {
			t.Error("expected validation error for empty weekdays")
		}
	})
}

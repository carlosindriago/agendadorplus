package usecases_test

import (
	"context"
	"testing"
	"time"

	"github.com/carlosindriago/agendadorplus/internal/domain"
	"github.com/carlosindriago/agendadorplus/internal/usecases"
	"github.com/google/uuid"
	"log/slog"
)

// --- Mock Slot Repository for Availability ---

type mockSlotRepo struct {
	slots []domain.TimeSlot
}

func (m *mockSlotRepo) GetAvailableByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) ([]domain.TimeSlot, error) {
	var result []domain.TimeSlot
	for _, s := range m.slots {
		if s.UserID == userID && s.IsAvailable {
			// Filter by date (same day in UTC)
			if s.StartTimeUTC.Year() == date.Year() &&
				s.StartTimeUTC.YearDay() == date.YearDay() {
				result = append(result, s)
			}
		}
	}
	return result, nil
}

func (m *mockSlotRepo) BulkCreate(ctx context.Context, slots []domain.TimeSlot) (int, error) {
	m.slots = append(m.slots, slots...)
	return len(slots), nil
}

func Test_ListAvailability(t *testing.T) {
	userID := uuid.New()
	tenantID := uuid.New()

	// Create some test slots — all times in UTC
	baseDate := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)

	t.Run("RETURNS_AVAILABLE_SLOTS_FOR_DATE", func(t *testing.T) {
		repo := &mockSlotRepo{
			slots: []domain.TimeSlot{
				{
					ID:           uuid.New(),
					TenantID:     tenantID,
					UserID:       userID,
					StartTimeUTC: baseDate.Add(9 * time.Hour),  // 09:00 UTC
					EndTimeUTC:   baseDate.Add(10 * time.Hour), // 10:00 UTC
					IsAvailable:  true,
				},
				{
					ID:           uuid.New(),
					TenantID:     tenantID,
					UserID:       userID,
					StartTimeUTC: baseDate.Add(10 * time.Hour), // 10:00 UTC
					EndTimeUTC:   baseDate.Add(11 * time.Hour), // 11:00 UTC
					IsAvailable:  true,
				},
			},
		}

		uc := usecases.NewAvailabilityUseCase(repo, slog.Default())
		slots, err := uc.GetAvailableSlots(context.Background(), userID, baseDate)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(slots) != 2 {
			t.Errorf("expected 2 slots, got %d", len(slots))
		}
	})

	t.Run("BOOKED_SLOT_NOT_IN_AVAILABILITY", func(t *testing.T) {
		repo := &mockSlotRepo{
			slots: []domain.TimeSlot{
				{
					ID:           uuid.New(),
					TenantID:     tenantID,
					UserID:       userID,
					StartTimeUTC: baseDate.Add(9 * time.Hour),
					EndTimeUTC:   baseDate.Add(10 * time.Hour),
					IsAvailable:  true,
				},
				{
					ID:           uuid.New(),
					TenantID:     tenantID,
					UserID:       userID,
					StartTimeUTC: baseDate.Add(10 * time.Hour),
					EndTimeUTC:   baseDate.Add(11 * time.Hour),
					IsAvailable:  false, // Already booked
				},
			},
		}

		uc := usecases.NewAvailabilityUseCase(repo, slog.Default())
		slots, err := uc.GetAvailableSlots(context.Background(), userID, baseDate)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(slots) != 1 {
			t.Errorf("expected 1 available slot (booked one excluded), got %d", len(slots))
		}
	})

	t.Run("EMPTY_DAY_RETURNS_EMPTY", func(t *testing.T) {
		repo := &mockSlotRepo{
			slots: []domain.TimeSlot{}, // No slots at all
		}

		uc := usecases.NewAvailabilityUseCase(repo, slog.Default())
		slots, err := uc.GetAvailableSlots(context.Background(), userID, baseDate)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(slots) != 0 {
			t.Errorf("expected 0 slots for empty day, got %d", len(slots))
		}
	})

	t.Run("INVALID_USER_ID_RETURNS_ERROR", func(t *testing.T) {
		repo := &mockSlotRepo{}
		uc := usecases.NewAvailabilityUseCase(repo, slog.Default())

		_, err := uc.GetAvailableSlots(context.Background(), uuid.Nil, baseDate)

		if err == nil {
			t.Error("expected validation error for nil user ID")
		}
	})
}

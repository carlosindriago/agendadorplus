package usecases_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/carlosindriago/agendadorplus/internal/domain"
	"github.com/carlosindriago/agendadorplus/internal/usecases"
	"github.com/google/uuid"
)

// --- Mock Repository ---

// mockAppointmentRepo simulates a PostgreSQL repository with Row-Level Lock.
type mockAppointmentRepo struct {
	mu           sync.Mutex
	bookedSlots  map[uuid.UUID]bool
	simulateWait time.Duration // Simulate DB latency
}

func newMockRepo() *mockAppointmentRepo {
	return &mockAppointmentRepo{
		bookedSlots: make(map[uuid.UUID]bool),
	}
}

func (m *mockAppointmentRepo) BookSlot(ctx context.Context, req domain.BookingRequest) (*domain.Appointment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Simulate DB latency (for timeout tests)
	if m.simulateWait > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(m.simulateWait):
			// Continue normal execution
		}
	}

	// Check context deadline AFTER acquiring lock (simulates FOR UPDATE NOWAIT behavior)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if m.bookedSlots[req.TimeSlotID] {
		return nil, domain.ErrSlotUnavailable
	}

	m.bookedSlots[req.TimeSlotID] = true

	return &domain.Appointment{
		ID:             uuid.New(),
		TimeSlotID:     req.TimeSlotID,
		ServiceID:      req.ServiceID,
		ClientName:     req.ClientName,
		ClientEmail:    req.ClientEmail,
		ClientTimezone: req.ClientTimezone,
		Status:         domain.StatusConfirmed,
		CreatedAt:      time.Now(),
	}, nil
}

func (m *mockAppointmentRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.Appointment, error) {
	return nil, nil
}

// --- Mock Notifier ---

type mockNotifier struct {
	called atomic.Int32
}

func (m *mockNotifier) SendBookingConfirmation(ctx context.Context, appointment *domain.Appointment) error {
	m.called.Add(1)
	return nil
}

// --- Tests ---

func Test_BookAppointment_Concurrency(t *testing.T) {
	t.Run("ONE_SUCCESS_REST_FAIL_on_concurrent_booking", func(t *testing.T) {
		repo := newMockRepo()
		notifier := &mockNotifier{}
		logger := slog.Default()
		uc := usecases.NewBookingUseCase(repo, notifier, logger)

		slotID := uuid.New()
		serviceID := uuid.New()
		concurrentUsers := 10

		var wg sync.WaitGroup
		var successCount atomic.Int32
		var errorCount atomic.Int32

		for i := 0; i < concurrentUsers; i++ {
			wg.Add(1)
			go func(userIndex int) {
				defer wg.Done()

				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()

				_, err := uc.Book(ctx, domain.BookingRequest{
					TimeSlotID:     slotID,
					ServiceID:      serviceID,
					ClientName:     "Test User",
					ClientEmail:    "test@example.com",
					ClientTimezone: "America/Lima",
				})

				if err == nil {
					successCount.Add(1)
				} else {
					errorCount.Add(1)
				}
			}(i)
		}

		wg.Wait()

		if got := successCount.Load(); got != 1 {
			t.Errorf("expected exactly 1 success, got %d", got)
		}
		expectedErrors := int32(concurrentUsers - 1)
		if got := errorCount.Load(); got != expectedErrors {
			t.Errorf("expected %d errors, got %d", expectedErrors, got)
		}
	})

	t.Run("TIMEOUT_EXPIRED_releases_resource", func(t *testing.T) {
		repo := newMockRepo()
		repo.simulateWait = 100 * time.Millisecond // Simulate slow DB
		notifier := &mockNotifier{}
		logger := slog.Default()
		uc := usecases.NewBookingUseCase(repo, notifier, logger)

		slotID := uuid.New()

		// Ridiculously short context to force timeout before DB responds
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		// Give the context time to expire
		time.Sleep(5 * time.Millisecond)

		_, err := uc.Book(ctx, domain.BookingRequest{
			TimeSlotID:     slotID,
			ServiceID:      uuid.New(),
			ClientName:     "Timeout User",
			ClientEmail:    "timeout@example.com",
			ClientTimezone: "America/Lima",
		})

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("expected context.DeadlineExceeded, got: %v", err)
		}

		// Verify the slot was NOT booked
		if repo.bookedSlots[slotID] {
			t.Error("slot should NOT be marked as booked after timeout")
		}
	})

	t.Run("VALIDATION_ERROR_on_invalid_request", func(t *testing.T) {
		repo := newMockRepo()
		notifier := &mockNotifier{}
		logger := slog.Default()
		uc := usecases.NewBookingUseCase(repo, notifier, logger)

		ctx := context.Background()

		// Empty slot ID should fail validation
		_, err := uc.Book(ctx, domain.BookingRequest{
			TimeSlotID:     uuid.Nil,
			ServiceID:      uuid.New(),
			ClientName:     "Test",
			ClientEmail:    "test@example.com",
			ClientTimezone: "America/Lima",
		})

		if err == nil {
			t.Error("expected validation error, got nil")
		}

		de, ok := domain.IsDomainError(err)
		if !ok {
			t.Errorf("expected DomainError, got %T", err)
		}
		if de.Code != "validation_error" {
			t.Errorf("expected code 'validation_error', got '%s'", de.Code)
		}
	})

	t.Run("INVALID_TIMEZONE_rejected", func(t *testing.T) {
		repo := newMockRepo()
		notifier := &mockNotifier{}
		logger := slog.Default()
		uc := usecases.NewBookingUseCase(repo, notifier, logger)

		ctx := context.Background()

		_, err := uc.Book(ctx, domain.BookingRequest{
			TimeSlotID:     uuid.New(),
			ServiceID:      uuid.New(),
			ClientName:     "Test",
			ClientEmail:    "test@example.com",
			ClientTimezone: "Invalid/Timezone",
		})

		if err == nil {
			t.Error("expected validation error for invalid timezone, got nil")
		}
	})
}

package postgres

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/carlosindriago/agendadorplus/internal/domain"
	"github.com/carlosindriago/agendadorplus/internal/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository implements all the database ports using pgx.
type Repository struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// Compile-time checks
var _ ports.AppointmentRepository = (*Repository)(nil)
var _ ports.SlotRepository = (*Repository)(nil)
var _ ports.ServiceRepository = (*Repository)(nil)
var _ ports.UserRepository = (*Repository)(nil)

// NewRepository creates a new Repository.
func NewRepository(pool *pgxpool.Pool, logger *slog.Logger) *Repository {
	return &Repository{
		pool:   pool,
		logger: logger,
	}
}

// --- AppointmentRepository ---

// BookSlot is the core transaction method. It uses SELECT FOR UPDATE NOWAIT
// to prevent double-booking.
func (r *Repository) BookSlot(ctx context.Context, req domain.BookingRequest) (*domain.Appointment, error) {
	// Begin transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	// Defer rollback. If commit was successful, this is a no-op.
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// 1. Lock the time slot
	// SELECT FOR UPDATE NOWAIT ensures we get an exclusive lock or fail immediately (no queueing).
	var slotAvailable bool
	err = tx.QueryRow(ctx, `
		SELECT is_available 
		FROM time_slots 
		WHERE id = $1 
		FOR UPDATE NOWAIT
	`, req.TimeSlotID).Scan(&slotAvailable)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			// Code 55P03 is lock_not_available
			if pgErr.Code == "55P03" {
				return nil, domain.ErrSlotLocked
			}
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	// 2. Validate availability
	if !slotAvailable {
		// Logically unavailable
		return nil, domain.ErrSlotUnavailable
	}

	// 3. Mark the slot as unavailable
	_, err = tx.Exec(ctx, `
		UPDATE time_slots 
		SET is_available = false 
		WHERE id = $1
	`, req.TimeSlotID)
	if err != nil {
		return nil, err
	}

	// 4. Create the appointment
	appointment := &domain.Appointment{
		ID:             uuid.New(),
		TimeSlotID:     req.TimeSlotID,
		ServiceID:      req.ServiceID,
		ClientName:     req.ClientName,
		ClientEmail:    req.ClientEmail,
		ClientTimezone: req.ClientTimezone,
		Status:         domain.StatusConfirmed,
		CreatedAt:      time.Now().UTC(),
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO appointments (id, time_slot_id, service_id, client_name, client_email, client_timezone, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, appointment.ID, appointment.TimeSlotID, appointment.ServiceID, appointment.ClientName, appointment.ClientEmail, appointment.ClientTimezone, appointment.Status, appointment.CreatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			// Code 23505 is unique_violation (failsafe if constraints are hit)
			if pgErr.Code == "23505" {
				return nil, domain.ErrSlotUnavailable
			}
		}
		return nil, err
	}

	// 5. Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return appointment, nil
}

func (r *Repository) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.Appointment, error) {
	// Implementation for Admin dashboard
	rows, err := r.pool.Query(ctx, `
		SELECT a.id, a.time_slot_id, a.service_id, a.client_name, a.client_email, a.client_timezone, a.status, a.created_at
		FROM appointments a
		JOIN time_slots ts ON a.time_slot_id = ts.id
		WHERE ts.user_id = $1
		ORDER BY ts.start_time_utc DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var appointments []domain.Appointment
	for rows.Next() {
		var a domain.Appointment
		if err := rows.Scan(&a.ID, &a.TimeSlotID, &a.ServiceID, &a.ClientName, &a.ClientEmail, &a.ClientTimezone, &a.Status, &a.CreatedAt); err != nil {
			return nil, err
		}
		appointments = append(appointments, a)
	}
	return appointments, rows.Err()
}

// --- SlotRepository ---

func (r *Repository) GetAvailableByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) ([]domain.TimeSlot, error) {
	// Date is in UTC. We want to find slots that START on that day (UTC).
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.AddDate(0, 0, 1)

	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, user_id, start_time_utc, end_time_utc, is_available, created_at
		FROM time_slots
		WHERE user_id = $1 AND start_time_utc >= $2 AND start_time_utc < $3 AND is_available = true
		ORDER BY start_time_utc ASC
	`, userID, startOfDay, endOfDay)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slots []domain.TimeSlot
	for rows.Next() {
		var s domain.TimeSlot
		if err := rows.Scan(&s.ID, &s.TenantID, &s.UserID, &s.StartTimeUTC, &s.EndTimeUTC, &s.IsAvailable, &s.CreatedAt); err != nil {
			return nil, err
		}
		slots = append(slots, s)
	}
	return slots, rows.Err()
}

func (r *Repository) BulkCreate(ctx context.Context, slots []domain.TimeSlot) (int, error) {
	if len(slots) == 0 {
		return 0, nil
	}

	// pgx way to do bulk inserts: CopyFrom
	// However, CopyFrom doesn't support ON CONFLICT DO NOTHING natively.
	// Since we need idempotency (skipping existing slots via the uq_user_start_time constraint),
	// we'll build a batch of insert queries.
	batch := &pgx.Batch{}

	for _, s := range slots {
		if s.CreatedAt.IsZero() {
			s.CreatedAt = time.Now().UTC()
		}
		batch.Queue(`
			INSERT INTO time_slots (id, tenant_id, user_id, start_time_utc, end_time_utc, is_available, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT ON CONSTRAINT uq_user_start_time DO NOTHING
		`, s.ID, s.TenantID, s.UserID, s.StartTimeUTC, s.EndTimeUTC, s.IsAvailable, s.CreatedAt)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	var insertedCount int
	for i := 0; i < len(slots); i++ {
		ct, err := br.Exec()
		if err != nil {
			return insertedCount, err
		}
		insertedCount += int(ct.RowsAffected())
	}

	return insertedCount, nil
}

// --- ServiceRepository ---

func (r *Repository) List(ctx context.Context, tenantID uuid.UUID) ([]domain.Service, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, name, duration_mins, is_active, created_at
		FROM services
		WHERE tenant_id = $1 AND is_active = true
		ORDER BY duration_mins ASC, name ASC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []domain.Service
	for rows.Next() {
		var s domain.Service
		if err := rows.Scan(&s.ID, &s.TenantID, &s.Name, &s.DurationMins, &s.IsActive, &s.CreatedAt); err != nil {
			return nil, err
		}
		services = append(services, s)
	}
	return services, rows.Err()
}

func (r *Repository) GetServiceByID(ctx context.Context, id uuid.UUID) (*domain.Service, error) {
	var s domain.Service
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, name, duration_mins, is_active, created_at
		FROM services
		WHERE id = $1
	`, id).Scan(&s.ID, &s.TenantID, &s.Name, &s.DurationMins, &s.IsActive, &s.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *Repository) Create(ctx context.Context, service *domain.Service) error {
	service.ID = uuid.New()
	service.CreatedAt = time.Now().UTC()

	_, err := r.pool.Exec(ctx, `
		INSERT INTO services (id, tenant_id, name, duration_mins, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, service.ID, service.TenantID, service.Name, service.DurationMins, service.IsActive, service.CreatedAt)
	return err
}

func (r *Repository) Update(ctx context.Context, service *domain.Service) error {
	ct, err := r.pool.Exec(ctx, `
		UPDATE services
		SET name = $1, duration_mins = $2, is_active = $3
		WHERE id = $4
	`, service.Name, service.DurationMins, service.IsActive, service.ID)

	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// --- UserRepository ---

func (r *Repository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, email, password_hash, name, created_at
		FROM users
		WHERE email = $1
	`, email).Scan(&u.ID, &u.TenantID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, email, password_hash, name, created_at
		FROM users
		WHERE id = $1
	`, id).Scan(&u.ID, &u.TenantID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

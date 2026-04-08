-- AgendadorPlus MVP - Initial Schema
-- All timestamps in UTC (TIMESTAMPTZ)
-- All IDs are UUIDv4

-- =============================================================
-- Tenant (Multi-tenant ready, 1 default record for MVP)
-- =============================================================
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =============================================================
-- Users (Profesionales / Dueños de la agenda)
-- =============================================================
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_tenant ON users(tenant_id);
CREATE INDEX idx_users_email ON users(email);

-- =============================================================
-- Services (Servicios ofrecidos por el profesional)
-- =============================================================
CREATE TABLE services (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    duration_mins INT NOT NULL CHECK (duration_mins > 0),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_services_tenant ON services(tenant_id);

-- =============================================================
-- Time Slots (Bloques de disponibilidad pre-calculados)
-- DECISIÓN TÉCNICA (MVP): Modelo de slots fijos.
-- Limitación asumida: no soporta duraciones variables ni buffers dinámicos.
-- En EE se migrará a modelo de rangos temporales con overlap check.
-- =============================================================
CREATE TABLE time_slots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    start_time_utc TIMESTAMPTZ NOT NULL,
    end_time_utc TIMESTAMPTZ NOT NULL,
    is_available BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraint: end must be after start
    CONSTRAINT chk_time_slot_range CHECK (end_time_utc > start_time_utc),
    -- Constraint: no duplicate slots for the same user at the same time
    CONSTRAINT uq_user_start_time UNIQUE (user_id, start_time_utc)
);

-- Optimized index for availability queries by tenant + date range
CREATE INDEX idx_time_slots_availability
    ON time_slots(tenant_id, user_id, start_time_utc)
    WHERE is_available = TRUE;

-- =============================================================
-- Appointments (Reservas reales)
-- DEUDA TÉCNICA (MVP): 1 slot = 1 cita (UNIQUE on time_slot_id).
-- Garantiza Row-Level Lock fácil con SELECT FOR UPDATE NOWAIT.
-- En EE, si se requiere multiplicity (clases grupales), se removerá UNIQUE.
-- =============================================================
CREATE TABLE appointments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    time_slot_id UUID NOT NULL REFERENCES time_slots(id) UNIQUE,
    service_id UUID NOT NULL REFERENCES services(id),
    client_name VARCHAR(255) NOT NULL,
    client_email VARCHAR(255) NOT NULL,
    client_timezone VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'confirmed',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_appointments_slot ON appointments(time_slot_id);
CREATE INDEX idx_appointments_status ON appointments(status);

-- =============================================================
-- Seed: Default tenant + admin user for MVP
-- Password: admin123 (bcrypt hash)
-- =============================================================
INSERT INTO tenants (id, name) VALUES
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'AgendadorPlus Default');

INSERT INTO users (id, tenant_id, email, password_hash, name) VALUES
    ('b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22',
     'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
     'admin@agendadorplus.com',
     -- bcrypt hash of 'admin123' (cost 10)
     '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
     'Admin AgendadorPlus');

-- Default service for quick testing
INSERT INTO services (tenant_id, name, duration_mins) VALUES
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Consulta General', 30),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Consulta Extendida', 60);

CREATE TABLE IF NOT EXISTS patients (
    id UUID PRIMARY KEY,
    full_name TEXT NOT NULL,
    phone TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    birth_date TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS appointments (
    id UUID PRIMARY KEY,
    patient_id UUID NOT NULL,
    doctor_id UUID NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('CREATED', 'COMPLETED', 'CANCELLED')),
    appointment_date TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS schedule_slots (
    id UUID PRIMARY KEY,
    doctor_id UUID NOT NULL,
    slot_date TIMESTAMPTZ NOT NULL,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    cabinet TEXT NOT NULL,
    is_booked BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS visits (
    id UUID PRIMARY KEY,
    appointment_id UUID NOT NULL,
    complaints TEXT NOT NULL,
    diagnosis TEXT NOT NULL DEFAULT '',
    recommendations TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL CHECK (status IN ('IN_PROGRESS', 'COMPLETED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS emergency_requests (
    id UUID PRIMARY KEY,
    patient_id UUID NOT NULL,
    doctor_id UUID NOT NULL,
    requested_date TIMESTAMPTZ NOT NULL,
    reason TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('WAITING_RESPONSE', 'SCHEDULED', 'NEEDS_REVIEW', 'CANCELLED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS reschedule_offers (
    id UUID PRIMARY KEY,
    emergency_request_id UUID NOT NULL,
    appointment_id UUID NOT NULL,
    patient_id UUID NOT NULL,
    doctor_id UUID NOT NULL,
    old_date TIMESTAMPTZ NOT NULL,
    proposed_date TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('PENDING', 'ACCEPTED', 'DECLINED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY,
    appointment_id UUID NOT NULL,
    amount_kopecks BIGINT NOT NULL CHECK (amount_kopecks > 0),
    currency TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL CHECK (status IN ('PENDING', 'WAITING_FOR_CAPTURE', 'SUCCEEDED', 'CANCELED')),
    provider TEXT NOT NULL,
    provider_payment_id TEXT NOT NULL,
    confirmation_url TEXT NOT NULL DEFAULT '',
    cancellation_reason TEXT NOT NULL DEFAULT '',
    test BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS payments_appointment_idx
    ON payments (appointment_id, created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS appointments_doctor_slot_unique
    ON appointments (doctor_id, appointment_date)
    WHERE status <> 'CANCELLED';

CREATE INDEX IF NOT EXISTS appointments_patient_idx
    ON appointments (patient_id);

CREATE INDEX IF NOT EXISTS schedule_slots_doctor_date_idx
    ON schedule_slots (doctor_id, slot_date, start_time);

CREATE UNIQUE INDEX IF NOT EXISTS visits_appointment_unique
    ON visits (appointment_id);

CREATE INDEX IF NOT EXISTS emergency_requests_status_idx
    ON emergency_requests (status, requested_date);

CREATE INDEX IF NOT EXISTS reschedule_offers_request_idx
    ON reschedule_offers (emergency_request_id, status);

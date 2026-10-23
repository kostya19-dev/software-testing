DELETE FROM reschedule_offers;
DELETE FROM emergency_requests;
DELETE FROM visits;
DELETE FROM appointments;
DELETE FROM schedule_slots;
DELETE FROM patients;

INSERT INTO patients (id, full_name, phone, email, birth_date, created_at)
VALUES
    ('11111111-1111-1111-1111-111111111111', 'Иванов Иван Иванович', '+79001234567', 'ivanov@example.com', '1990-05-15T00:00:00Z', NOW()),
    ('22222222-2222-2222-2222-222222222222', 'Петров Пётр Петрович', '+79007654321', 'petrov@example.com', '1985-03-20T00:00:00Z', NOW())
ON CONFLICT (id) DO UPDATE SET
    full_name = EXCLUDED.full_name,
    phone = EXCLUDED.phone,
    email = EXCLUDED.email,
    birth_date = EXCLUDED.birth_date;

INSERT INTO appointments (id, patient_id, doctor_id, status, appointment_date, created_at)
VALUES
    ('33333333-3333-3333-3333-333333333333', '11111111-1111-1111-1111-111111111111', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'CREATED', (CURRENT_DATE + 1) + TIME '09:00', NOW()),
    ('44444444-4444-4444-4444-444444444444', '22222222-2222-2222-2222-222222222222', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'COMPLETED', (CURRENT_DATE + 1) + TIME '10:00', NOW())
ON CONFLICT (id) DO UPDATE SET
    patient_id = EXCLUDED.patient_id,
    doctor_id = EXCLUDED.doctor_id,
    status = EXCLUDED.status,
    appointment_date = EXCLUDED.appointment_date;

INSERT INTO schedule_slots (id, doctor_id, slot_date, start_time, end_time, cabinet, is_booked)
VALUES
    ('55555555-5555-5555-5555-555555555555', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', CURRENT_DATE + 1, (CURRENT_DATE + 1) + TIME '09:00', (CURRENT_DATE + 1) + TIME '09:30', '101', TRUE),
    ('66666666-6666-6666-6666-666666666666', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', CURRENT_DATE + 1, (CURRENT_DATE + 1) + TIME '10:00', (CURRENT_DATE + 1) + TIME '10:30', '101', FALSE)
ON CONFLICT (id) DO UPDATE SET
    doctor_id = EXCLUDED.doctor_id,
    slot_date = EXCLUDED.slot_date,
    start_time = EXCLUDED.start_time,
    end_time = EXCLUDED.end_time,
    cabinet = EXCLUDED.cabinet,
    is_booked = EXCLUDED.is_booked;

INSERT INTO visits (id, appointment_id, complaints, diagnosis, recommendations, status, created_at)
VALUES
    ('77777777-7777-7777-7777-777777777777', '44444444-4444-4444-4444-444444444444', 'Головная боль, слабость', 'ОРВИ', 'Покой, питьевой режим, контроль температуры', 'COMPLETED', NOW())
ON CONFLICT (id) DO UPDATE SET
    appointment_id = EXCLUDED.appointment_id,
    complaints = EXCLUDED.complaints,
    diagnosis = EXCLUDED.diagnosis,
    recommendations = EXCLUDED.recommendations,
    status = EXCLUDED.status;

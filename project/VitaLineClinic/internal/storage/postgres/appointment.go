package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
)

const (
	tableAppointments = "appointments"
)

var appointmentColumns = []string{
	"id", "patient_id", "doctor_id", "status", "appointment_date", "created_at",
}

type AppointmentRepository struct {
	db *sql.DB
	sb sq.StatementBuilderType
}

func NewAppointmentRepository(db *sql.DB) *AppointmentRepository {
	return &AppointmentRepository{
		db: db,
		sb: psql.RunWith(db),
	}
}

func (r *AppointmentRepository) Save(ctx context.Context, app *domain.Appointment) error {
	ctx = ensureContext(ctx)
	if err := app.Validate(); err != nil {
		return err
	}

	_, err := r.sb.
		Insert(tableAppointments).
		Columns(appointmentColumns...).
		Values(app.ID, app.PatientID, app.DoctorID, string(app.Status), app.Date, createdAtOrNow(app.CreatedAt)).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("insert appointment: %w", err)
	}
	return nil
}

func (r *AppointmentRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Appointment, error) {
	ctx = ensureContext(ctx)
	row := r.sb.
		Select(appointmentColumns...).
		From(tableAppointments).
		Where(sq.Eq{"id": id}).
		QueryRowContext(ctx)
	return scanAppointment(row)
}

func (r *AppointmentRepository) FindAll(ctx context.Context) ([]*domain.Appointment, error) {
	ctx = ensureContext(ctx)
	rows, err := r.sb.
		Select(appointmentColumns...).
		From(tableAppointments).
		OrderBy("created_at", "id").
		QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("select appointments: %w", err)
	}
	defer rows.Close()

	result := make([]*domain.Appointment, 0)
	for rows.Next() {
		appointment, err := scanAppointment(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, appointment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate appointments: %w", err)
	}
	return result, nil
}

func (r *AppointmentRepository) FindByPatient(ctx context.Context, patientID uuid.UUID) ([]*domain.Appointment, error) {
	ctx = ensureContext(ctx)
	rows, err := r.sb.
		Select(appointmentColumns...).
		From(tableAppointments).
		Where(sq.Eq{"patient_id": patientID}).
		OrderBy("appointment_date", "id").
		QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("select appointments by patient: %w", err)
	}
	defer rows.Close()

	result := make([]*domain.Appointment, 0)
	for rows.Next() {
		appointment, err := scanAppointment(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, appointment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate appointments by patient: %w", err)
	}
	return result, nil
}

func (r *AppointmentRepository) FindActiveByDoctorAndDate(ctx context.Context, doctorID uuid.UUID, date time.Time) (*domain.Appointment, error) {
	ctx = ensureContext(ctx)
	row := r.sb.
		Select(appointmentColumns...).
		From(tableAppointments).
		Where(sq.Eq{"doctor_id": doctorID}).
		Where(sq.Eq{"appointment_date": date}).
		Where(sq.NotEq{"status": string(domain.StatusCancelled)}).
		Limit(1).
		QueryRowContext(ctx)
	return scanAppointment(row)
}

func (r *AppointmentRepository) CheckSlotBusy(ctx context.Context, doctorID uuid.UUID, date time.Time) (bool, error) {
	ctx = ensureContext(ctx)

	inner := psql.
		Select("1").
		From(tableAppointments).
		Where(sq.Eq{"doctor_id": doctorID}).
		Where(sq.Eq{"appointment_date": date}).
		Where(sq.NotEq{"status": string(domain.StatusCancelled)})

	innerSQL, innerArgs, err := inner.ToSql()
	if err != nil {
		return false, fmt.Errorf("build check slot busy: %w", err)
	}

	row := r.db.QueryRowContext(ctx, "SELECT EXISTS("+innerSQL+")", innerArgs...)

	var busy bool
	if err := row.Scan(&busy); err != nil {
		return false, fmt.Errorf("check slot busy: %w", err)
	}
	return busy, nil
}

func (r *AppointmentRepository) Update(ctx context.Context, app *domain.Appointment) error {
	ctx = ensureContext(ctx)
	result, err := r.sb.
		Update(tableAppointments).
		Set("patient_id", app.PatientID).
		Set("doctor_id", app.DoctorID).
		Set("status", string(app.Status)).
		Set("appointment_date", app.Date).
		Set("created_at", createdAtOrNow(app.CreatedAt)).
		Where(sq.Eq{"id": app.ID}).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("update appointment: %w", err)
	}
	affected, err := rowsAffected(result)
	if err != nil {
		return fmt.Errorf("update appointment rows: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *AppointmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx = ensureContext(ctx)
	result, err := r.sb.
		Delete(tableAppointments).
		Where(sq.Eq{"id": id}).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("delete appointment: %w", err)
	}
	affected, err := rowsAffected(result)
	if err != nil {
		return fmt.Errorf("delete appointment rows: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"

	"vitaline-clinic/internal/domain"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

type scanner interface {
	Scan(dest ...any) error
}

func ensureContext(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	return context.Background()
}

func mapScanError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}

func rowsAffected(result sql.Result) (int64, error) {
	if result == nil {
		return 0, nil
	}
	return result.RowsAffected()
}

func createdAtOrNow(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value
}

func scanPatient(s scanner) (*domain.Patient, error) {
	var patient domain.Patient
	if err := s.Scan(&patient.ID, &patient.FullName, &patient.Phone, &patient.Email, &patient.BirthDate, &patient.CreatedAt); err != nil {
		return nil, mapScanError(err)
	}
	return &patient, nil
}

func scanAppointment(s scanner) (*domain.Appointment, error) {
	var appointment domain.Appointment
	var status string
	if err := s.Scan(&appointment.ID, &appointment.PatientID, &appointment.DoctorID, &status, &appointment.Date, &appointment.CreatedAt); err != nil {
		return nil, mapScanError(err)
	}
	appointment.Status = domain.AppointmentStatus(status)
	return &appointment, nil
}

func scanScheduleSlot(s scanner) (*domain.ScheduleSlot, error) {
	var slot domain.ScheduleSlot
	if err := s.Scan(&slot.ID, &slot.DoctorID, &slot.Date, &slot.StartTime, &slot.EndTime, &slot.Cabinet, &slot.IsBooked); err != nil {
		return nil, mapScanError(err)
	}
	return &slot, nil
}

func scanVisit(s scanner) (*domain.Visit, error) {
	var visit domain.Visit
	var status string
	if err := s.Scan(&visit.ID, &visit.AppointmentID, &visit.Complaints, &visit.Diagnosis, &visit.Recommendations, &status, &visit.CreatedAt); err != nil {
		return nil, mapScanError(err)
	}
	visit.Status = domain.VisitStatus(status)
	return &visit, nil
}

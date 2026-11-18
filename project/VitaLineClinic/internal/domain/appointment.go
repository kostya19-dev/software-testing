package domain

import (
	"time"

	"github.com/google/uuid"
)

type AppointmentStatus string

const (
	StatusCreated   AppointmentStatus = "CREATED"
	StatusCompleted AppointmentStatus = "COMPLETED"
	StatusCancelled AppointmentStatus = "CANCELLED"
)

type Appointment struct {
	ID        uuid.UUID
	PatientID uuid.UUID
	DoctorID  uuid.UUID
	Status    AppointmentStatus
	Date      time.Time
	CreatedAt time.Time
}

func (a *Appointment) Validate() error {
	if a.PatientID == uuid.Nil {
		return NewValidationError("PatientID", "не может быть пустым")
	}
	if a.DoctorID == uuid.Nil {
		return NewValidationError("DoctorID", "не может быть пустым")
	}
	if a.Date.IsZero() {
		return NewValidationError("Date", "дата не может быть пустой")
	}
	if a.Status != "" && !isValidAppointmentStatus(a.Status) {
		return NewValidationError("Status", "недопустимый статус: "+string(a.Status))
	}
	return nil
}

func (a *Appointment) Cancel() error {
	if a.Status == StatusCancelled {
		return ErrInvalidStatus
	}
	if a.Status == StatusCompleted {
		return ErrInvalidStatus
	}
	a.Status = StatusCancelled
	return nil
}

func (a *Appointment) Complete() error {
	if a.Status != StatusCreated {
		return ErrInvalidStatus
	}
	a.Status = StatusCompleted
	return nil
}

func isValidAppointmentStatus(s AppointmentStatus) bool {
	return s == StatusCreated || s == StatusCompleted || s == StatusCancelled
}

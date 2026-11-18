package http

import (
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
)

type AppointmentDTO struct {
	ID        uuid.UUID                `json:"id"`
	PatientID uuid.UUID                `json:"patient_id"`
	DoctorID  uuid.UUID                `json:"doctor_id"`
	Status    domain.AppointmentStatus `json:"status"`
	Date      time.Time                `json:"date"`
	CreatedAt time.Time                `json:"created_at"`
}

func NewAppointmentDTO(app *domain.Appointment) AppointmentDTO {
	return AppointmentDTO{
		ID:        app.ID,
		PatientID: app.PatientID,
		DoctorID:  app.DoctorID,
		Status:    app.Status,
		Date:      app.Date,
		CreatedAt: app.CreatedAt,
	}
}

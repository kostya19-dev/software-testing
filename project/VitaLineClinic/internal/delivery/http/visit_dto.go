package http

import (
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
)

type VisitDTO struct {
	ID              uuid.UUID          `json:"id"`
	AppointmentID   uuid.UUID          `json:"appointment_id"`
	Complaints      string             `json:"complaints"`
	Diagnosis       string             `json:"diagnosis"`
	Recommendations string             `json:"recommendations"`
	Status          domain.VisitStatus `json:"status"`
	CreatedAt       time.Time          `json:"created_at"`
}

func NewVisitDTO(visit *domain.Visit) VisitDTO {
	return VisitDTO{
		ID:              visit.ID,
		AppointmentID:   visit.AppointmentID,
		Complaints:      visit.Complaints,
		Diagnosis:       visit.Diagnosis,
		Recommendations: visit.Recommendations,
		Status:          visit.Status,
		CreatedAt:       visit.CreatedAt,
	}
}

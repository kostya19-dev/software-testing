package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type VisitStatus string

const (
	VisitStatusInProgress VisitStatus = "IN_PROGRESS"
	VisitStatusCompleted  VisitStatus = "COMPLETED"
)

type Visit struct {
	ID              uuid.UUID
	AppointmentID   uuid.UUID
	Complaints      string
	Diagnosis       string
	Recommendations string
	Status          VisitStatus
	CreatedAt       time.Time
}

func (v *Visit) Validate() error {
	if v.AppointmentID == uuid.Nil {
		return NewValidationError("AppointmentID", "не может быть пустым")
	}
	if strings.TrimSpace(v.Complaints) == "" {
		return NewValidationError("Complaints", "жалобы не могут быть пустыми")
	}
	if v.Status != "" && !isValidVisitStatus(v.Status) {
		return NewValidationError("Status", "недопустимый статус: "+string(v.Status))
	}
	return nil
}

func (v *Visit) Complete(diagnosis, recommendations string) error {
	if v.Status == VisitStatusCompleted {
		return ErrInvalidStatus
	}
	if strings.TrimSpace(diagnosis) == "" {
		return NewValidationError("Diagnosis", "диагноз не может быть пустым при завершении")
	}
	if strings.TrimSpace(recommendations) == "" {
		return NewValidationError("Recommendations", "рекомендации не могут быть пустыми при завершении")
	}
	v.Diagnosis = diagnosis
	v.Recommendations = recommendations
	v.Status = VisitStatusCompleted
	return nil
}

func isValidVisitStatus(s VisitStatus) bool {
	return s == VisitStatusInProgress || s == VisitStatusCompleted
}

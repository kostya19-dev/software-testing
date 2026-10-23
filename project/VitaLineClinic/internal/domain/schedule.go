package domain

import (
	"time"

	"github.com/google/uuid"
)

type ScheduleSlot struct {
	ID        uuid.UUID
	DoctorID  uuid.UUID
	Date      time.Time
	StartTime time.Time
	EndTime   time.Time
	Cabinet   string
	IsBooked  bool
}

func (s *ScheduleSlot) Validate() error {
	if s.DoctorID == uuid.Nil {
		return NewValidationError("DoctorID", "не может быть пустым")
	}
	if s.StartTime.IsZero() {
		return NewValidationError("StartTime", "время начала не может быть пустым")
	}
	if s.EndTime.IsZero() {
		return NewValidationError("EndTime", "время окончания не может быть пустым")
	}
	if !s.EndTime.After(s.StartTime) {
		return NewValidationError("EndTime", "время окончания должно быть позже времени начала")
	}
	return nil
}

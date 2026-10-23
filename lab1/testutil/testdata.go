package testutil

import (
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
)

var (
	ID        = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	PatientID = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	DoctorID  = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	Date      = time.Date(2099, 5, 10, 10, 0, 0, 0, time.UTC)
)

func Patient() *domain.Patient {
	return &domain.Patient{
		ID:        ID,
		FullName:  "Иван Иванов",
		Phone:     "+79001234567",
		Email:     "ivan@example.com",
		BirthDate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		CreatedAt: Date,
	}
}

func Appointment() *domain.Appointment {
	return &domain.Appointment{
		ID:        ID,
		PatientID: PatientID,
		DoctorID:  DoctorID,
		Date:      Date,
		Status:    domain.StatusCreated,
		CreatedAt: Date,
	}
}

func Schedule() *domain.ScheduleSlot {
	return &domain.ScheduleSlot{
		ID:        ID,
		DoctorID:  DoctorID,
		Date:      Date,
		StartTime: Date,
		EndTime:   Date.Add(30 * time.Minute),
		Cabinet:   "101",
	}
}

func Visit() *domain.Visit {
	return &domain.Visit{
		ID:            ID,
		AppointmentID: ID,
		Complaints:    "Боль",
		Status:        domain.VisitStatusInProgress,
		CreatedAt:     Date,
	}
}

func EmergencyRequest() *domain.EmergencyRequest {
	return &domain.EmergencyRequest{
		ID:            ID,
		PatientID:     PatientID,
		DoctorID:      DoctorID,
		RequestedDate: Date,
		Reason:        "Боль",
		Status:        domain.EmergencyStatusWaitingResponse,
		CreatedAt:     Date,
	}
}

func RescheduleOffer() *domain.RescheduleOffer {
	return &domain.RescheduleOffer{
		ID:                 ID,
		EmergencyRequestID: ID,
		AppointmentID:      ID,
		PatientID:          PatientID,
		DoctorID:           DoctorID,
		OldDate:            Date,
		ProposedDate:       Date.Add(30 * time.Minute),
		Status:             domain.OfferStatusPending,
		CreatedAt:          Date,
	}
}

type PatientBuilder struct {
	value domain.Patient
}

func NewPatientBuilder() *PatientBuilder {
	return &PatientBuilder{value: *Patient()}
}

func (b *PatientBuilder) WithPhone(phone string) *PatientBuilder {
	b.value.Phone = phone
	return b
}

func (b *PatientBuilder) Build() *domain.Patient {
	value := b.value
	return &value
}

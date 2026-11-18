package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type EmergencyRequestStatus string

const (
	EmergencyStatusWaitingResponse EmergencyRequestStatus = "WAITING_RESPONSE"
	EmergencyStatusScheduled       EmergencyRequestStatus = "SCHEDULED"
	EmergencyStatusNeedsReview     EmergencyRequestStatus = "NEEDS_REVIEW"
	EmergencyStatusCancelled       EmergencyRequestStatus = "CANCELLED"
)

type RescheduleOfferStatus string

const (
	OfferStatusPending  RescheduleOfferStatus = "PENDING"
	OfferStatusAccepted RescheduleOfferStatus = "ACCEPTED"
	OfferStatusDeclined RescheduleOfferStatus = "DECLINED"
)

type EmergencyRequest struct {
	ID            uuid.UUID
	PatientID     uuid.UUID
	DoctorID      uuid.UUID
	RequestedDate time.Time
	Reason        string
	Status        EmergencyRequestStatus
	CreatedAt     time.Time
}

func (r *EmergencyRequest) Validate() error {
	if r.PatientID == uuid.Nil {
		return NewValidationError("PatientID", "не может быть пустым")
	}
	if r.DoctorID == uuid.Nil {
		return NewValidationError("DoctorID", "не может быть пустым")
	}
	if r.RequestedDate.IsZero() {
		return NewValidationError("RequestedDate", "дата срочной записи не может быть пустой")
	}
	if strings.TrimSpace(r.Reason) == "" {
		return NewValidationError("Reason", "причина срочной записи не может быть пустой")
	}
	if r.Status != "" && !isValidEmergencyStatus(r.Status) {
		return NewValidationError("Status", "недопустимый статус: "+string(r.Status))
	}
	return nil
}

type RescheduleOffer struct {
	ID                 uuid.UUID
	EmergencyRequestID uuid.UUID
	AppointmentID      uuid.UUID
	PatientID          uuid.UUID
	DoctorID           uuid.UUID
	OldDate            time.Time
	ProposedDate       time.Time
	Status             RescheduleOfferStatus
	CreatedAt          time.Time
}

func (o *RescheduleOffer) Validate() error {
	if o.EmergencyRequestID == uuid.Nil {
		return NewValidationError("EmergencyRequestID", "не может быть пустым")
	}
	if o.AppointmentID == uuid.Nil {
		return NewValidationError("AppointmentID", "не может быть пустым")
	}
	if o.PatientID == uuid.Nil {
		return NewValidationError("PatientID", "не может быть пустым")
	}
	if o.DoctorID == uuid.Nil {
		return NewValidationError("DoctorID", "не может быть пустым")
	}
	if o.OldDate.IsZero() {
		return NewValidationError("OldDate", "старое время записи не может быть пустым")
	}
	if o.ProposedDate.IsZero() {
		return NewValidationError("ProposedDate", "новое время записи не может быть пустым")
	}
	if o.Status != "" && !isValidOfferStatus(o.Status) {
		return NewValidationError("Status", "недопустимый статус: "+string(o.Status))
	}
	return nil
}

func (o *RescheduleOffer) Accept() error {
	if o.Status != OfferStatusPending {
		return ErrInvalidStatus
	}
	o.Status = OfferStatusAccepted
	return nil
}

func (o *RescheduleOffer) Decline() error {
	if o.Status != OfferStatusPending {
		return ErrInvalidStatus
	}
	o.Status = OfferStatusDeclined
	return nil
}

func isValidEmergencyStatus(status EmergencyRequestStatus) bool {
	return status == EmergencyStatusWaitingResponse ||
		status == EmergencyStatusScheduled ||
		status == EmergencyStatusNeedsReview ||
		status == EmergencyStatusCancelled
}

func isValidOfferStatus(status RescheduleOfferStatus) bool {
	return status == OfferStatusPending ||
		status == OfferStatusAccepted ||
		status == OfferStatusDeclined
}

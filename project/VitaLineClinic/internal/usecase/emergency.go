package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
)

type EmergencyRepository interface {
	SaveRequest(ctx context.Context, request *domain.EmergencyRequest) error
	FindRequestByID(ctx context.Context, id uuid.UUID) (*domain.EmergencyRequest, error)
	FindAllRequests(ctx context.Context) ([]*domain.EmergencyRequest, error)
	UpdateRequest(ctx context.Context, request *domain.EmergencyRequest) error
	SaveOffer(ctx context.Context, offer *domain.RescheduleOffer) error
	FindOfferByID(ctx context.Context, id uuid.UUID) (*domain.RescheduleOffer, error)
	FindAllOffers(ctx context.Context) ([]*domain.RescheduleOffer, error)
	UpdateOffer(ctx context.Context, offer *domain.RescheduleOffer) error
}

type CreateEmergencyRequestDTO struct {
	PatientID     uuid.UUID
	DoctorID      uuid.UUID
	RequestedDate time.Time
	Reason        string
}

func (d *CreateEmergencyRequestDTO) Validate() error {
	if d.PatientID == uuid.Nil {
		return domain.NewValidationError("PatientID", "не может быть пустым")
	}
	if d.DoctorID == uuid.Nil {
		return domain.NewValidationError("DoctorID", "не может быть пустым")
	}
	if d.RequestedDate.IsZero() {
		return domain.NewValidationError("RequestedDate", "дата срочной записи не может быть пустой")
	}
	if strings.TrimSpace(d.Reason) == "" {
		return domain.NewValidationError("Reason", "причина срочности не может быть пустой")
	}
	return nil
}

type EmergencyRequestResponseDTO struct {
	ID            uuid.UUID
	PatientID     uuid.UUID
	DoctorID      uuid.UUID
	RequestedDate time.Time
	Reason        string
	Status        domain.EmergencyRequestStatus
	CreatedAt     time.Time
}

type RescheduleOfferResponseDTO struct {
	ID                 uuid.UUID
	EmergencyRequestID uuid.UUID
	AppointmentID      uuid.UUID
	PatientID          uuid.UUID
	DoctorID           uuid.UUID
	OldDate            time.Time
	ProposedDate       time.Time
	Status             domain.RescheduleOfferStatus
	CreatedAt          time.Time
}

type IEmergencyService interface {
	Create(ctx context.Context, dto CreateEmergencyRequestDTO) (EmergencyRequestResponseDTO, error)
	GetRequests(ctx context.Context) ([]EmergencyRequestResponseDTO, error)
	GetOffers(ctx context.Context) ([]RescheduleOfferResponseDTO, error)
	AcceptOffer(ctx context.Context, offerID uuid.UUID) (RescheduleOfferResponseDTO, error)
	DeclineOffer(ctx context.Context, offerID uuid.UUID) (RescheduleOfferResponseDTO, error)
}

type EmergencyService struct {
	repo            EmergencyRepository
	appointmentRepo AppointmentRepository
}

func NewEmergencyService(repo EmergencyRepository, appointmentRepo AppointmentRepository) *EmergencyService {
	return &EmergencyService{
		repo:            repo,
		appointmentRepo: appointmentRepo,
	}
}

func (s *EmergencyService) Create(ctx context.Context, dto CreateEmergencyRequestDTO) (EmergencyRequestResponseDTO, error) {
	if err := dto.Validate(); err != nil {
		return EmergencyRequestResponseDTO{}, err
	}

	now := time.Now().UTC()
	request := &domain.EmergencyRequest{
		ID:            uuid.New(),
		PatientID:     dto.PatientID,
		DoctorID:      dto.DoctorID,
		RequestedDate: dto.RequestedDate,
		Reason:        strings.TrimSpace(dto.Reason),
		Status:        domain.EmergencyStatusScheduled,
		CreatedAt:     now,
	}

	occupied, err := s.appointmentRepo.FindActiveByDoctorAndDate(ctx, dto.DoctorID, dto.RequestedDate)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return EmergencyRequestResponseDTO{}, err
	}

	if errors.Is(err, domain.ErrNotFound) {
		if err := s.repo.SaveRequest(ctx, request); err != nil {
			return EmergencyRequestResponseDTO{}, err
		}
		appointment := &domain.Appointment{
			ID:        uuid.New(),
			PatientID: dto.PatientID,
			DoctorID:  dto.DoctorID,
			Status:    domain.StatusCreated,
			Date:      dto.RequestedDate,
			CreatedAt: now,
		}
		if err := s.appointmentRepo.Save(ctx, appointment); err != nil {
			return EmergencyRequestResponseDTO{}, err
		}
		return toEmergencyRequestResponse(request), nil
	}

	request.Status = domain.EmergencyStatusWaitingResponse
	if err := s.repo.SaveRequest(ctx, request); err != nil {
		return EmergencyRequestResponseDTO{}, err
	}

	proposedDate, err := s.findProposedDate(ctx, dto.DoctorID, dto.RequestedDate)
	if err != nil {
		return EmergencyRequestResponseDTO{}, err
	}

	offer := &domain.RescheduleOffer{
		ID:                 uuid.New(),
		EmergencyRequestID: request.ID,
		AppointmentID:      occupied.ID,
		PatientID:          occupied.PatientID,
		DoctorID:           occupied.DoctorID,
		OldDate:            occupied.Date,
		ProposedDate:       proposedDate,
		Status:             domain.OfferStatusPending,
		CreatedAt:          now,
	}
	if err := s.repo.SaveOffer(ctx, offer); err != nil {
		return EmergencyRequestResponseDTO{}, err
	}

	return toEmergencyRequestResponse(request), nil
}

func (s *EmergencyService) GetRequests(ctx context.Context) ([]EmergencyRequestResponseDTO, error) {
	requests, err := s.repo.FindAllRequests(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]EmergencyRequestResponseDTO, 0, len(requests))
	for _, request := range requests {
		result = append(result, toEmergencyRequestResponse(request))
	}
	return result, nil
}

func (s *EmergencyService) GetOffers(ctx context.Context) ([]RescheduleOfferResponseDTO, error) {
	offers, err := s.repo.FindAllOffers(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]RescheduleOfferResponseDTO, 0, len(offers))
	for _, offer := range offers {
		result = append(result, toRescheduleOfferResponse(offer))
	}
	return result, nil
}

func (s *EmergencyService) AcceptOffer(ctx context.Context, offerID uuid.UUID) (RescheduleOfferResponseDTO, error) {
	if offerID == uuid.Nil {
		return RescheduleOfferResponseDTO{}, domain.NewValidationError("OfferID", "не может быть пустым")
	}

	offer, err := s.repo.FindOfferByID(ctx, offerID)
	if err != nil {
		return RescheduleOfferResponseDTO{}, err
	}
	request, err := s.repo.FindRequestByID(ctx, offer.EmergencyRequestID)
	if err != nil {
		return RescheduleOfferResponseDTO{}, err
	}
	appointment, err := s.appointmentRepo.FindByID(ctx, offer.AppointmentID)
	if err != nil {
		return RescheduleOfferResponseDTO{}, err
	}

	if err := offer.Accept(); err != nil {
		return RescheduleOfferResponseDTO{}, err
	}

	appointment.Date = offer.ProposedDate
	if err := s.appointmentRepo.Update(ctx, appointment); err != nil {
		return RescheduleOfferResponseDTO{}, err
	}

	emergencyAppointment := &domain.Appointment{
		ID:        uuid.New(),
		PatientID: request.PatientID,
		DoctorID:  request.DoctorID,
		Status:    domain.StatusCreated,
		Date:      request.RequestedDate,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.appointmentRepo.Save(ctx, emergencyAppointment); err != nil {
		return RescheduleOfferResponseDTO{}, err
	}

	request.Status = domain.EmergencyStatusScheduled
	if err := s.repo.UpdateRequest(ctx, request); err != nil {
		return RescheduleOfferResponseDTO{}, err
	}
	if err := s.repo.UpdateOffer(ctx, offer); err != nil {
		return RescheduleOfferResponseDTO{}, err
	}

	return toRescheduleOfferResponse(offer), nil
}

func (s *EmergencyService) DeclineOffer(ctx context.Context, offerID uuid.UUID) (RescheduleOfferResponseDTO, error) {
	if offerID == uuid.Nil {
		return RescheduleOfferResponseDTO{}, domain.NewValidationError("OfferID", "не может быть пустым")
	}
	offer, err := s.repo.FindOfferByID(ctx, offerID)
	if err != nil {
		return RescheduleOfferResponseDTO{}, err
	}
	request, err := s.repo.FindRequestByID(ctx, offer.EmergencyRequestID)
	if err != nil {
		return RescheduleOfferResponseDTO{}, err
	}
	if err := offer.Decline(); err != nil {
		return RescheduleOfferResponseDTO{}, err
	}
	request.Status = domain.EmergencyStatusNeedsReview
	if err := s.repo.UpdateOffer(ctx, offer); err != nil {
		return RescheduleOfferResponseDTO{}, err
	}
	if err := s.repo.UpdateRequest(ctx, request); err != nil {
		return RescheduleOfferResponseDTO{}, err
	}
	return toRescheduleOfferResponse(offer), nil
}

func (s *EmergencyService) findProposedDate(ctx context.Context, doctorID uuid.UUID, requestedDate time.Time) (time.Time, error) {
	candidate := requestedDate.Add(30 * time.Minute)
	for i := 0; i < 14*16; i++ {
		if candidate.Hour() < 9 {
			candidate = time.Date(candidate.Year(), candidate.Month(), candidate.Day(), 9, 0, 0, 0, candidate.Location())
		}
		if candidate.Hour() >= 17 {
			nextDay := candidate.AddDate(0, 0, 1)
			candidate = time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 9, 0, 0, 0, candidate.Location())
			continue
		}
		busy, err := s.appointmentRepo.CheckSlotBusy(ctx, doctorID, candidate)
		if err != nil {
			return time.Time{}, err
		}
		if !busy {
			return candidate, nil
		}
		candidate = candidate.Add(30 * time.Minute)
	}
	return time.Time{}, domain.NewValidationError("ProposedDate", "не найдено свободное время для переноса")
}

func toEmergencyRequestResponse(r *domain.EmergencyRequest) EmergencyRequestResponseDTO {
	return EmergencyRequestResponseDTO{
		ID:            r.ID,
		PatientID:     r.PatientID,
		DoctorID:      r.DoctorID,
		RequestedDate: r.RequestedDate,
		Reason:        r.Reason,
		Status:        r.Status,
		CreatedAt:     r.CreatedAt,
	}
}

func toRescheduleOfferResponse(o *domain.RescheduleOffer) RescheduleOfferResponseDTO {
	return RescheduleOfferResponseDTO{
		ID:                 o.ID,
		EmergencyRequestID: o.EmergencyRequestID,
		AppointmentID:      o.AppointmentID,
		PatientID:          o.PatientID,
		DoctorID:           o.DoctorID,
		OldDate:            o.OldDate,
		ProposedDate:       o.ProposedDate,
		Status:             o.Status,
		CreatedAt:          o.CreatedAt,
	}
}

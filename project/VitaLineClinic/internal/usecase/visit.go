package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
)

type CreateVisitDTO struct {
	AppointmentID uuid.UUID
	Complaints    string
}

func (d *CreateVisitDTO) Validate() error {
	if d.AppointmentID == uuid.Nil {
		return domain.NewValidationError("AppointmentID", "не может быть пустым")
	}
	if strings.TrimSpace(d.Complaints) == "" {
		return domain.NewValidationError("Complaints", "жалобы не могут быть пустыми")
	}
	return nil
}

type UpdateVisitDTO struct {
	Diagnosis       string
	Recommendations string
}

func (d *UpdateVisitDTO) Validate() error {
	if strings.TrimSpace(d.Diagnosis) == "" {
		return domain.NewValidationError("Diagnosis", "диагноз не может быть пустым")
	}
	if strings.TrimSpace(d.Recommendations) == "" {
		return domain.NewValidationError("Recommendations", "рекомендации не могут быть пустыми")
	}
	return nil
}

type VisitResponseDTO struct {
	ID              uuid.UUID
	AppointmentID   uuid.UUID
	Complaints      string
	Diagnosis       string
	Recommendations string
	Status          domain.VisitStatus
}

type VisitRepository interface {
	Save(ctx context.Context, visit *domain.Visit) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Visit, error)
	FindByAppointment(ctx context.Context, appointmentID uuid.UUID) (*domain.Visit, error)
	Update(ctx context.Context, visit *domain.Visit) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type IVisitService interface {
	Create(ctx context.Context, dto CreateVisitDTO) (VisitResponseDTO, error)
	Complete(ctx context.Context, visitID uuid.UUID, dto UpdateVisitDTO) (VisitResponseDTO, error)
	GetByAppointment(ctx context.Context, appointmentID uuid.UUID) (VisitResponseDTO, error)
	GetByPatient(ctx context.Context, patientID uuid.UUID) ([]VisitResponseDTO, error)
}

type VisitService struct {
	repo    VisitRepository
	appRepo AppointmentRepository
}

func NewVisitService(r VisitRepository, ar AppointmentRepository) *VisitService {
	return &VisitService{repo: r, appRepo: ar}
}

func (s *VisitService) Create(ctx context.Context, dto CreateVisitDTO) (VisitResponseDTO, error) {
	if err := dto.Validate(); err != nil {
		return VisitResponseDTO{}, err
	}

	app, err := s.appRepo.FindByID(ctx, dto.AppointmentID)
	if err != nil {
		return VisitResponseDTO{}, err
	}
	if app.Status != domain.StatusCreated {
		return VisitResponseDTO{}, domain.ErrInvalidStatus
	}

	visit := &domain.Visit{
		ID:            uuid.New(),
		AppointmentID: dto.AppointmentID,
		Complaints:    dto.Complaints,
		Status:        domain.VisitStatusInProgress,
		CreatedAt:     time.Now(),
	}

	if err := visit.Validate(); err != nil {
		return VisitResponseDTO{}, err
	}

	if err := s.repo.Save(ctx, visit); err != nil {
		return VisitResponseDTO{}, err
	}

	if err := app.Complete(); err != nil {
		return VisitResponseDTO{}, err
	}
	if err := s.appRepo.Update(ctx, app); err != nil {
		return VisitResponseDTO{}, err
	}

	return toVisitResponse(visit), nil
}

func (s *VisitService) Complete(ctx context.Context, visitID uuid.UUID, dto UpdateVisitDTO) (VisitResponseDTO, error) {
	if err := dto.Validate(); err != nil {
		return VisitResponseDTO{}, err
	}

	visit, err := s.repo.FindByID(ctx, visitID)
	if err != nil {
		return VisitResponseDTO{}, err
	}

	if err := visit.Complete(dto.Diagnosis, dto.Recommendations); err != nil {
		return VisitResponseDTO{}, err
	}

	if err := s.repo.Update(ctx, visit); err != nil {
		return VisitResponseDTO{}, err
	}

	return toVisitResponse(visit), nil
}

func (s *VisitService) GetByAppointment(ctx context.Context, appointmentID uuid.UUID) (VisitResponseDTO, error) {
	visit, err := s.repo.FindByAppointment(ctx, appointmentID)
	if err != nil {
		return VisitResponseDTO{}, err
	}
	return toVisitResponse(visit), nil
}

func (s *VisitService) GetByPatient(ctx context.Context, patientID uuid.UUID) ([]VisitResponseDTO, error) {
	if patientID == uuid.Nil {
		return nil, domain.NewValidationError("patientID", "не может быть пустым")
	}

	appointments, err := s.appRepo.FindByPatient(ctx, patientID)
	if err != nil {
		return nil, err
	}

	result := make([]VisitResponseDTO, 0, len(appointments))
	for _, app := range appointments {
		visit, err := s.repo.FindByAppointment(ctx, app.ID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				continue
			}
			return nil, err
		}
		result = append(result, toVisitResponse(visit))
	}
	return result, nil
}

func toVisitResponse(v *domain.Visit) VisitResponseDTO {
	return VisitResponseDTO{
		ID:              v.ID,
		AppointmentID:   v.AppointmentID,
		Complaints:      v.Complaints,
		Diagnosis:       v.Diagnosis,
		Recommendations: v.Recommendations,
		Status:          v.Status,
	}
}

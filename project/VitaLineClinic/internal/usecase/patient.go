package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
)

type PatientRepository interface {
	Save(ctx context.Context, patient *domain.Patient) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Patient, error)
	FindAll(ctx context.Context) ([]*domain.Patient, error)
	FindByEmail(ctx context.Context, email string) (*domain.Patient, error)
	Update(ctx context.Context, patient *domain.Patient) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type CreatePatientDTO struct {
	FullName  string
	Phone     string
	Email     string
	BirthDate time.Time
}

func (d *CreatePatientDTO) Validate() error {
	if strings.TrimSpace(d.FullName) == "" {
		return domain.NewValidationError("FullName", "ФИО не может быть пустым")
	}
	if strings.TrimSpace(d.Phone) == "" {
		return domain.NewValidationError("Phone", "телефон не может быть пустым")
	}
	if strings.TrimSpace(d.Email) == "" {
		return domain.NewValidationError("Email", "email не может быть пустым")
	}
	if d.BirthDate.IsZero() {
		return domain.NewValidationError("BirthDate", "дата рождения не может быть пустой")
	}
	return nil
}

type PatientResponseDTO struct {
	ID        uuid.UUID
	FullName  string
	Phone     string
	Email     string
	BirthDate time.Time
	CreatedAt time.Time
}

type IPatientService interface {
	Register(ctx context.Context, dto CreatePatientDTO) (PatientResponseDTO, error)
	GetByID(ctx context.Context, id uuid.UUID) (PatientResponseDTO, error)
	GetAll(ctx context.Context) ([]PatientResponseDTO, error)
	FindByEmail(ctx context.Context, email string) (PatientResponseDTO, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type PatientService struct {
	repo PatientRepository
}

func NewPatientService(r PatientRepository) *PatientService {
	return &PatientService{repo: r}
}

func (s *PatientService) Register(ctx context.Context, dto CreatePatientDTO) (PatientResponseDTO, error) {
	if err := dto.Validate(); err != nil {
		return PatientResponseDTO{}, err
	}

	patient := &domain.Patient{
		ID:        uuid.New(),
		FullName:  dto.FullName,
		Phone:     dto.Phone,
		Email:     dto.Email,
		BirthDate: dto.BirthDate,
		CreatedAt: time.Now(),
	}

	if err := patient.Validate(); err != nil {
		return PatientResponseDTO{}, err
	}

	if err := s.repo.Save(ctx, patient); err != nil {
		return PatientResponseDTO{}, err
	}

	return toPatientResponse(patient), nil
}

func (s *PatientService) GetByID(ctx context.Context, id uuid.UUID) (PatientResponseDTO, error) {
	if id == uuid.Nil {
		return PatientResponseDTO{}, domain.NewValidationError("ID", "не может быть пустым")
	}
	patient, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return PatientResponseDTO{}, err
	}
	return toPatientResponse(patient), nil
}

func (s *PatientService) GetAll(ctx context.Context) ([]PatientResponseDTO, error) {
	patients, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]PatientResponseDTO, 0, len(patients))
	for _, p := range patients {
		result = append(result, toPatientResponse(p))
	}
	return result, nil
}

func (s *PatientService) FindByEmail(ctx context.Context, email string) (PatientResponseDTO, error) {
	if strings.TrimSpace(email) == "" {
		return PatientResponseDTO{}, domain.NewValidationError("Email", "не может быть пустым")
	}
	patient, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return PatientResponseDTO{}, err
	}
	return toPatientResponse(patient), nil
}

func (s *PatientService) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return domain.NewValidationError("ID", "не может быть пустым")
	}
	return s.repo.Delete(ctx, id)
}

func toPatientResponse(p *domain.Patient) PatientResponseDTO {
	return PatientResponseDTO{
		ID:        p.ID,
		FullName:  p.FullName,
		Phone:     p.Phone,
		Email:     p.Email,
		BirthDate: p.BirthDate,
		CreatedAt: p.CreatedAt,
	}
}

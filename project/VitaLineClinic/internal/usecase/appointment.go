package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
)

type CreateAppointmentDTO struct {
	PatientID uuid.UUID
	DoctorID  uuid.UUID
	Date      time.Time
}

func (d *CreateAppointmentDTO) Validate() error {
	if d.PatientID == uuid.Nil {
		return domain.NewValidationError("PatientID", "не может быть пустым")
	}
	if d.DoctorID == uuid.Nil {
		return domain.NewValidationError("DoctorID", "не может быть пустым")
	}
	if d.Date.IsZero() {
		return domain.NewValidationError("Date", "дата не может быть пустой")
	}
	return nil
}

type AppointmentResponseDTO struct {
	ID        uuid.UUID
	PatientID uuid.UUID
	DoctorID  uuid.UUID
	Status    domain.AppointmentStatus
	Date      time.Time
}

type AppointmentRepository interface {
	Save(ctx context.Context, app *domain.Appointment) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Appointment, error)
	FindAll(ctx context.Context) ([]*domain.Appointment, error)
	FindByPatient(ctx context.Context, patientID uuid.UUID) ([]*domain.Appointment, error)
	FindActiveByDoctorAndDate(ctx context.Context, doctorID uuid.UUID, date time.Time) (*domain.Appointment, error)
	CheckSlotBusy(ctx context.Context, doctorID uuid.UUID, date time.Time) (bool, error)
	Update(ctx context.Context, app *domain.Appointment) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type IAppointmentService interface {
	Create(ctx context.Context, dto CreateAppointmentDTO) (AppointmentResponseDTO, error)
	Cancel(ctx context.Context, appointmentID uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (AppointmentResponseDTO, error)
	GetAll(ctx context.Context) ([]AppointmentResponseDTO, error)
	GetByPatient(ctx context.Context, patientID uuid.UUID) ([]AppointmentResponseDTO, error)
}

type AppointmentService struct {
	repo        AppointmentRepository
	minLeadTime time.Duration
}

type AppointmentServiceOption func(*AppointmentService)

func WithAppointmentMinLeadTime(d time.Duration) AppointmentServiceOption {
	return func(s *AppointmentService) {
		if d > 0 {
			s.minLeadTime = d
		}
	}
}

func NewAppointmentService(r AppointmentRepository, opts ...AppointmentServiceOption) *AppointmentService {
	service := &AppointmentService{repo: r}
	for _, opt := range opts {
		opt(service)
	}
	return service
}

func (s *AppointmentService) Create(ctx context.Context, dto CreateAppointmentDTO) (AppointmentResponseDTO, error) {
	if err := dto.Validate(); err != nil {
		return AppointmentResponseDTO{}, err
	}

	if dto.Date.Before(time.Now()) {
		return AppointmentResponseDTO{}, domain.ErrPastDate
	}
	if s.minLeadTime > 0 && time.Until(dto.Date) < s.minLeadTime {
		return AppointmentResponseDTO{}, domain.ErrAppointmentLeadTime
	}

	isBusy, err := s.repo.CheckSlotBusy(ctx, dto.DoctorID, dto.Date)
	if err != nil {
		return AppointmentResponseDTO{}, err
	}
	if isBusy {
		return AppointmentResponseDTO{}, domain.ErrSlotBusy
	}

	app := &domain.Appointment{
		ID:        uuid.New(),
		PatientID: dto.PatientID,
		DoctorID:  dto.DoctorID,
		Status:    domain.StatusCreated,
		Date:      dto.Date,
		CreatedAt: time.Now(),
	}

	if err := app.Validate(); err != nil {
		return AppointmentResponseDTO{}, err
	}

	if err := s.repo.Save(ctx, app); err != nil {
		return AppointmentResponseDTO{}, err
	}

	return toAppointmentResponse(app), nil
}

func (s *AppointmentService) Cancel(ctx context.Context, appointmentID uuid.UUID) error {
	app, err := s.repo.FindByID(ctx, appointmentID)
	if err != nil {
		return err
	}

	if err := app.Cancel(); err != nil {
		return err
	}

	return s.repo.Update(ctx, app)
}

func (s *AppointmentService) GetByID(ctx context.Context, id uuid.UUID) (AppointmentResponseDTO, error) {
	app, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return AppointmentResponseDTO{}, err
	}
	return toAppointmentResponse(app), nil
}

func (s *AppointmentService) GetAll(ctx context.Context) ([]AppointmentResponseDTO, error) {
	apps, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]AppointmentResponseDTO, 0, len(apps))
	for _, app := range apps {
		result = append(result, toAppointmentResponse(app))
	}
	return result, nil
}

func (s *AppointmentService) GetByPatient(ctx context.Context, patientID uuid.UUID) ([]AppointmentResponseDTO, error) {
	if patientID == uuid.Nil {
		return nil, domain.NewValidationError("patientID", "не может быть пустым")
	}
	apps, err := s.repo.FindByPatient(ctx, patientID)
	if err != nil {
		return nil, err
	}
	result := make([]AppointmentResponseDTO, 0, len(apps))
	for _, app := range apps {
		result = append(result, toAppointmentResponse(app))
	}
	return result, nil
}

func toAppointmentResponse(a *domain.Appointment) AppointmentResponseDTO {
	return AppointmentResponseDTO{
		ID:        a.ID,
		PatientID: a.PatientID,
		DoctorID:  a.DoctorID,
		Status:    a.Status,
		Date:      a.Date,
	}
}

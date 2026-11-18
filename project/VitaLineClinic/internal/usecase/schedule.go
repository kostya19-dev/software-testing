package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
)

type ScheduleRepository interface {
	Save(ctx context.Context, slot *domain.ScheduleSlot) error
	FindByDoctorID(ctx context.Context, doctorID uuid.UUID) ([]*domain.ScheduleSlot, error)
	FindFreeSlots(ctx context.Context, doctorID uuid.UUID, date time.Time) ([]*domain.ScheduleSlot, error)
	Update(ctx context.Context, slot *domain.ScheduleSlot) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type CreateScheduleSlotDTO struct {
	DoctorID  uuid.UUID
	Date      time.Time
	StartTime time.Time
	EndTime   time.Time
	Cabinet   string
}

func (d *CreateScheduleSlotDTO) Validate() error {
	if d.DoctorID == uuid.Nil {
		return domain.NewValidationError("DoctorID", "не может быть пустым")
	}
	if d.Date.IsZero() {
		return domain.NewValidationError("Date", "дата не может быть пустой")
	}
	if d.StartTime.IsZero() {
		return domain.NewValidationError("StartTime", "время начала не может быть пустым")
	}
	if d.EndTime.IsZero() {
		return domain.NewValidationError("EndTime", "время окончания не может быть пустым")
	}
	if !d.EndTime.After(d.StartTime) {
		return domain.NewValidationError("EndTime", "время окончания должно быть позже времени начала")
	}
	if strings.TrimSpace(d.Cabinet) == "" {
		return domain.NewValidationError("Cabinet", "кабинет не может быть пустым")
	}
	return nil
}

type ScheduleSlotResponseDTO struct {
	ID        uuid.UUID
	DoctorID  uuid.UUID
	Date      time.Time
	StartTime time.Time
	EndTime   time.Time
	Cabinet   string
	IsBooked  bool
}

type IScheduleService interface {
	CreateSlot(ctx context.Context, dto CreateScheduleSlotDTO) (ScheduleSlotResponseDTO, error)
	GetByDoctor(ctx context.Context, doctorID uuid.UUID) ([]ScheduleSlotResponseDTO, error)
	GetFreeSlots(ctx context.Context, doctorID uuid.UUID, date time.Time) ([]ScheduleSlotResponseDTO, error)
	DeleteSlot(ctx context.Context, id uuid.UUID) error
}

type ScheduleService struct {
	repo ScheduleRepository
}

func NewScheduleService(r ScheduleRepository) *ScheduleService {
	return &ScheduleService{repo: r}
}

func (s *ScheduleService) CreateSlot(ctx context.Context, dto CreateScheduleSlotDTO) (ScheduleSlotResponseDTO, error) {
	if err := dto.Validate(); err != nil {
		return ScheduleSlotResponseDTO{}, err
	}

	slot := &domain.ScheduleSlot{
		ID:        uuid.New(),
		DoctorID:  dto.DoctorID,
		Date:      dto.Date,
		StartTime: dto.StartTime,
		EndTime:   dto.EndTime,
		Cabinet:   dto.Cabinet,
		IsBooked:  false,
	}

	if err := slot.Validate(); err != nil {
		return ScheduleSlotResponseDTO{}, err
	}

	if err := s.repo.Save(ctx, slot); err != nil {
		return ScheduleSlotResponseDTO{}, err
	}

	return toScheduleSlotResponse(slot), nil
}

func (s *ScheduleService) GetByDoctor(ctx context.Context, doctorID uuid.UUID) ([]ScheduleSlotResponseDTO, error) {
	if doctorID == uuid.Nil {
		return nil, domain.NewValidationError("DoctorID", "не может быть пустым")
	}
	slots, err := s.repo.FindByDoctorID(ctx, doctorID)
	if err != nil {
		return nil, err
	}
	result := make([]ScheduleSlotResponseDTO, 0, len(slots))
	for _, slot := range slots {
		result = append(result, toScheduleSlotResponse(slot))
	}
	return result, nil
}

func (s *ScheduleService) GetFreeSlots(ctx context.Context, doctorID uuid.UUID, date time.Time) ([]ScheduleSlotResponseDTO, error) {
	if doctorID == uuid.Nil {
		return nil, domain.NewValidationError("DoctorID", "не может быть пустым")
	}
	if date.IsZero() {
		return nil, domain.NewValidationError("Date", "дата не может быть пустой")
	}
	slots, err := s.repo.FindFreeSlots(ctx, doctorID, date)
	if err != nil {
		return nil, err
	}
	result := make([]ScheduleSlotResponseDTO, 0, len(slots))
	for _, slot := range slots {
		result = append(result, toScheduleSlotResponse(slot))
	}
	return result, nil
}

func (s *ScheduleService) DeleteSlot(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return domain.NewValidationError("ID", "не может быть пустым")
	}
	return s.repo.Delete(ctx, id)
}

func toScheduleSlotResponse(s *domain.ScheduleSlot) ScheduleSlotResponseDTO {
	return ScheduleSlotResponseDTO{
		ID:        s.ID,
		DoctorID:  s.DoctorID,
		Date:      s.Date,
		StartTime: s.StartTime,
		EndTime:   s.EndTime,
		Cabinet:   s.Cabinet,
		IsBooked:  s.IsBooked,
	}
}

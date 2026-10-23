package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
)

type mockScheduleRepo struct {
	slots map[uuid.UUID]*domain.ScheduleSlot
}

func newMockScheduleRepo() *mockScheduleRepo {
	return &mockScheduleRepo{slots: make(map[uuid.UUID]*domain.ScheduleSlot)}
}

func (m *mockScheduleRepo) Save(_ context.Context, slot *domain.ScheduleSlot) error {
	m.slots[slot.ID] = slot
	return nil
}

func (m *mockScheduleRepo) FindByDoctorID(_ context.Context, doctorID uuid.UUID) ([]*domain.ScheduleSlot, error) {
	var result []*domain.ScheduleSlot
	for _, slot := range m.slots {
		if slot.DoctorID == doctorID {
			result = append(result, slot)
		}
	}
	return result, nil
}

func (m *mockScheduleRepo) FindFreeSlots(_ context.Context, doctorID uuid.UUID, date time.Time) ([]*domain.ScheduleSlot, error) {
	var result []*domain.ScheduleSlot
	for _, slot := range m.slots {
		if slot.DoctorID == doctorID && sameDay(slot.Date, date) && !slot.IsBooked {
			result = append(result, slot)
		}
	}
	return result, nil
}

func (m *mockScheduleRepo) Update(_ context.Context, slot *domain.ScheduleSlot) error {
	m.slots[slot.ID] = slot
	return nil
}

func (m *mockScheduleRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.slots[id]; !ok {
		return domain.ErrNotFound
	}
	delete(m.slots, id)
	return nil
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func validScheduleDTO(doctorID uuid.UUID) CreateScheduleSlotDTO {
	date := time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC)
	return CreateScheduleSlotDTO{
		DoctorID:  doctorID,
		Date:      date,
		StartTime: time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 5, 22, 10, 30, 0, 0, time.UTC),
		Cabinet:   "203",
	}
}

func TestScheduleCreateSlotSuccess(t *testing.T) {
	repo := newMockScheduleRepo()
	service := NewScheduleService(repo)
	doctorID := uuid.New()

	res, err := service.CreateSlot(context.Background(), validScheduleDTO(doctorID))
	if err != nil {
		t.Fatalf("ожидался успех, получена ошибка: %v", err)
	}
	if res.ID == uuid.Nil {
		t.Fatal("ожидалось присвоение ID")
	}
	if res.DoctorID != doctorID {
		t.Fatalf("ожидался doctorID %s, получено: %s", doctorID, res.DoctorID)
	}
	if res.IsBooked {
		t.Fatal("новый слот должен быть свободным")
	}
}

func TestScheduleCreateSlotValidationError(t *testing.T) {
	repo := newMockScheduleRepo()
	service := NewScheduleService(repo)
	dto := validScheduleDTO(uuid.New())
	dto.Cabinet = ""

	_, err := service.CreateSlot(context.Background(), dto)
	if err == nil {
		t.Fatal("ожидалась ошибка валидации")
	}
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("ожидалась ErrValidation, получено: %v", err)
	}
}

func TestScheduleGetByDoctorSuccess(t *testing.T) {
	repo := newMockScheduleRepo()
	service := NewScheduleService(repo)
	doctorID := uuid.New()
	otherDoctorID := uuid.New()
	_, _ = service.CreateSlot(context.Background(), validScheduleDTO(doctorID))
	_, _ = service.CreateSlot(context.Background(), validScheduleDTO(otherDoctorID))

	result, err := service.GetByDoctor(context.Background(), doctorID)
	if err != nil {
		t.Fatalf("ожидался успех, получена ошибка: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("ожидался 1 слот, получено: %d", len(result))
	}
}

func TestScheduleGetByDoctorValidationError(t *testing.T) {
	repo := newMockScheduleRepo()
	service := NewScheduleService(repo)

	_, err := service.GetByDoctor(context.Background(), uuid.Nil)
	if err == nil {
		t.Fatal("ожидалась ошибка валидации")
	}
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("ожидалась ErrValidation, получено: %v", err)
	}
}

func TestScheduleGetFreeSlotsSuccess(t *testing.T) {
	repo := newMockScheduleRepo()
	service := NewScheduleService(repo)
	doctorID := uuid.New()
	date := time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC)
	freeSlot, _ := service.CreateSlot(context.Background(), validScheduleDTO(doctorID))
	booked := &domain.ScheduleSlot{
		ID:        uuid.New(),
		DoctorID:  doctorID,
		Date:      date,
		StartTime: time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 5, 22, 11, 30, 0, 0, time.UTC),
		Cabinet:   "204",
		IsBooked:  true,
	}
	_ = repo.Save(context.Background(), booked)

	result, err := service.GetFreeSlots(context.Background(), doctorID, date)
	if err != nil {
		t.Fatalf("ожидался успех, получена ошибка: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("ожидался 1 свободный слот, получено: %d", len(result))
	}
	if result[0].ID != freeSlot.ID {
		t.Fatalf("ожидался свободный слот %s, получено: %s", freeSlot.ID, result[0].ID)
	}
}

func TestScheduleGetFreeSlotsValidationError(t *testing.T) {
	repo := newMockScheduleRepo()
	service := NewScheduleService(repo)

	_, err := service.GetFreeSlots(context.Background(), uuid.Nil, time.Now())
	if err == nil {
		t.Fatal("ожидалась ошибка валидации")
	}
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("ожидалась ErrValidation, получено: %v", err)
	}
}

func TestScheduleDeleteSlotSuccess(t *testing.T) {
	repo := newMockScheduleRepo()
	service := NewScheduleService(repo)
	created, err := service.CreateSlot(context.Background(), validScheduleDTO(uuid.New()))
	if err != nil {
		t.Fatalf("create slot failed: %v", err)
	}

	if err := service.DeleteSlot(context.Background(), created.ID); err != nil {
		t.Fatalf("ожидался успех, получена ошибка: %v", err)
	}
}

func TestScheduleDeleteSlotValidationError(t *testing.T) {
	repo := newMockScheduleRepo()
	service := NewScheduleService(repo)

	err := service.DeleteSlot(context.Background(), uuid.Nil)
	if err == nil {
		t.Fatal("ожидалась ошибка валидации")
	}
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("ожидалась ErrValidation, получено: %v", err)
	}
}

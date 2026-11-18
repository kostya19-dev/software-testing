package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
)

type mockPatientRepo struct {
	patients map[uuid.UUID]*domain.Patient
	saveErr  error
}

func newMockPatientRepo() *mockPatientRepo {
	return &mockPatientRepo{patients: make(map[uuid.UUID]*domain.Patient)}
}

func (m *mockPatientRepo) Save(_ context.Context, patient *domain.Patient) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.patients[patient.ID] = patient
	return nil
}

func (m *mockPatientRepo) FindByID(_ context.Context, id uuid.UUID) (*domain.Patient, error) {
	if patient, ok := m.patients[id]; ok {
		return patient, nil
	}
	return nil, domain.ErrNotFound
}

func (m *mockPatientRepo) FindAll(_ context.Context) ([]*domain.Patient, error) {
	result := make([]*domain.Patient, 0, len(m.patients))
	for _, patient := range m.patients {
		result = append(result, patient)
	}
	return result, nil
}

func (m *mockPatientRepo) FindByEmail(_ context.Context, email string) (*domain.Patient, error) {
	for _, patient := range m.patients {
		if patient.Email == email {
			return patient, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockPatientRepo) Update(_ context.Context, patient *domain.Patient) error {
	m.patients[patient.ID] = patient
	return nil
}

func (m *mockPatientRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.patients[id]; !ok {
		return domain.ErrNotFound
	}
	delete(m.patients, id)
	return nil
}

func validPatientDTO() CreatePatientDTO {
	return CreatePatientDTO{
		FullName:  "Ivanov Ivan Ivanovich",
		Phone:     "+79001234567",
		Email:     "ivanov@example.com",
		BirthDate: time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC),
	}
}

func TestPatientRegisterSuccess(t *testing.T) {
	repo := newMockPatientRepo()
	service := NewPatientService(repo)

	res, err := service.Register(context.Background(), validPatientDTO())
	if err != nil {
		t.Fatalf("ожидался успех, получена ошибка: %v", err)
	}
	if res.ID == uuid.Nil {
		t.Fatal("ожидалось присвоение ID")
	}
	if res.Email != "ivanov@example.com" {
		t.Fatalf("ожидался email ivanov@example.com, получено: %s", res.Email)
	}
}

func TestPatientRegisterValidationError(t *testing.T) {
	repo := newMockPatientRepo()
	service := NewPatientService(repo)

	dto := validPatientDTO()
	dto.Email = ""

	_, err := service.Register(context.Background(), dto)
	if err == nil {
		t.Fatal("ожидалась ошибка валидации")
	}
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("ожидалась ErrValidation, получено: %v", err)
	}
}

func TestPatientRegisterRepoError(t *testing.T) {
	repo := newMockPatientRepo()
	repo.saveErr = errors.New("save failed")
	service := NewPatientService(repo)

	_, err := service.Register(context.Background(), validPatientDTO())
	if err == nil {
		t.Fatal("ожидалась ошибка репозитория")
	}
}

func TestPatientGetByIDSuccess(t *testing.T) {
	repo := newMockPatientRepo()
	service := NewPatientService(repo)
	patient := &domain.Patient{
		ID:        uuid.New(),
		FullName:  "Petrov Petr",
		Phone:     "+79007654321",
		Email:     "petrov@example.com",
		BirthDate: time.Date(1985, 3, 20, 0, 0, 0, 0, time.UTC),
		CreatedAt: time.Now(),
	}
	_ = repo.Save(context.Background(), patient)

	res, err := service.GetByID(context.Background(), patient.ID)
	if err != nil {
		t.Fatalf("ожидался успех, получена ошибка: %v", err)
	}
	if res.ID != patient.ID {
		t.Fatalf("ожидался ID %s, получено: %s", patient.ID, res.ID)
	}
}

func TestPatientGetByIDValidationError(t *testing.T) {
	repo := newMockPatientRepo()
	service := NewPatientService(repo)

	_, err := service.GetByID(context.Background(), uuid.Nil)
	if err == nil {
		t.Fatal("ожидалась ошибка валидации")
	}
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("ожидалась ErrValidation, получено: %v", err)
	}
}

func TestPatientGetAllSuccess(t *testing.T) {
	repo := newMockPatientRepo()
	service := NewPatientService(repo)
	_, _ = service.Register(context.Background(), validPatientDTO())

	result, err := service.GetAll(context.Background())
	if err != nil {
		t.Fatalf("ожидался успех, получена ошибка: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("ожидался 1 пациент, получено: %d", len(result))
	}
}

func TestPatientFindByEmailSuccess(t *testing.T) {
	repo := newMockPatientRepo()
	service := NewPatientService(repo)
	created, err := service.Register(context.Background(), validPatientDTO())
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	found, err := service.FindByEmail(context.Background(), created.Email)
	if err != nil {
		t.Fatalf("ожидался успех, получена ошибка: %v", err)
	}
	if found.ID != created.ID {
		t.Fatalf("ожидался ID %s, получено: %s", created.ID, found.ID)
	}
}

func TestPatientFindByEmailValidationError(t *testing.T) {
	repo := newMockPatientRepo()
	service := NewPatientService(repo)

	_, err := service.FindByEmail(context.Background(), " ")
	if err == nil {
		t.Fatal("ожидалась ошибка валидации")
	}
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("ожидалась ErrValidation, получено: %v", err)
	}
}

func TestPatientDeleteSuccess(t *testing.T) {
	repo := newMockPatientRepo()
	service := NewPatientService(repo)
	created, err := service.Register(context.Background(), validPatientDTO())
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	if err := service.Delete(context.Background(), created.ID); err != nil {
		t.Fatalf("ожидался успех, получена ошибка: %v", err)
	}
	if _, err := service.GetByID(context.Background(), created.ID); err != domain.ErrNotFound {
		t.Fatalf("ожидалась ErrNotFound после удаления, получено: %v", err)
	}
}

func TestPatientDeleteValidationError(t *testing.T) {
	repo := newMockPatientRepo()
	service := NewPatientService(repo)

	err := service.Delete(context.Background(), uuid.Nil)
	if err == nil {
		t.Fatal("ожидалась ошибка валидации")
	}
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("ожидалась ErrValidation, получено: %v", err)
	}
}

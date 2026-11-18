package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
)

// ручной мок репы

type mockAppRepo struct {
	appointments map[uuid.UUID]*domain.Appointment
	isBusy       bool
	saveErr      error
}

func newMockAppRepo() *mockAppRepo {
	return &mockAppRepo{
		appointments: make(map[uuid.UUID]*domain.Appointment),
	}
}

func (m *mockAppRepo) Save(_ context.Context, app *domain.Appointment) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.appointments[app.ID] = app
	return nil
}

func (m *mockAppRepo) FindByID(_ context.Context, id uuid.UUID) (*domain.Appointment, error) {
	if app, ok := m.appointments[id]; ok {
		return app, nil
	}
	return nil, domain.ErrNotFound
}

func (m *mockAppRepo) FindAll(_ context.Context) ([]*domain.Appointment, error) {
	result := make([]*domain.Appointment, 0, len(m.appointments))
	for _, app := range m.appointments {
		result = append(result, app)
	}
	return result, nil
}

func (m *mockAppRepo) FindByPatient(_ context.Context, patientID uuid.UUID) ([]*domain.Appointment, error) {
	var result []*domain.Appointment
	for _, app := range m.appointments {
		if app.PatientID == patientID {
			result = append(result, app)
		}
	}
	return result, nil
}

func (m *mockAppRepo) FindActiveByDoctorAndDate(_ context.Context, doctorID uuid.UUID, date time.Time) (*domain.Appointment, error) {
	for _, app := range m.appointments {
		if app.DoctorID == doctorID && app.Date.Equal(date) && app.Status != domain.StatusCancelled {
			return app, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockAppRepo) CheckSlotBusy(_ context.Context, _ uuid.UUID, _ time.Time) (bool, error) {
	return m.isBusy, nil
}

func (m *mockAppRepo) Update(_ context.Context, app *domain.Appointment) error {
	m.appointments[app.ID] = app
	return nil
}

func (m *mockAppRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.appointments[id]; !ok {
		return domain.ErrNotFound
	}
	delete(m.appointments, id)
	return nil
}

// ТЕСТЫ CREATE

func TestCreate_Success(t *testing.T) {
	repo := newMockAppRepo()
	service := NewAppointmentService(repo)
	ctx := context.Background()

	patientID := uuid.New()
	doctorID := uuid.New()

	dto := CreateAppointmentDTO{
		PatientID: patientID,
		DoctorID:  doctorID,
		Date:      time.Now().Add(24 * time.Hour),
	}

	res, err := service.Create(ctx, dto)
	if err != nil {
		t.Fatalf("ожидался успех, получена ошибка: %v", err)
	}
	if res.ID == uuid.Nil {
		t.Error("ожидалось присвоение ID")
	}
	if res.Status != domain.StatusCreated {
		t.Errorf("ожидался статус CREATED, получено: %s", res.Status)
	}
	if res.PatientID != patientID || res.DoctorID != doctorID {
		t.Error("PatientID или DoctorID не совпадают")
	}
}

func TestCreate_ValidationError_NilPatientID(t *testing.T) {
	repo := newMockAppRepo()
	service := NewAppointmentService(repo)
	ctx := context.Background()

	dto := CreateAppointmentDTO{
		PatientID: uuid.Nil,
		DoctorID:  uuid.New(),
		Date:      time.Now().Add(24 * time.Hour),
	}

	_, err := service.Create(ctx, dto)
	if err == nil {
		t.Fatal("ожидалась ошибка валидации, получен nil")
	}
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("ожидалась ErrValidation, получено: %v", err)
	}
}

func TestCreate_ValidationError_NilDoctorID(t *testing.T) {
	repo := newMockAppRepo()
	service := NewAppointmentService(repo)
	ctx := context.Background()

	dto := CreateAppointmentDTO{
		PatientID: uuid.New(),
		DoctorID:  uuid.Nil,
		Date:      time.Now().Add(24 * time.Hour),
	}

	_, err := service.Create(ctx, dto)
	if err == nil {
		t.Fatal("ожидалась ошибка валидации, получен nil")
	}
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("ожидалась ErrValidation, получено: %v", err)
	}
}

func TestCreate_ValidationError_EmptyDate(t *testing.T) {
	repo := newMockAppRepo()
	service := NewAppointmentService(repo)
	ctx := context.Background()

	dto := CreateAppointmentDTO{
		PatientID: uuid.New(),
		DoctorID:  uuid.New(),
	}

	_, err := service.Create(ctx, dto)
	if err == nil {
		t.Fatal("ожидалась ошибка валидации, получен nil")
	}
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("ожидалась ErrValidation, получено: %v", err)
	}
}

func TestCreate_PastDate(t *testing.T) {
	repo := newMockAppRepo()
	service := NewAppointmentService(repo)
	ctx := context.Background()

	dto := CreateAppointmentDTO{
		PatientID: uuid.New(),
		DoctorID:  uuid.New(),
		Date:      time.Now().Add(-24 * time.Hour),
	}

	_, err := service.Create(ctx, dto)
	if err != domain.ErrPastDate {
		t.Errorf("ожидалась ErrPastDate, получено: %v", err)
	}
}

func TestCreate_TooSoonByConfiguredLeadTime(t *testing.T) {
	repo := newMockAppRepo()
	service := NewAppointmentService(repo, WithAppointmentMinLeadTime(2*time.Hour))
	ctx := context.Background()

	dto := CreateAppointmentDTO{
		PatientID: uuid.New(),
		DoctorID:  uuid.New(),
		Date:      time.Now().Add(30 * time.Minute),
	}

	_, err := service.Create(ctx, dto)
	if err != domain.ErrAppointmentLeadTime {
		t.Errorf("ожидалась ErrAppointmentLeadTime, получено: %v", err)
	}
}

func TestCreate_SlotBusy(t *testing.T) {
	repo := newMockAppRepo()
	repo.isBusy = true
	service := NewAppointmentService(repo)
	ctx := context.Background()

	dto := CreateAppointmentDTO{
		PatientID: uuid.New(),
		DoctorID:  uuid.New(),
		Date:      time.Now().Add(24 * time.Hour),
	}

	_, err := service.Create(ctx, dto)
	if err != domain.ErrSlotBusy {
		t.Errorf("ожидалась ErrSlotBusy, получено: %v", err)
	}
}

func TestCreate_RepoSaveError(t *testing.T) {
	repo := newMockAppRepo()
	repo.saveErr = errors.New("ошибка базы данных")
	service := NewAppointmentService(repo)
	ctx := context.Background()

	dto := CreateAppointmentDTO{
		PatientID: uuid.New(),
		DoctorID:  uuid.New(),
		Date:      time.Now().Add(24 * time.Hour),
	}

	_, err := service.Create(ctx, dto)
	if err == nil {
		t.Fatal("ожидалась ошибка репозитория, получен nil")
	}
}

// ТЕСТЫ CANCEL

func TestCancel_Success(t *testing.T) {
	repo := newMockAppRepo()
	service := NewAppointmentService(repo)
	ctx := context.Background()

	appID := uuid.New()
	app := &domain.Appointment{
		ID:        appID,
		PatientID: uuid.New(),
		DoctorID:  uuid.New(),
		Status:    domain.StatusCreated,
		Date:      time.Now().Add(24 * time.Hour),
	}
	_ = repo.Save(ctx, app)

	err := service.Cancel(ctx, appID)
	if err != nil {
		t.Fatalf("ожидался успех, получена ошибка: %v", err)
	}

	updated, _ := repo.FindByID(ctx, appID)
	if updated.Status != domain.StatusCancelled {
		t.Errorf("ожидался статус CANCELLED, получено: %s", updated.Status)
	}
}

func TestCancel_NotFound(t *testing.T) {
	repo := newMockAppRepo()
	service := NewAppointmentService(repo)
	ctx := context.Background()

	err := service.Cancel(ctx, uuid.New())
	if err != domain.ErrNotFound {
		t.Errorf("ожидалась ErrNotFound, получено: %v", err)
	}
}

func TestCancel_AlreadyCancelled(t *testing.T) {
	repo := newMockAppRepo()
	service := NewAppointmentService(repo)
	ctx := context.Background()

	appID := uuid.New()
	app := &domain.Appointment{ID: appID, Status: domain.StatusCancelled}
	_ = repo.Save(ctx, app)

	err := service.Cancel(ctx, appID)
	if err != domain.ErrInvalidStatus {
		t.Errorf("ожидалась ErrInvalidStatus, получено: %v", err)
	}
}

func TestCancel_CompletedAppointment(t *testing.T) {
	repo := newMockAppRepo()
	service := NewAppointmentService(repo)
	ctx := context.Background()

	appID := uuid.New()
	app := &domain.Appointment{ID: appID, Status: domain.StatusCompleted}
	_ = repo.Save(ctx, app)

	err := service.Cancel(ctx, appID)
	if err != domain.ErrInvalidStatus {
		t.Errorf("ожидалась ErrInvalidStatus, получено: %v", err)
	}
}

// ТЕСТЫ GET

func TestGetByID_Success(t *testing.T) {
	repo := newMockAppRepo()
	service := NewAppointmentService(repo)
	ctx := context.Background()

	appID := uuid.New()
	app := &domain.Appointment{
		ID:        appID,
		PatientID: uuid.New(),
		DoctorID:  uuid.New(),
		Status:    domain.StatusCreated,
	}
	_ = repo.Save(ctx, app)

	res, err := service.GetByID(ctx, appID)
	if err != nil {
		t.Fatalf("ожидался успех, получена ошибка: %v", err)
	}
	if res.ID != appID {
		t.Errorf("ожидался ID=%s, получено ID=%s", appID, res.ID)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	repo := newMockAppRepo()
	service := NewAppointmentService(repo)
	ctx := context.Background()

	_, err := service.GetByID(ctx, uuid.New())
	if err != domain.ErrNotFound {
		t.Errorf("ожидалась ErrNotFound, получено: %v", err)
	}
}

func TestGetAll_Success(t *testing.T) {
	repo := newMockAppRepo()
	service := NewAppointmentService(repo)
	ctx := context.Background()

	_ = repo.Save(ctx, &domain.Appointment{ID: uuid.New(), PatientID: uuid.New(), DoctorID: uuid.New(), Status: domain.StatusCreated})
	_ = repo.Save(ctx, &domain.Appointment{ID: uuid.New(), PatientID: uuid.New(), DoctorID: uuid.New(), Status: domain.StatusCreated})

	result, err := service.GetAll(ctx)
	if err != nil {
		t.Fatalf("ожидался успех, получена ошибка: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("ожидалось 2 записи, получено: %d", len(result))
	}
}

func TestGetAll_EmptyList(t *testing.T) {
	repo := newMockAppRepo()
	service := NewAppointmentService(repo)
	ctx := context.Background()

	result, err := service.GetAll(ctx)
	if err != nil {
		t.Fatalf("ожидался успех, получена ошибка: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("ожидалось 0 записей, получено: %d", len(result))
	}
}

func TestGetByPatient_Success(t *testing.T) {
	repo := newMockAppRepo()
	service := NewAppointmentService(repo)
	ctx := context.Background()

	patientID := uuid.New()
	_ = repo.Save(ctx, &domain.Appointment{ID: uuid.New(), PatientID: patientID, DoctorID: uuid.New(), Status: domain.StatusCreated})
	_ = repo.Save(ctx, &domain.Appointment{ID: uuid.New(), PatientID: patientID, DoctorID: uuid.New(), Status: domain.StatusCreated})
	_ = repo.Save(ctx, &domain.Appointment{ID: uuid.New(), PatientID: uuid.New(), DoctorID: uuid.New(), Status: domain.StatusCreated})

	result, err := service.GetByPatient(ctx, patientID)
	if err != nil {
		t.Fatalf("ожидался успех, получена ошибка: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("ожидалось 2 записи для пациента, получено: %d", len(result))
	}
}

func TestGetByPatient_InvalidID(t *testing.T) {
	repo := newMockAppRepo()
	service := NewAppointmentService(repo)
	ctx := context.Background()

	_, err := service.GetByPatient(ctx, uuid.Nil)
	if err == nil {
		t.Fatal("ожидалась ошибка валидации, получен nil")
	}
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("ожидалась ErrValidation, получено: %v", err)
	}
}

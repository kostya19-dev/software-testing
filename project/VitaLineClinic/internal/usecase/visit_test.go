package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
)

//  РУЧНОЙ МОК VISIT РЕПОЗИТОРИЯ

type mockVisitRepo struct {
	visits map[uuid.UUID]*domain.Visit
}

func newMockVisitRepo() *mockVisitRepo {
	return &mockVisitRepo{
		visits: make(map[uuid.UUID]*domain.Visit),
	}
}

func (m *mockVisitRepo) Save(_ context.Context, v *domain.Visit) error {
	m.visits[v.ID] = v
	return nil
}

func (m *mockVisitRepo) FindByID(_ context.Context, id uuid.UUID) (*domain.Visit, error) {
	if v, ok := m.visits[id]; ok {
		return v, nil
	}
	return nil, domain.ErrNotFound
}

func (m *mockVisitRepo) FindByAppointment(_ context.Context, appointmentID uuid.UUID) (*domain.Visit, error) {
	for _, v := range m.visits {
		if v.AppointmentID == appointmentID {
			return v, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockVisitRepo) Update(_ context.Context, v *domain.Visit) error {
	m.visits[v.ID] = v
	return nil
}

func (m *mockVisitRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.visits[id]; !ok {
		return domain.ErrNotFound
	}
	delete(m.visits, id)
	return nil
}

// ТЕСТЫ CREATE

func TestVisitCreate_Success(t *testing.T) {
	visitRepo := newMockVisitRepo()
	appRepo := newMockAppRepo()
	service := NewVisitService(visitRepo, appRepo)
	ctx := context.Background()

	appID := uuid.New()
	app := &domain.Appointment{
		ID:        appID,
		PatientID: uuid.New(),
		DoctorID:  uuid.New(),
		Status:    domain.StatusCreated,
		Date:      time.Now().Add(24 * time.Hour),
	}
	_ = appRepo.Save(ctx, app)

	dto := CreateVisitDTO{
		AppointmentID: appID,
		Complaints:    "Головная боль, температура 38.5",
	}

	res, err := service.Create(ctx, dto)
	if err != nil {
		t.Fatalf("ожидался успех, получена ошибка: %v", err)
	}
	if res.ID == uuid.Nil {
		t.Error("ожидалось присвоение ID")
	}
	if res.Status != domain.VisitStatusInProgress {
		t.Errorf("ожидался статус IN_PROGRESS, получено: %s", res.Status)
	}
	if res.Complaints != "Головная боль, температура 38.5" {
		t.Error("жалобы не совпадают")
	}

	updatedApp, _ := appRepo.FindByID(ctx, appID)
	if updatedApp.Status != domain.StatusCompleted {
		t.Errorf("ожидался статус записи COMPLETED, получено: %s", updatedApp.Status)
	}
}

func TestVisitCreate_ValidationError_NilAppointmentID(t *testing.T) {
	visitRepo := newMockVisitRepo()
	appRepo := newMockAppRepo()
	service := NewVisitService(visitRepo, appRepo)
	ctx := context.Background()

	dto := CreateVisitDTO{
		AppointmentID: uuid.Nil,
		Complaints:    "Жалобы",
	}

	_, err := service.Create(ctx, dto)
	if err == nil {
		t.Fatal("ожидалась ошибка валидации, получен nil")
	}
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("ожидалась ErrValidation, получено: %v", err)
	}
}

func TestVisitCreate_ValidationError_EmptyComplaints(t *testing.T) {
	visitRepo := newMockVisitRepo()
	appRepo := newMockAppRepo()
	service := NewVisitService(visitRepo, appRepo)
	ctx := context.Background()

	dto := CreateVisitDTO{
		AppointmentID: uuid.New(),
		Complaints:    "",
	}

	_, err := service.Create(ctx, dto)
	if err == nil {
		t.Fatal("ожидалась ошибка валидации, получен nil")
	}
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("ожидалась ErrValidation, получено: %v", err)
	}
}

func TestVisitCreate_ValidationError_WhitespaceComplaints(t *testing.T) {
	visitRepo := newMockVisitRepo()
	appRepo := newMockAppRepo()
	service := NewVisitService(visitRepo, appRepo)
	ctx := context.Background()

	dto := CreateVisitDTO{
		AppointmentID: uuid.New(),
		Complaints:    "   ",
	}

	_, err := service.Create(ctx, dto)
	if err == nil {
		t.Fatal("ожидалась ошибка валидации, получен nil")
	}
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("ожидалась ErrValidation, получено: %v", err)
	}
}

func TestVisitCreate_AppointmentNotFound(t *testing.T) {
	visitRepo := newMockVisitRepo()
	appRepo := newMockAppRepo()
	service := NewVisitService(visitRepo, appRepo)
	ctx := context.Background()

	dto := CreateVisitDTO{
		AppointmentID: uuid.New(),
		Complaints:    "Жалобы",
	}

	_, err := service.Create(ctx, dto)
	if err != domain.ErrNotFound {
		t.Errorf("ожидалась ErrNotFound, получено: %v", err)
	}
}

func TestVisitCreate_AppointmentAlreadyCancelled(t *testing.T) {
	visitRepo := newMockVisitRepo()
	appRepo := newMockAppRepo()
	service := NewVisitService(visitRepo, appRepo)
	ctx := context.Background()

	appID := uuid.New()
	app := &domain.Appointment{ID: appID, Status: domain.StatusCancelled}
	_ = appRepo.Save(ctx, app)

	dto := CreateVisitDTO{
		AppointmentID: appID,
		Complaints:    "Жалобы",
	}

	_, err := service.Create(ctx, dto)
	if err != domain.ErrInvalidStatus {
		t.Errorf("ожидалась ErrInvalidStatus, получено: %v", err)
	}
}

func TestVisitCreate_AppointmentAlreadyCompleted(t *testing.T) {
	visitRepo := newMockVisitRepo()
	appRepo := newMockAppRepo()
	service := NewVisitService(visitRepo, appRepo)
	ctx := context.Background()

	appID := uuid.New()
	app := &domain.Appointment{ID: appID, Status: domain.StatusCompleted}
	_ = appRepo.Save(ctx, app)

	dto := CreateVisitDTO{
		AppointmentID: appID,
		Complaints:    "Жалобы",
	}

	_, err := service.Create(ctx, dto)
	if err != domain.ErrInvalidStatus {
		t.Errorf("ожидалась ErrInvalidStatus, получено: %v", err)
	}
}

//  ТЕСТЫ COMPLETE

func TestVisitComplete_Success(t *testing.T) {
	visitRepo := newMockVisitRepo()
	appRepo := newMockAppRepo()
	service := NewVisitService(visitRepo, appRepo)
	ctx := context.Background()

	visitID := uuid.New()
	visit := &domain.Visit{
		ID:            visitID,
		AppointmentID: uuid.New(),
		Complaints:    "Боль в горле",
		Status:        domain.VisitStatusInProgress,
		CreatedAt:     time.Now(),
	}
	_ = visitRepo.Save(ctx, visit)

	dto := UpdateVisitDTO{
		Diagnosis:       "ОРВИ",
		Recommendations: "Постельный режим, обильное питьё",
	}

	res, err := service.Complete(ctx, visitID, dto)
	if err != nil {
		t.Fatalf("ожидался успех, получена ошибка: %v", err)
	}
	if res.Status != domain.VisitStatusCompleted {
		t.Errorf("ожидался статус COMPLETED, получено: %s", res.Status)
	}
	if res.Diagnosis != "ОРВИ" {
		t.Errorf("ожидался диагноз 'ОРВИ', получено: %s", res.Diagnosis)
	}
}

func TestVisitComplete_NotFound(t *testing.T) {
	visitRepo := newMockVisitRepo()
	appRepo := newMockAppRepo()
	service := NewVisitService(visitRepo, appRepo)
	ctx := context.Background()

	dto := UpdateVisitDTO{
		Diagnosis:       "ОРВИ",
		Recommendations: "Отдых",
	}

	_, err := service.Complete(ctx, uuid.New(), dto)
	if err != domain.ErrNotFound {
		t.Errorf("ожидалась ErrNotFound, получено: %v", err)
	}
}

func TestVisitComplete_AlreadyCompleted(t *testing.T) {
	visitRepo := newMockVisitRepo()
	appRepo := newMockAppRepo()
	service := NewVisitService(visitRepo, appRepo)
	ctx := context.Background()

	visitID := uuid.New()
	visit := &domain.Visit{
		ID:            visitID,
		AppointmentID: uuid.New(),
		Complaints:    "Жалобы",
		Status:        domain.VisitStatusCompleted,
		Diagnosis:     "Диагноз",
	}
	_ = visitRepo.Save(ctx, visit)

	dto := UpdateVisitDTO{
		Diagnosis:       "Новый диагноз",
		Recommendations: "Новые рекомендации",
	}

	_, err := service.Complete(ctx, visitID, dto)
	if err != domain.ErrInvalidStatus {
		t.Errorf("ожидалась ErrInvalidStatus, получено: %v", err)
	}
}

func TestVisitComplete_ValidationError_EmptyDiagnosis(t *testing.T) {
	visitRepo := newMockVisitRepo()
	appRepo := newMockAppRepo()
	service := NewVisitService(visitRepo, appRepo)
	ctx := context.Background()

	dto := UpdateVisitDTO{
		Diagnosis:       "",
		Recommendations: "Рекомендации",
	}

	_, err := service.Complete(ctx, uuid.New(), dto)
	if err == nil {
		t.Fatal("ожидалась ошибка валидации, получен nil")
	}
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("ожидалась ErrValidation, получено: %v", err)
	}
}

func TestVisitComplete_ValidationError_EmptyRecommendations(t *testing.T) {
	visitRepo := newMockVisitRepo()
	appRepo := newMockAppRepo()
	service := NewVisitService(visitRepo, appRepo)
	ctx := context.Background()

	dto := UpdateVisitDTO{
		Diagnosis:       "Диагноз",
		Recommendations: "",
	}

	_, err := service.Complete(ctx, uuid.New(), dto)
	if err == nil {
		t.Fatal("ожидалась ошибка валидации, получен nil")
	}
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("ожидалась ErrValidation, получено: %v", err)
	}
}

// ТЕСТЫ GET

func TestVisitGetByAppointment_Success(t *testing.T) {
	visitRepo := newMockVisitRepo()
	appRepo := newMockAppRepo()
	service := NewVisitService(visitRepo, appRepo)
	ctx := context.Background()

	appID := uuid.New()
	visitID := uuid.New()
	visit := &domain.Visit{
		ID:            visitID,
		AppointmentID: appID,
		Complaints:    "Кашель",
		Status:        domain.VisitStatusInProgress,
	}
	_ = visitRepo.Save(ctx, visit)

	res, err := service.GetByAppointment(ctx, appID)
	if err != nil {
		t.Fatalf("ожидался успех, получена ошибка: %v", err)
	}
	if res.AppointmentID != appID {
		t.Errorf("ожидался AppointmentID=%s, получено: %s", appID, res.AppointmentID)
	}
}

func TestVisitGetByAppointment_NotFound(t *testing.T) {
	visitRepo := newMockVisitRepo()
	appRepo := newMockAppRepo()
	service := NewVisitService(visitRepo, appRepo)
	ctx := context.Background()

	_, err := service.GetByAppointment(ctx, uuid.New())
	if err != domain.ErrNotFound {
		t.Errorf("ожидалась ErrNotFound, получено: %v", err)
	}
}

func TestVisitGetByPatient_Success(t *testing.T) {
	visitRepo := newMockVisitRepo()
	appRepo := newMockAppRepo()
	service := NewVisitService(visitRepo, appRepo)
	ctx := context.Background()

	patientID := uuid.New()
	otherPatientID := uuid.New()

	app1 := &domain.Appointment{
		ID:        uuid.New(),
		PatientID: patientID,
		DoctorID:  uuid.New(),
		Status:    domain.StatusCreated,
		Date:      time.Now().Add(24 * time.Hour),
	}
	app2 := &domain.Appointment{
		ID:        uuid.New(),
		PatientID: otherPatientID,
		DoctorID:  uuid.New(),
		Status:    domain.StatusCreated,
		Date:      time.Now().Add(48 * time.Hour),
	}

	_ = appRepo.Save(ctx, app1)
	_ = appRepo.Save(ctx, app2)

	_ = visitRepo.Save(ctx, &domain.Visit{
		ID:            uuid.New(),
		AppointmentID: app1.ID,
		Complaints:    "Complaint 1",
		Status:        domain.VisitStatusCompleted,
	})
	_ = visitRepo.Save(ctx, &domain.Visit{
		ID:            uuid.New(),
		AppointmentID: app2.ID,
		Complaints:    "Complaint 2",
		Status:        domain.VisitStatusInProgress,
	})

	result, err := service.GetByPatient(ctx, patientID)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 visit, got: %d", len(result))
	}
	if result[0].AppointmentID != app1.ID {
		t.Errorf("expected visit for appointment %s, got: %s", app1.ID, result[0].AppointmentID)
	}
}

func TestVisitGetByPatient_InvalidID(t *testing.T) {
	visitRepo := newMockVisitRepo()
	appRepo := newMockAppRepo()
	service := NewVisitService(visitRepo, appRepo)
	ctx := context.Background()

	_, err := service.GetByPatient(ctx, uuid.Nil)
	if err == nil {
		t.Fatal("ожидалась ошибка валидации, получен nil")
	}
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("ожидалась ErrValidation, получено: %v", err)
	}
}

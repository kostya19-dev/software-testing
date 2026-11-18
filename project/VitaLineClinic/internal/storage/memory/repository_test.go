package memory

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
	"vitaline-clinic/internal/usecase"
)

// Проверка реализации интерфейсов на этапе компиляции
var (
	_ usecase.AppointmentRepository = (*AppointmentRepository)(nil)
	_ usecase.VisitRepository       = (*VisitRepository)(nil)
	_ usecase.PatientRepository     = (*PatientRepository)(nil)
	_ usecase.ScheduleRepository    = (*ScheduleRepository)(nil)
)

// Тест 1: Сохранение и извлечение данных

func TestMemory_SaveAndRetrieve(t *testing.T) {
	ctx := context.Background()

	// Patient
	patientRepo := NewPatientRepository()
	patientID := uuid.New()
	patient := &domain.Patient{
		ID:        patientID,
		FullName:  "Иванов Иван Иванович",
		Phone:     "+79001234567",
		Email:     "ivanov@example.com",
		BirthDate: time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC),
	}

	if err := patientRepo.Save(ctx, patient); err != nil {
		t.Fatalf("ошибка сохранения пациента: %v", err)
	}

	found, err := patientRepo.FindByID(ctx, patientID)
	if err != nil {
		t.Fatalf("ошибка поиска пациента: %v", err)
	}
	if found.FullName != "Иванов Иван Иванович" {
		t.Errorf("ожидалось ФИО 'Иванов Иван Иванович', получено: %s", found.FullName)
	}
	if found.Email != "ivanov@example.com" {
		t.Errorf("ожидался email 'ivanov@example.com', получено: %s", found.Email)
	}

	// Поиск по email
	byEmail, err := patientRepo.FindByEmail(ctx, "ivanov@example.com")
	if err != nil {
		t.Fatalf("ошибка поиска по email: %v", err)
	}
	if byEmail.ID != patientID {
		t.Errorf("ожидался ID=%s, получено ID=%s", patientID, byEmail.ID)
	}

	// Appointment
	appRepo := NewAppointmentRepository()
	futureDate := time.Now().Add(48 * time.Hour)
	appID := uuid.New()
	doctorID := uuid.New()

	app := &domain.Appointment{
		ID:        appID,
		PatientID: patientID,
		DoctorID:  doctorID,
		Status:    domain.StatusCreated,
		Date:      futureDate,
		CreatedAt: time.Now(),
	}

	if err := appRepo.Save(ctx, app); err != nil {
		t.Fatalf("ошибка сохранения записи: %v", err)
	}

	foundApp, err := appRepo.FindByID(ctx, appID)
	if err != nil {
		t.Fatalf("ошибка поиска записи: %v", err)
	}
	if foundApp.PatientID != patientID {
		t.Errorf("ожидался PatientID=%s, получено: %s", patientID, foundApp.PatientID)
	}
	if foundApp.Status != domain.StatusCreated {
		t.Errorf("ожидался статус CREATED, получено: %s", foundApp.Status)
	}

	// Поиск по пациенту
	byPatient, err := appRepo.FindByPatient(ctx, patientID)
	if err != nil {
		t.Fatalf("ошибка поиска по пациенту: %v", err)
	}
	if len(byPatient) != 1 {
		t.Errorf("ожидалась 1 запись, получено: %d", len(byPatient))
	}

	// Проверка занятости слота
	isBusy, err := appRepo.CheckSlotBusy(ctx, doctorID, futureDate)
	if err != nil {
		t.Fatalf("ошибка проверки слота: %v", err)
	}
	if !isBusy {
		t.Error("ожидалось, что слот занят")
	}

	// Visit
	visitRepo := NewVisitRepository()
	visitID := uuid.New()
	visit := &domain.Visit{
		ID:            visitID,
		AppointmentID: appID,
		Complaints:    "Головная боль",
		Status:        domain.VisitStatusInProgress,
		CreatedAt:     time.Now(),
	}

	if err := visitRepo.Save(ctx, visit); err != nil {
		t.Fatalf("ошибка сохранения визита: %v", err)
	}

	foundVisit, err := visitRepo.FindByAppointment(ctx, appID)
	if err != nil {
		t.Fatalf("ошибка поиска визита: %v", err)
	}
	if foundVisit.Complaints != "Головная боль" {
		t.Errorf("ожидались жалобы 'Головная боль', получено: %s", foundVisit.Complaints)
	}

	//  Schedule
	schedRepo := NewScheduleRepository()
	tomorrow := time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour)
	slotID := uuid.New()

	slot := &domain.ScheduleSlot{
		ID:        slotID,
		DoctorID:  doctorID,
		Date:      tomorrow,
		StartTime: tomorrow.Add(9 * time.Hour),
		EndTime:   tomorrow.Add(10 * time.Hour),
		Cabinet:   "101",
		IsBooked:  false,
	}

	if err := schedRepo.Save(ctx, slot); err != nil {
		t.Fatalf("ошибка сохранения слота: %v", err)
	}

	freeSlots, err := schedRepo.FindFreeSlots(ctx, doctorID, tomorrow)
	if err != nil {
		t.Fatalf("ошибка поиска свободных слотов: %v", err)
	}
	if len(freeSlots) != 1 {
		t.Errorf("ожидался 1 свободный слот, получено: %d", len(freeSlots))
	}
}

//Тест 2: Изменение данных

func TestMemory_Update(t *testing.T) {
	ctx := context.Background()

	// Создаём и обновляем запись на приём
	appRepo := NewAppointmentRepository()
	appID := uuid.New()
	app := &domain.Appointment{
		ID:        appID,
		PatientID: uuid.New(),
		DoctorID:  uuid.New(),
		Status:    domain.StatusCreated,
		Date:      time.Now().Add(48 * time.Hour),
		CreatedAt: time.Now(),
	}
	if err := appRepo.Save(ctx, app); err != nil {
		t.Fatalf("ошибка сохранения записи: %v", err)
	}

	// Отменяем запись
	app.Status = domain.StatusCancelled
	if err := appRepo.Update(ctx, app); err != nil {
		t.Fatalf("ошибка обновления записи: %v", err)
	}

	updated, err := appRepo.FindByID(ctx, appID)
	if err != nil {
		t.Fatalf("ошибка поиска: %v", err)
	}
	if updated.Status != domain.StatusCancelled {
		t.Errorf("ожидался статус CANCELLED, получено: %s", updated.Status)
	}

	// Создаём и завершаем визит
	visitRepo := NewVisitRepository()
	visitID := uuid.New()
	visit := &domain.Visit{
		ID:            visitID,
		AppointmentID: uuid.New(),
		Complaints:    "Кашель",
		Status:        domain.VisitStatusInProgress,
		CreatedAt:     time.Now(),
	}
	if err := visitRepo.Save(ctx, visit); err != nil {
		t.Fatalf("ошибка сохранения визита: %v", err)
	}

	visit.Diagnosis = "ОРВИ"
	visit.Recommendations = "Постельный режим"
	visit.Status = domain.VisitStatusCompleted
	if err := visitRepo.Update(ctx, visit); err != nil {
		t.Fatalf("ошибка обновления визита: %v", err)
	}

	updatedVisit, err := visitRepo.FindByID(ctx, visitID)
	if err != nil {
		t.Fatalf("ошибка поиска визита: %v", err)
	}
	if updatedVisit.Status != domain.VisitStatusCompleted {
		t.Errorf("ожидался статус COMPLETED, получено: %s", updatedVisit.Status)
	}
	if updatedVisit.Diagnosis != "ОРВИ" {
		t.Errorf("ожидался диагноз 'ОРВИ', получено: %s", updatedVisit.Diagnosis)
	}

	// Обновляем телефон пациента
	patientRepo := NewPatientRepository()
	patientID := uuid.New()
	patient := &domain.Patient{
		ID:        patientID,
		FullName:  "Петров Пётр",
		Phone:     "+79009876543",
		Email:     "petrov@example.com",
		BirthDate: time.Date(1985, 3, 20, 0, 0, 0, 0, time.UTC),
	}
	if err := patientRepo.Save(ctx, patient); err != nil {
		t.Fatalf("ошибка сохранения пациента: %v", err)
	}

	patient.Phone = "+79001111111"
	if err := patientRepo.Update(ctx, patient); err != nil {
		t.Fatalf("ошибка обновления пациента: %v", err)
	}

	updatedPatient, err := patientRepo.FindByID(ctx, patientID)
	if err != nil {
		t.Fatalf("ошибка поиска пациента: %v", err)
	}
	if updatedPatient.Phone != "+79001111111" {
		t.Errorf("ожидался телефон '+79001111111', получено: %s", updatedPatient.Phone)
	}
}

// Тест 3: Удаление данных

func TestMemory_Delete(t *testing.T) {
	ctx := context.Background()

	// Удаление записи на приём
	appRepo := NewAppointmentRepository()
	appID := uuid.New()
	app := &domain.Appointment{
		ID:        appID,
		PatientID: uuid.New(),
		DoctorID:  uuid.New(),
		Status:    domain.StatusCreated,
		Date:      time.Now().Add(48 * time.Hour),
		CreatedAt: time.Now(),
	}
	if err := appRepo.Save(ctx, app); err != nil {
		t.Fatalf("ошибка сохранения: %v", err)
	}

	if err := appRepo.Delete(ctx, appID); err != nil {
		t.Fatalf("ошибка удаления: %v", err)
	}

	_, err := appRepo.FindByID(ctx, appID)
	if err != domain.ErrNotFound {
		t.Errorf("ожидалась ErrNotFound после удаления, получено: %v", err)
	}

	// Повторное удаление — ошибка
	err = appRepo.Delete(ctx, appID)
	if err != domain.ErrNotFound {
		t.Errorf("ожидалась ErrNotFound при повторном удалении, получено: %v", err)
	}

	// Удаление пациента
	patientRepo := NewPatientRepository()
	patientID := uuid.New()
	patient := &domain.Patient{
		ID:        patientID,
		FullName:  "Сидоров Сидор",
		Phone:     "+79005555555",
		Email:     "sidorov@example.com",
		BirthDate: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	if err := patientRepo.Save(ctx, patient); err != nil {
		t.Fatalf("ошибка сохранения пациента: %v", err)
	}

	if err := patientRepo.Delete(ctx, patientID); err != nil {
		t.Fatalf("ошибка удаления пациента: %v", err)
	}

	_, err = patientRepo.FindByID(ctx, patientID)
	if err != domain.ErrNotFound {
		t.Errorf("ожидалась ErrNotFound после удаления пациента, получено: %v", err)
	}

	all, err := patientRepo.FindAll(ctx)
	if err != nil {
		t.Fatalf("ошибка FindAll: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("ожидалось 0 пациентов после удаления, получено: %d", len(all))
	}

	// Удаление визита
	visitRepo := NewVisitRepository()
	visitID := uuid.New()
	visit := &domain.Visit{
		ID:            visitID,
		AppointmentID: uuid.New(),
		Complaints:    "Жалобы",
		Status:        domain.VisitStatusInProgress,
		CreatedAt:     time.Now(),
	}
	if err := visitRepo.Save(ctx, visit); err != nil {
		t.Fatalf("ошибка сохранения визита: %v", err)
	}

	if err := visitRepo.Delete(ctx, visitID); err != nil {
		t.Fatalf("ошибка удаления визита: %v", err)
	}

	_, err = visitRepo.FindByID(ctx, visitID)
	if err != domain.ErrNotFound {
		t.Errorf("ожидалась ErrNotFound после удаления визита, получено: %v", err)
	}

	// Удаление слота расписания
	schedRepo := NewScheduleRepository()
	tomorrow := time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour)
	slotID := uuid.New()
	slot := &domain.ScheduleSlot{
		ID:        slotID,
		DoctorID:  uuid.New(),
		Date:      tomorrow,
		StartTime: tomorrow.Add(9 * time.Hour),
		EndTime:   tomorrow.Add(10 * time.Hour),
		Cabinet:   "205",
	}
	if err := schedRepo.Save(ctx, slot); err != nil {
		t.Fatalf("ошибка сохранения слота: %v", err)
	}

	if err := schedRepo.Delete(ctx, slotID); err != nil {
		t.Fatalf("ошибка удаления слота: %v", err)
	}

	slots, err := schedRepo.FindByDoctorID(ctx, slot.DoctorID)
	if err != nil {
		t.Fatalf("ошибка FindByDoctorID: %v", err)
	}
	if len(slots) != 0 {
		t.Errorf("ожидалось 0 слотов после удаления, получено: %d", len(slots))
	}
}

package memory

import (
	"context"
	"errors"
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

// must останавливает тест, если операция репозитория вернула ошибку.
func must(t *testing.T, err error, action string) {
	t.Helper()
	if err != nil {
		t.Fatalf("ошибка %s: %v", action, err)
	}
}

// expectEqual сравнивает полученное значение с ожидаемым.
func expectEqual[T comparable](t *testing.T, what string, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("ожидалось %s=%v, получено: %v", what, want, got)
	}
}

func expectNotFound(t *testing.T, err error, when string) {
	t.Helper()
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("ожидалась ErrNotFound %s, получено: %v", when, err)
	}
}

func tomorrow() time.Time {
	return time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour)
}

func newSlot(doctorID uuid.UUID, cabinet string) *domain.ScheduleSlot {
	day := tomorrow()
	return &domain.ScheduleSlot{
		ID:        uuid.New(),
		DoctorID:  doctorID,
		Date:      day,
		StartTime: day.Add(9 * time.Hour),
		EndTime:   day.Add(10 * time.Hour),
		Cabinet:   cabinet,
	}
}

func newAppointment(patientID, doctorID uuid.UUID) *domain.Appointment {
	return &domain.Appointment{
		ID:        uuid.New(),
		PatientID: patientID,
		DoctorID:  doctorID,
		Status:    domain.StatusCreated,
		Date:      time.Now().Add(48 * time.Hour),
		CreatedAt: time.Now(),
	}
}

func newVisit(appointmentID uuid.UUID, complaints string) *domain.Visit {
	return &domain.Visit{
		ID:            uuid.New(),
		AppointmentID: appointmentID,
		Complaints:    complaints,
		Status:        domain.VisitStatusInProgress,
		CreatedAt:     time.Now(),
	}
}

// Тест 1: Сохранение и извлечение данных

func TestMemory_SaveAndRetrieve(t *testing.T) {
	ctx := context.Background()
	patientID := savePatientAndRetrieve(ctx, t)
	doctorID := uuid.New()
	appID := saveAppointmentAndRetrieve(ctx, t, patientID, doctorID)
	saveVisitAndRetrieve(ctx, t, appID)
	saveSlotAndFindFree(ctx, t, doctorID)
}

func savePatientAndRetrieve(ctx context.Context, t *testing.T) uuid.UUID {
	t.Helper()
	repo := NewPatientRepository()
	patient := &domain.Patient{
		ID:        uuid.New(),
		FullName:  "Иванов Иван Иванович",
		Phone:     "+79001234567",
		Email:     "ivanov@example.com",
		BirthDate: time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC),
	}
	must(t, repo.Save(ctx, patient), "сохранения пациента")

	found, err := repo.FindByID(ctx, patient.ID)
	must(t, err, "поиска пациента")
	expectEqual(t, "ФИО", found.FullName, "Иванов Иван Иванович")
	expectEqual(t, "email", found.Email, "ivanov@example.com")

	byEmail, err := repo.FindByEmail(ctx, "ivanov@example.com")
	must(t, err, "поиска по email")
	expectEqual(t, "ID", byEmail.ID, patient.ID)
	return patient.ID
}

func saveAppointmentAndRetrieve(ctx context.Context, t *testing.T, patientID, doctorID uuid.UUID) uuid.UUID {
	t.Helper()
	repo := NewAppointmentRepository()
	app := newAppointment(patientID, doctorID)
	must(t, repo.Save(ctx, app), "сохранения записи")

	found, err := repo.FindByID(ctx, app.ID)
	must(t, err, "поиска записи")
	expectEqual(t, "PatientID", found.PatientID, patientID)
	expectEqual(t, "статус", found.Status, domain.StatusCreated)

	byPatient, err := repo.FindByPatient(ctx, patientID)
	must(t, err, "поиска по пациенту")
	expectEqual(t, "число записей", len(byPatient), 1)

	busy, err := repo.CheckSlotBusy(ctx, doctorID, app.Date)
	must(t, err, "проверки слота")
	expectEqual(t, "слот занят", busy, true)
	return app.ID
}

func saveVisitAndRetrieve(ctx context.Context, t *testing.T, appID uuid.UUID) {
	t.Helper()
	repo := NewVisitRepository()
	must(t, repo.Save(ctx, newVisit(appID, "Головная боль")), "сохранения визита")

	found, err := repo.FindByAppointment(ctx, appID)
	must(t, err, "поиска визита")
	expectEqual(t, "жалобы", found.Complaints, "Головная боль")
}

func saveSlotAndFindFree(ctx context.Context, t *testing.T, doctorID uuid.UUID) {
	t.Helper()
	repo := NewScheduleRepository()
	slot := newSlot(doctorID, "101")
	must(t, repo.Save(ctx, slot), "сохранения слота")

	free, err := repo.FindFreeSlots(ctx, doctorID, slot.Date)
	must(t, err, "поиска свободных слотов")
	expectEqual(t, "число свободных слотов", len(free), 1)
}

// Тест 2: Изменение данных

func TestMemory_Update(t *testing.T) {
	ctx := context.Background()
	t.Run("отмена записи", func(t *testing.T) { updateAppointment(ctx, t) })
	t.Run("завершение визита", func(t *testing.T) { updateVisit(ctx, t) })
	t.Run("телефон пациента", func(t *testing.T) { updatePatient(ctx, t) })
}

func updateAppointment(ctx context.Context, t *testing.T) {
	repo := NewAppointmentRepository()
	app := newAppointment(uuid.New(), uuid.New())
	must(t, repo.Save(ctx, app), "сохранения записи")

	app.Status = domain.StatusCancelled
	must(t, repo.Update(ctx, app), "обновления записи")

	updated, err := repo.FindByID(ctx, app.ID)
	must(t, err, "поиска")
	expectEqual(t, "статус", updated.Status, domain.StatusCancelled)
}

func updateVisit(ctx context.Context, t *testing.T) {
	repo := NewVisitRepository()
	visit := newVisit(uuid.New(), "Кашель")
	must(t, repo.Save(ctx, visit), "сохранения визита")

	visit.Diagnosis = "ОРВИ"
	visit.Recommendations = "Постельный режим"
	visit.Status = domain.VisitStatusCompleted
	must(t, repo.Update(ctx, visit), "обновления визита")

	updated, err := repo.FindByID(ctx, visit.ID)
	must(t, err, "поиска визита")
	expectEqual(t, "статус", updated.Status, domain.VisitStatusCompleted)
	expectEqual(t, "диагноз", updated.Diagnosis, "ОРВИ")
}

func updatePatient(ctx context.Context, t *testing.T) {
	repo := NewPatientRepository()
	patient := &domain.Patient{
		ID:        uuid.New(),
		FullName:  "Петров Пётр",
		Phone:     "+79009876543",
		Email:     "petrov@example.com",
		BirthDate: time.Date(1985, 3, 20, 0, 0, 0, 0, time.UTC),
	}
	must(t, repo.Save(ctx, patient), "сохранения пациента")

	patient.Phone = "+79001111111"
	must(t, repo.Update(ctx, patient), "обновления пациента")

	updated, err := repo.FindByID(ctx, patient.ID)
	must(t, err, "поиска пациента")
	expectEqual(t, "телефон", updated.Phone, "+79001111111")
}

// Тест 3: Удаление данных

func TestMemory_Delete(t *testing.T) {
	ctx := context.Background()
	t.Run("запись на приём", func(t *testing.T) { deleteAppointment(ctx, t) })
	t.Run("пациент", func(t *testing.T) { deletePatient(ctx, t) })
	t.Run("визит", func(t *testing.T) { deleteVisit(ctx, t) })
	t.Run("слот расписания", func(t *testing.T) { deleteSlot(ctx, t) })
}

func deleteAppointment(ctx context.Context, t *testing.T) {
	repo := NewAppointmentRepository()
	app := newAppointment(uuid.New(), uuid.New())
	must(t, repo.Save(ctx, app), "сохранения")
	must(t, repo.Delete(ctx, app.ID), "удаления")

	_, err := repo.FindByID(ctx, app.ID)
	expectNotFound(t, err, "после удаления")

	// Повторное удаление — ошибка
	expectNotFound(t, repo.Delete(ctx, app.ID), "при повторном удалении")
}

func deletePatient(ctx context.Context, t *testing.T) {
	repo := NewPatientRepository()
	patient := &domain.Patient{
		ID:        uuid.New(),
		FullName:  "Сидоров Сидор",
		Phone:     "+79005555555",
		Email:     "sidorov@example.com",
		BirthDate: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	must(t, repo.Save(ctx, patient), "сохранения пациента")
	must(t, repo.Delete(ctx, patient.ID), "удаления пациента")

	_, err := repo.FindByID(ctx, patient.ID)
	expectNotFound(t, err, "после удаления пациента")

	all, err := repo.FindAll(ctx)
	must(t, err, "FindAll")
	expectEqual(t, "число пациентов", len(all), 0)
}

func deleteVisit(ctx context.Context, t *testing.T) {
	repo := NewVisitRepository()
	visit := newVisit(uuid.New(), "Жалобы")
	must(t, repo.Save(ctx, visit), "сохранения визита")
	must(t, repo.Delete(ctx, visit.ID), "удаления визита")

	_, err := repo.FindByID(ctx, visit.ID)
	expectNotFound(t, err, "после удаления визита")
}

func deleteSlot(ctx context.Context, t *testing.T) {
	repo := NewScheduleRepository()
	slot := newSlot(uuid.New(), "205")
	must(t, repo.Save(ctx, slot), "сохранения слота")
	must(t, repo.Delete(ctx, slot.ID), "удаления слота")

	slots, err := repo.FindByDoctorID(ctx, slot.DoctorID)
	must(t, err, "FindByDoctorID")
	expectEqual(t, "число слотов", len(slots), 0)
}

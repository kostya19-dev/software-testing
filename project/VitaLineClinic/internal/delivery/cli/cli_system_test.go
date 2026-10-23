package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
	"vitaline-clinic/internal/storage/memory"
	"vitaline-clinic/internal/usecase"
)

func TestTechnicalUIRunsMainUseCases(t *testing.T) {
	patientRepo := memory.NewPatientRepository()
	appointmentRepo := memory.NewAppointmentRepository()
	visitRepo := memory.NewVisitRepository()
	scheduleRepo := memory.NewScheduleRepository()

	ctx := context.Background()
	patientID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	doctorID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	cancelAppointmentID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	visitAppointmentID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	completeVisitID := uuid.MustParse("55555555-5555-5555-5555-555555555555")
	slotID := uuid.MustParse("66666666-6666-6666-6666-666666666666")

	mustSavePatient(t, ctx, patientRepo, &domain.Patient{
		ID:        patientID,
		FullName:  "Ivanov Ivan Ivanovich",
		Phone:     "+79001234567",
		Email:     "ivanov@example.com",
		BirthDate: time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC),
		CreatedAt: time.Now(),
	})
	mustSaveAppointment(t, ctx, appointmentRepo, &domain.Appointment{
		ID:        cancelAppointmentID,
		PatientID: patientID,
		DoctorID:  doctorID,
		Status:    domain.StatusCreated,
		Date:      time.Now().Add(48 * time.Hour),
		CreatedAt: time.Now(),
	})
	mustSaveAppointment(t, ctx, appointmentRepo, &domain.Appointment{
		ID:        visitAppointmentID,
		PatientID: patientID,
		DoctorID:  doctorID,
		Status:    domain.StatusCreated,
		Date:      time.Now().Add(72 * time.Hour),
		CreatedAt: time.Now(),
	})
	mustSaveVisit(t, ctx, visitRepo, &domain.Visit{
		ID:            completeVisitID,
		AppointmentID: cancelAppointmentID,
		Complaints:    "Cough",
		Status:        domain.VisitStatusInProgress,
		CreatedAt:     time.Now(),
	})
	mustSaveSlot(t, ctx, scheduleRepo, &domain.ScheduleSlot{
		ID:        slotID,
		DoctorID:  doctorID,
		Date:      time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC),
		StartTime: time.Date(2026, 5, 22, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 5, 22, 9, 30, 0, 0, time.UTC),
		Cabinet:   "101",
		IsBooked:  false,
	})

	input := strings.Join([]string{
		"1", "1", "Petrov Petr Petrovich", "+79007654321", "petrov@example.com", "1985-03-20",
		"2", patientID.String(),
		"3", "ivanov@example.com",
		"4",
		"0",
		"2", "1", doctorID.String(), "2026-05-22", "10:00", "10:30", "102",
		"2", doctorID.String(),
		"3", doctorID.String(), "2026-05-22",
		"4", slotID.String(),
		"0",
		"3", "1", patientID.String(), doctorID.String(), time.Now().Add(96*time.Hour).Format(dateTimeLayout),
		"2", cancelAppointmentID.String(),
		"3", visitAppointmentID.String(),
		"4",
		"5", patientID.String(),
		"0",
		"4", "1", visitAppointmentID.String(), "Headache",
		"2", completeVisitID.String(), "ARVI", "Rest and warm drinks",
		"3", visitAppointmentID.String(),
		"4", patientID.String(),
		"0",
		"0",
	}, "\n") + "\n"

	var output bytes.Buffer
	app := NewApp(
		usecase.NewPatientService(patientRepo),
		usecase.NewAppointmentService(appointmentRepo),
		usecase.NewVisitService(visitRepo, appointmentRepo),
		usecase.NewScheduleService(scheduleRepo),
		strings.NewReader(input),
		&output,
	)

	app.Run()

	text := output.String()
	for _, want := range []string{
		"Пациент зарегистрирован",
		"Слот создан",
		"Запись создана",
		"Запись отменена",
		"Приём начат",
		"Приём завершён",
		"До свидания",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("ожидался вывод %q, полный вывод:\n%s", want, text)
		}
	}
	if strings.Contains(text, "Ошибка") {
		t.Fatalf("технический UI завершился с ошибкой:\n%s", text)
	}
}

func mustSavePatient(t *testing.T, ctx context.Context, repo *memory.PatientRepository, patient *domain.Patient) {
	t.Helper()
	if err := repo.Save(ctx, patient); err != nil {
		t.Fatalf("save patient: %v", err)
	}
}

func mustSaveAppointment(t *testing.T, ctx context.Context, repo *memory.AppointmentRepository, appointment *domain.Appointment) {
	t.Helper()
	if err := repo.Save(ctx, appointment); err != nil {
		t.Fatalf("save appointment: %v", err)
	}
}

func mustSaveVisit(t *testing.T, ctx context.Context, repo *memory.VisitRepository, visit *domain.Visit) {
	t.Helper()
	if err := repo.Save(ctx, visit); err != nil {
		t.Fatalf("save visit: %v", err)
	}
}

func mustSaveSlot(t *testing.T, ctx context.Context, repo *memory.ScheduleRepository, slot *domain.ScheduleSlot) {
	t.Helper()
	if err := repo.Save(ctx, slot); err != nil {
		t.Fatalf("save slot: %v", err)
	}
}

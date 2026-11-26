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

// uiIDs are the fixed identifiers of the records seeded for the menu test.
type uiIDs struct {
	patient, doctor, cancelAppointment, visitAppointment, completeVisit, slot uuid.UUID
}

type uiRepos struct {
	patients     *memory.PatientRepository
	appointments *memory.AppointmentRepository
	visits       *memory.VisitRepository
	schedule     *memory.ScheduleRepository
}

func TestTechnicalUIRunsMainUseCases(t *testing.T) {
	repos := uiRepos{
		patients:     memory.NewPatientRepository(),
		appointments: memory.NewAppointmentRepository(),
		visits:       memory.NewVisitRepository(),
		schedule:     memory.NewScheduleRepository(),
	}
	ids := uiIDs{
		patient:           uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		doctor:            uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		cancelAppointment: uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		visitAppointment:  uuid.MustParse("44444444-4444-4444-4444-444444444444"),
		completeVisit:     uuid.MustParse("55555555-5555-5555-5555-555555555555"),
		slot:              uuid.MustParse("66666666-6666-6666-6666-666666666666"),
	}
	seedTechnicalUI(context.Background(), t, repos, ids)

	var output bytes.Buffer
	app := NewApp(
		usecase.NewPatientService(repos.patients),
		usecase.NewAppointmentService(repos.appointments),
		usecase.NewVisitService(repos.visits, repos.appointments),
		usecase.NewScheduleService(repos.schedule),
		strings.NewReader(technicalUIInput(ids)),
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

func seedTechnicalUI(ctx context.Context, t *testing.T, repos uiRepos, ids uiIDs) {
	t.Helper()
	mustSavePatient(ctx, t, repos.patients, &domain.Patient{
		ID:        ids.patient,
		FullName:  "Ivanov Ivan Ivanovich",
		Phone:     "+79001234567",
		Email:     "ivanov@example.com",
		BirthDate: time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC),
		CreatedAt: time.Now(),
	})
	for _, appointment := range []struct {
		id    uuid.UUID
		after time.Duration
	}{{ids.cancelAppointment, 48 * time.Hour}, {ids.visitAppointment, 72 * time.Hour}} {
		mustSaveAppointment(ctx, t, repos.appointments, &domain.Appointment{
			ID:        appointment.id,
			PatientID: ids.patient,
			DoctorID:  ids.doctor,
			Status:    domain.StatusCreated,
			Date:      time.Now().Add(appointment.after),
			CreatedAt: time.Now(),
		})
	}
	mustSaveVisit(ctx, t, repos.visits, &domain.Visit{
		ID:            ids.completeVisit,
		AppointmentID: ids.cancelAppointment,
		Complaints:    "Cough",
		Status:        domain.VisitStatusInProgress,
		CreatedAt:     time.Now(),
	})
	mustSaveSlot(ctx, t, repos.schedule, &domain.ScheduleSlot{
		ID:        ids.slot,
		DoctorID:  ids.doctor,
		Date:      time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC),
		StartTime: time.Date(2026, 5, 22, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 5, 22, 9, 30, 0, 0, time.UTC),
		Cabinet:   "101",
	})
}

// technicalUIInput is the keyboard input that walks through every menu.
func technicalUIInput(ids uiIDs) string {
	patient, doctor := ids.patient.String(), ids.doctor.String()
	return strings.Join([]string{
		"1", "1", "Petrov Petr Petrovich", "+79007654321", "petrov@example.com", "1985-03-20",
		"2", patient,
		"3", "ivanov@example.com",
		"4",
		"0",
		"2", "1", doctor, "2026-05-22", "10:00", "10:30", "102",
		"2", doctor,
		"3", doctor, "2026-05-22",
		"4", ids.slot.String(),
		"0",
		"3", "1", patient, doctor, time.Now().Add(96 * time.Hour).Format(dateTimeLayout),
		"2", ids.cancelAppointment.String(),
		"3", ids.visitAppointment.String(),
		"4",
		"5", patient,
		"0",
		"4", "1", ids.visitAppointment.String(), "Headache",
		"2", ids.completeVisit.String(), "ARVI", "Rest and warm drinks",
		"3", ids.visitAppointment.String(),
		"4", patient,
		"0",
		"0",
	}, "\n") + "\n"
}

func mustSavePatient(ctx context.Context, t *testing.T, repo *memory.PatientRepository, patient *domain.Patient) {
	t.Helper()
	if err := repo.Save(ctx, patient); err != nil {
		t.Fatalf("save patient: %v", err)
	}
}

func mustSaveAppointment(ctx context.Context, t *testing.T, repo *memory.AppointmentRepository, appointment *domain.Appointment) {
	t.Helper()
	if err := repo.Save(ctx, appointment); err != nil {
		t.Fatalf("save appointment: %v", err)
	}
}

func mustSaveVisit(ctx context.Context, t *testing.T, repo *memory.VisitRepository, visit *domain.Visit) {
	t.Helper()
	if err := repo.Save(ctx, visit); err != nil {
		t.Fatalf("save visit: %v", err)
	}
}

func mustSaveSlot(ctx context.Context, t *testing.T, repo *memory.ScheduleRepository, slot *domain.ScheduleSlot) {
	t.Helper()
	if err := repo.Save(ctx, slot); err != nil {
		t.Fatalf("save slot: %v", err)
	}
}

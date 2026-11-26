package cli

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
	"vitaline-clinic/internal/storage/memory"
	"vitaline-clinic/internal/usecase"
)

type commandFixture struct {
	app         *CommandApp
	out         *bytes.Buffer
	patientID   uuid.UUID
	doctorID    uuid.UUID
	appointment uuid.UUID
	visit       uuid.UUID
	slot        uuid.UUID
}

func newCommandFixture(t *testing.T) commandFixture {
	t.Helper()
	ctx := context.Background()
	patients := memory.NewPatientRepository()
	appointments := memory.NewAppointmentRepository()
	visits := memory.NewVisitRepository()
	schedule := memory.NewScheduleRepository()

	f := commandFixture{
		out:         &bytes.Buffer{},
		patientID:   uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		doctorID:    uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		appointment: uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		visit:       uuid.MustParse("55555555-5555-5555-5555-555555555555"),
		slot:        uuid.MustParse("66666666-6666-6666-6666-666666666666"),
	}
	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	mustSavePatient(ctx, t, patients, &domain.Patient{
		ID: f.patientID, FullName: "Ivanov Ivan", Phone: "+79001234567",
		Email: "ivanov@example.com", BirthDate: time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC), CreatedAt: created,
	})
	mustSaveAppointment(ctx, t, appointments, &domain.Appointment{
		ID: f.appointment, PatientID: f.patientID, DoctorID: f.doctorID, Status: domain.StatusCreated,
		Date: time.Date(2030, 5, 10, 9, 0, 0, 0, time.UTC), CreatedAt: created,
	})
	mustSaveVisit(ctx, t, visits, &domain.Visit{
		ID: f.visit, AppointmentID: f.appointment, Complaints: "Cough",
		Status: domain.VisitStatusInProgress, CreatedAt: created,
	})
	mustSaveSlot(ctx, t, schedule, &domain.ScheduleSlot{
		ID: f.slot, DoctorID: f.doctorID, Date: time.Date(2030, 5, 22, 0, 0, 0, 0, time.UTC),
		StartTime: time.Date(2030, 5, 22, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2030, 5, 22, 9, 30, 0, 0, time.UTC), Cabinet: "101",
	})

	f.app = NewCommandApp("clinic-cli", slog.New(slog.NewTextHandler(io.Discard, nil)),
		usecase.NewPatientService(patients),
		usecase.NewAppointmentService(appointments),
		usecase.NewVisitService(visits, appointments),
		usecase.NewScheduleService(schedule),
		f.out,
	)
	return f
}

func commandScript(f commandFixture) [][]string {
	patient, doctor := f.patientID.String(), f.doctorID.String()
	appointment, visit, slot := f.appointment.String(), f.visit.String(), f.slot.String()
	return [][]string{
		{},
		{"help"},
		{"unknown"},
		{"patient"},
		{"patient", "nope"},
		{"patient", "get", "--id", patient},
		{"patient", "get"},
		{"patient", "get", "--id", "bad"},
		{"patient", "get", "--bad-flag"},
		{"patient", "search", "--email", "ivanov@example.com"},
		{"patient", "list"},
		{"patient", "register", "--birth-date", "bad"},
		{"patient", "register", "--full-name", "Petrov Petr", "--phone", "+79007654321", "--email", "petrov@example.com", "--birth-date", "1985-03-20"},
		{"patient", "list"},
		{"schedule"},
		{"schedule", "nope"},
		{"schedule", "list", "--doctor-id", doctor},
		{"schedule", "free", "--doctor-id", doctor, "--date", "2030-05-22"},
		{"schedule", "free", "--doctor-id", doctor, "--date", "2030-05-23"},
		{"schedule", "free", "--doctor-id", doctor, "--date", "bad"},
		{"schedule", "create", "--doctor-id", doctor, "--date", "2030-05-22", "--start", "10:00", "--end", "10:30", "--cabinet", "102"},
		{"schedule", "create", "--doctor-id", doctor, "--date", "2030-05-22", "--start", "", "--end", "10:30"},
		{"schedule", "create", "--doctor-id", doctor, "--date", "2030-05-22", "--start", "10:00", "--end", "xx"},
		{"schedule", "delete", "--id", slot},
		{"schedule", "list", "--doctor-id", uuid.Nil.String()},
		{"schedule", "list", "--doctor-id", patient},
		{"appointment"},
		{"appointment", "nope"},
		{"appointment", "get", "--id", appointment},
		{"appointment", "list"},
		{"appointment", "list-patient", "--patient-id", patient},
		{"appointment", "list-patient", "--patient-id", doctor},
		{"appointment", "create", "--patient-id", patient, "--doctor-id", doctor, "--date-time", "bad"},
		{"appointment", "create", "--patient-id", patient, "--doctor-id", doctor, "--date-time", "2030-06-01 10:00"},
		{"visit"},
		{"visit", "nope"},
		{"visit", "get-appointment", "--appointment-id", appointment},
		{"visit", "list-patient", "--patient-id", doctor},
		{"visit", "complete", "--visit-id", visit, "--diagnosis", "ARVI", "--recommendations", "Rest"},
		{"visit", "start", "--appointment-id", "bad"},
		{"appointment", "cancel", "--id", appointment},
		{"patient", "delete", "--id", patient},
	}
}

func runCommandScript(t *testing.T) string {
	t.Helper()
	f := newCommandFixture(t)
	for _, args := range commandScript(f) {
		f.out.WriteString("$ " + strings.Join(args, " ") + "\n")
		if err := f.app.Run(context.Background(), args); err != nil {
			f.out.WriteString("error: " + err.Error() + "\n")
		}
	}
	return f.out.String()
}

func TestCommandAppRunsEverySubcommand(t *testing.T) {
	text := runCommandScript(t)
	for _, want := range []string{
		"Команды:",
		"error: неизвестная команда: unknown",
		"error: неизвестная команда patient: nope",
		"error: неизвестная команда schedule: nope",
		"error: неизвестная команда appointment: nope",
		"error: неизвестная команда visit: nope",
		"error: флаг --id обязателен",
		"Использование: clinic-cli patient get --id <uuid>",
		"ФИО: Petrov Petr",
		"--- Пациент 2 ---",
		"--- Свободный слот 1 ---",
		"Свободных слотов нет",
		"Кабинет: 102",
		"error: пустое значение времени",
		"Слот удалён",
		"Расписание пусто",
		"--- Запись 1 ---",
		"Записей нет",
		"error: разбор date-time:",
		"Приёмов нет",
		"Диагноз: ARVI",
		"error: разбор --appointment-id:",
		"Запись отменена",
		"Пациент удалён",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("в выводе нет %q", want)
		}
	}
	if t.Failed() {
		t.Logf("полный вывод:\n%s", text)
	}
}

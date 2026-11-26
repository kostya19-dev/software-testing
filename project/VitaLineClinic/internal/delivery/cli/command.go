package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/usecase"
)

type CommandApp struct {
	appName string
	logger  *slog.Logger
	out     io.Writer

	patients     usecase.IPatientService
	appointments usecase.IAppointmentService
	visits       usecase.IVisitService
	schedule     usecase.IScheduleService
}

func NewCommandApp(
	appName string,
	logger *slog.Logger,
	patients usecase.IPatientService,
	appointments usecase.IAppointmentService,
	visits usecase.IVisitService,
	schedule usecase.IScheduleService,
	out io.Writer,
) *CommandApp {
	if out == nil {
		out = os.Stdout
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &CommandApp{
		appName:      appName,
		logger:       logger,
		out:          out,
		patients:     patients,
		appointments: appointments,
		visits:       visits,
		schedule:     schedule,
	}
}

func (a *CommandApp) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		a.printRootUsage()
		return nil
	}

	a.logger.Info("cli command invoked", "args", strings.Join(args, " "))

	switch args[0] {
	case "help", "--help", "-h":
		a.printRootUsage()
		return nil
	case "patient":
		return a.runPatient(ctx, args[1:])
	case "schedule":
		return a.runSchedule(ctx, args[1:])
	case "appointment":
		return a.runAppointment(ctx, args[1:])
	case "visit":
		return a.runVisit(ctx, args[1:])
	default:
		return fmt.Errorf("неизвестная команда: %s", args[0])
	}
}

// subcommands maps a subcommand name to its handler; handlers receive the
// arguments that follow the subcommand name.
type subcommands map[string]func(ctx context.Context, args []string) error

func (a *CommandApp) dispatch(ctx context.Context, group string, args []string, usage func(), commands subcommands) error {
	if len(args) == 0 {
		usage()
		return nil
	}
	handler, ok := commands[args[0]]
	if !ok {
		usage()
		return fmt.Errorf("неизвестная команда %s: %s", group, args[0])
	}
	return handler(ctx, args[1:])
}

func (a *CommandApp) newFlagSet(name, usage string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(a.out)
	fs.Usage = func() {
		fmt.Fprintln(a.out, "Использование: clinic-cli "+usage)
	}
	return fs
}

// parseIDFlag handles subcommands whose only argument is a single UUID flag.
func (a *CommandApp) parseIDFlag(command, name, description string, args []string) (uuid.UUID, error) {
	fs := a.newFlagSet(command, command+" --"+name+" <uuid>")
	value := fs.String(name, "", description)
	if err := fs.Parse(args); err != nil {
		return uuid.Nil, err
	}
	return parseUUIDFlag(*value, name)
}

func (a *CommandApp) actionFailed(command string, err error, attrs ...any) error {
	args := append([]any{"command", command}, attrs...)
	args = append(args, "error", err)
	a.logger.Error("cli action failed", args...)
	return err
}

func printList[T any](a *CommandApp, items []T, empty, title string, show func(T)) {
	if len(items) == 0 {
		fmt.Fprintln(a.out, empty)
		return
	}
	for i, item := range items {
		fmt.Fprintf(a.out, "--- %s %d ---\n", title, i+1)
		show(item)
		fmt.Fprintln(a.out)
	}
}

func (a *CommandApp) printRootUsage() {
	fmt.Fprintf(a.out, "%s — технологический CLI для VitaLine Clinic\n\n", a.appName)
	fmt.Fprintln(a.out, "Команды:")
	fmt.Fprintln(a.out, "  patient register|get|search|list|delete")
	fmt.Fprintln(a.out, "  schedule create|list|free|delete")
	fmt.Fprintln(a.out, "  appointment create|cancel|get|list|list-patient")
	fmt.Fprintln(a.out, "  visit start|complete|get-appointment|list-patient")
	fmt.Fprintln(a.out)
	fmt.Fprintln(a.out, "Пример:")
	fmt.Fprintln(a.out, "  clinic-cli patient register --full-name \"Иван Иванов\" --phone +79001234567 --email ivanov@example.com --birth-date 2000-05-10")
}

func (a *CommandApp) printPatientUsage() {
	fmt.Fprintln(a.out, "Команды patient:")
	fmt.Fprintln(a.out, "  patient register --full-name ... --phone ... --email ... --birth-date 2000-05-10")
	fmt.Fprintln(a.out, "  patient get --id <uuid>")
	fmt.Fprintln(a.out, "  patient search --email ivanov@example.com")
	fmt.Fprintln(a.out, "  patient list")
	fmt.Fprintln(a.out, "  patient delete --id <uuid>")
}

func (a *CommandApp) printScheduleUsage() {
	fmt.Fprintln(a.out, "Команды schedule:")
	fmt.Fprintln(a.out, "  schedule create --doctor-id <uuid> --date 2026-04-25 --start 09:00 --end 09:30 --cabinet 101")
	fmt.Fprintln(a.out, "  schedule list --doctor-id <uuid>")
	fmt.Fprintln(a.out, "  schedule free --doctor-id <uuid> --date 2026-04-25")
	fmt.Fprintln(a.out, "  schedule delete --id <uuid>")
}

func (a *CommandApp) printAppointmentUsage() {
	fmt.Fprintln(a.out, "Команды appointment:")
	fmt.Fprintln(a.out, "  appointment create --patient-id <uuid> --doctor-id <uuid> --date-time \"2026-04-25 10:00\"")
	fmt.Fprintln(a.out, "  appointment cancel --id <uuid>")
	fmt.Fprintln(a.out, "  appointment get --id <uuid>")
	fmt.Fprintln(a.out, "  appointment list")
	fmt.Fprintln(a.out, "  appointment list-patient --patient-id <uuid>")
}

func (a *CommandApp) printVisitUsage() {
	fmt.Fprintln(a.out, "Команды visit:")
	fmt.Fprintln(a.out, "  visit start --appointment-id <uuid> --complaints \"Головная боль\"")
	fmt.Fprintln(a.out, "  visit complete --visit-id <uuid> --diagnosis \"ОРВИ\" --recommendations \"Покой\"")
	fmt.Fprintln(a.out, "  visit get-appointment --appointment-id <uuid>")
	fmt.Fprintln(a.out, "  visit list-patient --patient-id <uuid>")
}

func (a *CommandApp) printPatient(patient usecase.PatientResponseDTO) {
	fmt.Fprintf(a.out, "ID: %s\n", patient.ID)
	fmt.Fprintf(a.out, "ФИО: %s\n", patient.FullName)
	fmt.Fprintf(a.out, "Телефон: %s\n", patient.Phone)
	fmt.Fprintf(a.out, "Email: %s\n", patient.Email)
	fmt.Fprintf(a.out, "Дата рождения: %s\n", patient.BirthDate.Format(dateLayout))
	fmt.Fprintf(a.out, "Создан: %s\n", patient.CreatedAt.Format(time.RFC3339))
}

func (a *CommandApp) printSlot(slot usecase.ScheduleSlotResponseDTO) {
	fmt.Fprintf(a.out, "ID: %s\n", slot.ID)
	fmt.Fprintf(a.out, "ID врача: %s\n", slot.DoctorID)
	fmt.Fprintf(a.out, "Дата: %s\n", slot.Date.Format(dateLayout))
	fmt.Fprintf(a.out, "Начало: %s\n", slot.StartTime.Format(timeLayout))
	fmt.Fprintf(a.out, "Окончание: %s\n", slot.EndTime.Format(timeLayout))
	fmt.Fprintf(a.out, "Кабинет: %s\n", slot.Cabinet)
	fmt.Fprintf(a.out, "Занят: %t\n", slot.IsBooked)
}

func (a *CommandApp) printAppointment(app usecase.AppointmentResponseDTO) {
	fmt.Fprintf(a.out, "ID: %s\n", app.ID)
	fmt.Fprintf(a.out, "ID пациента: %s\n", app.PatientID)
	fmt.Fprintf(a.out, "ID врача: %s\n", app.DoctorID)
	fmt.Fprintf(a.out, "Статус: %s\n", app.Status)
	fmt.Fprintf(a.out, "Дата: %s\n", app.Date.Format(dateTimeLayout))
}

func (a *CommandApp) printVisit(visit usecase.VisitResponseDTO) {
	fmt.Fprintf(a.out, "ID: %s\n", visit.ID)
	fmt.Fprintf(a.out, "ID записи: %s\n", visit.AppointmentID)
	fmt.Fprintf(a.out, "Жалобы: %s\n", visit.Complaints)
	fmt.Fprintf(a.out, "Диагноз: %s\n", visit.Diagnosis)
	fmt.Fprintf(a.out, "Рекомендации: %s\n", visit.Recommendations)
	fmt.Fprintf(a.out, "Статус: %s\n", visit.Status)
}

func parseUUIDFlag(value, field string) (uuid.UUID, error) {
	if strings.TrimSpace(value) == "" {
		return uuid.Nil, fmt.Errorf("флаг --%s обязателен", field)
	}

	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return uuid.Nil, fmt.Errorf("разбор --%s: %w", field, err)
	}

	return id, nil
}

func parseTimeOnDate(date time.Time, value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, fmt.Errorf("пустое значение времени")
	}

	parsed, err := time.Parse(timeLayout, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, fmt.Errorf("разбор времени: %w", err)
	}

	return time.Date(
		date.Year(), date.Month(), date.Day(),
		parsed.Hour(), parsed.Minute(), 0, 0,
		date.Location(),
	), nil
}

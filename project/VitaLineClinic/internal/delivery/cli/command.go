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

func (a *CommandApp) runPatient(ctx context.Context, args []string) error {
	if len(args) == 0 {
		a.printPatientUsage()
		return nil
	}

	switch args[0] {
	case "register":
		fs := flag.NewFlagSet("patient register", flag.ContinueOnError)
		fs.SetOutput(a.out)
		fullName := fs.String("full-name", "", "ФИО пациента")
		phone := fs.String("phone", "", "Телефон пациента")
		email := fs.String("email", "", "Email пациента")
		birthDate := fs.String("birth-date", "", "Дата рождения в формате ГГГГ-ММ-ДД")
		fs.Usage = func() {
			fmt.Fprintln(a.out, "Использование: clinic-cli patient register --full-name \"Иван Иванов\" --phone +79001234567 --email ivanov@example.com --birth-date 2000-05-10")
		}
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}

		date, err := time.Parse(dateLayout, strings.TrimSpace(*birthDate))
		if err != nil {
			return fmt.Errorf("разбор birth-date: %w", err)
		}

		a.logger.Info("cli action started", "command", "patient register", "email", strings.TrimSpace(*email))
		res, err := a.patients.Register(ctx, usecase.CreatePatientDTO{
			FullName:  strings.TrimSpace(*fullName),
			Phone:     strings.TrimSpace(*phone),
			Email:     strings.TrimSpace(*email),
			BirthDate: date,
		})
		if err != nil {
			a.logger.Error("cli action failed", "command", "patient register", "error", err)
			return err
		}

		a.logger.Info("cli action completed", "command", "patient register", "patient_id", res.ID)
		a.printPatient(res)
		return nil

	case "get":
		fs := flag.NewFlagSet("patient get", flag.ContinueOnError)
		fs.SetOutput(a.out)
		id := fs.String("id", "", "UUID пациента")
		fs.Usage = func() {
			fmt.Fprintln(a.out, "Использование: clinic-cli patient get --id <uuid>")
		}
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}

		patientID, err := parseUUIDFlag(*id, "id")
		if err != nil {
			return err
		}

		a.logger.Info("cli action started", "command", "patient get", "patient_id", patientID)
		res, err := a.patients.GetByID(ctx, patientID)
		if err != nil {
			a.logger.Error("cli action failed", "command", "patient get", "patient_id", patientID, "error", err)
			return err
		}

		a.printPatient(res)
		return nil

	case "search":
		fs := flag.NewFlagSet("patient search", flag.ContinueOnError)
		fs.SetOutput(a.out)
		email := fs.String("email", "", "Email пациента")
		fs.Usage = func() {
			fmt.Fprintln(a.out, "Использование: clinic-cli patient search --email ivanov@example.com")
		}
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}

		a.logger.Info("cli action started", "command", "patient search", "email", strings.TrimSpace(*email))
		res, err := a.patients.FindByEmail(ctx, strings.TrimSpace(*email))
		if err != nil {
			a.logger.Error("cli action failed", "command", "patient search", "email", strings.TrimSpace(*email), "error", err)
			return err
		}

		a.printPatient(res)
		return nil

	case "list":
		a.logger.Info("cli action started", "command", "patient list")
		list, err := a.patients.GetAll(ctx)
		if err != nil {
			a.logger.Error("cli action failed", "command", "patient list", "error", err)
			return err
		}
		if len(list) == 0 {
			fmt.Fprintln(a.out, "Список пациентов пуст")
			return nil
		}
		for i, patient := range list {
			fmt.Fprintf(a.out, "--- Пациент %d ---\n", i+1)
			a.printPatient(patient)
			fmt.Fprintln(a.out)
		}
		return nil

	case "delete":
		fs := flag.NewFlagSet("patient delete", flag.ContinueOnError)
		fs.SetOutput(a.out)
		id := fs.String("id", "", "UUID пациента")
		fs.Usage = func() {
			fmt.Fprintln(a.out, "Использование: clinic-cli patient delete --id <uuid>")
		}
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}

		patientID, err := parseUUIDFlag(*id, "id")
		if err != nil {
			return err
		}

		a.logger.Info("cli action started", "command", "patient delete", "patient_id", patientID)
		if err := a.patients.Delete(ctx, patientID); err != nil {
			a.logger.Error("cli action failed", "command", "patient delete", "patient_id", patientID, "error", err)
			return err
		}

		a.logger.Info("cli action completed", "command", "patient delete", "patient_id", patientID)
		fmt.Fprintln(a.out, "Пациент удалён")
		return nil

	default:
		a.printPatientUsage()
		return fmt.Errorf("неизвестная команда patient: %s", args[0])
	}
}

func (a *CommandApp) runSchedule(ctx context.Context, args []string) error {
	if len(args) == 0 {
		a.printScheduleUsage()
		return nil
	}

	switch args[0] {
	case "create":
		fs := flag.NewFlagSet("schedule create", flag.ContinueOnError)
		fs.SetOutput(a.out)
		doctorID := fs.String("doctor-id", "", "UUID врача")
		date := fs.String("date", "", "Дата в формате ГГГГ-ММ-ДД")
		start := fs.String("start", "", "Время начала в формате ЧЧ:ММ")
		end := fs.String("end", "", "Время окончания в формате ЧЧ:ММ")
		cabinet := fs.String("cabinet", "", "Кабинет")
		fs.Usage = func() {
			fmt.Fprintln(a.out, "Использование: clinic-cli schedule create --doctor-id <uuid> --date 2026-04-25 --start 09:00 --end 09:30 --cabinet 101")
		}
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}

		parsedDoctorID, err := parseUUIDFlag(*doctorID, "doctor-id")
		if err != nil {
			return err
		}
		parsedDate, err := time.Parse(dateLayout, strings.TrimSpace(*date))
		if err != nil {
			return fmt.Errorf("разбор date: %w", err)
		}
		parsedStart, err := parseTimeOnDate(parsedDate, *start)
		if err != nil {
			return err
		}
		parsedEnd, err := parseTimeOnDate(parsedDate, *end)
		if err != nil {
			return err
		}

		a.logger.Info("cli action started", "command", "schedule create", "doctor_id", parsedDoctorID, "date", parsedDate.Format(dateLayout))
		res, err := a.schedule.CreateSlot(ctx, usecase.CreateScheduleSlotDTO{
			DoctorID:  parsedDoctorID,
			Date:      parsedDate,
			StartTime: parsedStart,
			EndTime:   parsedEnd,
			Cabinet:   strings.TrimSpace(*cabinet),
		})
		if err != nil {
			a.logger.Error("cli action failed", "command", "schedule create", "doctor_id", parsedDoctorID, "error", err)
			return err
		}

		a.printSlot(res)
		return nil

	case "list":
		fs := flag.NewFlagSet("schedule list", flag.ContinueOnError)
		fs.SetOutput(a.out)
		doctorID := fs.String("doctor-id", "", "UUID врача")
		fs.Usage = func() {
			fmt.Fprintln(a.out, "Использование: clinic-cli schedule list --doctor-id <uuid>")
		}
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}

		parsedDoctorID, err := parseUUIDFlag(*doctorID, "doctor-id")
		if err != nil {
			return err
		}

		list, err := a.schedule.GetByDoctor(ctx, parsedDoctorID)
		if err != nil {
			a.logger.Error("cli action failed", "command", "schedule list", "doctor_id", parsedDoctorID, "error", err)
			return err
		}
		if len(list) == 0 {
			fmt.Fprintln(a.out, "Расписание пусто")
			return nil
		}
		for i, slot := range list {
			fmt.Fprintf(a.out, "--- Слот %d ---\n", i+1)
			a.printSlot(slot)
			fmt.Fprintln(a.out)
		}
		return nil

	case "free":
		fs := flag.NewFlagSet("schedule free", flag.ContinueOnError)
		fs.SetOutput(a.out)
		doctorID := fs.String("doctor-id", "", "UUID врача")
		date := fs.String("date", "", "Дата в формате ГГГГ-ММ-ДД")
		fs.Usage = func() {
			fmt.Fprintln(a.out, "Использование: clinic-cli schedule free --doctor-id <uuid> --date 2026-04-25")
		}
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}

		parsedDoctorID, err := parseUUIDFlag(*doctorID, "doctor-id")
		if err != nil {
			return err
		}
		parsedDate, err := time.Parse(dateLayout, strings.TrimSpace(*date))
		if err != nil {
			return fmt.Errorf("разбор date: %w", err)
		}

		list, err := a.schedule.GetFreeSlots(ctx, parsedDoctorID, parsedDate)
		if err != nil {
			a.logger.Error("cli action failed", "command", "schedule free", "doctor_id", parsedDoctorID, "error", err)
			return err
		}
		if len(list) == 0 {
			fmt.Fprintln(a.out, "Свободных слотов нет")
			return nil
		}
		for i, slot := range list {
			fmt.Fprintf(a.out, "--- Свободный слот %d ---\n", i+1)
			a.printSlot(slot)
			fmt.Fprintln(a.out)
		}
		return nil

	case "delete":
		fs := flag.NewFlagSet("schedule delete", flag.ContinueOnError)
		fs.SetOutput(a.out)
		id := fs.String("id", "", "UUID слота")
		fs.Usage = func() {
			fmt.Fprintln(a.out, "Использование: clinic-cli schedule delete --id <uuid>")
		}
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}

		slotID, err := parseUUIDFlag(*id, "id")
		if err != nil {
			return err
		}
		if err := a.schedule.DeleteSlot(ctx, slotID); err != nil {
			a.logger.Error("cli action failed", "command", "schedule delete", "slot_id", slotID, "error", err)
			return err
		}

		fmt.Fprintln(a.out, "Слот удалён")
		return nil

	default:
		a.printScheduleUsage()
		return fmt.Errorf("неизвестная команда schedule: %s", args[0])
	}
}

func (a *CommandApp) runAppointment(ctx context.Context, args []string) error {
	if len(args) == 0 {
		a.printAppointmentUsage()
		return nil
	}

	switch args[0] {
	case "create":
		fs := flag.NewFlagSet("appointment create", flag.ContinueOnError)
		fs.SetOutput(a.out)
		patientID := fs.String("patient-id", "", "UUID пациента")
		doctorID := fs.String("doctor-id", "", "UUID врача")
		dateTime := fs.String("date-time", "", "Дата и время в формате ГГГГ-ММ-ДД ЧЧ:ММ")
		fs.Usage = func() {
			fmt.Fprintln(a.out, "Использование: clinic-cli appointment create --patient-id <uuid> --doctor-id <uuid> --date-time \"2026-04-25 10:00\"")
		}
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}

		parsedPatientID, err := parseUUIDFlag(*patientID, "patient-id")
		if err != nil {
			return err
		}
		parsedDoctorID, err := parseUUIDFlag(*doctorID, "doctor-id")
		if err != nil {
			return err
		}
		when, err := time.Parse(dateTimeLayout, strings.TrimSpace(*dateTime))
		if err != nil {
			return fmt.Errorf("разбор date-time: %w", err)
		}

		res, err := a.appointments.Create(ctx, usecase.CreateAppointmentDTO{
			PatientID: parsedPatientID,
			DoctorID:  parsedDoctorID,
			Date:      when,
		})
		if err != nil {
			a.logger.Error("cli action failed", "command", "appointment create", "patient_id", parsedPatientID, "doctor_id", parsedDoctorID, "error", err)
			return err
		}

		a.printAppointment(res)
		return nil

	case "cancel":
		fs := flag.NewFlagSet("appointment cancel", flag.ContinueOnError)
		fs.SetOutput(a.out)
		id := fs.String("id", "", "UUID записи")
		fs.Usage = func() {
			fmt.Fprintln(a.out, "Использование: clinic-cli appointment cancel --id <uuid>")
		}
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}

		appointmentID, err := parseUUIDFlag(*id, "id")
		if err != nil {
			return err
		}
		if err := a.appointments.Cancel(ctx, appointmentID); err != nil {
			a.logger.Error("cli action failed", "command", "appointment cancel", "appointment_id", appointmentID, "error", err)
			return err
		}

		fmt.Fprintln(a.out, "Запись отменена")
		return nil

	case "get":
		fs := flag.NewFlagSet("appointment get", flag.ContinueOnError)
		fs.SetOutput(a.out)
		id := fs.String("id", "", "UUID записи")
		fs.Usage = func() {
			fmt.Fprintln(a.out, "Использование: clinic-cli appointment get --id <uuid>")
		}
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}

		appointmentID, err := parseUUIDFlag(*id, "id")
		if err != nil {
			return err
		}
		res, err := a.appointments.GetByID(ctx, appointmentID)
		if err != nil {
			a.logger.Error("cli action failed", "command", "appointment get", "appointment_id", appointmentID, "error", err)
			return err
		}

		a.printAppointment(res)
		return nil

	case "list":
		list, err := a.appointments.GetAll(ctx)
		if err != nil {
			a.logger.Error("cli action failed", "command", "appointment list", "error", err)
			return err
		}
		if len(list) == 0 {
			fmt.Fprintln(a.out, "Записей нет")
			return nil
		}
		for i, app := range list {
			fmt.Fprintf(a.out, "--- Запись %d ---\n", i+1)
			a.printAppointment(app)
			fmt.Fprintln(a.out)
		}
		return nil

	case "list-patient":
		fs := flag.NewFlagSet("appointment list-patient", flag.ContinueOnError)
		fs.SetOutput(a.out)
		patientID := fs.String("patient-id", "", "UUID пациента")
		fs.Usage = func() {
			fmt.Fprintln(a.out, "Использование: clinic-cli appointment list-patient --patient-id <uuid>")
		}
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}

		parsedPatientID, err := parseUUIDFlag(*patientID, "patient-id")
		if err != nil {
			return err
		}
		list, err := a.appointments.GetByPatient(ctx, parsedPatientID)
		if err != nil {
			a.logger.Error("cli action failed", "command", "appointment list-patient", "patient_id", parsedPatientID, "error", err)
			return err
		}
		if len(list) == 0 {
			fmt.Fprintln(a.out, "Записей нет")
			return nil
		}
		for i, app := range list {
			fmt.Fprintf(a.out, "--- Запись %d ---\n", i+1)
			a.printAppointment(app)
			fmt.Fprintln(a.out)
		}
		return nil

	default:
		a.printAppointmentUsage()
		return fmt.Errorf("неизвестная команда appointment: %s", args[0])
	}
}

func (a *CommandApp) runVisit(ctx context.Context, args []string) error {
	if len(args) == 0 {
		a.printVisitUsage()
		return nil
	}

	switch args[0] {
	case "start":
		fs := flag.NewFlagSet("visit start", flag.ContinueOnError)
		fs.SetOutput(a.out)
		appointmentID := fs.String("appointment-id", "", "UUID записи на приём")
		complaints := fs.String("complaints", "", "Жалобы пациента")
		fs.Usage = func() {
			fmt.Fprintln(a.out, "Использование: clinic-cli visit start --appointment-id <uuid> --complaints \"Головная боль\"")
		}
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}

		parsedAppointmentID, err := parseUUIDFlag(*appointmentID, "appointment-id")
		if err != nil {
			return err
		}
		res, err := a.visits.Create(ctx, usecase.CreateVisitDTO{
			AppointmentID: parsedAppointmentID,
			Complaints:    strings.TrimSpace(*complaints),
		})
		if err != nil {
			a.logger.Error("cli action failed", "command", "visit start", "appointment_id", parsedAppointmentID, "error", err)
			return err
		}

		a.printVisit(res)
		return nil

	case "complete":
		fs := flag.NewFlagSet("visit complete", flag.ContinueOnError)
		fs.SetOutput(a.out)
		visitID := fs.String("visit-id", "", "UUID приёма")
		diagnosis := fs.String("diagnosis", "", "Диагноз")
		recommendations := fs.String("recommendations", "", "Рекомендации")
		fs.Usage = func() {
			fmt.Fprintln(a.out, "Использование: clinic-cli visit complete --visit-id <uuid> --diagnosis \"ОРВИ\" --recommendations \"Покой\"")
		}
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}

		parsedVisitID, err := parseUUIDFlag(*visitID, "visit-id")
		if err != nil {
			return err
		}
		res, err := a.visits.Complete(ctx, parsedVisitID, usecase.UpdateVisitDTO{
			Diagnosis:       strings.TrimSpace(*diagnosis),
			Recommendations: strings.TrimSpace(*recommendations),
		})
		if err != nil {
			a.logger.Error("cli action failed", "command", "visit complete", "visit_id", parsedVisitID, "error", err)
			return err
		}

		a.printVisit(res)
		return nil

	case "get-appointment":
		fs := flag.NewFlagSet("visit get-appointment", flag.ContinueOnError)
		fs.SetOutput(a.out)
		appointmentID := fs.String("appointment-id", "", "UUID записи на приём")
		fs.Usage = func() {
			fmt.Fprintln(a.out, "Использование: clinic-cli visit get-appointment --appointment-id <uuid>")
		}
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}

		parsedAppointmentID, err := parseUUIDFlag(*appointmentID, "appointment-id")
		if err != nil {
			return err
		}
		res, err := a.visits.GetByAppointment(ctx, parsedAppointmentID)
		if err != nil {
			a.logger.Error("cli action failed", "command", "visit get-appointment", "appointment_id", parsedAppointmentID, "error", err)
			return err
		}

		a.printVisit(res)
		return nil

	case "list-patient":
		fs := flag.NewFlagSet("visit list-patient", flag.ContinueOnError)
		fs.SetOutput(a.out)
		patientID := fs.String("patient-id", "", "UUID пациента")
		fs.Usage = func() {
			fmt.Fprintln(a.out, "Использование: clinic-cli visit list-patient --patient-id <uuid>")
		}
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}

		parsedPatientID, err := parseUUIDFlag(*patientID, "patient-id")
		if err != nil {
			return err
		}
		list, err := a.visits.GetByPatient(ctx, parsedPatientID)
		if err != nil {
			a.logger.Error("cli action failed", "command", "visit list-patient", "patient_id", parsedPatientID, "error", err)
			return err
		}
		if len(list) == 0 {
			fmt.Fprintln(a.out, "Приёмов нет")
			return nil
		}
		for i, visit := range list {
			fmt.Fprintf(a.out, "--- Приём %d ---\n", i+1)
			a.printVisit(visit)
			fmt.Fprintln(a.out)
		}
		return nil

	default:
		a.printVisitUsage()
		return fmt.Errorf("неизвестная команда visit: %s", args[0])
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

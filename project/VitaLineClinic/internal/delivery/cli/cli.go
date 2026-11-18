package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
	"vitaline-clinic/internal/usecase"
)

const (
	dateLayout     = "2006-01-02"
	timeLayout     = "15:04"
	dateTimeLayout = "2006-01-02 15:04"
)

type App struct {
	patients     usecase.IPatientService
	appointments usecase.IAppointmentService
	visits       usecase.IVisitService
	schedule     usecase.IScheduleService

	in  *bufio.Scanner
	out io.Writer
	ctx context.Context
}

func NewApp(
	patients usecase.IPatientService,
	appointments usecase.IAppointmentService,
	visits usecase.IVisitService,
	schedule usecase.IScheduleService,
	in io.Reader,
	out io.Writer,
) *App {
	return &App{
		patients:     patients,
		appointments: appointments,
		visits:       visits,
		schedule:     schedule,
		in:           bufio.NewScanner(in),
		out:          out,
		ctx:          context.Background(),
	}
}

func (a *App) Run() {
	a.println("=== VitaLine Clinic — Технологический UI ===")
	a.println("Системное тестирование всех Use Case.")
	a.println("")

	for {
		a.println("--- Главное меню ---")
		a.println("1. Пациенты")
		a.println("2. Расписание врачей")
		a.println("3. Записи на приём")
		a.println("4. Приёмы (визиты)")
		a.println("0. Выход")

		choice, ok := a.readLine("Выбор")
		if !ok {
			a.println("Ввод завершён")
			return
		}

		switch choice {
		case "1":
			a.patientMenu()
		case "2":
			a.scheduleMenu()
		case "3":
			a.appointmentMenu()
		case "4":
			a.visitMenu()
		case "0":
			a.println("До свидания!")
			return
		default:
			a.println("Неверный пункт меню")
		}
		a.println("")
	}
}

func (a *App) patientMenu() {
	for {
		a.println("")
		a.println("--- Пациенты ---")
		a.println("1. Зарегистрировать пациента")
		a.println("2. Найти по ID")
		a.println("3. Найти по email")
		a.println("4. Список всех пациентов")
		a.println("5. Удалить пациента")
		a.println("0. Назад")

		choice, ok := a.readLine("Выбор")
		if !ok {
			return
		}

		switch choice {
		case "1":
			a.registerPatient()
		case "2":
			a.findPatientByID()
		case "3":
			a.findPatientByEmail()
		case "4":
			a.listPatients()
		case "5":
			a.deletePatient()
		case "0":
			return
		default:
			a.println("Неверный пункт меню")
		}
	}
}

func (a *App) registerPatient() {
	fullName, ok := a.readLine("ФИО")
	if !ok {
		return
	}
	phone, ok := a.readLine("Телефон (например +79001234567)")
	if !ok {
		return
	}
	email, ok := a.readLine("Email")
	if !ok {
		return
	}
	birthDate, err := a.readDate("Дата рождения (ГГГГ-ММ-ДД)")
	if err != nil {
		a.printError(err)
		return
	}

	dto := usecase.CreatePatientDTO{
		FullName:  fullName,
		Phone:     phone,
		Email:     email,
		BirthDate: birthDate,
	}

	res, err := a.patients.Register(a.ctx, dto)
	if err != nil {
		a.printError(err)
		return
	}

	a.println("Пациент зарегистрирован:")
	a.printPatient(res)
}

func (a *App) findPatientByID() {
	id, err := a.readUUID("ID пациента")
	if err != nil {
		a.printError(err)
		return
	}
	res, err := a.patients.GetByID(a.ctx, id)
	if err != nil {
		a.printError(err)
		return
	}
	a.printPatient(res)
}

func (a *App) findPatientByEmail() {
	email, ok := a.readLine("Email")
	if !ok {
		return
	}
	res, err := a.patients.FindByEmail(a.ctx, email)
	if err != nil {
		a.printError(err)
		return
	}
	a.printPatient(res)
}

func (a *App) listPatients() {
	patients, err := a.patients.GetAll(a.ctx)
	if err != nil {
		a.printError(err)
		return
	}
	if len(patients) == 0 {
		a.println("Список пациентов пуст")
		return
	}
	for i, p := range patients {
		a.printf("\n--- Пациент %d ---\n", i+1)
		a.printPatient(p)
	}
}

func (a *App) deletePatient() {
	id, err := a.readUUID("ID пациента")
	if err != nil {
		a.printError(err)
		return
	}
	if err := a.patients.Delete(a.ctx, id); err != nil {
		a.printError(err)
		return
	}
	a.println("Пациент удалён")
}

func (a *App) scheduleMenu() {
	for {
		a.println("")
		a.println("--- Расписание врачей ---")
		a.println("1. Добавить слот")
		a.println("2. Расписание врача")
		a.println("3. Свободные слоты врача")
		a.println("4. Удалить слот")
		a.println("0. Назад")

		choice, ok := a.readLine("Выбор")
		if !ok {
			return
		}

		switch choice {
		case "1":
			a.createSlot()
		case "2":
			a.viewDoctorSchedule()
		case "3":
			a.viewFreeSlots()
		case "4":
			a.deleteSlot()
		case "0":
			return
		default:
			a.println("Неверный пункт меню")
		}
	}
}

func (a *App) createSlot() {
	doctorID, err := a.readUUID("ID врача")
	if err != nil {
		a.printError(err)
		return
	}
	date, err := a.readDate("Дата (ГГГГ-ММ-ДД)")
	if err != nil {
		a.printError(err)
		return
	}
	start, err := a.readTimeOnDate("Время начала (ЧЧ:ММ)", date)
	if err != nil {
		a.printError(err)
		return
	}
	end, err := a.readTimeOnDate("Время окончания (ЧЧ:ММ)", date)
	if err != nil {
		a.printError(err)
		return
	}
	cabinet, ok := a.readLine("Кабинет")
	if !ok {
		return
	}

	dto := usecase.CreateScheduleSlotDTO{
		DoctorID:  doctorID,
		Date:      date,
		StartTime: start,
		EndTime:   end,
		Cabinet:   cabinet,
	}

	res, err := a.schedule.CreateSlot(a.ctx, dto)
	if err != nil {
		a.printError(err)
		return
	}

	a.println("Слот создан:")
	a.printSlot(res)
}

func (a *App) viewDoctorSchedule() {
	doctorID, err := a.readUUID("ID врача")
	if err != nil {
		a.printError(err)
		return
	}
	slots, err := a.schedule.GetByDoctor(a.ctx, doctorID)
	if err != nil {
		a.printError(err)
		return
	}
	if len(slots) == 0 {
		a.println("Расписание пусто")
		return
	}
	for i, s := range slots {
		a.printf("\n--- Слот %d ---\n", i+1)
		a.printSlot(s)
	}
}

func (a *App) viewFreeSlots() {
	doctorID, err := a.readUUID("ID врача")
	if err != nil {
		a.printError(err)
		return
	}
	date, err := a.readDate("Дата (ГГГГ-ММ-ДД)")
	if err != nil {
		a.printError(err)
		return
	}
	slots, err := a.schedule.GetFreeSlots(a.ctx, doctorID, date)
	if err != nil {
		a.printError(err)
		return
	}
	if len(slots) == 0 {
		a.println("Свободных слотов нет")
		return
	}
	for i, s := range slots {
		a.printf("\n--- Свободный слот %d ---\n", i+1)
		a.printSlot(s)
	}
}

func (a *App) deleteSlot() {
	id, err := a.readUUID("ID слота")
	if err != nil {
		a.printError(err)
		return
	}
	if err := a.schedule.DeleteSlot(a.ctx, id); err != nil {
		a.printError(err)
		return
	}
	a.println("Слот удалён")
}

func (a *App) appointmentMenu() {
	for {
		a.println("")
		a.println("--- Записи на приём ---")
		a.println("1. Создать запись")
		a.println("2. Отменить запись")
		a.println("3. Просмотр по ID")
		a.println("4. Все записи")
		a.println("5. Записи пациента")
		a.println("0. Назад")

		choice, ok := a.readLine("Выбор")
		if !ok {
			return
		}

		switch choice {
		case "1":
			a.createAppointment()
		case "2":
			a.cancelAppointment()
		case "3":
			a.viewAppointment()
		case "4":
			a.listAppointments()
		case "5":
			a.patientAppointments()
		case "0":
			return
		default:
			a.println("Неверный пункт меню")
		}
	}
}

func (a *App) createAppointment() {
	patientID, err := a.readUUID("ID пациента")
	if err != nil {
		a.printError(err)
		return
	}
	doctorID, err := a.readUUID("ID врача")
	if err != nil {
		a.printError(err)
		return
	}
	when, err := a.readDateTime("Дата и время (ГГГГ-ММ-ДД ЧЧ:ММ)")
	if err != nil {
		a.printError(err)
		return
	}

	dto := usecase.CreateAppointmentDTO{
		PatientID: patientID,
		DoctorID:  doctorID,
		Date:      when,
	}

	res, err := a.appointments.Create(a.ctx, dto)
	if err != nil {
		a.printError(err)
		return
	}

	a.println("Запись создана:")
	a.printAppointment(res)
}

func (a *App) cancelAppointment() {
	id, err := a.readUUID("ID записи")
	if err != nil {
		a.printError(err)
		return
	}
	if err := a.appointments.Cancel(a.ctx, id); err != nil {
		a.printError(err)
		return
	}
	a.println("Запись отменена")
}

func (a *App) viewAppointment() {
	id, err := a.readUUID("ID записи")
	if err != nil {
		a.printError(err)
		return
	}
	res, err := a.appointments.GetByID(a.ctx, id)
	if err != nil {
		a.printError(err)
		return
	}
	a.printAppointment(res)
}

func (a *App) listAppointments() {
	apps, err := a.appointments.GetAll(a.ctx)
	if err != nil {
		a.printError(err)
		return
	}
	if len(apps) == 0 {
		a.println("Записей нет")
		return
	}
	for i, app := range apps {
		a.printf("\n--- Запись %d ---\n", i+1)
		a.printAppointment(app)
	}
}

func (a *App) patientAppointments() {
	patientID, err := a.readUUID("ID пациента")
	if err != nil {
		a.printError(err)
		return
	}
	apps, err := a.appointments.GetByPatient(a.ctx, patientID)
	if err != nil {
		a.printError(err)
		return
	}
	if len(apps) == 0 {
		a.println("Записей нет")
		return
	}
	for i, app := range apps {
		a.printf("\n--- Запись %d ---\n", i+1)
		a.printAppointment(app)
	}
}

func (a *App) visitMenu() {
	for {
		a.println("")
		a.println("--- Приёмы (визиты) ---")
		a.println("1. Начать приём")
		a.println("2. Завершить приём")
		a.println("3. Просмотр по записи")
		a.println("4. Приёмы пациента")
		a.println("0. Назад")

		choice, ok := a.readLine("Выбор")
		if !ok {
			return
		}

		switch choice {
		case "1":
			a.startVisit()
		case "2":
			a.completeVisit()
		case "3":
			a.viewVisitByAppointment()
		case "4":
			a.patientVisits()
		case "0":
			return
		default:
			a.println("Неверный пункт меню")
		}
	}
}

func (a *App) startVisit() {
	appID, err := a.readUUID("ID записи на приём")
	if err != nil {
		a.printError(err)
		return
	}
	complaints, ok := a.readLine("Жалобы пациента")
	if !ok {
		return
	}

	dto := usecase.CreateVisitDTO{
		AppointmentID: appID,
		Complaints:    complaints,
	}

	res, err := a.visits.Create(a.ctx, dto)
	if err != nil {
		a.printError(err)
		return
	}

	a.println("Приём начат:")
	a.printVisit(res)
}

func (a *App) completeVisit() {
	visitID, err := a.readUUID("ID приёма")
	if err != nil {
		a.printError(err)
		return
	}
	diagnosis, ok := a.readLine("Диагноз")
	if !ok {
		return
	}
	recommendations, ok := a.readLine("Рекомендации")
	if !ok {
		return
	}

	dto := usecase.UpdateVisitDTO{
		Diagnosis:       diagnosis,
		Recommendations: recommendations,
	}

	res, err := a.visits.Complete(a.ctx, visitID, dto)
	if err != nil {
		a.printError(err)
		return
	}

	a.println("Приём завершён:")
	a.printVisit(res)
}

func (a *App) viewVisitByAppointment() {
	appID, err := a.readUUID("ID записи на приём")
	if err != nil {
		a.printError(err)
		return
	}
	res, err := a.visits.GetByAppointment(a.ctx, appID)
	if err != nil {
		a.printError(err)
		return
	}
	a.printVisit(res)
}

func (a *App) patientVisits() {
	patientID, err := a.readUUID("ID пациента")
	if err != nil {
		a.printError(err)
		return
	}
	visits, err := a.visits.GetByPatient(a.ctx, patientID)
	if err != nil {
		a.printError(err)
		return
	}
	if len(visits) == 0 {
		a.println("Приёмов нет")
		return
	}
	for i, v := range visits {
		a.printf("\n--- Приём %d ---\n", i+1)
		a.printVisit(v)
	}
}

func (a *App) readLine(prompt string) (string, bool) {
	a.printf("%s: ", prompt)
	if !a.in.Scan() {
		return "", false
	}
	return strings.TrimSpace(a.in.Text()), true
}

func (a *App) readUUID(prompt string) (uuid.UUID, error) {
	s, ok := a.readLine(prompt)
	if !ok {
		return uuid.Nil, errors.New("ввод завершён")
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, fmt.Errorf("некорректный UUID: %w", err)
	}
	return id, nil
}

func (a *App) readDate(prompt string) (time.Time, error) {
	s, ok := a.readLine(prompt)
	if !ok {
		return time.Time{}, errors.New("ввод завершён")
	}
	t, err := time.ParseInLocation(dateLayout, s, time.UTC)
	if err != nil {
		return time.Time{}, fmt.Errorf("некорректная дата: %w", err)
	}
	return t, nil
}

func (a *App) readDateTime(prompt string) (time.Time, error) {
	s, ok := a.readLine(prompt)
	if !ok {
		return time.Time{}, errors.New("ввод завершён")
	}
	t, err := time.ParseInLocation(dateTimeLayout, s, time.UTC)
	if err != nil {
		return time.Time{}, fmt.Errorf("некорректная дата/время: %w", err)
	}
	return t, nil
}

func (a *App) readTimeOnDate(prompt string, date time.Time) (time.Time, error) {
	s, ok := a.readLine(prompt)
	if !ok {
		return time.Time{}, errors.New("ввод завершён")
	}
	t, err := time.ParseInLocation(timeLayout, s, time.UTC)
	if err != nil {
		return time.Time{}, fmt.Errorf("некорректное время: %w", err)
	}
	return time.Date(date.Year(), date.Month(), date.Day(), t.Hour(), t.Minute(), 0, 0, time.UTC), nil
}

func (a *App) println(s string) {
	fmt.Fprintln(a.out, s)
}

func (a *App) printf(format string, args ...any) {
	fmt.Fprintf(a.out, format, args...)
}

func (a *App) printError(err error) {
	var validationErr *domain.ValidationError
	if errors.As(err, &validationErr) {
		a.printf("Ошибка валидации: %s\n", validationErr.Error())
		return
	}
	switch {
	case errors.Is(err, domain.ErrNotFound):
		a.println("Не найдено")
	case errors.Is(err, domain.ErrSlotBusy):
		a.println("Слот занят")
	case errors.Is(err, domain.ErrInvalidStatus):
		a.println("Недопустимый статус для операции")
	case errors.Is(err, domain.ErrPastDate):
		a.println("Нельзя записаться в прошлое")
	default:
		a.printf("Ошибка: %v\n", err)
	}
}

func (a *App) printPatient(p usecase.PatientResponseDTO) {
	a.printf("  ID:            %s\n", p.ID)
	a.printf("  ФИО:           %s\n", p.FullName)
	a.printf("  Телефон:       %s\n", p.Phone)
	a.printf("  Email:         %s\n", p.Email)
	a.printf("  Дата рождения: %s\n", p.BirthDate.Format(dateLayout))
}

func (a *App) printAppointment(app usecase.AppointmentResponseDTO) {
	a.printf("  ID:         %s\n", app.ID)
	a.printf("  Пациент:    %s\n", app.PatientID)
	a.printf("  Врач:       %s\n", app.DoctorID)
	a.printf("  Статус:     %s\n", app.Status)
	a.printf("  Дата:       %s\n", app.Date.Format(dateTimeLayout))
}

func (a *App) printSlot(s usecase.ScheduleSlotResponseDTO) {
	a.printf("  ID:       %s\n", s.ID)
	a.printf("  Врач:     %s\n", s.DoctorID)
	a.printf("  Дата:     %s\n", s.Date.Format(dateLayout))
	a.printf("  Время:    %s — %s\n", s.StartTime.Format(timeLayout), s.EndTime.Format(timeLayout))
	a.printf("  Кабинет:  %s\n", s.Cabinet)
	if s.IsBooked {
		a.println("  Статус:   Забронирован")
	} else {
		a.println("  Статус:   Свободен")
	}
}

func (a *App) printVisit(v usecase.VisitResponseDTO) {
	a.printf("  ID:              %s\n", v.ID)
	a.printf("  Запись:          %s\n", v.AppointmentID)
	a.printf("  Жалобы:          %s\n", v.Complaints)
	a.printf("  Диагноз:         %s\n", v.Diagnosis)
	a.printf("  Рекомендации:    %s\n", v.Recommendations)
	a.printf("  Статус:          %s\n", v.Status)
}

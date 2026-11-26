package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"vitaline-clinic/internal/usecase"
)

func (a *CommandApp) runSchedule(ctx context.Context, args []string) error {
	return a.dispatch(ctx, "schedule", args, a.printScheduleUsage, subcommands{
		"create": a.scheduleCreate,
		"list":   a.scheduleList,
		"free":   a.scheduleFree,
		"delete": a.scheduleDelete,
	})
}

func (a *CommandApp) scheduleCreate(ctx context.Context, args []string) error {
	fs := a.newFlagSet("schedule create", "schedule create --doctor-id <uuid> --date 2026-04-25 --start 09:00 --end 09:30 --cabinet 101")
	doctorID := fs.String("doctor-id", "", "UUID врача")
	date := fs.String("date", "", "Дата в формате ГГГГ-ММ-ДД")
	start := fs.String("start", "", "Время начала в формате ЧЧ:ММ")
	end := fs.String("end", "", "Время окончания в формате ЧЧ:ММ")
	cabinet := fs.String("cabinet", "", "Кабинет")
	if err := fs.Parse(args); err != nil {
		return err
	}

	dto, err := slotFromFlags(*doctorID, *date, *start, *end)
	if err != nil {
		return err
	}
	dto.Cabinet = strings.TrimSpace(*cabinet)

	a.logger.Info("cli action started", "command", "schedule create", "doctor_id", dto.DoctorID, "date", dto.Date.Format(dateLayout))
	res, err := a.schedule.CreateSlot(ctx, dto)
	if err != nil {
		return a.actionFailed("schedule create", err, "doctor_id", dto.DoctorID)
	}

	a.printSlot(res)
	return nil
}

func slotFromFlags(doctorID, date, start, end string) (usecase.CreateScheduleSlotDTO, error) {
	parsedDoctorID, err := parseUUIDFlag(doctorID, "doctor-id")
	if err != nil {
		return usecase.CreateScheduleSlotDTO{}, err
	}
	parsedDate, err := parseDateFlag(date)
	if err != nil {
		return usecase.CreateScheduleSlotDTO{}, err
	}
	parsedStart, err := parseTimeOnDate(parsedDate, start)
	if err != nil {
		return usecase.CreateScheduleSlotDTO{}, err
	}
	parsedEnd, err := parseTimeOnDate(parsedDate, end)
	if err != nil {
		return usecase.CreateScheduleSlotDTO{}, err
	}
	return usecase.CreateScheduleSlotDTO{
		DoctorID:  parsedDoctorID,
		Date:      parsedDate,
		StartTime: parsedStart,
		EndTime:   parsedEnd,
	}, nil
}

func (a *CommandApp) scheduleList(ctx context.Context, args []string) error {
	doctorID, err := a.parseIDFlag("schedule list", "doctor-id", "UUID врача", args)
	if err != nil {
		return err
	}

	list, err := a.schedule.GetByDoctor(ctx, doctorID)
	if err != nil {
		return a.actionFailed("schedule list", err, "doctor_id", doctorID)
	}
	printList(a, list, "Расписание пусто", "Слот", a.printSlot)
	return nil
}

func (a *CommandApp) scheduleFree(ctx context.Context, args []string) error {
	fs := a.newFlagSet("schedule free", "schedule free --doctor-id <uuid> --date 2026-04-25")
	doctorID := fs.String("doctor-id", "", "UUID врача")
	date := fs.String("date", "", "Дата в формате ГГГГ-ММ-ДД")
	if err := fs.Parse(args); err != nil {
		return err
	}

	parsedDoctorID, err := parseUUIDFlag(*doctorID, "doctor-id")
	if err != nil {
		return err
	}
	parsedDate, err := parseDateFlag(*date)
	if err != nil {
		return err
	}

	list, err := a.schedule.GetFreeSlots(ctx, parsedDoctorID, parsedDate)
	if err != nil {
		return a.actionFailed("schedule free", err, "doctor_id", parsedDoctorID)
	}
	printList(a, list, "Свободных слотов нет", "Свободный слот", a.printSlot)
	return nil
}

func (a *CommandApp) scheduleDelete(ctx context.Context, args []string) error {
	slotID, err := a.parseIDFlag("schedule delete", "id", "UUID слота", args)
	if err != nil {
		return err
	}
	if err := a.schedule.DeleteSlot(ctx, slotID); err != nil {
		return a.actionFailed("schedule delete", err, "slot_id", slotID)
	}

	fmt.Fprintln(a.out, "Слот удалён")
	return nil
}

func parseDateFlag(value string) (time.Time, error) {
	parsed, err := time.Parse(dateLayout, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, fmt.Errorf("разбор date: %w", err)
	}
	return parsed, nil
}

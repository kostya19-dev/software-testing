package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"vitaline-clinic/internal/usecase"
)

func (a *CommandApp) runAppointment(ctx context.Context, args []string) error {
	return a.dispatch(ctx, "appointment", args, a.printAppointmentUsage, subcommands{
		"create":       a.appointmentCreate,
		"cancel":       a.appointmentCancel,
		"get":          a.appointmentGet,
		"list":         a.appointmentList,
		"list-patient": a.appointmentListPatient,
	})
}

func (a *CommandApp) appointmentCreate(ctx context.Context, args []string) error {
	fs := a.newFlagSet("appointment create", "appointment create --patient-id <uuid> --doctor-id <uuid> --date-time \"2026-04-25 10:00\"")
	patientID := fs.String("patient-id", "", "UUID пациента")
	doctorID := fs.String("doctor-id", "", "UUID врача")
	dateTime := fs.String("date-time", "", "Дата и время в формате ГГГГ-ММ-ДД ЧЧ:ММ")
	if err := fs.Parse(args); err != nil {
		return err
	}

	dto, err := appointmentFromFlags(*patientID, *doctorID, *dateTime)
	if err != nil {
		return err
	}

	res, err := a.appointments.Create(ctx, dto)
	if err != nil {
		return a.actionFailed("appointment create", err, "patient_id", dto.PatientID, "doctor_id", dto.DoctorID)
	}

	a.printAppointment(res)
	return nil
}

func appointmentFromFlags(patientID, doctorID, dateTime string) (usecase.CreateAppointmentDTO, error) {
	parsedPatientID, err := parseUUIDFlag(patientID, "patient-id")
	if err != nil {
		return usecase.CreateAppointmentDTO{}, err
	}
	parsedDoctorID, err := parseUUIDFlag(doctorID, "doctor-id")
	if err != nil {
		return usecase.CreateAppointmentDTO{}, err
	}
	when, err := time.Parse(dateTimeLayout, strings.TrimSpace(dateTime))
	if err != nil {
		return usecase.CreateAppointmentDTO{}, fmt.Errorf("разбор date-time: %w", err)
	}
	return usecase.CreateAppointmentDTO{PatientID: parsedPatientID, DoctorID: parsedDoctorID, Date: when}, nil
}

func (a *CommandApp) appointmentCancel(ctx context.Context, args []string) error {
	appointmentID, err := a.parseIDFlag("appointment cancel", "id", "UUID записи", args)
	if err != nil {
		return err
	}
	if err := a.appointments.Cancel(ctx, appointmentID); err != nil {
		return a.actionFailed("appointment cancel", err, "appointment_id", appointmentID)
	}

	fmt.Fprintln(a.out, "Запись отменена")
	return nil
}

func (a *CommandApp) appointmentGet(ctx context.Context, args []string) error {
	appointmentID, err := a.parseIDFlag("appointment get", "id", "UUID записи", args)
	if err != nil {
		return err
	}
	res, err := a.appointments.GetByID(ctx, appointmentID)
	if err != nil {
		return a.actionFailed("appointment get", err, "appointment_id", appointmentID)
	}

	a.printAppointment(res)
	return nil
}

func (a *CommandApp) appointmentList(ctx context.Context, _ []string) error {
	list, err := a.appointments.GetAll(ctx)
	if err != nil {
		return a.actionFailed("appointment list", err)
	}
	printList(a, list, "Записей нет", "Запись", a.printAppointment)
	return nil
}

func (a *CommandApp) appointmentListPatient(ctx context.Context, args []string) error {
	patientID, err := a.parseIDFlag("appointment list-patient", "patient-id", "UUID пациента", args)
	if err != nil {
		return err
	}
	list, err := a.appointments.GetByPatient(ctx, patientID)
	if err != nil {
		return a.actionFailed("appointment list-patient", err, "patient_id", patientID)
	}
	printList(a, list, "Записей нет", "Запись", a.printAppointment)
	return nil
}

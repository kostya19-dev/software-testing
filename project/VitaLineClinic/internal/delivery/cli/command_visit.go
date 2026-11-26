package cli

import (
	"context"
	"strings"

	"vitaline-clinic/internal/usecase"
)

func (a *CommandApp) runVisit(ctx context.Context, args []string) error {
	return a.dispatch(ctx, "visit", args, a.printVisitUsage, subcommands{
		"start":           a.visitStart,
		"complete":        a.visitComplete,
		"get-appointment": a.visitGetAppointment,
		"list-patient":    a.visitListPatient,
	})
}

func (a *CommandApp) visitStart(ctx context.Context, args []string) error {
	fs := a.newFlagSet("visit start", "visit start --appointment-id <uuid> --complaints \"Головная боль\"")
	appointmentID := fs.String("appointment-id", "", "UUID записи на приём")
	complaints := fs.String("complaints", "", "Жалобы пациента")
	if err := fs.Parse(args); err != nil {
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
		return a.actionFailed("visit start", err, "appointment_id", parsedAppointmentID)
	}

	a.printVisit(res)
	return nil
}

func (a *CommandApp) visitComplete(ctx context.Context, args []string) error {
	fs := a.newFlagSet("visit complete", "visit complete --visit-id <uuid> --diagnosis \"ОРВИ\" --recommendations \"Покой\"")
	visitID := fs.String("visit-id", "", "UUID приёма")
	diagnosis := fs.String("diagnosis", "", "Диагноз")
	recommendations := fs.String("recommendations", "", "Рекомендации")
	if err := fs.Parse(args); err != nil {
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
		return a.actionFailed("visit complete", err, "visit_id", parsedVisitID)
	}

	a.printVisit(res)
	return nil
}

func (a *CommandApp) visitGetAppointment(ctx context.Context, args []string) error {
	appointmentID, err := a.parseIDFlag("visit get-appointment", "appointment-id", "UUID записи на приём", args)
	if err != nil {
		return err
	}
	res, err := a.visits.GetByAppointment(ctx, appointmentID)
	if err != nil {
		return a.actionFailed("visit get-appointment", err, "appointment_id", appointmentID)
	}

	a.printVisit(res)
	return nil
}

func (a *CommandApp) visitListPatient(ctx context.Context, args []string) error {
	patientID, err := a.parseIDFlag("visit list-patient", "patient-id", "UUID пациента", args)
	if err != nil {
		return err
	}
	list, err := a.visits.GetByPatient(ctx, patientID)
	if err != nil {
		return a.actionFailed("visit list-patient", err, "patient_id", patientID)
	}
	printList(a, list, "Приёмов нет", "Приём", a.printVisit)
	return nil
}

package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"vitaline-clinic/internal/usecase"
)

func (a *CommandApp) runPatient(ctx context.Context, args []string) error {
	return a.dispatch(ctx, "patient", args, a.printPatientUsage, subcommands{
		"register": a.patientRegister,
		"get":      a.patientGet,
		"search":   a.patientSearch,
		"list":     a.patientList,
		"delete":   a.patientDelete,
	})
}

func (a *CommandApp) patientRegister(ctx context.Context, args []string) error {
	fs := a.newFlagSet("patient register", "patient register --full-name \"Иван Иванов\" --phone +79001234567 --email ivanov@example.com --birth-date 2000-05-10")
	fullName := fs.String("full-name", "", "ФИО пациента")
	phone := fs.String("phone", "", "Телефон пациента")
	email := fs.String("email", "", "Email пациента")
	birthDate := fs.String("birth-date", "", "Дата рождения в формате ГГГГ-ММ-ДД")
	if err := fs.Parse(args); err != nil {
		return err
	}

	date, err := time.Parse(dateLayout, strings.TrimSpace(*birthDate))
	if err != nil {
		return fmt.Errorf("разбор birth-date: %w", err)
	}

	const command = "patient register"
	a.logger.Info("cli action started", "command", command, "email", strings.TrimSpace(*email))
	res, err := a.patients.Register(ctx, usecase.CreatePatientDTO{
		FullName:  strings.TrimSpace(*fullName),
		Phone:     strings.TrimSpace(*phone),
		Email:     strings.TrimSpace(*email),
		BirthDate: date,
	})
	if err != nil {
		return a.actionFailed(command, err)
	}

	a.logger.Info("cli action completed", "command", command, "patient_id", res.ID)
	a.printPatient(res)
	return nil
}

func (a *CommandApp) patientGet(ctx context.Context, args []string) error {
	patientID, err := a.parseIDFlag("patient get", "id", "UUID пациента", args)
	if err != nil {
		return err
	}

	a.logger.Info("cli action started", "command", "patient get", "patient_id", patientID)
	res, err := a.patients.GetByID(ctx, patientID)
	if err != nil {
		return a.actionFailed("patient get", err, "patient_id", patientID)
	}

	a.printPatient(res)
	return nil
}

func (a *CommandApp) patientSearch(ctx context.Context, args []string) error {
	fs := a.newFlagSet("patient search", "patient search --email ivanov@example.com")
	email := fs.String("email", "", "Email пациента")
	if err := fs.Parse(args); err != nil {
		return err
	}

	address := strings.TrimSpace(*email)
	a.logger.Info("cli action started", "command", "patient search", "email", address)
	res, err := a.patients.FindByEmail(ctx, address)
	if err != nil {
		return a.actionFailed("patient search", err, "email", address)
	}

	a.printPatient(res)
	return nil
}

func (a *CommandApp) patientList(ctx context.Context, _ []string) error {
	a.logger.Info("cli action started", "command", "patient list")
	list, err := a.patients.GetAll(ctx)
	if err != nil {
		return a.actionFailed("patient list", err)
	}
	printList(a, list, "Список пациентов пуст", "Пациент", a.printPatient)
	return nil
}

func (a *CommandApp) patientDelete(ctx context.Context, args []string) error {
	patientID, err := a.parseIDFlag("patient delete", "id", "UUID пациента", args)
	if err != nil {
		return err
	}

	a.logger.Info("cli action started", "command", "patient delete", "patient_id", patientID)
	if err := a.patients.Delete(ctx, patientID); err != nil {
		return a.actionFailed("patient delete", err, "patient_id", patientID)
	}

	a.logger.Info("cli action completed", "command", "patient delete", "patient_id", patientID)
	fmt.Fprintln(a.out, "Пациент удалён")
	return nil
}

package integration_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
	storage "vitaline-clinic/internal/storage/mongodb"
	"vitaline-clinic/internal/usecase"
)

func TestVisitCompletionAndPatientHistory(t *testing.T) {
	store, ctx := database(t)
	patients := storage.NewPatientRepository(store.Database())
	apps := storage.NewAppointmentRepository(store.Database())
	visits := storage.NewVisitRepository(store.Database())
	appointments := usecase.NewAppointmentService(apps)
	service := usecase.NewVisitService(visits, apps)
	p := patient()
	must(t, patients.Save(ctx, p))
	app, err := appointments.Create(ctx, usecase.CreateAppointmentDTO{PatientID: p.ID, DoctorID: uuid.New(), Date: tomorrow()})
	must(t, err)

	visit, err := service.Create(ctx, usecase.CreateVisitDTO{AppointmentID: app.ID, Complaints: "Головная боль"})
	must(t, err)
	persisted, err := apps.FindByID(ctx, app.ID)
	must(t, err)
	if persisted.Status != domain.StatusCompleted {
		t.Fatal("visit did not complete the appointment")
	}

	diagnosis := usecase.UpdateVisitDTO{Diagnosis: "Тестовый диагноз", Recommendations: "Контрольный визит"}
	_, err = service.Complete(ctx, visit.ID, diagnosis)
	must(t, err)
	history, err := service.GetByPatient(ctx, p.ID)
	must(t, err)
	if len(history) != 1 || history[0].ID != visit.ID || history[0].Status != domain.VisitStatusCompleted || history[0].Diagnosis != diagnosis.Diagnosis || history[0].Recommendations != diagnosis.Recommendations {
		t.Fatalf("incorrect patient history: %+v", history)
	}

	_, err = service.Complete(ctx, visit.ID, usecase.UpdateVisitDTO{Diagnosis: "Другой диагноз", Recommendations: "Другие рекомендации"})
	if !errors.Is(err, domain.ErrInvalidStatus) {
		t.Fatalf("repeated completion: %v", err)
	}
	saved, err := visits.FindByID(ctx, visit.ID)
	must(t, err)
	if saved.Diagnosis != diagnosis.Diagnosis {
		t.Fatal("rejected completion overwrote the diagnosis")
	}
}

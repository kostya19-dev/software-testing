package usecase_test

import (
	"context"
	"errors"
	"testing"
	"vitaline-clinic/internal/domain"
	"vitaline-clinic/internal/usecase"
	"vitaline-clinic/lab1/testutil"
)

func TestVisitServiceLondon(t *testing.T) {
	t.Run("Create/success", func(t *testing.T) {
		x := newServiceFixture()
		s := usecase.NewVisitService(x.visit, x.appointment)
		got, err := s.Create(context.Background(), usecase.CreateVisitDTO{AppointmentID: testutil.ID, Complaints: "Боль"})
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != domain.VisitStatusInProgress || x.savedVisit == nil || x.savedAppointment == nil || x.savedAppointment.Status != domain.StatusCompleted {
			t.Fatalf("unexpected response: %+v", got)
		}
	})
	t.Run("Create/appointment.Update", func(t *testing.T) {
		x := newServiceFixture()
		x.fail = "appointment.Update"
		s := usecase.NewVisitService(x.visit, x.appointment)
		_, err := s.Create(context.Background(), usecase.CreateVisitDTO{AppointmentID: testutil.ID, Complaints: "Боль"})
		if !errors.Is(err, storageFailure) {
			t.Fatalf("expected original error: %v", err)
		}
	})
	t.Run("Complete/success", func(t *testing.T) {
		x := newServiceFixture()
		s := usecase.NewVisitService(x.visit, x.appointment)
		got, err := s.Complete(context.Background(), testutil.ID, usecase.UpdateVisitDTO{Diagnosis: "ARVI", Recommendations: "Rest"})
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != domain.VisitStatusCompleted || got.Diagnosis != "ARVI" || x.savedVisit == nil || x.savedVisit.Recommendations != "Rest" {
			t.Fatalf("unexpected response: %+v", got)
		}
	})
	t.Run("Complete/visit.Update", func(t *testing.T) {
		x := newServiceFixture()
		x.fail = "visit.Update"
		s := usecase.NewVisitService(x.visit, x.appointment)
		_, err := s.Complete(context.Background(), testutil.ID, usecase.UpdateVisitDTO{Diagnosis: "ARVI", Recommendations: "Rest"})
		if !errors.Is(err, storageFailure) {
			t.Fatalf("expected original error: %v", err)
		}
	})
	t.Run("GetByAppointment/success", func(t *testing.T) {
		x := newServiceFixture()
		s := usecase.NewVisitService(x.visit, x.appointment)
		got, err := s.GetByAppointment(context.Background(), testutil.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.AppointmentID != testutil.ID {
			t.Fatalf("unexpected response: %+v", got)
		}
	})
	t.Run("GetByAppointment/visit.FindByAppointment", func(t *testing.T) {
		x := newServiceFixture()
		x.fail = "visit.FindByAppointment"
		s := usecase.NewVisitService(x.visit, x.appointment)
		_, err := s.GetByAppointment(context.Background(), testutil.ID)
		if !errors.Is(err, storageFailure) {
			t.Fatalf("expected original error: %v", err)
		}
	})
	t.Run("GetByPatient/success", func(t *testing.T) {
		x := newServiceFixture()
		s := usecase.NewVisitService(x.visit, x.appointment)
		got, err := s.GetByPatient(context.Background(), testutil.PatientID)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].AppointmentID != testutil.ID {
			t.Fatalf("unexpected response: %+v", got)
		}
	})
	t.Run("GetByPatient/visit.FindByAppointment", func(t *testing.T) {
		x := newServiceFixture()
		x.fail = "visit.FindByAppointment"
		s := usecase.NewVisitService(x.visit, x.appointment)
		_, err := s.GetByPatient(context.Background(), testutil.PatientID)
		if !errors.Is(err, storageFailure) {
			t.Fatalf("expected original error: %v", err)
		}
	})
}

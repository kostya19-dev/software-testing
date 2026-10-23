package usecase_test

import (
	"context"
	"errors"
	"testing"
	"vitaline-clinic/internal/domain"
	"vitaline-clinic/internal/usecase"
	"vitaline-clinic/lab1/testutil"
)

func TestAppointmentServiceLondon(t *testing.T) {
	t.Run("Create/success", func(t *testing.T) {
		x := newServiceFixture()
		s := usecase.NewAppointmentService(x.appointment)
		got, err := s.Create(context.Background(), appointmentDTO())
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != domain.StatusCreated || x.savedAppointment == nil || x.savedAppointment.PatientID != testutil.PatientID || x.savedAppointment.DoctorID != testutil.DoctorID || x.savedAppointment.Status != domain.StatusCreated || !x.savedAppointment.Date.Equal(testutil.Date) {
			t.Fatalf("unexpected response: %+v", got)
		}
	})
	t.Run("Create/appointment.Save", func(t *testing.T) {
		x := newServiceFixture()
		x.fail = "appointment.Save"
		s := usecase.NewAppointmentService(x.appointment)
		_, err := s.Create(context.Background(), appointmentDTO())
		if !errors.Is(err, storageFailure) {
			t.Fatalf("expected original error: %v", err)
		}
	})
	t.Run("Cancel/success", func(t *testing.T) {
		x := newServiceFixture()
		s := usecase.NewAppointmentService(x.appointment)
		err := s.Cancel(context.Background(), testutil.ID)
		if err != nil {
			t.Fatal(err)
		}
		if x.savedAppointment == nil || x.savedAppointment.Status != domain.StatusCancelled {
			t.Fatal("status not persisted")
		}
	})
	t.Run("Cancel/appointment.Update", func(t *testing.T) {
		x := newServiceFixture()
		x.fail = "appointment.Update"
		s := usecase.NewAppointmentService(x.appointment)
		err := s.Cancel(context.Background(), testutil.ID)
		if !errors.Is(err, storageFailure) {
			t.Fatalf("expected original error: %v", err)
		}
	})
	t.Run("GetByID/success", func(t *testing.T) {
		x := newServiceFixture()
		s := usecase.NewAppointmentService(x.appointment)
		got, err := s.GetByID(context.Background(), testutil.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.ID != testutil.ID {
			t.Fatalf("unexpected response: %+v", got)
		}
	})
	t.Run("GetByID/appointment.FindByID", func(t *testing.T) {
		x := newServiceFixture()
		x.fail = "appointment.FindByID"
		s := usecase.NewAppointmentService(x.appointment)
		_, err := s.GetByID(context.Background(), testutil.ID)
		if !errors.Is(err, storageFailure) {
			t.Fatalf("expected original error: %v", err)
		}
	})
	t.Run("GetAll/success", func(t *testing.T) {
		x := newServiceFixture()
		s := usecase.NewAppointmentService(x.appointment)
		got, err := s.GetAll(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].ID != testutil.ID {
			t.Fatalf("unexpected response: %+v", got)
		}
	})
	t.Run("GetAll/appointment.FindAll", func(t *testing.T) {
		x := newServiceFixture()
		x.fail = "appointment.FindAll"
		s := usecase.NewAppointmentService(x.appointment)
		_, err := s.GetAll(context.Background())
		if !errors.Is(err, storageFailure) {
			t.Fatalf("expected original error: %v", err)
		}
	})
	t.Run("GetByPatient/success", func(t *testing.T) {
		x := newServiceFixture()
		s := usecase.NewAppointmentService(x.appointment)
		got, err := s.GetByPatient(context.Background(), testutil.PatientID)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].PatientID != testutil.PatientID {
			t.Fatalf("unexpected response: %+v", got)
		}
	})
	t.Run("GetByPatient/appointment.FindByPatient", func(t *testing.T) {
		x := newServiceFixture()
		x.fail = "appointment.FindByPatient"
		s := usecase.NewAppointmentService(x.appointment)
		_, err := s.GetByPatient(context.Background(), testutil.PatientID)
		if !errors.Is(err, storageFailure) {
			t.Fatalf("expected original error: %v", err)
		}
	})
}

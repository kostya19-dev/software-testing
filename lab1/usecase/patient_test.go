package usecase_test

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"testing"
	"vitaline-clinic/internal/domain"
	"vitaline-clinic/internal/storage/memory"
	"vitaline-clinic/internal/usecase"
	"vitaline-clinic/lab1/testutil"
)

func TestPatientServiceLondon(t *testing.T) {
	t.Run("Register/success", func(t *testing.T) {
		x := newServiceFixture()
		s := usecase.NewPatientService(x.patient)
		got, err := s.Register(context.Background(), patientDTO())
		if err != nil {
			t.Fatal(err)
		}
		if got.Email != testutil.Patient().Email || x.savedPatient == nil || got.ID == uuid.Nil || got.ID != x.savedPatient.ID || x.savedPatient.FullName != testutil.Patient().FullName || x.savedPatient.Email != testutil.Patient().Email {
			t.Fatalf("unexpected response: %+v", got)
		}
	})
	t.Run("Register/patient.Save", func(t *testing.T) {
		x := newServiceFixture()
		x.fail = "patient.Save"
		s := usecase.NewPatientService(x.patient)
		_, err := s.Register(context.Background(), patientDTO())
		if !errors.Is(err, storageFailure) {
			t.Fatalf("expected original error: %v", err)
		}
	})
	t.Run("GetByID/success", func(t *testing.T) {
		x := newServiceFixture()
		s := usecase.NewPatientService(x.patient)
		got, err := s.GetByID(context.Background(), testutil.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.ID != testutil.ID || got.Email != testutil.Patient().Email {
			t.Fatalf("unexpected response: %+v", got)
		}
	})
	t.Run("GetByID/patient.FindByID", func(t *testing.T) {
		x := newServiceFixture()
		x.fail = "patient.FindByID"
		s := usecase.NewPatientService(x.patient)
		_, err := s.GetByID(context.Background(), testutil.ID)
		if !errors.Is(err, storageFailure) {
			t.Fatalf("expected original error: %v", err)
		}
	})
	t.Run("GetAll/success", func(t *testing.T) {
		x := newServiceFixture()
		s := usecase.NewPatientService(x.patient)
		got, err := s.GetAll(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].ID != testutil.ID {
			t.Fatalf("unexpected response: %+v", got)
		}
	})
	t.Run("GetAll/patient.FindAll", func(t *testing.T) {
		x := newServiceFixture()
		x.fail = "patient.FindAll"
		s := usecase.NewPatientService(x.patient)
		_, err := s.GetAll(context.Background())
		if !errors.Is(err, storageFailure) {
			t.Fatalf("expected original error: %v", err)
		}
	})
	t.Run("FindByEmail/success", func(t *testing.T) {
		x := newServiceFixture()
		s := usecase.NewPatientService(x.patient)
		got, err := s.FindByEmail(context.Background(), testutil.Patient().Email)
		if err != nil {
			t.Fatal(err)
		}
		if got.ID != testutil.ID {
			t.Fatalf("unexpected response: %+v", got)
		}
	})
	t.Run("FindByEmail/patient.FindByEmail", func(t *testing.T) {
		x := newServiceFixture()
		x.fail = "patient.FindByEmail"
		s := usecase.NewPatientService(x.patient)
		_, err := s.FindByEmail(context.Background(), testutil.Patient().Email)
		if !errors.Is(err, storageFailure) {
			t.Fatalf("expected original error: %v", err)
		}
	})
	t.Run("Delete/success", func(t *testing.T) {
		x := newServiceFixture()
		s := usecase.NewPatientService(x.patient)
		err := s.Delete(context.Background(), testutil.ID)
		if err != nil {
			t.Fatal(err)
		}

		if x.deletedPatientID != testutil.ID {
			t.Fatal("requested record was not deleted")
		}
	})
	t.Run("Delete/patient.Delete", func(t *testing.T) {
		x := newServiceFixture()
		x.fail = "patient.Delete"
		s := usecase.NewPatientService(x.patient)
		err := s.Delete(context.Background(), testutil.ID)
		if !errors.Is(err, storageFailure) {
			t.Fatalf("expected original error: %v", err)
		}
	})
}

func TestPatientServiceClassical(t *testing.T) {
	for _, valid := range []bool{true, false} {
		t.Run(map[bool]string{true: "valid", false: "invalid"}[valid], func(t *testing.T) {
			repo := memory.NewPatientRepository()
			service := usecase.NewPatientService(repo)
			dto := patientDTO()
			if !valid {
				dto.Email = "bad"
			}
			got, err := service.Register(context.Background(), dto)
			if valid {
				if err != nil {
					t.Fatal(err)
				}
				stored, e := repo.FindByID(context.Background(), got.ID)
				if e != nil || stored.Email != dto.Email {
					t.Fatalf("%+v %v", stored, e)
				}
			} else {
				var validation *domain.ValidationError
				if !errors.As(err, &validation) || validation.Field != "Email" {
					t.Fatalf("%v", err)
				}
			}
		})
	}
}

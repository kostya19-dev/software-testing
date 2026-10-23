package memory_test

import (
	"context"
	"errors"
	"testing"

	"vitaline-clinic/internal/domain"
	"vitaline-clinic/internal/storage/memory"
	"vitaline-clinic/lab1/testutil"
)

func TestPatientRepositoryContract(t *testing.T) {
	t.Run("SaveValid", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewPatientRepository()
		p := testutil.Patient()
		err := r.Save(ctx, p)
		if err != nil {
			t.Fatalf("contract failed: %v", err)
		}
		stored, readErr := r.FindByID(ctx, p.ID)
		if readErr != nil || stored == nil || *stored != *p {
			t.Fatalf("stored=%+v want=%+v error=%v", stored, p, readErr)
		}
	})
	t.Run("SaveInvalid", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewPatientRepository()
		p := testutil.Patient()
		p.Email = "bad"
		err := r.Save(ctx, p)
		if !errors.Is(err, domain.ErrValidation) {
			t.Fatalf("contract failed: %v", err)
		}
		remaining, readErr := r.FindByID(ctx, p.ID)
		if !errors.Is(readErr, domain.ErrNotFound) || remaining != nil {
			t.Fatalf("unexpected record: %+v, error: %v", remaining, readErr)
		}
	})
	t.Run("UpdatePresent", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewPatientRepository()
		p := testutil.Patient()
		mustSave(t, r.Save(ctx, p))
		p.Email = "updated@example.com"
		err := r.Update(ctx, p)
		if err != nil {
			t.Fatalf("contract failed: %v", err)
		}
		stored, readErr := r.FindByID(ctx, p.ID)
		if readErr != nil || stored == nil || *stored != *p {
			t.Fatalf("stored=%+v want=%+v error=%v", stored, p, readErr)
		}
	})
	t.Run("UpdateMissing", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewPatientRepository()
		p := testutil.Patient()
		err := r.Update(ctx, p)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("contract failed: %v", err)
		}
	})
	t.Run("DeletePresent", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewPatientRepository()
		p := testutil.Patient()
		mustSave(t, r.Save(ctx, p))
		err := r.Delete(ctx, p.ID)
		if err != nil {
			t.Fatalf("contract failed: %v", err)
		}
		remaining, readErr := r.FindByID(ctx, p.ID)
		if !errors.Is(readErr, domain.ErrNotFound) || remaining != nil {
			t.Fatalf("unexpected record: %+v, error: %v", remaining, readErr)
		}
	})
	t.Run("DeleteMissing", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewPatientRepository()
		p := testutil.Patient()
		err := r.Delete(ctx, p.ID)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("contract failed: %v", err)
		}
	})
	t.Run("FindByIDPresent", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewPatientRepository()
		p := testutil.Patient()
		mustSave(t, r.Save(ctx, p))
		got, err := r.FindByID(ctx, p.ID)
		if err != nil || got == nil || got.ID != p.ID {
			t.Fatalf("contract failed: %v", err)
		}
	})
	t.Run("FindByIDMissing", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewPatientRepository()
		p := testutil.Patient()
		got, err := r.FindByID(ctx, p.ID)
		if !errors.Is(err, domain.ErrNotFound) || got != nil {
			t.Fatalf("contract failed: %v", err)
		}
	})
	t.Run("FindByEmailPresent", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewPatientRepository()
		p := testutil.Patient()
		mustSave(t, r.Save(ctx, p))
		got, err := r.FindByEmail(ctx, p.Email)
		if err != nil || got == nil || got.ID != p.ID {
			t.Fatalf("contract failed: %v", err)
		}
	})
	t.Run("FindByEmailMissing", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewPatientRepository()
		p := testutil.Patient()
		got, err := r.FindByEmail(ctx, p.Email)
		if !errors.Is(err, domain.ErrNotFound) || got != nil {
			t.Fatalf("contract failed: %v", err)
		}
	})
	t.Run("FindAllPresent", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewPatientRepository()
		p := testutil.Patient()
		mustSave(t, r.Save(ctx, p))
		got, err := r.FindAll(ctx)
		if err != nil || len(got) != 1 || got[0].ID != p.ID {
			t.Fatalf("contract failed: %v", err)
		}
	})
	t.Run("FindAllMissing", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewPatientRepository()
		got, err := r.FindAll(ctx)
		if err != nil || len(got) != 0 {
			t.Fatalf("contract failed: %v", err)
		}
	})
}

func mustSave(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestAppointmentRepositoryContract(t *testing.T) {
	t.Run("SaveValid", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewAppointmentRepository()
		p := testutil.Appointment()
		err := r.Save(ctx, p)
		if err != nil {
			t.Fatalf("contract failed: %v", err)
		}
		stored, readErr := r.FindByID(ctx, p.ID)
		if readErr != nil || stored == nil || *stored != *p {
			t.Fatalf("stored=%+v want=%+v error=%v", stored, p, readErr)
		}
	})
	t.Run("UpdatePresent", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewAppointmentRepository()
		p := testutil.Appointment()
		mustSave(t, r.Save(ctx, p))
		p.Status = domain.StatusCancelled
		err := r.Update(ctx, p)
		if err != nil {
			t.Fatalf("contract failed: %v", err)
		}
		stored, readErr := r.FindByID(ctx, p.ID)
		if readErr != nil || stored == nil || *stored != *p {
			t.Fatalf("stored=%+v want=%+v error=%v", stored, p, readErr)
		}
	})
	t.Run("DeletePresent", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewAppointmentRepository()
		p := testutil.Appointment()
		mustSave(t, r.Save(ctx, p))
		err := r.Delete(ctx, p.ID)
		if err != nil {
			t.Fatalf("contract failed: %v", err)
		}
		remaining, readErr := r.FindByID(ctx, p.ID)
		if !errors.Is(readErr, domain.ErrNotFound) || remaining != nil {
			t.Fatalf("unexpected record: %+v, error: %v", remaining, readErr)
		}
	})
	t.Run("FindByIDPresent", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewAppointmentRepository()
		p := testutil.Appointment()
		mustSave(t, r.Save(ctx, p))
		got, err := r.FindByID(ctx, p.ID)
		if err != nil || got == nil || got.ID != p.ID {
			t.Fatalf("contract failed: %v", err)
		}
	})
	t.Run("FindAllPresent", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewAppointmentRepository()
		p := testutil.Appointment()
		mustSave(t, r.Save(ctx, p))
		got, err := r.FindAll(ctx)
		if err != nil || len(got) != 1 || got[0].ID != p.ID {
			t.Fatalf("contract failed: %v", err)
		}
	})
	t.Run("FindByPatientPresent", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewAppointmentRepository()
		p := testutil.Appointment()
		mustSave(t, r.Save(ctx, p))
		got, err := r.FindByPatient(ctx, p.PatientID)
		if err != nil || len(got) != 1 || got[0].ID != p.ID {
			t.Fatalf("contract failed: %v", err)
		}
	})
	t.Run("FindActiveByDoctorAndDatePresent", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewAppointmentRepository()
		p := testutil.Appointment()
		mustSave(t, r.Save(ctx, p))
		got, err := r.FindActiveByDoctorAndDate(ctx, p.DoctorID, p.Date)
		if err != nil || got == nil || got.ID != p.ID {
			t.Fatalf("contract failed: %v", err)
		}
	})
	t.Run("CheckSlotBusyCreated", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewAppointmentRepository()
		p := testutil.Appointment()
		p.Status = domain.StatusCreated
		mustSave(t, r.Save(ctx, p))
		got, err := r.CheckSlotBusy(ctx, p.DoctorID, p.Date)
		if err != nil || got != true {
			t.Fatalf("contract failed: %v", err)
		}
	})
	t.Run("CheckSlotBusyCancelled", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewAppointmentRepository()
		p := testutil.Appointment()
		p.Status = domain.StatusCancelled
		mustSave(t, r.Save(ctx, p))
		got, err := r.CheckSlotBusy(ctx, p.DoctorID, p.Date)
		if err != nil || got != false {
			t.Fatalf("contract failed: %v", err)
		}
	})
}

func TestScheduleRepositoryContract(t *testing.T) {
	t.Run("SaveValid", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewScheduleRepository()
		p := testutil.Schedule()
		err := r.Save(ctx, p)
		if err != nil {
			t.Fatalf("contract failed: %v", err)
		}
		stored, readErr := r.FindByDoctorID(ctx, p.DoctorID)
		if readErr != nil || len(stored) != 1 || *stored[0] != *p {
			t.Fatalf("stored=%+v want=%+v error=%v", stored, p, readErr)
		}
	})
	t.Run("UpdatePresent", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewScheduleRepository()
		p := testutil.Schedule()
		mustSave(t, r.Save(ctx, p))
		p.Cabinet = "202"
		err := r.Update(ctx, p)
		if err != nil {
			t.Fatalf("contract failed: %v", err)
		}
		stored, readErr := r.FindByDoctorID(ctx, p.DoctorID)
		if readErr != nil || len(stored) != 1 || *stored[0] != *p {
			t.Fatalf("stored=%+v want=%+v error=%v", stored, p, readErr)
		}
	})
	t.Run("DeletePresent", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewScheduleRepository()
		p := testutil.Schedule()
		mustSave(t, r.Save(ctx, p))
		err := r.Delete(ctx, p.ID)
		if err != nil {
			t.Fatalf("contract failed: %v", err)
		}
		remaining, readErr := r.FindByDoctorID(ctx, p.DoctorID)
		if readErr != nil || len(remaining) != 0 {
			t.Fatalf("unexpected records: %+v, error: %v", remaining, readErr)
		}
	})
	t.Run("FindByDoctorIDPresent", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewScheduleRepository()
		p := testutil.Schedule()
		mustSave(t, r.Save(ctx, p))
		got, err := r.FindByDoctorID(ctx, p.DoctorID)
		if err != nil || len(got) != 1 || got[0].ID != p.ID {
			t.Fatalf("contract failed: %v", err)
		}
	})
	t.Run("FindFreeSlotsPresent", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewScheduleRepository()
		p := testutil.Schedule()
		mustSave(t, r.Save(ctx, p))
		got, err := r.FindFreeSlots(ctx, p.DoctorID, p.Date)
		if err != nil || len(got) != 1 || got[0].ID != p.ID {
			t.Fatalf("contract failed: %v", err)
		}
	})
}

func TestVisitRepositoryContract(t *testing.T) {
	t.Run("SaveValid", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewVisitRepository()
		p := testutil.Visit()
		err := r.Save(ctx, p)
		if err != nil {
			t.Fatalf("contract failed: %v", err)
		}
		stored, readErr := r.FindByID(ctx, p.ID)
		if readErr != nil || stored == nil || *stored != *p {
			t.Fatalf("stored=%+v want=%+v error=%v", stored, p, readErr)
		}
	})
	t.Run("UpdatePresent", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewVisitRepository()
		p := testutil.Visit()
		mustSave(t, r.Save(ctx, p))
		p.Complaints = "Updated complaints"
		err := r.Update(ctx, p)
		if err != nil {
			t.Fatalf("contract failed: %v", err)
		}
		stored, readErr := r.FindByID(ctx, p.ID)
		if readErr != nil || stored == nil || *stored != *p {
			t.Fatalf("stored=%+v want=%+v error=%v", stored, p, readErr)
		}
	})
	t.Run("DeletePresent", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewVisitRepository()
		p := testutil.Visit()
		mustSave(t, r.Save(ctx, p))
		err := r.Delete(ctx, p.ID)
		if err != nil {
			t.Fatalf("contract failed: %v", err)
		}
		remaining, readErr := r.FindByID(ctx, p.ID)
		if !errors.Is(readErr, domain.ErrNotFound) || remaining != nil {
			t.Fatalf("unexpected record: %+v, error: %v", remaining, readErr)
		}
	})
	t.Run("FindByIDPresent", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewVisitRepository()
		p := testutil.Visit()
		mustSave(t, r.Save(ctx, p))
		got, err := r.FindByID(ctx, p.ID)
		if err != nil || got == nil || got.ID != p.ID {
			t.Fatalf("contract failed: %v", err)
		}
	})
	t.Run("FindByAppointmentPresent", func(t *testing.T) {
		ctx := context.Background()
		r := memory.NewVisitRepository()
		p := testutil.Visit()
		mustSave(t, r.Save(ctx, p))
		got, err := r.FindByAppointment(ctx, p.AppointmentID)
		if err != nil || got == nil || got.ID != p.ID {
			t.Fatalf("contract failed: %v", err)
		}
	})
}

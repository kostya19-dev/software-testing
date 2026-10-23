package integration_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
	storage "vitaline-clinic/internal/storage/mongodb"
	"vitaline-clinic/internal/usecase"
)

func TestAppointmentBookingAndCancellation(t *testing.T) {
	store, ctx := database(t)
	patients := storage.NewPatientRepository(store.Database())
	repo := storage.NewAppointmentRepository(store.Database())
	service := usecase.NewAppointmentService(repo)
	p := patient()
	must(t, patients.Save(ctx, p))
	input := usecase.CreateAppointmentDTO{PatientID: p.ID, DoctorID: uuid.New(), Date: tomorrow()}

	booking, err := service.Create(ctx, input)
	must(t, err)
	saved, err := repo.FindByID(ctx, booking.ID)
	must(t, err)
	if saved.Status != domain.StatusCreated || saved.PatientID != p.ID {
		t.Fatal("booking was not persisted")
	}

	_, err = service.Create(ctx, input)
	if !errors.Is(err, domain.ErrSlotBusy) {
		t.Fatalf("occupied time: %v", err)
	}
	bookings, err := repo.FindByPatient(ctx, p.ID)
	must(t, err)
	if len(bookings) != 1 {
		t.Fatal("rejected booking changed the database")
	}

	must(t, service.Cancel(ctx, booking.ID))
	saved, err = repo.FindByID(ctx, booking.ID)
	must(t, err)
	if saved.Status != domain.StatusCancelled {
		t.Fatal("cancellation was not persisted")
	}
	replacement, err := service.Create(ctx, input)
	must(t, err)
	if replacement.ID == booking.ID {
		t.Fatal("new booking reused the cancelled booking")
	}
}

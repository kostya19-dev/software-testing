package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
)

// repos bundles the repositories used by the integration tests.
type repos struct {
	patients     *PatientRepository
	appointments *AppointmentRepository
	visits       *VisitRepository
	schedule     *ScheduleRepository
	emergency    *EmergencyRepository
}

func newRepos(db *sql.DB) repos {
	return repos{
		patients:     NewPatientRepository(db),
		appointments: NewAppointmentRepository(db),
		visits:       NewVisitRepository(db),
		schedule:     NewScheduleRepository(db),
		emergency:    NewEmergencyRepository(db),
	}
}

func check(t *testing.T, err error, action string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", action, err)
	}
}

func expect(t *testing.T, ok bool, format string, args ...any) {
	t.Helper()
	if !ok {
		t.Fatalf(format, args...)
	}
}

func expectNotFound(t *testing.T, err error, after string) {
	t.Helper()
	expect(t, errors.Is(err, domain.ErrNotFound), "expected ErrNotFound after %s delete, got: %v", after, err)
}

func utc(year int, month time.Month, day, hour, minute int) time.Time {
	return time.Date(year, month, day, hour, minute, 0, 0, time.UTC)
}

func TestIntegrationPostgres_SaveAndRetrieve(t *testing.T) {
	r := newRepos(openIntegrationDB(t))
	ctx := context.Background()
	doctorID := uuid.New()
	appointmentTime := utc(2026, 4, 12, 10, 30)

	patientID := saveAndFindPatient(ctx, t, r)
	appointmentID := saveAndFindAppointment(ctx, t, r, patientID, doctorID, appointmentTime)
	saveEmergencyAndOffer(ctx, t, r, appointmentID, patientID, doctorID, appointmentTime)
	checkPatientAppointments(ctx, t, r, patientID, doctorID, appointmentTime)
	saveAndFindVisit(ctx, t, r, appointmentID)
	saveAndFindFreeSlot(ctx, t, r, doctorID)
}

func saveAndFindPatient(ctx context.Context, t *testing.T, r repos) uuid.UUID {
	t.Helper()
	patient := &domain.Patient{
		ID:        uuid.New(),
		FullName:  "Ivan Ivanov",
		Phone:     "+79001234567",
		Email:     "ivanov.integration@example.com",
		BirthDate: utc(1990, 5, 15, 0, 0),
		CreatedAt: utc(2026, 4, 10, 9, 0),
	}
	check(t, r.patients.Save(ctx, patient), "save patient")

	found, err := r.patients.FindByID(ctx, patient.ID)
	check(t, err, "find patient by id")
	expect(t, found.FullName == patient.FullName, "unexpected patient full name: %s", found.FullName)

	byEmail, err := r.patients.FindByEmail(ctx, patient.Email)
	check(t, err, "find patient by email")
	expect(t, byEmail.ID == patient.ID, "unexpected patient id by email: %s", byEmail.ID)
	return patient.ID
}

func saveAndFindAppointment(ctx context.Context, t *testing.T, r repos, patientID, doctorID uuid.UUID, when time.Time) uuid.UUID {
	t.Helper()
	appointment := &domain.Appointment{
		ID:        uuid.New(),
		PatientID: patientID,
		DoctorID:  doctorID,
		Status:    domain.StatusCreated,
		Date:      when,
		CreatedAt: utc(2026, 4, 10, 9, 30),
	}
	check(t, r.appointments.Save(ctx, appointment), "save appointment")

	found, err := r.appointments.FindByID(ctx, appointment.ID)
	check(t, err, "find appointment by id")
	expect(t, found.PatientID == patientID, "unexpected patient id in appointment: %s", found.PatientID)
	return appointment.ID
}

func saveEmergencyAndOffer(ctx context.Context, t *testing.T, r repos, appointmentID, patientID, doctorID uuid.UUID, when time.Time) {
	t.Helper()
	request := &domain.EmergencyRequest{
		ID:            uuid.New(),
		PatientID:     patientID,
		DoctorID:      doctorID,
		RequestedDate: when,
		Reason:        "Urgent pain",
		Status:        domain.EmergencyStatusWaitingResponse,
		CreatedAt:     utc(2026, 4, 12, 8, 0),
	}
	check(t, r.emergency.SaveRequest(ctx, request), "save emergency request")

	offer := &domain.RescheduleOffer{
		ID:                 uuid.New(),
		EmergencyRequestID: request.ID,
		AppointmentID:      appointmentID,
		PatientID:          patientID,
		DoctorID:           doctorID,
		OldDate:            when,
		ProposedDate:       when.Add(30 * time.Minute),
		Status:             domain.OfferStatusPending,
		CreatedAt:          utc(2026, 4, 12, 8, 5),
	}
	check(t, r.emergency.SaveOffer(ctx, offer), "save reschedule offer")

	requests, err := r.emergency.FindAllRequests(ctx)
	check(t, err, "find emergency requests")
	expect(t, len(requests) == 1 && requests[0].Status == domain.EmergencyStatusWaitingResponse,
		"unexpected emergency requests: %+v", requests)

	offer.Status = domain.OfferStatusAccepted
	check(t, r.emergency.UpdateOffer(ctx, offer), "update reschedule offer")
	updated, err := r.emergency.FindOfferByID(ctx, offer.ID)
	check(t, err, "find reschedule offer")
	expect(t, updated.Status == domain.OfferStatusAccepted, "unexpected offer status: %s", updated.Status)
}

func checkPatientAppointments(ctx context.Context, t *testing.T, r repos, patientID, doctorID uuid.UUID, when time.Time) {
	t.Helper()
	list, err := r.appointments.FindByPatient(ctx, patientID)
	check(t, err, "find appointments by patient")
	expect(t, len(list) == 1, "expected 1 appointment, got %d", len(list))

	busy, err := r.appointments.CheckSlotBusy(ctx, doctorID, when)
	check(t, err, "check slot busy")
	expect(t, busy, "expected slot to be busy")
}

func saveAndFindVisit(ctx context.Context, t *testing.T, r repos, appointmentID uuid.UUID) {
	t.Helper()
	visit := &domain.Visit{
		ID:            uuid.New(),
		AppointmentID: appointmentID,
		Complaints:    "Headache",
		Status:        domain.VisitStatusInProgress,
		CreatedAt:     utc(2026, 4, 12, 11, 0),
	}
	check(t, r.visits.Save(ctx, visit), "save visit")

	found, err := r.visits.FindByAppointment(ctx, appointmentID)
	check(t, err, "find visit by appointment")
	expect(t, found.Complaints == visit.Complaints, "unexpected complaints: %s", found.Complaints)
}

func saveAndFindFreeSlot(ctx context.Context, t *testing.T, r repos, doctorID uuid.UUID) {
	t.Helper()
	slot := &domain.ScheduleSlot{
		ID:        uuid.New(),
		DoctorID:  doctorID,
		Date:      utc(2026, 4, 13, 0, 0),
		StartTime: utc(2026, 4, 13, 9, 0),
		EndTime:   utc(2026, 4, 13, 10, 0),
		Cabinet:   "101",
	}
	check(t, r.schedule.Save(ctx, slot), "save schedule slot")

	free, err := r.schedule.FindFreeSlots(ctx, doctorID, slot.Date)
	check(t, err, "find free slots")
	expect(t, len(free) == 1, "expected 1 free slot, got %d", len(free))
}

func TestIntegrationPostgres_Update(t *testing.T) {
	r := newRepos(openIntegrationDB(t))
	ctx := context.Background()
	doctorID := uuid.New()

	patientID := updatePatientPhone(ctx, t, r)
	appointmentID := cancelAppointment(ctx, t, r, patientID, doctorID)
	bookSlot(ctx, t, r, doctorID)
	completeVisit(ctx, t, r, appointmentID)
}

func updatePatientPhone(ctx context.Context, t *testing.T, r repos) uuid.UUID {
	t.Helper()
	patient := &domain.Patient{
		ID:        uuid.New(),
		FullName:  "Petr Petrov",
		Phone:     "+79005554433",
		Email:     "petrov.integration@example.com",
		BirthDate: utc(1985, 3, 20, 0, 0),
		CreatedAt: utc(2026, 4, 10, 8, 0),
	}
	check(t, r.patients.Save(ctx, patient), "save patient")

	patient.Phone = "+79001112233"
	check(t, r.patients.Update(ctx, patient), "update patient")

	updated, err := r.patients.FindByID(ctx, patient.ID)
	check(t, err, "find updated patient")
	expect(t, updated.Phone == patient.Phone, "unexpected patient phone: %s", updated.Phone)
	return patient.ID
}

func cancelAppointment(ctx context.Context, t *testing.T, r repos, patientID, doctorID uuid.UUID) uuid.UUID {
	t.Helper()
	appointment := &domain.Appointment{
		ID:        uuid.New(),
		PatientID: patientID,
		DoctorID:  doctorID,
		Status:    domain.StatusCreated,
		Date:      utc(2026, 4, 15, 12, 0),
		CreatedAt: utc(2026, 4, 10, 8, 30),
	}
	check(t, r.appointments.Save(ctx, appointment), "save appointment")

	appointment.Status = domain.StatusCancelled
	check(t, r.appointments.Update(ctx, appointment), "update appointment")

	updated, err := r.appointments.FindByID(ctx, appointment.ID)
	check(t, err, "find updated appointment")
	expect(t, updated.Status == domain.StatusCancelled, "unexpected appointment status: %s", updated.Status)
	return appointment.ID
}

func bookSlot(ctx context.Context, t *testing.T, r repos, doctorID uuid.UUID) {
	t.Helper()
	slot := &domain.ScheduleSlot{
		ID:        uuid.New(),
		DoctorID:  doctorID,
		Date:      utc(2026, 4, 16, 0, 0),
		StartTime: utc(2026, 4, 16, 14, 0),
		EndTime:   utc(2026, 4, 16, 15, 0),
		Cabinet:   "205",
	}
	check(t, r.schedule.Save(ctx, slot), "save schedule slot")

	slot.IsBooked = true
	slot.Cabinet = "206"
	check(t, r.schedule.Update(ctx, slot), "update schedule slot")

	slots, err := r.schedule.FindByDoctorID(ctx, doctorID)
	check(t, err, "find doctor slots")
	expect(t, len(slots) == 1, "expected 1 schedule slot, got %d", len(slots))
	expect(t, slots[0].IsBooked && slots[0].Cabinet == "206", "schedule slot was not updated correctly")
}

func completeVisit(ctx context.Context, t *testing.T, r repos, appointmentID uuid.UUID) {
	t.Helper()
	visit := &domain.Visit{
		ID:            uuid.New(),
		AppointmentID: appointmentID,
		Complaints:    "Cough",
		Status:        domain.VisitStatusInProgress,
		CreatedAt:     utc(2026, 4, 15, 13, 0),
	}
	check(t, r.visits.Save(ctx, visit), "save visit")

	visit.Diagnosis = "ARVI"
	visit.Recommendations = "Rest"
	visit.Status = domain.VisitStatusCompleted
	check(t, r.visits.Update(ctx, visit), "update visit")

	updated, err := r.visits.FindByID(ctx, visit.ID)
	check(t, err, "find updated visit")
	expect(t, updated.Status == domain.VisitStatusCompleted, "unexpected visit status: %s", updated.Status)
	expect(t, updated.Diagnosis == "ARVI" && updated.Recommendations == "Rest", "visit fields were not updated correctly")
}

// deleteFixture holds the ids of the records created for the delete test.
type deleteFixture struct {
	patientID, doctorID, appointmentID, visitID, slotID uuid.UUID
}

func seedForDelete(ctx context.Context, t *testing.T, r repos) deleteFixture {
	t.Helper()
	f := deleteFixture{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}
	patientID, doctorID, appointmentID := f.patientID, f.doctorID, f.appointmentID
	visitID, slotID := f.visitID, f.slotID

	check(t, r.patients.Save(ctx, &domain.Patient{
		ID: patientID, FullName: "Sidor Sidorov", Phone: "+79009998877",
		Email: "sidorov.integration@example.com", BirthDate: utc(2000, 1, 1, 0, 0), CreatedAt: utc(2026, 4, 10, 7, 0),
	}), "save patient")
	check(t, r.appointments.Save(ctx, &domain.Appointment{
		ID: appointmentID, PatientID: patientID, DoctorID: doctorID, Status: domain.StatusCreated,
		Date: utc(2026, 4, 17, 16, 0), CreatedAt: utc(2026, 4, 10, 7, 30),
	}), "save appointment")
	check(t, r.visits.Save(ctx, &domain.Visit{
		ID: visitID, AppointmentID: appointmentID, Complaints: "Pain",
		Status: domain.VisitStatusInProgress, CreatedAt: utc(2026, 4, 17, 16, 30),
	}), "save visit")
	check(t, r.schedule.Save(ctx, &domain.ScheduleSlot{
		ID: slotID, DoctorID: doctorID, Date: utc(2026, 4, 18, 0, 0),
		StartTime: utc(2026, 4, 18, 9, 0), EndTime: utc(2026, 4, 18, 10, 0), Cabinet: "305",
	}), "save schedule slot")
	return f
}

func TestIntegrationPostgres_Delete(t *testing.T) {
	r := newRepos(openIntegrationDB(t))
	ctx := context.Background()
	f := seedForDelete(ctx, t, r)

	check(t, r.visits.Delete(ctx, f.visitID), "delete visit")
	_, err := r.visits.FindByID(ctx, f.visitID)
	expectNotFound(t, err, "visit")

	check(t, r.appointments.Delete(ctx, f.appointmentID), "delete appointment")
	_, err = r.appointments.FindByID(ctx, f.appointmentID)
	expectNotFound(t, err, "appointment")

	check(t, r.schedule.Delete(ctx, f.slotID), "delete schedule slot")
	slots, err := r.schedule.FindByDoctorID(ctx, f.doctorID)
	check(t, err, "find doctor slots after delete")
	expect(t, len(slots) == 0, "expected 0 schedule slots after delete, got %d", len(slots))

	check(t, r.patients.Delete(ctx, f.patientID), "delete patient")
	_, err = r.patients.FindByID(ctx, f.patientID)
	expectNotFound(t, err, "patient")

	patients, err := r.patients.FindAll(ctx)
	check(t, err, "find all patients after delete")
	expect(t, len(patients) == 0, "expected 0 patients after delete, got %d", len(patients))
}

package postgres

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
	"vitaline-clinic/internal/usecase"
)

var (
	_ usecase.AppointmentRepository = (*AppointmentRepository)(nil)
	_ usecase.VisitRepository       = (*VisitRepository)(nil)
	_ usecase.PatientRepository     = (*PatientRepository)(nil)
	_ usecase.ScheduleRepository    = (*ScheduleRepository)(nil)
	_ usecase.EmergencyRepository   = (*EmergencyRepository)(nil)
)

func integrationDSN() string {
	if dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL")); dsn != "" {
		return dsn
	}
	return strings.TrimSpace(os.Getenv("DATABASE_URL"))
}

func openIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dsn := integrationDSN()
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL or DATABASE_URL to run PostgreSQL integration tests")
	}

	db, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open integration database: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	applySchema(t, db)
	resetTables(t, db)
	return db
}

func applySchema(t *testing.T, db *sql.DB) {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve current test file path")
	}

	schemaPath := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../migrations/schema.sql"))
	raw, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema file: %v", err)
	}

	statements := strings.Split(string(raw), ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("apply schema statement %q: %v", stmt, err)
		}
	}
}

func resetTables(t *testing.T, db *sql.DB) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.ExecContext(ctx, `TRUNCATE TABLE reschedule_offers, emergency_requests, visits, schedule_slots, appointments, patients`)
	if err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}

func TestIntegrationPostgres_SaveAndRetrieve(t *testing.T) {
	db := openIntegrationDB(t)
	ctx := context.Background()

	patientRepo := NewPatientRepository(db)
	appointmentRepo := NewAppointmentRepository(db)
	visitRepo := NewVisitRepository(db)
	scheduleRepo := NewScheduleRepository(db)
	emergencyRepo := NewEmergencyRepository(db)

	patientID := uuid.New()
	doctorID := uuid.New()
	appointmentID := uuid.New()
	visitID := uuid.New()
	slotID := uuid.New()
	emergencyID := uuid.New()
	offerID := uuid.New()

	patient := &domain.Patient{
		ID:        patientID,
		FullName:  "Ivan Ivanov",
		Phone:     "+79001234567",
		Email:     "ivanov.integration@example.com",
		BirthDate: time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC),
		CreatedAt: time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC),
	}
	if err := patientRepo.Save(ctx, patient); err != nil {
		t.Fatalf("save patient: %v", err)
	}

	foundPatient, err := patientRepo.FindByID(ctx, patientID)
	if err != nil {
		t.Fatalf("find patient by id: %v", err)
	}
	if foundPatient.FullName != patient.FullName {
		t.Fatalf("unexpected patient full name: %s", foundPatient.FullName)
	}

	foundByEmail, err := patientRepo.FindByEmail(ctx, patient.Email)
	if err != nil {
		t.Fatalf("find patient by email: %v", err)
	}
	if foundByEmail.ID != patientID {
		t.Fatalf("unexpected patient id by email: %s", foundByEmail.ID)
	}

	appointmentTime := time.Date(2026, 4, 12, 10, 30, 0, 0, time.UTC)
	appointment := &domain.Appointment{
		ID:        appointmentID,
		PatientID: patientID,
		DoctorID:  doctorID,
		Status:    domain.StatusCreated,
		Date:      appointmentTime,
		CreatedAt: time.Date(2026, 4, 10, 9, 30, 0, 0, time.UTC),
	}
	if err := appointmentRepo.Save(ctx, appointment); err != nil {
		t.Fatalf("save appointment: %v", err)
	}

	foundAppointment, err := appointmentRepo.FindByID(ctx, appointmentID)
	if err != nil {
		t.Fatalf("find appointment by id: %v", err)
	}
	if foundAppointment.PatientID != patientID {
		t.Fatalf("unexpected patient id in appointment: %s", foundAppointment.PatientID)
	}

	emergencyRequest := &domain.EmergencyRequest{
		ID:            emergencyID,
		PatientID:     patientID,
		DoctorID:      doctorID,
		RequestedDate: appointmentTime,
		Reason:        "Urgent pain",
		Status:        domain.EmergencyStatusWaitingResponse,
		CreatedAt:     time.Date(2026, 4, 12, 8, 0, 0, 0, time.UTC),
	}
	if err := emergencyRepo.SaveRequest(ctx, emergencyRequest); err != nil {
		t.Fatalf("save emergency request: %v", err)
	}

	rescheduleOffer := &domain.RescheduleOffer{
		ID:                 offerID,
		EmergencyRequestID: emergencyID,
		AppointmentID:      appointmentID,
		PatientID:          patientID,
		DoctorID:           doctorID,
		OldDate:            appointmentTime,
		ProposedDate:       appointmentTime.Add(30 * time.Minute),
		Status:             domain.OfferStatusPending,
		CreatedAt:          time.Date(2026, 4, 12, 8, 5, 0, 0, time.UTC),
	}
	if err := emergencyRepo.SaveOffer(ctx, rescheduleOffer); err != nil {
		t.Fatalf("save reschedule offer: %v", err)
	}

	emergencyRequests, err := emergencyRepo.FindAllRequests(ctx)
	if err != nil {
		t.Fatalf("find emergency requests: %v", err)
	}
	if len(emergencyRequests) != 1 || emergencyRequests[0].Status != domain.EmergencyStatusWaitingResponse {
		t.Fatalf("unexpected emergency requests: %+v", emergencyRequests)
	}

	rescheduleOffer.Status = domain.OfferStatusAccepted
	if err := emergencyRepo.UpdateOffer(ctx, rescheduleOffer); err != nil {
		t.Fatalf("update reschedule offer: %v", err)
	}
	updatedOffer, err := emergencyRepo.FindOfferByID(ctx, offerID)
	if err != nil {
		t.Fatalf("find reschedule offer: %v", err)
	}
	if updatedOffer.Status != domain.OfferStatusAccepted {
		t.Fatalf("unexpected offer status: %s", updatedOffer.Status)
	}

	patientAppointments, err := appointmentRepo.FindByPatient(ctx, patientID)
	if err != nil {
		t.Fatalf("find appointments by patient: %v", err)
	}
	if len(patientAppointments) != 1 {
		t.Fatalf("expected 1 appointment, got %d", len(patientAppointments))
	}

	busy, err := appointmentRepo.CheckSlotBusy(ctx, doctorID, appointmentTime)
	if err != nil {
		t.Fatalf("check slot busy: %v", err)
	}
	if !busy {
		t.Fatal("expected slot to be busy")
	}

	visit := &domain.Visit{
		ID:            visitID,
		AppointmentID: appointmentID,
		Complaints:    "Headache",
		Status:        domain.VisitStatusInProgress,
		CreatedAt:     time.Date(2026, 4, 12, 11, 0, 0, 0, time.UTC),
	}
	if err := visitRepo.Save(ctx, visit); err != nil {
		t.Fatalf("save visit: %v", err)
	}

	foundVisit, err := visitRepo.FindByAppointment(ctx, appointmentID)
	if err != nil {
		t.Fatalf("find visit by appointment: %v", err)
	}
	if foundVisit.Complaints != visit.Complaints {
		t.Fatalf("unexpected complaints: %s", foundVisit.Complaints)
	}

	slotDate := time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC)
	slot := &domain.ScheduleSlot{
		ID:        slotID,
		DoctorID:  doctorID,
		Date:      slotDate,
		StartTime: time.Date(2026, 4, 13, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 4, 13, 10, 0, 0, 0, time.UTC),
		Cabinet:   "101",
		IsBooked:  false,
	}
	if err := scheduleRepo.Save(ctx, slot); err != nil {
		t.Fatalf("save schedule slot: %v", err)
	}

	freeSlots, err := scheduleRepo.FindFreeSlots(ctx, doctorID, slotDate)
	if err != nil {
		t.Fatalf("find free slots: %v", err)
	}
	if len(freeSlots) != 1 {
		t.Fatalf("expected 1 free slot, got %d", len(freeSlots))
	}
}

func TestIntegrationPostgres_Update(t *testing.T) {
	db := openIntegrationDB(t)
	ctx := context.Background()

	patientRepo := NewPatientRepository(db)
	appointmentRepo := NewAppointmentRepository(db)
	visitRepo := NewVisitRepository(db)
	scheduleRepo := NewScheduleRepository(db)

	patientID := uuid.New()
	patient := &domain.Patient{
		ID:        patientID,
		FullName:  "Petr Petrov",
		Phone:     "+79005554433",
		Email:     "petrov.integration@example.com",
		BirthDate: time.Date(1985, 3, 20, 0, 0, 0, 0, time.UTC),
		CreatedAt: time.Date(2026, 4, 10, 8, 0, 0, 0, time.UTC),
	}
	if err := patientRepo.Save(ctx, patient); err != nil {
		t.Fatalf("save patient: %v", err)
	}

	patient.Phone = "+79001112233"
	if err := patientRepo.Update(ctx, patient); err != nil {
		t.Fatalf("update patient: %v", err)
	}

	updatedPatient, err := patientRepo.FindByID(ctx, patientID)
	if err != nil {
		t.Fatalf("find updated patient: %v", err)
	}
	if updatedPatient.Phone != patient.Phone {
		t.Fatalf("unexpected patient phone: %s", updatedPatient.Phone)
	}

	appointmentID := uuid.New()
	doctorID := uuid.New()
	appointment := &domain.Appointment{
		ID:        appointmentID,
		PatientID: patientID,
		DoctorID:  doctorID,
		Status:    domain.StatusCreated,
		Date:      time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC),
		CreatedAt: time.Date(2026, 4, 10, 8, 30, 0, 0, time.UTC),
	}
	if err := appointmentRepo.Save(ctx, appointment); err != nil {
		t.Fatalf("save appointment: %v", err)
	}

	appointment.Status = domain.StatusCancelled
	if err := appointmentRepo.Update(ctx, appointment); err != nil {
		t.Fatalf("update appointment: %v", err)
	}

	updatedAppointment, err := appointmentRepo.FindByID(ctx, appointmentID)
	if err != nil {
		t.Fatalf("find updated appointment: %v", err)
	}
	if updatedAppointment.Status != domain.StatusCancelled {
		t.Fatalf("unexpected appointment status: %s", updatedAppointment.Status)
	}

	slotID := uuid.New()
	slot := &domain.ScheduleSlot{
		ID:        slotID,
		DoctorID:  doctorID,
		Date:      time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC),
		StartTime: time.Date(2026, 4, 16, 14, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 4, 16, 15, 0, 0, 0, time.UTC),
		Cabinet:   "205",
		IsBooked:  false,
	}
	if err := scheduleRepo.Save(ctx, slot); err != nil {
		t.Fatalf("save schedule slot: %v", err)
	}

	slot.IsBooked = true
	slot.Cabinet = "206"
	if err := scheduleRepo.Update(ctx, slot); err != nil {
		t.Fatalf("update schedule slot: %v", err)
	}

	doctorSlots, err := scheduleRepo.FindByDoctorID(ctx, doctorID)
	if err != nil {
		t.Fatalf("find doctor slots: %v", err)
	}
	if len(doctorSlots) != 1 {
		t.Fatalf("expected 1 schedule slot, got %d", len(doctorSlots))
	}
	if !doctorSlots[0].IsBooked || doctorSlots[0].Cabinet != "206" {
		t.Fatalf("schedule slot was not updated correctly")
	}

	visitID := uuid.New()
	visit := &domain.Visit{
		ID:            visitID,
		AppointmentID: appointmentID,
		Complaints:    "Cough",
		Status:        domain.VisitStatusInProgress,
		CreatedAt:     time.Date(2026, 4, 15, 13, 0, 0, 0, time.UTC),
	}
	if err := visitRepo.Save(ctx, visit); err != nil {
		t.Fatalf("save visit: %v", err)
	}

	visit.Diagnosis = "ARVI"
	visit.Recommendations = "Rest"
	visit.Status = domain.VisitStatusCompleted
	if err := visitRepo.Update(ctx, visit); err != nil {
		t.Fatalf("update visit: %v", err)
	}

	updatedVisit, err := visitRepo.FindByID(ctx, visitID)
	if err != nil {
		t.Fatalf("find updated visit: %v", err)
	}
	if updatedVisit.Status != domain.VisitStatusCompleted {
		t.Fatalf("unexpected visit status: %s", updatedVisit.Status)
	}
	if updatedVisit.Diagnosis != "ARVI" || updatedVisit.Recommendations != "Rest" {
		t.Fatalf("visit fields were not updated correctly")
	}
}

func TestIntegrationPostgres_Delete(t *testing.T) {
	db := openIntegrationDB(t)
	ctx := context.Background()

	patientRepo := NewPatientRepository(db)
	appointmentRepo := NewAppointmentRepository(db)
	visitRepo := NewVisitRepository(db)
	scheduleRepo := NewScheduleRepository(db)

	patientID := uuid.New()
	doctorID := uuid.New()
	appointmentID := uuid.New()
	visitID := uuid.New()
	slotID := uuid.New()

	patient := &domain.Patient{
		ID:        patientID,
		FullName:  "Sidor Sidorov",
		Phone:     "+79009998877",
		Email:     "sidorov.integration@example.com",
		BirthDate: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		CreatedAt: time.Date(2026, 4, 10, 7, 0, 0, 0, time.UTC),
	}
	if err := patientRepo.Save(ctx, patient); err != nil {
		t.Fatalf("save patient: %v", err)
	}

	appointment := &domain.Appointment{
		ID:        appointmentID,
		PatientID: patientID,
		DoctorID:  doctorID,
		Status:    domain.StatusCreated,
		Date:      time.Date(2026, 4, 17, 16, 0, 0, 0, time.UTC),
		CreatedAt: time.Date(2026, 4, 10, 7, 30, 0, 0, time.UTC),
	}
	if err := appointmentRepo.Save(ctx, appointment); err != nil {
		t.Fatalf("save appointment: %v", err)
	}

	visit := &domain.Visit{
		ID:            visitID,
		AppointmentID: appointmentID,
		Complaints:    "Pain",
		Status:        domain.VisitStatusInProgress,
		CreatedAt:     time.Date(2026, 4, 17, 16, 30, 0, 0, time.UTC),
	}
	if err := visitRepo.Save(ctx, visit); err != nil {
		t.Fatalf("save visit: %v", err)
	}

	slot := &domain.ScheduleSlot{
		ID:        slotID,
		DoctorID:  doctorID,
		Date:      time.Date(2026, 4, 18, 0, 0, 0, 0, time.UTC),
		StartTime: time.Date(2026, 4, 18, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC),
		Cabinet:   "305",
		IsBooked:  false,
	}
	if err := scheduleRepo.Save(ctx, slot); err != nil {
		t.Fatalf("save schedule slot: %v", err)
	}

	if err := visitRepo.Delete(ctx, visitID); err != nil {
		t.Fatalf("delete visit: %v", err)
	}
	if _, err := visitRepo.FindByID(ctx, visitID); err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound after visit delete, got: %v", err)
	}

	if err := appointmentRepo.Delete(ctx, appointmentID); err != nil {
		t.Fatalf("delete appointment: %v", err)
	}
	if _, err := appointmentRepo.FindByID(ctx, appointmentID); err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound after appointment delete, got: %v", err)
	}

	if err := scheduleRepo.Delete(ctx, slotID); err != nil {
		t.Fatalf("delete schedule slot: %v", err)
	}
	doctorSlots, err := scheduleRepo.FindByDoctorID(ctx, doctorID)
	if err != nil {
		t.Fatalf("find doctor slots after delete: %v", err)
	}
	if len(doctorSlots) != 0 {
		t.Fatalf("expected 0 schedule slots after delete, got %d", len(doctorSlots))
	}

	if err := patientRepo.Delete(ctx, patientID); err != nil {
		t.Fatalf("delete patient: %v", err)
	}
	if _, err := patientRepo.FindByID(ctx, patientID); err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound after patient delete, got: %v", err)
	}

	allPatients, err := patientRepo.FindAll(ctx)
	if err != nil {
		t.Fatalf("find all patients after delete: %v", err)
	}
	if len(allPatients) != 0 {
		t.Fatalf("expected 0 patients after delete, got %d", len(allPatients))
	}
}

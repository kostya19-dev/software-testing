package usecase_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"vitaline-clinic/internal/domain"
	"vitaline-clinic/internal/storage/memory"
	"vitaline-clinic/internal/usecase"
	"vitaline-clinic/lab1/testutil"
)

func TestPatientValidationErrors(t *testing.T) {
	tests := []struct {
		name   string
		field  string
		change func(*usecase.CreatePatientDTO)
	}{
		{"empty_name", "FullName", func(dto *usecase.CreatePatientDTO) { dto.FullName = " " }},
		{"empty_phone", "Phone", func(dto *usecase.CreatePatientDTO) { dto.Phone = "" }},
		{"invalid_phone", "Phone", func(dto *usecase.CreatePatientDTO) { dto.Phone = "abc" }},
		{"empty_email", "Email", func(dto *usecase.CreatePatientDTO) { dto.Email = "" }},
		{"invalid_email", "Email", func(dto *usecase.CreatePatientDTO) { dto.Email = "abc" }},
		{"missing_birth_date", "BirthDate", func(dto *usecase.CreatePatientDTO) { dto.BirthDate = time.Time{} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repository := memory.NewPatientRepository()
			service := usecase.NewPatientService(repository)
			input := patientDTO()
			tt.change(&input)

			result, err := service.Register(ctx, input)

			require.ErrorIs(t, err, domain.ErrValidation)
			var validation *domain.ValidationError
			require.ErrorAs(t, err, &validation)
			require.Equal(t, tt.field, validation.Field)
			require.Equal(t, usecase.PatientResponseDTO{}, result)
			patients, err := repository.FindAll(ctx)
			require.NoError(t, err)
			require.Empty(t, patients)
		})
	}
}

func TestPatientRepositoryErrors(t *testing.T) {
	tests := []struct {
		name  string
		cause error
	}{
		{"not_found", domain.ErrNotFound},
		{"storage_failure", storageFailure},
		{"cancelled_context", context.Canceled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := fmt.Errorf("read patient: %w", tt.cause)
			repository := &patientMock{
				FindByIDFunc: func(context.Context, uuid.UUID) (*domain.Patient, error) {
					return nil, expected
				},
			}
			service := usecase.NewPatientService(repository)

			result, err := service.GetByID(context.Background(), testutil.ID)

			require.ErrorIs(t, err, tt.cause)
			require.Equal(t, usecase.PatientResponseDTO{}, result)
		})
	}
}

func TestAppointmentCreateErrors(t *testing.T) {
	tests := []struct {
		name         string
		date         time.Time
		busy         bool
		storageError error
		want         error
	}{
		{"past_date", time.Now().Add(-24 * time.Hour), false, nil, domain.ErrPastDate},
		{"insufficient_lead_time", time.Now().Add(time.Hour), false, nil, domain.ErrAppointmentLeadTime},
		{"busy_slot", testutil.Date, true, nil, domain.ErrSlotBusy},
		{"storage_failure", testutil.Date, false, storageFailure, storageFailure},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var saved *domain.Appointment
			repository := &appointmentMock{
				CheckSlotBusyFunc: func(context.Context, uuid.UUID, time.Time) (bool, error) {
					return tt.busy, tt.storageError
				},
				SaveFunc: func(_ context.Context, appointment *domain.Appointment) error {
					saved = appointment
					return nil
				},
			}
			service := usecase.NewAppointmentService(
				repository, usecase.WithAppointmentMinLeadTime(24*time.Hour),
			)
			input := appointmentDTO()
			input.Date = tt.date

			result, err := service.Create(context.Background(), input)

			require.ErrorIs(t, err, tt.want)
			require.Equal(t, usecase.AppointmentResponseDTO{}, result)
			require.Nil(t, saved)
		})
	}
}

func TestScheduleValidationErrors(t *testing.T) {
	tests := []struct {
		name   string
		field  string
		change func(*usecase.CreateScheduleSlotDTO)
	}{
		{"missing_doctor", "DoctorID", func(dto *usecase.CreateScheduleSlotDTO) { dto.DoctorID = uuid.Nil }},
		{"missing_date", "Date", func(dto *usecase.CreateScheduleSlotDTO) { dto.Date = time.Time{} }},
		{"missing_start", "StartTime", func(dto *usecase.CreateScheduleSlotDTO) { dto.StartTime = time.Time{} }},
		{"missing_end", "EndTime", func(dto *usecase.CreateScheduleSlotDTO) { dto.EndTime = time.Time{} }},
		{"equal_times", "EndTime", func(dto *usecase.CreateScheduleSlotDTO) { dto.EndTime = dto.StartTime }},
		{"end_before_start", "EndTime", func(dto *usecase.CreateScheduleSlotDTO) { dto.EndTime = dto.StartTime.Add(-time.Minute) }},
		{"empty_cabinet", "Cabinet", func(dto *usecase.CreateScheduleSlotDTO) { dto.Cabinet = " " }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup := newServiceFixture()
			service := usecase.NewScheduleService(setup.schedule)
			input := scheduleDTO()
			tt.change(&input)

			result, err := service.CreateSlot(context.Background(), input)

			require.ErrorIs(t, err, domain.ErrValidation)
			var validation *domain.ValidationError
			require.ErrorAs(t, err, &validation)
			require.Equal(t, tt.field, validation.Field)
			require.Equal(t, usecase.ScheduleSlotResponseDTO{}, result)
			require.Nil(t, setup.savedSlot)
		})
	}
}

func TestAppointmentValidationErrors(t *testing.T) {
	tests := []struct {
		name   string
		field  string
		change func(*usecase.CreateAppointmentDTO)
	}{
		{"missing_patient", "PatientID", func(dto *usecase.CreateAppointmentDTO) { dto.PatientID = uuid.Nil }},
		{"missing_doctor", "DoctorID", func(dto *usecase.CreateAppointmentDTO) { dto.DoctorID = uuid.Nil }},
		{"missing_date", "Date", func(dto *usecase.CreateAppointmentDTO) { dto.Date = time.Time{} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup := newServiceFixture()
			service := usecase.NewAppointmentService(setup.appointment)
			input := appointmentDTO()
			tt.change(&input)

			result, err := service.Create(context.Background(), input)

			require.ErrorIs(t, err, domain.ErrValidation)
			var validation *domain.ValidationError
			require.ErrorAs(t, err, &validation)
			require.Equal(t, tt.field, validation.Field)
			require.Equal(t, usecase.AppointmentResponseDTO{}, result)
			require.Nil(t, setup.savedAppointment)
		})
	}
}

func TestVisitCreateErrors(t *testing.T) {
	tests := []struct {
		name          string
		appointmentID uuid.UUID
		complaints    string
		status        domain.AppointmentStatus
		want          error
		field         string
	}{
		{"missing_appointment", uuid.Nil, "Боль", domain.StatusCreated, domain.ErrValidation, "AppointmentID"},
		{"empty_complaints", testutil.ID, " ", domain.StatusCreated, domain.ErrValidation, "Complaints"},
		{"cancelled_appointment", testutil.ID, "Боль", domain.StatusCancelled, domain.ErrInvalidStatus, ""},
		{"completed_appointment", testutil.ID, "Боль", domain.StatusCompleted, domain.ErrInvalidStatus, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup := newServiceFixture()
			appointment := testutil.Appointment()
			appointment.Status = tt.status
			before := *appointment
			setup.appointment.FindByIDFunc = func(context.Context, uuid.UUID) (*domain.Appointment, error) {
				return appointment, nil
			}
			service := usecase.NewVisitService(setup.visit, setup.appointment)
			input := usecase.CreateVisitDTO{AppointmentID: tt.appointmentID, Complaints: tt.complaints}

			result, err := service.Create(context.Background(), input)

			require.ErrorIs(t, err, tt.want)
			if tt.field != "" {
				var validation *domain.ValidationError
				require.ErrorAs(t, err, &validation)
				require.Equal(t, tt.field, validation.Field)
			}
			require.Equal(t, usecase.VisitResponseDTO{}, result)
			require.Equal(t, before, *appointment)
			require.Nil(t, setup.savedVisit)
			require.Nil(t, setup.savedAppointment)
		})
	}
}

func TestVisitCompleteErrors(t *testing.T) {
	tests := []struct {
		name            string
		diagnosis       string
		recommendations string
		status          domain.VisitStatus
		want            error
		field           string
	}{
		{"empty_diagnosis", " ", "Отдых", domain.VisitStatusInProgress, domain.ErrValidation, "Diagnosis"},
		{"empty_recommendations", "Диагноз", " ", domain.VisitStatusInProgress, domain.ErrValidation, "Recommendations"},
		{"already_completed", "Диагноз", "Отдых", domain.VisitStatusCompleted, domain.ErrInvalidStatus, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup := newServiceFixture()
			visit := testutil.Visit()
			visit.Status = tt.status
			before := *visit
			setup.visit.FindByIDFunc = func(context.Context, uuid.UUID) (*domain.Visit, error) {
				return visit, nil
			}
			service := usecase.NewVisitService(setup.visit, setup.appointment)
			input := usecase.UpdateVisitDTO{Diagnosis: tt.diagnosis, Recommendations: tt.recommendations}

			result, err := service.Complete(context.Background(), testutil.ID, input)

			require.ErrorIs(t, err, tt.want)
			if tt.field != "" {
				var validation *domain.ValidationError
				require.ErrorAs(t, err, &validation)
				require.Equal(t, tt.field, validation.Field)
			}
			require.Equal(t, usecase.VisitResponseDTO{}, result)
			require.Equal(t, before, *visit)
			require.Nil(t, setup.savedVisit)
		})
	}
}

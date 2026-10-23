package usecase_test

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"time"
	"vitaline-clinic/internal/domain"
	"vitaline-clinic/internal/usecase"
	"vitaline-clinic/lab1/testutil"
)

var storageFailure = errors.New("injected storage failure")

type serviceFixture struct {
	deletedPatientID uuid.UUID
	deletedSlotID    uuid.UUID
	fail             string
	patient          *patientMock
	appointment      *appointmentMock
	schedule         *scheduleMock
	visit            *visitMock
	emergency        *emergencyMock
	savedPatient     *domain.Patient
	savedAppointment *domain.Appointment
	savedRequest     *domain.EmergencyRequest
	savedOffer       *domain.RescheduleOffer
	savedVisit       *domain.Visit
	savedSlot        *domain.ScheduleSlot
}

func (fixture *serviceFixture) failureFor(name string) error {
	if name == fixture.fail {
		return storageFailure
	}
	return nil
}

func newServiceFixture() *serviceFixture {
	x := &serviceFixture{}
	x.patient = &patientMock{
		SaveFunc: func(ctx context.Context, patient *domain.Patient) error {
			if err := x.failureFor("patient.Save"); err != nil {
				return err
			}
			copy := *patient
			x.savedPatient = &copy
			return nil
		},
		FindByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Patient, error) {
			if err := x.failureFor("patient.FindByID"); err != nil {
				return nil, err
			}
			return testutil.Patient(), nil
		},
		FindAllFunc: func(ctx context.Context) ([]*domain.Patient, error) {
			if err := x.failureFor("patient.FindAll"); err != nil {
				return nil, err
			}
			return []*domain.Patient{testutil.Patient()}, nil
		},
		FindByEmailFunc: func(ctx context.Context, email string) (*domain.Patient, error) {
			if err := x.failureFor("patient.FindByEmail"); err != nil {
				return nil, err
			}
			return testutil.Patient(), nil
		},
		UpdateFunc: func(ctx context.Context, patient *domain.Patient) error {
			if err := x.failureFor("patient.Update"); err != nil {
				return err
			}
			copy := *patient
			x.savedPatient = &copy
			return nil
		},
		DeleteFunc: func(ctx context.Context, id uuid.UUID) error {
			if err := x.failureFor("patient.Delete"); err != nil {
				return err
			}
			x.deletedPatientID = id
			return nil
		},
	}
	x.appointment = &appointmentMock{
		SaveFunc: func(ctx context.Context, app *domain.Appointment) error {
			if err := x.failureFor("appointment.Save"); err != nil {
				return err
			}
			copy := *app
			x.savedAppointment = &copy
			return nil
		},
		FindByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Appointment, error) {
			if err := x.failureFor("appointment.FindByID"); err != nil {
				return nil, err
			}
			return testutil.Appointment(), nil
		},
		FindAllFunc: func(ctx context.Context) ([]*domain.Appointment, error) {
			if err := x.failureFor("appointment.FindAll"); err != nil {
				return nil, err
			}
			return []*domain.Appointment{testutil.Appointment()}, nil
		},
		FindByPatientFunc: func(ctx context.Context, patientID uuid.UUID) ([]*domain.Appointment, error) {
			if err := x.failureFor("appointment.FindByPatient"); err != nil {
				return nil, err
			}
			return []*domain.Appointment{testutil.Appointment()}, nil
		},
		FindActiveByDoctorAndDateFunc: func(ctx context.Context, doctorID uuid.UUID, date time.Time) (*domain.Appointment, error) {
			if err := x.failureFor("appointment.FindActiveByDoctorAndDate"); err != nil {
				return nil, err
			}
			return testutil.Appointment(), nil
		},
		CheckSlotBusyFunc: func(ctx context.Context, doctorID uuid.UUID, date time.Time) (bool, error) {
			if err := x.failureFor("appointment.CheckSlotBusy"); err != nil {
				return false, err
			}
			return false, nil
		},
		UpdateFunc: func(ctx context.Context, app *domain.Appointment) error {
			if err := x.failureFor("appointment.Update"); err != nil {
				return err
			}
			copy := *app
			x.savedAppointment = &copy
			return nil
		},
		DeleteFunc: func(ctx context.Context, id uuid.UUID) error {
			if err := x.failureFor("appointment.Delete"); err != nil {
				return err
			}
			return nil
		},
	}
	x.schedule = &scheduleMock{
		SaveFunc: func(ctx context.Context, slot *domain.ScheduleSlot) error {
			if err := x.failureFor("schedule.Save"); err != nil {
				return err
			}
			copy := *slot
			x.savedSlot = &copy
			return nil
		},
		FindByDoctorIDFunc: func(ctx context.Context, doctorID uuid.UUID) ([]*domain.ScheduleSlot, error) {
			if err := x.failureFor("schedule.FindByDoctorID"); err != nil {
				return nil, err
			}
			return []*domain.ScheduleSlot{testutil.Schedule()}, nil
		},
		FindFreeSlotsFunc: func(ctx context.Context, doctorID uuid.UUID, date time.Time) ([]*domain.ScheduleSlot, error) {
			if err := x.failureFor("schedule.FindFreeSlots"); err != nil {
				return nil, err
			}
			return []*domain.ScheduleSlot{testutil.Schedule()}, nil
		},
		UpdateFunc: func(ctx context.Context, slot *domain.ScheduleSlot) error {
			if err := x.failureFor("schedule.Update"); err != nil {
				return err
			}
			copy := *slot
			x.savedSlot = &copy
			return nil
		},
		DeleteFunc: func(ctx context.Context, id uuid.UUID) error {
			if err := x.failureFor("schedule.Delete"); err != nil {
				return err
			}
			x.deletedSlotID = id
			return nil
		},
	}
	x.visit = &visitMock{
		SaveFunc: func(ctx context.Context, visit *domain.Visit) error {
			if err := x.failureFor("visit.Save"); err != nil {
				return err
			}
			copy := *visit
			x.savedVisit = &copy
			return nil
		},
		FindByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Visit, error) {
			if err := x.failureFor("visit.FindByID"); err != nil {
				return nil, err
			}
			return testutil.Visit(), nil
		},
		FindByAppointmentFunc: func(ctx context.Context, appointmentID uuid.UUID) (*domain.Visit, error) {
			if err := x.failureFor("visit.FindByAppointment"); err != nil {
				return nil, err
			}
			return testutil.Visit(), nil
		},
		UpdateFunc: func(ctx context.Context, visit *domain.Visit) error {
			if err := x.failureFor("visit.Update"); err != nil {
				return err
			}
			copy := *visit
			x.savedVisit = &copy
			return nil
		},
		DeleteFunc: func(ctx context.Context, id uuid.UUID) error {
			if err := x.failureFor("visit.Delete"); err != nil {
				return err
			}
			return nil
		},
	}
	x.emergency = &emergencyMock{
		SaveRequestFunc: func(ctx context.Context, request *domain.EmergencyRequest) error {
			if err := x.failureFor("emergency.SaveRequest"); err != nil {
				return err
			}
			copy := *request
			x.savedRequest = &copy
			return nil
		},
		FindRequestByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.EmergencyRequest, error) {
			if err := x.failureFor("emergency.FindRequestByID"); err != nil {
				return nil, err
			}
			return testutil.EmergencyRequest(), nil
		},
		FindAllRequestsFunc: func(ctx context.Context) ([]*domain.EmergencyRequest, error) {
			if err := x.failureFor("emergency.FindAllRequests"); err != nil {
				return nil, err
			}
			return []*domain.EmergencyRequest{testutil.EmergencyRequest()}, nil
		},
		UpdateRequestFunc: func(ctx context.Context, request *domain.EmergencyRequest) error {
			if err := x.failureFor("emergency.UpdateRequest"); err != nil {
				return err
			}
			copy := *request
			x.savedRequest = &copy
			return nil
		},
		SaveOfferFunc: func(ctx context.Context, offer *domain.RescheduleOffer) error {
			if err := x.failureFor("emergency.SaveOffer"); err != nil {
				return err
			}
			copy := *offer
			x.savedOffer = &copy
			return nil
		},
		FindOfferByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.RescheduleOffer, error) {
			if err := x.failureFor("emergency.FindOfferByID"); err != nil {
				return nil, err
			}
			return testutil.RescheduleOffer(), nil
		},
		FindAllOffersFunc: func(ctx context.Context) ([]*domain.RescheduleOffer, error) {
			if err := x.failureFor("emergency.FindAllOffers"); err != nil {
				return nil, err
			}
			return []*domain.RescheduleOffer{testutil.RescheduleOffer()}, nil
		},
		UpdateOfferFunc: func(ctx context.Context, offer *domain.RescheduleOffer) error {
			if err := x.failureFor("emergency.UpdateOffer"); err != nil {
				return err
			}
			copy := *offer
			x.savedOffer = &copy
			return nil
		},
	}
	return x
}

func patientDTO() usecase.CreatePatientDTO {
	p := testutil.Patient()
	return usecase.CreatePatientDTO{FullName: p.FullName, Phone: p.Phone, Email: p.Email, BirthDate: p.BirthDate}
}

func appointmentDTO() usecase.CreateAppointmentDTO {
	p := testutil.Appointment()
	return usecase.CreateAppointmentDTO{PatientID: p.PatientID, DoctorID: p.DoctorID, Date: p.Date}
}

func scheduleDTO() usecase.CreateScheduleSlotDTO {
	p := testutil.Schedule()
	return usecase.CreateScheduleSlotDTO{DoctorID: p.DoctorID, Date: p.Date, StartTime: p.StartTime, EndTime: p.EndTime, Cabinet: p.Cabinet}
}

func emergencyDTO() usecase.CreateEmergencyRequestDTO {
	p := testutil.EmergencyRequest()
	return usecase.CreateEmergencyRequestDTO{PatientID: p.PatientID, DoctorID: p.DoctorID, RequestedDate: p.RequestedDate, Reason: p.Reason}
}

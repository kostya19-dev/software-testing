package usecase_test

import (
	"context"
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
	"vitaline-clinic/internal/usecase"
)

type patientMock struct {
	SaveFunc        func(ctx context.Context, patient *domain.Patient) error
	FindByIDFunc    func(ctx context.Context, id uuid.UUID) (*domain.Patient, error)
	FindAllFunc     func(ctx context.Context) ([]*domain.Patient, error)
	FindByEmailFunc func(ctx context.Context, email string) (*domain.Patient, error)
	UpdateFunc      func(ctx context.Context, patient *domain.Patient) error
	DeleteFunc      func(ctx context.Context, id uuid.UUID) error
}

func (m *patientMock) Save(ctx context.Context, patient *domain.Patient) error {
	if m.SaveFunc == nil {
		panic("unexpected PatientRepository.Save")
	}
	return m.SaveFunc(ctx, patient)
}

func (m *patientMock) FindByID(ctx context.Context, id uuid.UUID) (*domain.Patient, error) {
	if m.FindByIDFunc == nil {
		panic("unexpected PatientRepository.FindByID")
	}
	return m.FindByIDFunc(ctx, id)
}

func (m *patientMock) FindAll(ctx context.Context) ([]*domain.Patient, error) {
	if m.FindAllFunc == nil {
		panic("unexpected PatientRepository.FindAll")
	}
	return m.FindAllFunc(ctx)
}

func (m *patientMock) FindByEmail(ctx context.Context, email string) (*domain.Patient, error) {
	if m.FindByEmailFunc == nil {
		panic("unexpected PatientRepository.FindByEmail")
	}
	return m.FindByEmailFunc(ctx, email)
}

func (m *patientMock) Update(ctx context.Context, patient *domain.Patient) error {
	if m.UpdateFunc == nil {
		panic("unexpected PatientRepository.Update")
	}
	return m.UpdateFunc(ctx, patient)
}

func (m *patientMock) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFunc == nil {
		panic("unexpected PatientRepository.Delete")
	}
	return m.DeleteFunc(ctx, id)
}

var _ usecase.PatientRepository = (*patientMock)(nil)

type appointmentMock struct {
	SaveFunc                      func(ctx context.Context, app *domain.Appointment) error
	FindByIDFunc                  func(ctx context.Context, id uuid.UUID) (*domain.Appointment, error)
	FindAllFunc                   func(ctx context.Context) ([]*domain.Appointment, error)
	FindByPatientFunc             func(ctx context.Context, patientID uuid.UUID) ([]*domain.Appointment, error)
	FindActiveByDoctorAndDateFunc func(ctx context.Context, doctorID uuid.UUID, date time.Time) (*domain.Appointment, error)
	CheckSlotBusyFunc             func(ctx context.Context, doctorID uuid.UUID, date time.Time) (bool, error)
	UpdateFunc                    func(ctx context.Context, app *domain.Appointment) error
	DeleteFunc                    func(ctx context.Context, id uuid.UUID) error
}

func (m *appointmentMock) Save(ctx context.Context, app *domain.Appointment) error {
	if m.SaveFunc == nil {
		panic("unexpected AppointmentRepository.Save")
	}
	return m.SaveFunc(ctx, app)
}

func (m *appointmentMock) FindByID(ctx context.Context, id uuid.UUID) (*domain.Appointment, error) {
	if m.FindByIDFunc == nil {
		panic("unexpected AppointmentRepository.FindByID")
	}
	return m.FindByIDFunc(ctx, id)
}

func (m *appointmentMock) FindAll(ctx context.Context) ([]*domain.Appointment, error) {
	if m.FindAllFunc == nil {
		panic("unexpected AppointmentRepository.FindAll")
	}
	return m.FindAllFunc(ctx)
}

func (m *appointmentMock) FindByPatient(ctx context.Context, patientID uuid.UUID) ([]*domain.Appointment, error) {
	if m.FindByPatientFunc == nil {
		panic("unexpected AppointmentRepository.FindByPatient")
	}
	return m.FindByPatientFunc(ctx, patientID)
}

func (m *appointmentMock) FindActiveByDoctorAndDate(ctx context.Context, doctorID uuid.UUID, date time.Time) (*domain.Appointment, error) {
	if m.FindActiveByDoctorAndDateFunc == nil {
		panic("unexpected AppointmentRepository.FindActiveByDoctorAndDate")
	}
	return m.FindActiveByDoctorAndDateFunc(ctx, doctorID, date)
}

func (m *appointmentMock) CheckSlotBusy(ctx context.Context, doctorID uuid.UUID, date time.Time) (bool, error) {
	if m.CheckSlotBusyFunc == nil {
		panic("unexpected AppointmentRepository.CheckSlotBusy")
	}
	return m.CheckSlotBusyFunc(ctx, doctorID, date)
}

func (m *appointmentMock) Update(ctx context.Context, app *domain.Appointment) error {
	if m.UpdateFunc == nil {
		panic("unexpected AppointmentRepository.Update")
	}
	return m.UpdateFunc(ctx, app)
}

func (m *appointmentMock) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFunc == nil {
		panic("unexpected AppointmentRepository.Delete")
	}
	return m.DeleteFunc(ctx, id)
}

var _ usecase.AppointmentRepository = (*appointmentMock)(nil)

type scheduleMock struct {
	SaveFunc           func(ctx context.Context, slot *domain.ScheduleSlot) error
	FindByDoctorIDFunc func(ctx context.Context, doctorID uuid.UUID) ([]*domain.ScheduleSlot, error)
	FindFreeSlotsFunc  func(ctx context.Context, doctorID uuid.UUID, date time.Time) ([]*domain.ScheduleSlot, error)
	UpdateFunc         func(ctx context.Context, slot *domain.ScheduleSlot) error
	DeleteFunc         func(ctx context.Context, id uuid.UUID) error
}

func (m *scheduleMock) Save(ctx context.Context, slot *domain.ScheduleSlot) error {
	if m.SaveFunc == nil {
		panic("unexpected ScheduleRepository.Save")
	}
	return m.SaveFunc(ctx, slot)
}

func (m *scheduleMock) FindByDoctorID(ctx context.Context, doctorID uuid.UUID) ([]*domain.ScheduleSlot, error) {
	if m.FindByDoctorIDFunc == nil {
		panic("unexpected ScheduleRepository.FindByDoctorID")
	}
	return m.FindByDoctorIDFunc(ctx, doctorID)
}

func (m *scheduleMock) FindFreeSlots(ctx context.Context, doctorID uuid.UUID, date time.Time) ([]*domain.ScheduleSlot, error) {
	if m.FindFreeSlotsFunc == nil {
		panic("unexpected ScheduleRepository.FindFreeSlots")
	}
	return m.FindFreeSlotsFunc(ctx, doctorID, date)
}

func (m *scheduleMock) Update(ctx context.Context, slot *domain.ScheduleSlot) error {
	if m.UpdateFunc == nil {
		panic("unexpected ScheduleRepository.Update")
	}
	return m.UpdateFunc(ctx, slot)
}

func (m *scheduleMock) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFunc == nil {
		panic("unexpected ScheduleRepository.Delete")
	}
	return m.DeleteFunc(ctx, id)
}

var _ usecase.ScheduleRepository = (*scheduleMock)(nil)

type visitMock struct {
	SaveFunc              func(ctx context.Context, visit *domain.Visit) error
	FindByIDFunc          func(ctx context.Context, id uuid.UUID) (*domain.Visit, error)
	FindByAppointmentFunc func(ctx context.Context, appointmentID uuid.UUID) (*domain.Visit, error)
	UpdateFunc            func(ctx context.Context, visit *domain.Visit) error
	DeleteFunc            func(ctx context.Context, id uuid.UUID) error
}

func (m *visitMock) Save(ctx context.Context, visit *domain.Visit) error {
	if m.SaveFunc == nil {
		panic("unexpected VisitRepository.Save")
	}
	return m.SaveFunc(ctx, visit)
}

func (m *visitMock) FindByID(ctx context.Context, id uuid.UUID) (*domain.Visit, error) {
	if m.FindByIDFunc == nil {
		panic("unexpected VisitRepository.FindByID")
	}
	return m.FindByIDFunc(ctx, id)
}

func (m *visitMock) FindByAppointment(ctx context.Context, appointmentID uuid.UUID) (*domain.Visit, error) {
	if m.FindByAppointmentFunc == nil {
		panic("unexpected VisitRepository.FindByAppointment")
	}
	return m.FindByAppointmentFunc(ctx, appointmentID)
}

func (m *visitMock) Update(ctx context.Context, visit *domain.Visit) error {
	if m.UpdateFunc == nil {
		panic("unexpected VisitRepository.Update")
	}
	return m.UpdateFunc(ctx, visit)
}

func (m *visitMock) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFunc == nil {
		panic("unexpected VisitRepository.Delete")
	}
	return m.DeleteFunc(ctx, id)
}

var _ usecase.VisitRepository = (*visitMock)(nil)

type emergencyMock struct {
	SaveRequestFunc     func(ctx context.Context, request *domain.EmergencyRequest) error
	FindRequestByIDFunc func(ctx context.Context, id uuid.UUID) (*domain.EmergencyRequest, error)
	FindAllRequestsFunc func(ctx context.Context) ([]*domain.EmergencyRequest, error)
	UpdateRequestFunc   func(ctx context.Context, request *domain.EmergencyRequest) error
	SaveOfferFunc       func(ctx context.Context, offer *domain.RescheduleOffer) error
	FindOfferByIDFunc   func(ctx context.Context, id uuid.UUID) (*domain.RescheduleOffer, error)
	FindAllOffersFunc   func(ctx context.Context) ([]*domain.RescheduleOffer, error)
	UpdateOfferFunc     func(ctx context.Context, offer *domain.RescheduleOffer) error
}

func (m *emergencyMock) SaveRequest(ctx context.Context, request *domain.EmergencyRequest) error {
	if m.SaveRequestFunc == nil {
		panic("unexpected EmergencyRepository.SaveRequest")
	}
	return m.SaveRequestFunc(ctx, request)
}

func (m *emergencyMock) FindRequestByID(ctx context.Context, id uuid.UUID) (*domain.EmergencyRequest, error) {
	if m.FindRequestByIDFunc == nil {
		panic("unexpected EmergencyRepository.FindRequestByID")
	}
	return m.FindRequestByIDFunc(ctx, id)
}

func (m *emergencyMock) FindAllRequests(ctx context.Context) ([]*domain.EmergencyRequest, error) {
	if m.FindAllRequestsFunc == nil {
		panic("unexpected EmergencyRepository.FindAllRequests")
	}
	return m.FindAllRequestsFunc(ctx)
}

func (m *emergencyMock) UpdateRequest(ctx context.Context, request *domain.EmergencyRequest) error {
	if m.UpdateRequestFunc == nil {
		panic("unexpected EmergencyRepository.UpdateRequest")
	}
	return m.UpdateRequestFunc(ctx, request)
}

func (m *emergencyMock) SaveOffer(ctx context.Context, offer *domain.RescheduleOffer) error {
	if m.SaveOfferFunc == nil {
		panic("unexpected EmergencyRepository.SaveOffer")
	}
	return m.SaveOfferFunc(ctx, offer)
}

func (m *emergencyMock) FindOfferByID(ctx context.Context, id uuid.UUID) (*domain.RescheduleOffer, error) {
	if m.FindOfferByIDFunc == nil {
		panic("unexpected EmergencyRepository.FindOfferByID")
	}
	return m.FindOfferByIDFunc(ctx, id)
}

func (m *emergencyMock) FindAllOffers(ctx context.Context) ([]*domain.RescheduleOffer, error) {
	if m.FindAllOffersFunc == nil {
		panic("unexpected EmergencyRepository.FindAllOffers")
	}
	return m.FindAllOffersFunc(ctx)
}

func (m *emergencyMock) UpdateOffer(ctx context.Context, offer *domain.RescheduleOffer) error {
	if m.UpdateOfferFunc == nil {
		panic("unexpected EmergencyRepository.UpdateOffer")
	}
	return m.UpdateOfferFunc(ctx, offer)
}

var _ usecase.EmergencyRepository = (*emergencyMock)(nil)

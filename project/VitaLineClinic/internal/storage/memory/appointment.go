package memory

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
)

type AppointmentRepository struct {
	mu           sync.RWMutex
	appointments map[uuid.UUID]*domain.Appointment
}

func NewAppointmentRepository() *AppointmentRepository {
	return &AppointmentRepository{
		appointments: make(map[uuid.UUID]*domain.Appointment),
	}
}

func (r *AppointmentRepository) Save(_ context.Context, app *domain.Appointment) error {
	if err := app.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	stored := *app
	r.appointments[app.ID] = &stored
	return nil
}

func (r *AppointmentRepository) FindByID(_ context.Context, id uuid.UUID) (*domain.Appointment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	app, ok := r.appointments[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *app
	return &cp, nil
}

func (r *AppointmentRepository) FindAll(_ context.Context) ([]*domain.Appointment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*domain.Appointment, 0, len(r.appointments))
	for _, app := range r.appointments {
		cp := *app
		result = append(result, &cp)
	}
	return result, nil
}

func (r *AppointmentRepository) FindByPatient(_ context.Context, patientID uuid.UUID) ([]*domain.Appointment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*domain.Appointment
	for _, app := range r.appointments {
		if app.PatientID == patientID {
			cp := *app
			result = append(result, &cp)
		}
	}
	return result, nil
}

func (r *AppointmentRepository) FindActiveByDoctorAndDate(_ context.Context, doctorID uuid.UUID, date time.Time) (*domain.Appointment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, app := range r.appointments {
		if app.DoctorID == doctorID && app.Date.Equal(date) && app.Status != domain.StatusCancelled {
			cp := *app
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *AppointmentRepository) CheckSlotBusy(_ context.Context, doctorID uuid.UUID, date time.Time) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, app := range r.appointments {
		if app.DoctorID == doctorID && app.Date.Equal(date) && app.Status != domain.StatusCancelled {
			return true, nil
		}
	}
	return false, nil
}

func (r *AppointmentRepository) Update(_ context.Context, app *domain.Appointment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.appointments[app.ID]; !ok {
		return domain.ErrNotFound
	}
	stored := *app
	r.appointments[app.ID] = &stored
	return nil
}

func (r *AppointmentRepository) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.appointments[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.appointments, id)
	return nil
}

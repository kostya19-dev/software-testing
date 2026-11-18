package memory

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
)

type PatientRepository struct {
	mu       sync.RWMutex
	patients map[uuid.UUID]*domain.Patient
}

func NewPatientRepository() *PatientRepository {
	return &PatientRepository{
		patients: make(map[uuid.UUID]*domain.Patient),
	}
}

func (r *PatientRepository) Save(_ context.Context, patient *domain.Patient) error {
	if err := patient.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	stored := *patient
	r.patients[patient.ID] = &stored
	return nil
}

func (r *PatientRepository) FindByID(_ context.Context, id uuid.UUID) (*domain.Patient, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.patients[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func (r *PatientRepository) FindAll(_ context.Context) ([]*domain.Patient, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*domain.Patient, 0, len(r.patients))
	for _, p := range r.patients {
		cp := *p
		result = append(result, &cp)
	}
	return result, nil
}

func (r *PatientRepository) FindByEmail(_ context.Context, email string) (*domain.Patient, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.patients {
		if p.Email == email {
			cp := *p
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *PatientRepository) Update(_ context.Context, patient *domain.Patient) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.patients[patient.ID]; !ok {
		return domain.ErrNotFound
	}
	stored := *patient
	r.patients[patient.ID] = &stored
	return nil
}

func (r *PatientRepository) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.patients[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.patients, id)
	return nil
}

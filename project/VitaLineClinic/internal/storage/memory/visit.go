package memory

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
)

type VisitRepository struct {
	mu     sync.RWMutex
	visits map[uuid.UUID]*domain.Visit
}

func NewVisitRepository() *VisitRepository {
	return &VisitRepository{
		visits: make(map[uuid.UUID]*domain.Visit),
	}
}

func (r *VisitRepository) Save(_ context.Context, visit *domain.Visit) error {
	if err := visit.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	stored := *visit
	r.visits[visit.ID] = &stored
	return nil
}

func (r *VisitRepository) FindByID(_ context.Context, id uuid.UUID) (*domain.Visit, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	v, ok := r.visits[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *v
	return &cp, nil
}

func (r *VisitRepository) FindByAppointment(_ context.Context, appointmentID uuid.UUID) (*domain.Visit, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, v := range r.visits {
		if v.AppointmentID == appointmentID {
			cp := *v
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *VisitRepository) Update(_ context.Context, visit *domain.Visit) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.visits[visit.ID]; !ok {
		return domain.ErrNotFound
	}
	stored := *visit
	r.visits[visit.ID] = &stored
	return nil
}

func (r *VisitRepository) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.visits[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.visits, id)
	return nil
}

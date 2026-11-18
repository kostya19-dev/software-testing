package memory

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
)

type ScheduleRepository struct {
	mu    sync.RWMutex
	slots map[uuid.UUID]*domain.ScheduleSlot
}

func NewScheduleRepository() *ScheduleRepository {
	return &ScheduleRepository{
		slots: make(map[uuid.UUID]*domain.ScheduleSlot),
	}
}

func (r *ScheduleRepository) Save(_ context.Context, slot *domain.ScheduleSlot) error {
	if err := slot.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	stored := *slot
	r.slots[slot.ID] = &stored
	return nil
}

func (r *ScheduleRepository) FindByDoctorID(_ context.Context, doctorID uuid.UUID) ([]*domain.ScheduleSlot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*domain.ScheduleSlot
	for _, s := range r.slots {
		if s.DoctorID == doctorID {
			cp := *s
			result = append(result, &cp)
		}
	}
	return result, nil
}

func (r *ScheduleRepository) FindFreeSlots(_ context.Context, doctorID uuid.UUID, date time.Time) ([]*domain.ScheduleSlot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*domain.ScheduleSlot
	for _, s := range r.slots {
		if s.DoctorID == doctorID && s.Date.Equal(date) && !s.IsBooked {
			cp := *s
			result = append(result, &cp)
		}
	}
	return result, nil
}

func (r *ScheduleRepository) Update(_ context.Context, slot *domain.ScheduleSlot) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.slots[slot.ID]; !ok {
		return domain.ErrNotFound
	}
	stored := *slot
	r.slots[slot.ID] = &stored
	return nil
}

func (r *ScheduleRepository) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.slots[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.slots, id)
	return nil
}

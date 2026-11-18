package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
)

const (
	tableScheduleSlots = "schedule_slots"
)

var scheduleColumns = []string{
	"id", "doctor_id", "slot_date", "start_time", "end_time", "cabinet", "is_booked",
}

type ScheduleRepository struct {
	db *sql.DB
	sb sq.StatementBuilderType
}

func NewScheduleRepository(db *sql.DB) *ScheduleRepository {
	return &ScheduleRepository{
		db: db,
		sb: psql.RunWith(db),
	}
}

func (r *ScheduleRepository) Save(ctx context.Context, slot *domain.ScheduleSlot) error {
	ctx = ensureContext(ctx)
	if err := slot.Validate(); err != nil {
		return err
	}

	_, err := r.sb.
		Insert(tableScheduleSlots).
		Columns(scheduleColumns...).
		Values(slot.ID, slot.DoctorID, slot.Date, slot.StartTime, slot.EndTime, slot.Cabinet, slot.IsBooked).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("insert schedule slot: %w", err)
	}
	return nil
}

func (r *ScheduleRepository) FindByDoctorID(ctx context.Context, doctorID uuid.UUID) ([]*domain.ScheduleSlot, error) {
	ctx = ensureContext(ctx)
	rows, err := r.sb.
		Select(scheduleColumns...).
		From(tableScheduleSlots).
		Where(sq.Eq{"doctor_id": doctorID}).
		OrderBy("slot_date", "start_time", "id").
		QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("select schedule by doctor: %w", err)
	}
	defer rows.Close()

	result := make([]*domain.ScheduleSlot, 0)
	for rows.Next() {
		slot, err := scanScheduleSlot(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, slot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schedule by doctor: %w", err)
	}
	return result, nil
}

func (r *ScheduleRepository) FindFreeSlots(ctx context.Context, doctorID uuid.UUID, date time.Time) ([]*domain.ScheduleSlot, error) {
	ctx = ensureContext(ctx)
	rows, err := r.sb.
		Select(scheduleColumns...).
		From(tableScheduleSlots).
		Where(sq.Eq{"doctor_id": doctorID}).
		Where(sq.Eq{"slot_date": date}).
		Where(sq.Eq{"is_booked": false}).
		OrderBy("start_time", "id").
		QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("select free schedule slots: %w", err)
	}
	defer rows.Close()

	result := make([]*domain.ScheduleSlot, 0)
	for rows.Next() {
		slot, err := scanScheduleSlot(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, slot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate free schedule slots: %w", err)
	}
	return result, nil
}

func (r *ScheduleRepository) Update(ctx context.Context, slot *domain.ScheduleSlot) error {
	ctx = ensureContext(ctx)
	result, err := r.sb.
		Update(tableScheduleSlots).
		Set("doctor_id", slot.DoctorID).
		Set("slot_date", slot.Date).
		Set("start_time", slot.StartTime).
		Set("end_time", slot.EndTime).
		Set("cabinet", slot.Cabinet).
		Set("is_booked", slot.IsBooked).
		Where(sq.Eq{"id": slot.ID}).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("update schedule slot: %w", err)
	}
	affected, err := rowsAffected(result)
	if err != nil {
		return fmt.Errorf("update schedule slot rows: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ScheduleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx = ensureContext(ctx)
	result, err := r.sb.
		Delete(tableScheduleSlots).
		Where(sq.Eq{"id": id}).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("delete schedule slot: %w", err)
	}
	affected, err := rowsAffected(result)
	if err != nil {
		return fmt.Errorf("delete schedule slot rows: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

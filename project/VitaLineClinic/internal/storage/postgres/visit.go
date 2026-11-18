package postgres

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
)

const (
	tableVisits = "visits"
)

var visitColumns = []string{
	"id", "appointment_id", "complaints", "diagnosis", "recommendations", "status", "created_at",
}

type VisitRepository struct {
	db *sql.DB
	sb sq.StatementBuilderType
}

func NewVisitRepository(db *sql.DB) *VisitRepository {
	return &VisitRepository{
		db: db,
		sb: psql.RunWith(db),
	}
}

func (r *VisitRepository) Save(ctx context.Context, visit *domain.Visit) error {
	ctx = ensureContext(ctx)
	if err := visit.Validate(); err != nil {
		return err
	}

	_, err := r.sb.
		Insert(tableVisits).
		Columns(visitColumns...).
		Values(visit.ID, visit.AppointmentID, visit.Complaints, visit.Diagnosis, visit.Recommendations, string(visit.Status), createdAtOrNow(visit.CreatedAt)).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("insert visit: %w", err)
	}
	return nil
}

func (r *VisitRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Visit, error) {
	ctx = ensureContext(ctx)
	row := r.sb.
		Select(visitColumns...).
		From(tableVisits).
		Where(sq.Eq{"id": id}).
		QueryRowContext(ctx)
	return scanVisit(row)
}

func (r *VisitRepository) FindByAppointment(ctx context.Context, appointmentID uuid.UUID) (*domain.Visit, error) {
	ctx = ensureContext(ctx)
	row := r.sb.
		Select(visitColumns...).
		From(tableVisits).
		Where(sq.Eq{"appointment_id": appointmentID}).
		QueryRowContext(ctx)
	return scanVisit(row)
}

func (r *VisitRepository) Update(ctx context.Context, visit *domain.Visit) error {
	ctx = ensureContext(ctx)
	result, err := r.sb.
		Update(tableVisits).
		Set("appointment_id", visit.AppointmentID).
		Set("complaints", visit.Complaints).
		Set("diagnosis", visit.Diagnosis).
		Set("recommendations", visit.Recommendations).
		Set("status", string(visit.Status)).
		Set("created_at", createdAtOrNow(visit.CreatedAt)).
		Where(sq.Eq{"id": visit.ID}).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("update visit: %w", err)
	}
	affected, err := rowsAffected(result)
	if err != nil {
		return fmt.Errorf("update visit rows: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *VisitRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx = ensureContext(ctx)
	result, err := r.sb.
		Delete(tableVisits).
		Where(sq.Eq{"id": id}).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("delete visit: %w", err)
	}
	affected, err := rowsAffected(result)
	if err != nil {
		return fmt.Errorf("delete visit rows: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

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
	tablePatients = "patients"
)

var patientColumns = []string{
	"id", "full_name", "phone", "email", "birth_date", "created_at",
}

type PatientRepository struct {
	db *sql.DB
	sb sq.StatementBuilderType
}

func NewPatientRepository(db *sql.DB) *PatientRepository {
	return &PatientRepository{
		db: db,
		sb: psql.RunWith(db),
	}
}

func (r *PatientRepository) Save(ctx context.Context, patient *domain.Patient) error {
	ctx = ensureContext(ctx)
	if err := patient.Validate(); err != nil {
		return err
	}

	_, err := r.sb.
		Insert(tablePatients).
		Columns(patientColumns...).
		Values(patient.ID, patient.FullName, patient.Phone, patient.Email, patient.BirthDate, createdAtOrNow(patient.CreatedAt)).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("insert patient: %w", err)
	}
	return nil
}

func (r *PatientRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Patient, error) {
	ctx = ensureContext(ctx)
	row := r.sb.
		Select(patientColumns...).
		From(tablePatients).
		Where(sq.Eq{"id": id}).
		QueryRowContext(ctx)
	return scanPatient(row)
}

func (r *PatientRepository) FindAll(ctx context.Context) ([]*domain.Patient, error) {
	ctx = ensureContext(ctx)
	rows, err := r.sb.
		Select(patientColumns...).
		From(tablePatients).
		OrderBy("created_at", "id").
		QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("select patients: %w", err)
	}
	defer rows.Close()

	result := make([]*domain.Patient, 0)
	for rows.Next() {
		patient, err := scanPatient(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, patient)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate patients: %w", err)
	}
	return result, nil
}

func (r *PatientRepository) FindByEmail(ctx context.Context, email string) (*domain.Patient, error) {
	ctx = ensureContext(ctx)
	row := r.sb.
		Select(patientColumns...).
		From(tablePatients).
		Where(sq.Eq{"email": email}).
		QueryRowContext(ctx)
	return scanPatient(row)
}

func (r *PatientRepository) Update(ctx context.Context, patient *domain.Patient) error {
	ctx = ensureContext(ctx)
	result, err := r.sb.
		Update(tablePatients).
		Set("full_name", patient.FullName).
		Set("phone", patient.Phone).
		Set("email", patient.Email).
		Set("birth_date", patient.BirthDate).
		Set("created_at", createdAtOrNow(patient.CreatedAt)).
		Where(sq.Eq{"id": patient.ID}).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("update patient: %w", err)
	}
	affected, err := rowsAffected(result)
	if err != nil {
		return fmt.Errorf("update patient rows: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PatientRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx = ensureContext(ctx)
	result, err := r.sb.
		Delete(tablePatients).
		Where(sq.Eq{"id": id}).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("delete patient: %w", err)
	}
	affected, err := rowsAffected(result)
	if err != nil {
		return fmt.Errorf("delete patient rows: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

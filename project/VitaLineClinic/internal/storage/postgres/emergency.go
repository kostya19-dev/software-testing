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
	tableEmergencyRequests = "emergency_requests"
	tableRescheduleOffers  = "reschedule_offers"
)

var emergencyRequestColumns = []string{
	"id", "patient_id", "doctor_id", "requested_date", "reason", "status", "created_at",
}

var rescheduleOfferColumns = []string{
	"id", "emergency_request_id", "appointment_id", "patient_id", "doctor_id", "old_date", "proposed_date", "status", "created_at",
}

type EmergencyRepository struct {
	db *sql.DB
	sb sq.StatementBuilderType
}

func NewEmergencyRepository(db *sql.DB) *EmergencyRepository {
	return &EmergencyRepository{
		db: db,
		sb: psql.RunWith(db),
	}
}

func (r *EmergencyRepository) SaveRequest(ctx context.Context, request *domain.EmergencyRequest) error {
	ctx = ensureContext(ctx)
	if err := request.Validate(); err != nil {
		return err
	}
	_, err := r.sb.
		Insert(tableEmergencyRequests).
		Columns(emergencyRequestColumns...).
		Values(request.ID, request.PatientID, request.DoctorID, request.RequestedDate, request.Reason, string(request.Status), createdAtOrNow(request.CreatedAt)).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("insert emergency request: %w", err)
	}
	return nil
}

func (r *EmergencyRepository) FindRequestByID(ctx context.Context, id uuid.UUID) (*domain.EmergencyRequest, error) {
	ctx = ensureContext(ctx)
	row := r.sb.
		Select(emergencyRequestColumns...).
		From(tableEmergencyRequests).
		Where(sq.Eq{"id": id}).
		QueryRowContext(ctx)
	return scanEmergencyRequest(row)
}

func (r *EmergencyRepository) FindAllRequests(ctx context.Context) ([]*domain.EmergencyRequest, error) {
	ctx = ensureContext(ctx)
	rows, err := r.sb.
		Select(emergencyRequestColumns...).
		From(tableEmergencyRequests).
		OrderBy("created_at DESC", "id").
		QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("select emergency requests: %w", err)
	}
	defer rows.Close()

	result := make([]*domain.EmergencyRequest, 0)
	for rows.Next() {
		request, err := scanEmergencyRequest(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, request)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate emergency requests: %w", err)
	}
	return result, nil
}

func (r *EmergencyRepository) UpdateRequest(ctx context.Context, request *domain.EmergencyRequest) error {
	ctx = ensureContext(ctx)
	if err := request.Validate(); err != nil {
		return err
	}
	result, err := r.sb.
		Update(tableEmergencyRequests).
		Set("patient_id", request.PatientID).
		Set("doctor_id", request.DoctorID).
		Set("requested_date", request.RequestedDate).
		Set("reason", request.Reason).
		Set("status", string(request.Status)).
		Set("created_at", createdAtOrNow(request.CreatedAt)).
		Where(sq.Eq{"id": request.ID}).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("update emergency request: %w", err)
	}
	affected, err := rowsAffected(result)
	if err != nil {
		return fmt.Errorf("update emergency request rows: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *EmergencyRepository) SaveOffer(ctx context.Context, offer *domain.RescheduleOffer) error {
	ctx = ensureContext(ctx)
	if err := offer.Validate(); err != nil {
		return err
	}
	_, err := r.sb.
		Insert(tableRescheduleOffers).
		Columns(rescheduleOfferColumns...).
		Values(offer.ID, offer.EmergencyRequestID, offer.AppointmentID, offer.PatientID, offer.DoctorID, offer.OldDate, offer.ProposedDate, string(offer.Status), createdAtOrNow(offer.CreatedAt)).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("insert reschedule offer: %w", err)
	}
	return nil
}

func (r *EmergencyRepository) FindOfferByID(ctx context.Context, id uuid.UUID) (*domain.RescheduleOffer, error) {
	ctx = ensureContext(ctx)
	row := r.sb.
		Select(rescheduleOfferColumns...).
		From(tableRescheduleOffers).
		Where(sq.Eq{"id": id}).
		QueryRowContext(ctx)
	return scanRescheduleOffer(row)
}

func (r *EmergencyRepository) FindAllOffers(ctx context.Context) ([]*domain.RescheduleOffer, error) {
	ctx = ensureContext(ctx)
	rows, err := r.sb.
		Select(rescheduleOfferColumns...).
		From(tableRescheduleOffers).
		OrderBy("created_at DESC", "id").
		QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("select reschedule offers: %w", err)
	}
	defer rows.Close()

	result := make([]*domain.RescheduleOffer, 0)
	for rows.Next() {
		offer, err := scanRescheduleOffer(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, offer)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate reschedule offers: %w", err)
	}
	return result, nil
}

func (r *EmergencyRepository) UpdateOffer(ctx context.Context, offer *domain.RescheduleOffer) error {
	ctx = ensureContext(ctx)
	if err := offer.Validate(); err != nil {
		return err
	}
	result, err := r.sb.
		Update(tableRescheduleOffers).
		Set("emergency_request_id", offer.EmergencyRequestID).
		Set("appointment_id", offer.AppointmentID).
		Set("patient_id", offer.PatientID).
		Set("doctor_id", offer.DoctorID).
		Set("old_date", offer.OldDate).
		Set("proposed_date", offer.ProposedDate).
		Set("status", string(offer.Status)).
		Set("created_at", createdAtOrNow(offer.CreatedAt)).
		Where(sq.Eq{"id": offer.ID}).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("update reschedule offer: %w", err)
	}
	affected, err := rowsAffected(result)
	if err != nil {
		return fmt.Errorf("update reschedule offer rows: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func scanEmergencyRequest(s scanner) (*domain.EmergencyRequest, error) {
	var request domain.EmergencyRequest
	var status string
	if err := s.Scan(&request.ID, &request.PatientID, &request.DoctorID, &request.RequestedDate, &request.Reason, &status, &request.CreatedAt); err != nil {
		return nil, mapScanError(err)
	}
	request.Status = domain.EmergencyRequestStatus(status)
	return &request, nil
}

func scanRescheduleOffer(s scanner) (*domain.RescheduleOffer, error) {
	var offer domain.RescheduleOffer
	var status string
	if err := s.Scan(&offer.ID, &offer.EmergencyRequestID, &offer.AppointmentID, &offer.PatientID, &offer.DoctorID, &offer.OldDate, &offer.ProposedDate, &status, &offer.CreatedAt); err != nil {
		return nil, mapScanError(err)
	}
	offer.Status = domain.RescheduleOfferStatus(status)
	return &offer, nil
}

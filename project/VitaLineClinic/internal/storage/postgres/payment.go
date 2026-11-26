package postgres

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
)

const tablePayments = "payments"

var paymentColumns = []string{
	"id", "appointment_id", "amount_kopecks", "currency", "description", "status", "provider",
	"provider_payment_id", "confirmation_url", "cancellation_reason", "test", "created_at", "updated_at",
}

type PaymentRepository struct {
	sb sq.StatementBuilderType
}

func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{sb: psql.RunWith(db)}
}

func (r *PaymentRepository) Save(ctx context.Context, p *domain.Payment) error {
	ctx = ensureContext(ctx)
	if err := p.Validate(); err != nil {
		return err
	}
	_, err := r.sb.
		Insert(tablePayments).
		Columns(paymentColumns...).
		Values(p.ID, p.AppointmentID, p.Amount.Kopecks, p.Amount.Currency, p.Description, string(p.Status), p.Provider,
			p.ProviderPaymentID, p.ConfirmationURL, p.CancellationReason, p.Test, createdAtOrNow(p.CreatedAt), createdAtOrNow(p.UpdatedAt)).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("insert payment: %w", err)
	}
	return nil
}

func (r *PaymentRepository) Update(ctx context.Context, p *domain.Payment) error {
	ctx = ensureContext(ctx)
	if err := p.Validate(); err != nil {
		return err
	}
	result, err := r.sb.
		Update(tablePayments).
		Set("status", string(p.Status)).
		Set("cancellation_reason", p.CancellationReason).
		Set("updated_at", createdAtOrNow(p.UpdatedAt)).
		Where(sq.Eq{"id": p.ID}).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("update payment: %w", err)
	}
	affected, err := rowsAffected(result)
	if err != nil {
		return fmt.Errorf("update payment rows: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PaymentRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Payment, error) {
	ctx = ensureContext(ctx)
	row := r.sb.Select(paymentColumns...).From(tablePayments).Where(sq.Eq{"id": id}).QueryRowContext(ctx)
	return scanPayment(row)
}

func (r *PaymentRepository) FindByAppointment(ctx context.Context, appointmentID uuid.UUID) ([]*domain.Payment, error) {
	ctx = ensureContext(ctx)
	rows, err := r.sb.
		Select(paymentColumns...).
		From(tablePayments).
		Where(sq.Eq{"appointment_id": appointmentID}).
		OrderBy("created_at DESC").
		QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("select payments: %w", err)
	}
	defer rows.Close()

	result := make([]*domain.Payment, 0)
	for rows.Next() {
		payment, err := scanPayment(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, payment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate payments: %w", err)
	}
	return result, nil
}

func scanPayment(s scanner) (*domain.Payment, error) {
	var p domain.Payment
	var status string
	err := s.Scan(&p.ID, &p.AppointmentID, &p.Amount.Kopecks, &p.Amount.Currency, &p.Description, &status, &p.Provider,
		&p.ProviderPaymentID, &p.ConfirmationURL, &p.CancellationReason, &p.Test, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, mapScanError(err)
	}
	p.Status = domain.PaymentStatus(status)
	return &p, nil
}

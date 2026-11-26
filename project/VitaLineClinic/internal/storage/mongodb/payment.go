package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"vitaline-clinic/internal/domain"
)

const collectionPayments = "payments"

type paymentDocument struct {
	ID                 string    `bson:"_id"`
	AppointmentID      string    `bson:"appointment_id"`
	AmountKopecks      int64     `bson:"amount_kopecks"`
	Currency           string    `bson:"currency"`
	Description        string    `bson:"description"`
	Status             string    `bson:"status"`
	Provider           string    `bson:"provider"`
	ProviderPaymentID  string    `bson:"provider_payment_id"`
	ConfirmationURL    string    `bson:"confirmation_url"`
	CancellationReason string    `bson:"cancellation_reason,omitempty"`
	Test               bool      `bson:"test"`
	CreatedAt          time.Time `bson:"created_at"`
	UpdatedAt          time.Time `bson:"updated_at"`
}

type PaymentRepository struct {
	collection *mongo.Collection
}

func NewPaymentRepository(db *mongo.Database) *PaymentRepository {
	return &PaymentRepository{collection: db.Collection(collectionPayments)}
}

func (r *PaymentRepository) Save(ctx context.Context, payment *domain.Payment) error {
	ctx = ensureContext(ctx)
	if err := payment.Validate(); err != nil {
		return err
	}
	if _, err := r.collection.InsertOne(ctx, paymentToDocument(payment)); err != nil {
		return fmt.Errorf("insert payment: %w", err)
	}
	return nil
}

func (r *PaymentRepository) Update(ctx context.Context, payment *domain.Payment) error {
	ctx = ensureContext(ctx)
	if err := payment.Validate(); err != nil {
		return err
	}
	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": uuidString(payment.ID)}, bson.M{"$set": bson.M{
		"status":              string(payment.Status),
		"cancellation_reason": payment.CancellationReason,
		"updated_at":          payment.UpdatedAt,
	}})
	if err != nil {
		return fmt.Errorf("update payment: %w", err)
	}
	return rowsMatched(result.MatchedCount)
}

func (r *PaymentRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Payment, error) {
	ctx = ensureContext(ctx)
	var doc paymentDocument
	if err := r.collection.FindOne(ctx, bson.M{"_id": uuidString(id)}).Decode(&doc); err != nil {
		return nil, mapMongoError(err)
	}
	return doc.toDomain()
}

func (r *PaymentRepository) FindByAppointment(ctx context.Context, appointmentID uuid.UUID) ([]*domain.Payment, error) {
	ctx = ensureContext(ctx)
	cursor, err := r.collection.Find(ctx,
		bson.M{"appointment_id": uuidString(appointmentID)},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("select payments: %w", err)
	}
	defer cursor.Close(ctx)

	result := make([]*domain.Payment, 0)
	for cursor.Next(ctx) {
		var doc paymentDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("decode payment: %w", err)
		}
		payment, err := doc.toDomain()
		if err != nil {
			return nil, err
		}
		result = append(result, payment)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate payments: %w", err)
	}
	return result, nil
}

func paymentToDocument(p *domain.Payment) paymentDocument {
	return paymentDocument{
		ID:                 uuidString(p.ID),
		AppointmentID:      uuidString(p.AppointmentID),
		AmountKopecks:      p.Amount.Kopecks,
		Currency:           p.Amount.Currency,
		Description:        p.Description,
		Status:             string(p.Status),
		Provider:           p.Provider,
		ProviderPaymentID:  p.ProviderPaymentID,
		ConfirmationURL:    p.ConfirmationURL,
		CancellationReason: p.CancellationReason,
		Test:               p.Test,
		CreatedAt:          createdAtOrNow(p.CreatedAt),
		UpdatedAt:          createdAtOrNow(p.UpdatedAt),
	}
}

func (d paymentDocument) toDomain() (*domain.Payment, error) {
	id, err := parseUUID(d.ID, "payment id")
	if err != nil {
		return nil, err
	}
	appointmentID, err := parseUUID(d.AppointmentID, "appointment id")
	if err != nil {
		return nil, err
	}
	return &domain.Payment{
		ID:                 id,
		AppointmentID:      appointmentID,
		Amount:             domain.Money{Kopecks: d.AmountKopecks, Currency: d.Currency},
		Description:        d.Description,
		Status:             domain.PaymentStatus(d.Status),
		Provider:           d.Provider,
		ProviderPaymentID:  d.ProviderPaymentID,
		ConfirmationURL:    d.ConfirmationURL,
		CancellationReason: d.CancellationReason,
		Test:               d.Test,
		CreatedAt:          d.CreatedAt,
		UpdatedAt:          d.UpdatedAt,
	}, nil
}

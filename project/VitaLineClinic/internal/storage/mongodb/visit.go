package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"vitaline-clinic/internal/domain"
)

const collectionVisits = "visits"

type visitDocument struct {
	ID              string    `bson:"_id"`
	AppointmentID   string    `bson:"appointment_id"`
	Complaints      string    `bson:"complaints"`
	Diagnosis       string    `bson:"diagnosis"`
	Recommendations string    `bson:"recommendations"`
	Status          string    `bson:"status"`
	CreatedAt       time.Time `bson:"created_at"`
}

type VisitRepository struct {
	collection *mongo.Collection
}

func NewVisitRepository(db *mongo.Database) *VisitRepository {
	return &VisitRepository{collection: db.Collection(collectionVisits)}
}

func (r *VisitRepository) Save(ctx context.Context, visit *domain.Visit) error {
	ctx = ensureContext(ctx)
	if err := visit.Validate(); err != nil {
		return err
	}
	_, err := r.collection.InsertOne(ctx, visitToDocument(visit))
	if err != nil {
		return fmt.Errorf("insert visit: %w", err)
	}
	return nil
}

func (r *VisitRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Visit, error) {
	ctx = ensureContext(ctx)
	var doc visitDocument
	err := r.collection.FindOne(ctx, bson.M{"_id": uuidString(id)}).Decode(&doc)
	if err != nil {
		return nil, mapMongoError(err)
	}
	return doc.toDomain()
}

func (r *VisitRepository) FindByAppointment(ctx context.Context, appointmentID uuid.UUID) (*domain.Visit, error) {
	ctx = ensureContext(ctx)
	var doc visitDocument
	err := r.collection.FindOne(ctx, bson.M{"appointment_id": uuidString(appointmentID)}).Decode(&doc)
	if err != nil {
		return nil, mapMongoError(err)
	}
	return doc.toDomain()
}

func (r *VisitRepository) Update(ctx context.Context, visit *domain.Visit) error {
	ctx = ensureContext(ctx)
	if err := visit.Validate(); err != nil {
		return err
	}
	result, err := r.collection.ReplaceOne(ctx, bson.M{"_id": uuidString(visit.ID)}, visitToDocument(visit))
	if err != nil {
		return fmt.Errorf("update visit: %w", err)
	}
	return rowsMatched(result.MatchedCount)
}

func (r *VisitRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx = ensureContext(ctx)
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": uuidString(id)})
	if err != nil {
		return fmt.Errorf("delete visit: %w", err)
	}
	return rowsMatched(result.DeletedCount)
}

func visitToDocument(visit *domain.Visit) visitDocument {
	return visitDocument{
		ID:              uuidString(visit.ID),
		AppointmentID:   uuidString(visit.AppointmentID),
		Complaints:      visit.Complaints,
		Diagnosis:       visit.Diagnosis,
		Recommendations: visit.Recommendations,
		Status:          string(visit.Status),
		CreatedAt:       createdAtOrNow(visit.CreatedAt),
	}
}

func (d visitDocument) toDomain() (*domain.Visit, error) {
	id, err := parseUUID(d.ID, "visit._id")
	if err != nil {
		return nil, err
	}
	appointmentID, err := parseUUID(d.AppointmentID, "visit.appointment_id")
	if err != nil {
		return nil, err
	}
	return &domain.Visit{
		ID:              id,
		AppointmentID:   appointmentID,
		Complaints:      d.Complaints,
		Diagnosis:       d.Diagnosis,
		Recommendations: d.Recommendations,
		Status:          domain.VisitStatus(d.Status),
		CreatedAt:       d.CreatedAt,
	}, nil
}

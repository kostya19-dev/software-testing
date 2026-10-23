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

const collectionPatients = "patients"

type patientDocument struct {
	ID        string    `bson:"_id"`
	FullName  string    `bson:"full_name"`
	Phone     string    `bson:"phone"`
	Email     string    `bson:"email"`
	BirthDate time.Time `bson:"birth_date"`
	CreatedAt time.Time `bson:"created_at"`
}

type PatientRepository struct {
	collection *mongo.Collection
}

func NewPatientRepository(db *mongo.Database) *PatientRepository {
	return &PatientRepository{collection: db.Collection(collectionPatients)}
}

func (r *PatientRepository) Save(ctx context.Context, patient *domain.Patient) error {
	ctx = ensureContext(ctx)
	if err := patient.Validate(); err != nil {
		return err
	}
	_, err := r.collection.InsertOne(ctx, patientToDocument(patient))
	if err != nil {
		return fmt.Errorf("insert patient: %w", err)
	}
	return nil
}

func (r *PatientRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Patient, error) {
	ctx = ensureContext(ctx)
	var doc patientDocument
	err := r.collection.FindOne(ctx, bson.M{"_id": uuidString(id)}).Decode(&doc)
	if err != nil {
		return nil, mapMongoError(err)
	}
	return doc.toDomain()
}

func (r *PatientRepository) FindAll(ctx context.Context) ([]*domain.Patient, error) {
	ctx = ensureContext(ctx)
	cursor, err := r.collection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}, {Key: "_id", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("select patients: %w", err)
	}
	defer cursor.Close(ctx)

	result := make([]*domain.Patient, 0)
	for cursor.Next(ctx) {
		var doc patientDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("decode patient: %w", err)
		}
		patient, err := doc.toDomain()
		if err != nil {
			return nil, err
		}
		result = append(result, patient)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate patients: %w", err)
	}
	return result, nil
}

func (r *PatientRepository) FindByEmail(ctx context.Context, email string) (*domain.Patient, error) {
	ctx = ensureContext(ctx)
	var doc patientDocument
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&doc)
	if err != nil {
		return nil, mapMongoError(err)
	}
	return doc.toDomain()
}

func (r *PatientRepository) Update(ctx context.Context, patient *domain.Patient) error {
	ctx = ensureContext(ctx)
	if err := patient.Validate(); err != nil {
		return err
	}
	result, err := r.collection.ReplaceOne(ctx, bson.M{"_id": uuidString(patient.ID)}, patientToDocument(patient))
	if err != nil {
		return fmt.Errorf("update patient: %w", err)
	}
	return rowsMatched(result.MatchedCount)
}

func (r *PatientRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx = ensureContext(ctx)
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": uuidString(id)})
	if err != nil {
		return fmt.Errorf("delete patient: %w", err)
	}
	return rowsMatched(result.DeletedCount)
}

func patientToDocument(patient *domain.Patient) patientDocument {
	return patientDocument{
		ID:        uuidString(patient.ID),
		FullName:  patient.FullName,
		Phone:     patient.Phone,
		Email:     patient.Email,
		BirthDate: patient.BirthDate,
		CreatedAt: createdAtOrNow(patient.CreatedAt),
	}
}

func (d patientDocument) toDomain() (*domain.Patient, error) {
	id, err := parseUUID(d.ID, "patient._id")
	if err != nil {
		return nil, err
	}
	return &domain.Patient{
		ID:        id,
		FullName:  d.FullName,
		Phone:     d.Phone,
		Email:     d.Email,
		BirthDate: d.BirthDate,
		CreatedAt: d.CreatedAt,
	}, nil
}

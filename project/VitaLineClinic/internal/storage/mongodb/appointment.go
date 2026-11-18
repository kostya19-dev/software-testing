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

const collectionAppointments = "appointments"

type appointmentDocument struct {
	ID        string    `bson:"_id"`
	PatientID string    `bson:"patient_id"`
	DoctorID  string    `bson:"doctor_id"`
	Status    string    `bson:"status"`
	Date      time.Time `bson:"appointment_date"`
	CreatedAt time.Time `bson:"created_at"`
}

type AppointmentRepository struct {
	collection *mongo.Collection
}

func NewAppointmentRepository(db *mongo.Database) *AppointmentRepository {
	return &AppointmentRepository{collection: db.Collection(collectionAppointments)}
}

func (r *AppointmentRepository) Save(ctx context.Context, app *domain.Appointment) error {
	ctx = ensureContext(ctx)
	if err := app.Validate(); err != nil {
		return err
	}
	_, err := r.collection.InsertOne(ctx, appointmentToDocument(app))
	if err != nil {
		return fmt.Errorf("insert appointment: %w", err)
	}
	return nil
}

func (r *AppointmentRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Appointment, error) {
	ctx = ensureContext(ctx)
	var doc appointmentDocument
	err := r.collection.FindOne(ctx, bson.M{"_id": uuidString(id)}).Decode(&doc)
	if err != nil {
		return nil, mapMongoError(err)
	}
	return doc.toDomain()
}

func (r *AppointmentRepository) FindAll(ctx context.Context) ([]*domain.Appointment, error) {
	ctx = ensureContext(ctx)
	cursor, err := r.collection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}, {Key: "_id", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("select appointments: %w", err)
	}
	defer cursor.Close(ctx)
	return decodeAppointments(ctx, cursor)
}

func (r *AppointmentRepository) FindByPatient(ctx context.Context, patientID uuid.UUID) ([]*domain.Appointment, error) {
	ctx = ensureContext(ctx)
	cursor, err := r.collection.Find(
		ctx,
		bson.M{"patient_id": uuidString(patientID)},
		options.Find().SetSort(bson.D{{Key: "appointment_date", Value: 1}, {Key: "_id", Value: 1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("select appointments by patient: %w", err)
	}
	defer cursor.Close(ctx)
	return decodeAppointments(ctx, cursor)
}

func (r *AppointmentRepository) FindActiveByDoctorAndDate(ctx context.Context, doctorID uuid.UUID, date time.Time) (*domain.Appointment, error) {
	ctx = ensureContext(ctx)
	var doc appointmentDocument
	err := r.collection.FindOne(ctx, activeAppointmentFilter(doctorID, date)).Decode(&doc)
	if err != nil {
		return nil, mapMongoError(err)
	}
	return doc.toDomain()
}

func (r *AppointmentRepository) CheckSlotBusy(ctx context.Context, doctorID uuid.UUID, date time.Time) (bool, error) {
	ctx = ensureContext(ctx)
	count, err := r.collection.CountDocuments(ctx, activeAppointmentFilter(doctorID, date))
	if err != nil {
		return false, fmt.Errorf("check slot busy: %w", err)
	}
	return count > 0, nil
}

func (r *AppointmentRepository) Update(ctx context.Context, app *domain.Appointment) error {
	ctx = ensureContext(ctx)
	if err := app.Validate(); err != nil {
		return err
	}
	result, err := r.collection.ReplaceOne(ctx, bson.M{"_id": uuidString(app.ID)}, appointmentToDocument(app))
	if err != nil {
		return fmt.Errorf("update appointment: %w", err)
	}
	return rowsMatched(result.MatchedCount)
}

func (r *AppointmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx = ensureContext(ctx)
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": uuidString(id)})
	if err != nil {
		return fmt.Errorf("delete appointment: %w", err)
	}
	return rowsMatched(result.DeletedCount)
}

func activeAppointmentFilter(doctorID uuid.UUID, date time.Time) bson.M {
	return bson.M{
		"doctor_id":        uuidString(doctorID),
		"appointment_date": date,
		"status":           bson.M{"$ne": string(domain.StatusCancelled)},
	}
}

func decodeAppointments(ctx context.Context, cursor *mongo.Cursor) ([]*domain.Appointment, error) {
	result := make([]*domain.Appointment, 0)
	for cursor.Next(ctx) {
		var doc appointmentDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("decode appointment: %w", err)
		}
		appointment, err := doc.toDomain()
		if err != nil {
			return nil, err
		}
		result = append(result, appointment)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate appointments: %w", err)
	}
	return result, nil
}

func appointmentToDocument(app *domain.Appointment) appointmentDocument {
	return appointmentDocument{
		ID:        uuidString(app.ID),
		PatientID: uuidString(app.PatientID),
		DoctorID:  uuidString(app.DoctorID),
		Status:    string(app.Status),
		Date:      app.Date,
		CreatedAt: createdAtOrNow(app.CreatedAt),
	}
}

func (d appointmentDocument) toDomain() (*domain.Appointment, error) {
	id, err := parseUUID(d.ID, "appointment._id")
	if err != nil {
		return nil, err
	}
	patientID, err := parseUUID(d.PatientID, "appointment.patient_id")
	if err != nil {
		return nil, err
	}
	doctorID, err := parseUUID(d.DoctorID, "appointment.doctor_id")
	if err != nil {
		return nil, err
	}
	return &domain.Appointment{
		ID:        id,
		PatientID: patientID,
		DoctorID:  doctorID,
		Status:    domain.AppointmentStatus(d.Status),
		Date:      d.Date,
		CreatedAt: d.CreatedAt,
	}, nil
}

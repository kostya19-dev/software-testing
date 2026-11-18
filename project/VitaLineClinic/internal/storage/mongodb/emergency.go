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

const (
	collectionEmergencyRequests = "emergency_requests"
	collectionRescheduleOffers  = "reschedule_offers"
)

type emergencyRequestDocument struct {
	ID            string    `bson:"_id"`
	PatientID     string    `bson:"patient_id"`
	DoctorID      string    `bson:"doctor_id"`
	RequestedDate time.Time `bson:"requested_date"`
	Reason        string    `bson:"reason"`
	Status        string    `bson:"status"`
	CreatedAt     time.Time `bson:"created_at"`
}

type rescheduleOfferDocument struct {
	ID                 string    `bson:"_id"`
	EmergencyRequestID string    `bson:"emergency_request_id"`
	AppointmentID      string    `bson:"appointment_id"`
	PatientID          string    `bson:"patient_id"`
	DoctorID           string    `bson:"doctor_id"`
	OldDate            time.Time `bson:"old_date"`
	ProposedDate       time.Time `bson:"proposed_date"`
	Status             string    `bson:"status"`
	CreatedAt          time.Time `bson:"created_at"`
}

type EmergencyRepository struct {
	requests *mongo.Collection
	offers   *mongo.Collection
}

func NewEmergencyRepository(db *mongo.Database) *EmergencyRepository {
	return &EmergencyRepository{
		requests: db.Collection(collectionEmergencyRequests),
		offers:   db.Collection(collectionRescheduleOffers),
	}
}

func (r *EmergencyRepository) SaveRequest(ctx context.Context, request *domain.EmergencyRequest) error {
	ctx = ensureContext(ctx)
	if err := request.Validate(); err != nil {
		return err
	}
	_, err := r.requests.InsertOne(ctx, emergencyRequestToDocument(request))
	if err != nil {
		return fmt.Errorf("insert emergency request: %w", err)
	}
	return nil
}

func (r *EmergencyRepository) FindRequestByID(ctx context.Context, id uuid.UUID) (*domain.EmergencyRequest, error) {
	ctx = ensureContext(ctx)
	var doc emergencyRequestDocument
	err := r.requests.FindOne(ctx, bson.M{"_id": uuidString(id)}).Decode(&doc)
	if err != nil {
		return nil, mapMongoError(err)
	}
	return doc.toDomain()
}

func (r *EmergencyRepository) FindAllRequests(ctx context.Context) ([]*domain.EmergencyRequest, error) {
	ctx = ensureContext(ctx)
	cursor, err := r.requests.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("select emergency requests: %w", err)
	}
	defer cursor.Close(ctx)

	result := make([]*domain.EmergencyRequest, 0)
	for cursor.Next(ctx) {
		var doc emergencyRequestDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("decode emergency request: %w", err)
		}
		request, err := doc.toDomain()
		if err != nil {
			return nil, err
		}
		result = append(result, request)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate emergency requests: %w", err)
	}
	return result, nil
}

func (r *EmergencyRepository) UpdateRequest(ctx context.Context, request *domain.EmergencyRequest) error {
	ctx = ensureContext(ctx)
	if err := request.Validate(); err != nil {
		return err
	}
	result, err := r.requests.ReplaceOne(ctx, bson.M{"_id": uuidString(request.ID)}, emergencyRequestToDocument(request))
	if err != nil {
		return fmt.Errorf("update emergency request: %w", err)
	}
	return rowsMatched(result.MatchedCount)
}

func (r *EmergencyRepository) SaveOffer(ctx context.Context, offer *domain.RescheduleOffer) error {
	ctx = ensureContext(ctx)
	if err := offer.Validate(); err != nil {
		return err
	}
	_, err := r.offers.InsertOne(ctx, rescheduleOfferToDocument(offer))
	if err != nil {
		return fmt.Errorf("insert reschedule offer: %w", err)
	}
	return nil
}

func (r *EmergencyRepository) FindOfferByID(ctx context.Context, id uuid.UUID) (*domain.RescheduleOffer, error) {
	ctx = ensureContext(ctx)
	var doc rescheduleOfferDocument
	err := r.offers.FindOne(ctx, bson.M{"_id": uuidString(id)}).Decode(&doc)
	if err != nil {
		return nil, mapMongoError(err)
	}
	return doc.toDomain()
}

func (r *EmergencyRepository) FindAllOffers(ctx context.Context) ([]*domain.RescheduleOffer, error) {
	ctx = ensureContext(ctx)
	cursor, err := r.offers.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("select reschedule offers: %w", err)
	}
	defer cursor.Close(ctx)

	result := make([]*domain.RescheduleOffer, 0)
	for cursor.Next(ctx) {
		var doc rescheduleOfferDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("decode reschedule offer: %w", err)
		}
		offer, err := doc.toDomain()
		if err != nil {
			return nil, err
		}
		result = append(result, offer)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate reschedule offers: %w", err)
	}
	return result, nil
}

func (r *EmergencyRepository) UpdateOffer(ctx context.Context, offer *domain.RescheduleOffer) error {
	ctx = ensureContext(ctx)
	if err := offer.Validate(); err != nil {
		return err
	}
	result, err := r.offers.ReplaceOne(ctx, bson.M{"_id": uuidString(offer.ID)}, rescheduleOfferToDocument(offer))
	if err != nil {
		return fmt.Errorf("update reschedule offer: %w", err)
	}
	return rowsMatched(result.MatchedCount)
}

func emergencyRequestToDocument(request *domain.EmergencyRequest) emergencyRequestDocument {
	return emergencyRequestDocument{
		ID:            uuidString(request.ID),
		PatientID:     uuidString(request.PatientID),
		DoctorID:      uuidString(request.DoctorID),
		RequestedDate: request.RequestedDate,
		Reason:        request.Reason,
		Status:        string(request.Status),
		CreatedAt:     createdAtOrNow(request.CreatedAt),
	}
}

func (d emergencyRequestDocument) toDomain() (*domain.EmergencyRequest, error) {
	id, err := parseUUID(d.ID, "emergency_request._id")
	if err != nil {
		return nil, err
	}
	patientID, err := parseUUID(d.PatientID, "emergency_request.patient_id")
	if err != nil {
		return nil, err
	}
	doctorID, err := parseUUID(d.DoctorID, "emergency_request.doctor_id")
	if err != nil {
		return nil, err
	}
	return &domain.EmergencyRequest{
		ID:            id,
		PatientID:     patientID,
		DoctorID:      doctorID,
		RequestedDate: d.RequestedDate,
		Reason:        d.Reason,
		Status:        domain.EmergencyRequestStatus(d.Status),
		CreatedAt:     d.CreatedAt,
	}, nil
}

func rescheduleOfferToDocument(offer *domain.RescheduleOffer) rescheduleOfferDocument {
	return rescheduleOfferDocument{
		ID:                 uuidString(offer.ID),
		EmergencyRequestID: uuidString(offer.EmergencyRequestID),
		AppointmentID:      uuidString(offer.AppointmentID),
		PatientID:          uuidString(offer.PatientID),
		DoctorID:           uuidString(offer.DoctorID),
		OldDate:            offer.OldDate,
		ProposedDate:       offer.ProposedDate,
		Status:             string(offer.Status),
		CreatedAt:          createdAtOrNow(offer.CreatedAt),
	}
}

func (d rescheduleOfferDocument) toDomain() (*domain.RescheduleOffer, error) {
	id, err := parseUUID(d.ID, "reschedule_offer._id")
	if err != nil {
		return nil, err
	}
	requestID, err := parseUUID(d.EmergencyRequestID, "reschedule_offer.emergency_request_id")
	if err != nil {
		return nil, err
	}
	appointmentID, err := parseUUID(d.AppointmentID, "reschedule_offer.appointment_id")
	if err != nil {
		return nil, err
	}
	patientID, err := parseUUID(d.PatientID, "reschedule_offer.patient_id")
	if err != nil {
		return nil, err
	}
	doctorID, err := parseUUID(d.DoctorID, "reschedule_offer.doctor_id")
	if err != nil {
		return nil, err
	}
	return &domain.RescheduleOffer{
		ID:                 id,
		EmergencyRequestID: requestID,
		AppointmentID:      appointmentID,
		PatientID:          patientID,
		DoctorID:           doctorID,
		OldDate:            d.OldDate,
		ProposedDate:       d.ProposedDate,
		Status:             domain.RescheduleOfferStatus(d.Status),
		CreatedAt:          d.CreatedAt,
	}, nil
}

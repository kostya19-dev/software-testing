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

const collectionScheduleSlots = "schedule_slots"

type scheduleSlotDocument struct {
	ID        string    `bson:"_id"`
	DoctorID  string    `bson:"doctor_id"`
	Date      time.Time `bson:"slot_date"`
	StartTime time.Time `bson:"start_time"`
	EndTime   time.Time `bson:"end_time"`
	Cabinet   string    `bson:"cabinet"`
	IsBooked  bool      `bson:"is_booked"`
}

type ScheduleRepository struct {
	collection *mongo.Collection
}

func NewScheduleRepository(db *mongo.Database) *ScheduleRepository {
	return &ScheduleRepository{collection: db.Collection(collectionScheduleSlots)}
}

func (r *ScheduleRepository) Save(ctx context.Context, slot *domain.ScheduleSlot) error {
	ctx = ensureContext(ctx)
	if err := slot.Validate(); err != nil {
		return err
	}
	_, err := r.collection.InsertOne(ctx, scheduleSlotToDocument(slot))
	if err != nil {
		return fmt.Errorf("insert schedule slot: %w", err)
	}
	return nil
}

func (r *ScheduleRepository) FindByDoctorID(ctx context.Context, doctorID uuid.UUID) ([]*domain.ScheduleSlot, error) {
	ctx = ensureContext(ctx)
	cursor, err := r.collection.Find(
		ctx,
		bson.M{"doctor_id": uuidString(doctorID)},
		options.Find().SetSort(bson.D{{Key: "slot_date", Value: 1}, {Key: "start_time", Value: 1}, {Key: "_id", Value: 1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("select schedule slots: %w", err)
	}
	defer cursor.Close(ctx)
	return decodeScheduleSlots(ctx, cursor)
}

func (r *ScheduleRepository) FindFreeSlots(ctx context.Context, doctorID uuid.UUID, date time.Time) ([]*domain.ScheduleSlot, error) {
	ctx = ensureContext(ctx)
	start := beginningOfDay(date)
	end := start.AddDate(0, 0, 1)
	cursor, err := r.collection.Find(
		ctx,
		bson.M{
			"doctor_id": uuidString(doctorID),
			"slot_date": bson.M{"$gte": start, "$lt": end},
			"is_booked": false,
		},
		options.Find().SetSort(bson.D{{Key: "start_time", Value: 1}, {Key: "_id", Value: 1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("select free schedule slots: %w", err)
	}
	defer cursor.Close(ctx)
	return decodeScheduleSlots(ctx, cursor)
}

func (r *ScheduleRepository) Update(ctx context.Context, slot *domain.ScheduleSlot) error {
	ctx = ensureContext(ctx)
	if err := slot.Validate(); err != nil {
		return err
	}
	result, err := r.collection.ReplaceOne(ctx, bson.M{"_id": uuidString(slot.ID)}, scheduleSlotToDocument(slot))
	if err != nil {
		return fmt.Errorf("update schedule slot: %w", err)
	}
	return rowsMatched(result.MatchedCount)
}

func (r *ScheduleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx = ensureContext(ctx)
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": uuidString(id)})
	if err != nil {
		return fmt.Errorf("delete schedule slot: %w", err)
	}
	return rowsMatched(result.DeletedCount)
}

func decodeScheduleSlots(ctx context.Context, cursor *mongo.Cursor) ([]*domain.ScheduleSlot, error) {
	result := make([]*domain.ScheduleSlot, 0)
	for cursor.Next(ctx) {
		var doc scheduleSlotDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("decode schedule slot: %w", err)
		}
		slot, err := doc.toDomain()
		if err != nil {
			return nil, err
		}
		result = append(result, slot)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate schedule slots: %w", err)
	}
	return result, nil
}

func scheduleSlotToDocument(slot *domain.ScheduleSlot) scheduleSlotDocument {
	return scheduleSlotDocument{
		ID:        uuidString(slot.ID),
		DoctorID:  uuidString(slot.DoctorID),
		Date:      slot.Date,
		StartTime: slot.StartTime,
		EndTime:   slot.EndTime,
		Cabinet:   slot.Cabinet,
		IsBooked:  slot.IsBooked,
	}
}

func (d scheduleSlotDocument) toDomain() (*domain.ScheduleSlot, error) {
	id, err := parseUUID(d.ID, "schedule_slot._id")
	if err != nil {
		return nil, err
	}
	doctorID, err := parseUUID(d.DoctorID, "schedule_slot.doctor_id")
	if err != nil {
		return nil, err
	}
	return &domain.ScheduleSlot{
		ID:        id,
		DoctorID:  doctorID,
		Date:      d.Date,
		StartTime: d.StartTime,
		EndTime:   d.EndTime,
		Cabinet:   d.Cabinet,
		IsBooked:  d.IsBooked,
	}, nil
}

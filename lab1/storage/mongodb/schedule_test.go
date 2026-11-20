package mongodb_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"

	"vitaline-clinic/internal/domain"
	"vitaline-clinic/internal/storage/mongodb"
	"vitaline-clinic/lab1/testutil"
)

func TestScheduleMongoRepository(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("Save/success", func(mt *mtest.T) {
		p := testutil.Schedule()
		r := mongodb.NewScheduleRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}))
		err := r.Save(ctx, p)
		t := mt.T
		if err != nil {
			t.Fatal(err)
		}
	})
	mt.Run("Save/server_error", func(mt *mtest.T) {
		p := testutil.Schedule()
		r := mongodb.NewScheduleRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{Code: 123, Message: "injected"}))
		err := r.Save(ctx, p)
		t := mt.T
		assertStorageError(t, err)
	})
	mt.Run("Update/success", func(mt *mtest.T) {
		p := testutil.Schedule()
		r := mongodb.NewScheduleRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}))
		err := r.Update(ctx, p)
		t := mt.T
		if err != nil {
			t.Fatal(err)
		}
	})
	mt.Run("Update/not_found", func(mt *mtest.T) {
		p := testutil.Schedule()
		r := mongodb.NewScheduleRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 0}))
		err := r.Update(ctx, p)
		t := mt.T
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("%v", err)
		}
	})
	mt.Run("Delete/success", func(mt *mtest.T) {
		p := testutil.Schedule()
		r := mongodb.NewScheduleRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}))
		err := r.Delete(ctx, p.ID)
		t := mt.T
		if err != nil {
			t.Fatal(err)
		}
	})
	mt.Run("Delete/not_found", func(mt *mtest.T) {
		p := testutil.Schedule()
		r := mongodb.NewScheduleRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 0}))
		err := r.Delete(ctx, p.ID)
		t := mt.T
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("%v", err)
		}
	})
	mt.Run("FindByDoctorID/success", func(mt *mtest.T) {
		p := testutil.Schedule()
		r := mongodb.NewScheduleRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".schedule_slots", mtest.FirstBatch, mongoDoc(mt.T, bson.M{"_id": p.ID.String(), "doctor_id": p.DoctorID.String(), "slot_date": p.Date, "start_time": p.StartTime, "end_time": p.EndTime, "cabinet": p.Cabinet, "is_booked": p.IsBooked})))
		got, err := r.FindByDoctorID(ctx, p.DoctorID)
		t := mt.T
		if err != nil || len(got) != 1 || !reflect.DeepEqual(got[0], p) {
			t.Fatalf("got=%+v want=%+v err=%v", got, p, err)
		}
	})
	mt.Run("FindByDoctorID/empty", func(mt *mtest.T) {
		p := testutil.Schedule()
		r := mongodb.NewScheduleRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".schedule_slots", mtest.FirstBatch))
		got, err := r.FindByDoctorID(ctx, p.DoctorID)
		t := mt.T
		if err != nil || len(got) != 0 {
			t.Fatalf("%v %v", got, err)
		}
	})
	mt.Run("FindFreeSlots/success", func(mt *mtest.T) {
		p := testutil.Schedule()
		r := mongodb.NewScheduleRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".schedule_slots", mtest.FirstBatch, mongoDoc(mt.T, bson.M{"_id": p.ID.String(), "doctor_id": p.DoctorID.String(), "slot_date": p.Date, "start_time": p.StartTime, "end_time": p.EndTime, "cabinet": p.Cabinet, "is_booked": p.IsBooked})))
		got, err := r.FindFreeSlots(ctx, p.DoctorID, p.Date)
		t := mt.T
		if err != nil || len(got) != 1 || !reflect.DeepEqual(got[0], p) {
			t.Fatalf("got=%+v want=%+v err=%v", got, p, err)
		}
	})
	mt.Run("FindFreeSlots/empty", func(mt *mtest.T) {
		p := testutil.Schedule()
		r := mongodb.NewScheduleRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".schedule_slots", mtest.FirstBatch))
		got, err := r.FindFreeSlots(ctx, p.DoctorID, p.Date)
		t := mt.T
		if err != nil || len(got) != 0 {
			t.Fatalf("%v %v", got, err)
		}
	})
}

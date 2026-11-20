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

func TestEmergencyRequestMongoRepository(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("SaveRequest/success", func(mt *mtest.T) {
		p := testutil.EmergencyRequest()
		r := mongodb.NewEmergencyRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}))
		err := r.SaveRequest(ctx, p)
		t := mt.T
		if err != nil {
			t.Fatal(err)
		}
	})
	mt.Run("SaveRequest/server_error", func(mt *mtest.T) {
		p := testutil.EmergencyRequest()
		r := mongodb.NewEmergencyRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{Code: 123, Message: "injected"}))
		err := r.SaveRequest(ctx, p)
		t := mt.T
		assertStorageError(t, err)
	})
	mt.Run("UpdateRequest/success", func(mt *mtest.T) {
		p := testutil.EmergencyRequest()
		r := mongodb.NewEmergencyRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}))
		err := r.UpdateRequest(ctx, p)
		t := mt.T
		if err != nil {
			t.Fatal(err)
		}
	})
	mt.Run("UpdateRequest/not_found", func(mt *mtest.T) {
		p := testutil.EmergencyRequest()
		r := mongodb.NewEmergencyRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 0}))
		err := r.UpdateRequest(ctx, p)
		t := mt.T
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("%v", err)
		}
	})
	mt.Run("FindRequestByID/success", func(mt *mtest.T) {
		p := testutil.EmergencyRequest()
		r := mongodb.NewEmergencyRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".emergency_requests", mtest.FirstBatch, mongoDoc(mt.T, bson.M{"_id": p.ID.String(), "patient_id": p.PatientID.String(), "doctor_id": p.DoctorID.String(), "requested_date": p.RequestedDate, "reason": p.Reason, "status": string(p.Status), "created_at": p.CreatedAt})))
		got, err := r.FindRequestByID(ctx, p.ID)
		t := mt.T
		if err != nil || !reflect.DeepEqual(got, p) {
			t.Fatalf("got=%+v want=%+v err=%v", got, p, err)
		}
	})
	mt.Run("FindRequestByID/empty", func(mt *mtest.T) {
		p := testutil.EmergencyRequest()
		r := mongodb.NewEmergencyRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".emergency_requests", mtest.FirstBatch))
		got, err := r.FindRequestByID(ctx, p.ID)
		t := mt.T
		if !errors.Is(err, domain.ErrNotFound) || got != nil {
			t.Fatalf("%v %v", got, err)
		}
	})
	mt.Run("FindAllRequests/success", func(mt *mtest.T) {
		p := testutil.EmergencyRequest()
		r := mongodb.NewEmergencyRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".emergency_requests", mtest.FirstBatch, mongoDoc(mt.T, bson.M{"_id": p.ID.String(), "patient_id": p.PatientID.String(), "doctor_id": p.DoctorID.String(), "requested_date": p.RequestedDate, "reason": p.Reason, "status": string(p.Status), "created_at": p.CreatedAt})))
		got, err := r.FindAllRequests(ctx)
		t := mt.T
		if err != nil || len(got) != 1 || !reflect.DeepEqual(got[0], p) {
			t.Fatalf("got=%+v want=%+v err=%v", got, p, err)
		}
	})
	mt.Run("FindAllRequests/empty", func(mt *mtest.T) {
		r := mongodb.NewEmergencyRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".emergency_requests", mtest.FirstBatch))
		got, err := r.FindAllRequests(ctx)
		t := mt.T
		if err != nil || len(got) != 0 {
			t.Fatalf("%v %v", got, err)
		}
	})
}

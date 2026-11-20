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

func TestRescheduleOfferMongoRepository(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("SaveOffer/success", func(mt *mtest.T) {
		p := testutil.RescheduleOffer()
		r := mongodb.NewEmergencyRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}))
		err := r.SaveOffer(ctx, p)
		t := mt.T
		if err != nil {
			t.Fatal(err)
		}
	})
	mt.Run("SaveOffer/server_error", func(mt *mtest.T) {
		p := testutil.RescheduleOffer()
		r := mongodb.NewEmergencyRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{Code: 123, Message: "injected"}))
		err := r.SaveOffer(ctx, p)
		t := mt.T
		assertStorageError(t, err)
	})
	mt.Run("UpdateOffer/success", func(mt *mtest.T) {
		p := testutil.RescheduleOffer()
		r := mongodb.NewEmergencyRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}))
		err := r.UpdateOffer(ctx, p)
		t := mt.T
		if err != nil {
			t.Fatal(err)
		}
	})
	mt.Run("UpdateOffer/not_found", func(mt *mtest.T) {
		p := testutil.RescheduleOffer()
		r := mongodb.NewEmergencyRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 0}))
		err := r.UpdateOffer(ctx, p)
		t := mt.T
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("%v", err)
		}
	})
	mt.Run("FindOfferByID/success", func(mt *mtest.T) {
		p := testutil.RescheduleOffer()
		r := mongodb.NewEmergencyRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".reschedule_offers", mtest.FirstBatch, mongoDoc(mt.T, bson.M{"_id": p.ID.String(), "emergency_request_id": p.EmergencyRequestID.String(), "appointment_id": p.AppointmentID.String(), "patient_id": p.PatientID.String(), "doctor_id": p.DoctorID.String(), "old_date": p.OldDate, "proposed_date": p.ProposedDate, "status": string(p.Status), "created_at": p.CreatedAt})))
		got, err := r.FindOfferByID(ctx, p.ID)
		t := mt.T
		if err != nil || !reflect.DeepEqual(got, p) {
			t.Fatalf("got=%+v want=%+v err=%v", got, p, err)
		}
	})
	mt.Run("FindOfferByID/empty", func(mt *mtest.T) {
		p := testutil.RescheduleOffer()
		r := mongodb.NewEmergencyRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".reschedule_offers", mtest.FirstBatch))
		got, err := r.FindOfferByID(ctx, p.ID)
		t := mt.T
		if !errors.Is(err, domain.ErrNotFound) || got != nil {
			t.Fatalf("%v %v", got, err)
		}
	})
	mt.Run("FindAllOffers/success", func(mt *mtest.T) {
		p := testutil.RescheduleOffer()
		r := mongodb.NewEmergencyRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".reschedule_offers", mtest.FirstBatch, mongoDoc(mt.T, bson.M{"_id": p.ID.String(), "emergency_request_id": p.EmergencyRequestID.String(), "appointment_id": p.AppointmentID.String(), "patient_id": p.PatientID.String(), "doctor_id": p.DoctorID.String(), "old_date": p.OldDate, "proposed_date": p.ProposedDate, "status": string(p.Status), "created_at": p.CreatedAt})))
		got, err := r.FindAllOffers(ctx)
		t := mt.T
		if err != nil || len(got) != 1 || !reflect.DeepEqual(got[0], p) {
			t.Fatalf("got=%+v want=%+v err=%v", got, p, err)
		}
	})
	mt.Run("FindAllOffers/empty", func(mt *mtest.T) {
		r := mongodb.NewEmergencyRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".reschedule_offers", mtest.FirstBatch))
		got, err := r.FindAllOffers(ctx)
		t := mt.T
		if err != nil || len(got) != 0 {
			t.Fatalf("%v %v", got, err)
		}
	})
}

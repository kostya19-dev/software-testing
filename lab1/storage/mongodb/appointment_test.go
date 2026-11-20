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

func TestAppointmentMongoRepository(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("Save/success", func(mt *mtest.T) {
		p := testutil.Appointment()
		r := mongodb.NewAppointmentRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}))
		err := r.Save(ctx, p)
		t := mt.T
		if err != nil {
			t.Fatal(err)
		}
	})
	mt.Run("Save/server_error", func(mt *mtest.T) {
		p := testutil.Appointment()
		r := mongodb.NewAppointmentRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{Code: 123, Message: "injected"}))
		err := r.Save(ctx, p)
		t := mt.T
		assertStorageError(t, err)
	})
	mt.Run("Update/success", func(mt *mtest.T) {
		p := testutil.Appointment()
		r := mongodb.NewAppointmentRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}))
		err := r.Update(ctx, p)
		t := mt.T
		if err != nil {
			t.Fatal(err)
		}
	})
	mt.Run("Update/not_found", func(mt *mtest.T) {
		p := testutil.Appointment()
		r := mongodb.NewAppointmentRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 0}))
		err := r.Update(ctx, p)
		t := mt.T
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("%v", err)
		}
	})
	mt.Run("Delete/success", func(mt *mtest.T) {
		p := testutil.Appointment()
		r := mongodb.NewAppointmentRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}))
		err := r.Delete(ctx, p.ID)
		t := mt.T
		if err != nil {
			t.Fatal(err)
		}
	})
	mt.Run("Delete/not_found", func(mt *mtest.T) {
		p := testutil.Appointment()
		r := mongodb.NewAppointmentRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 0}))
		err := r.Delete(ctx, p.ID)
		t := mt.T
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("%v", err)
		}
	})
	mt.Run("FindByID/success", func(mt *mtest.T) {
		p := testutil.Appointment()
		r := mongodb.NewAppointmentRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".appointments", mtest.FirstBatch, mongoDoc(mt.T, bson.M{"_id": p.ID.String(), "patient_id": p.PatientID.String(), "doctor_id": p.DoctorID.String(), "status": string(p.Status), "appointment_date": p.Date, "created_at": p.CreatedAt})))
		got, err := r.FindByID(ctx, p.ID)
		t := mt.T
		if err != nil || !reflect.DeepEqual(got, p) {
			t.Fatalf("got=%+v want=%+v err=%v", got, p, err)
		}
	})
	mt.Run("FindByID/empty", func(mt *mtest.T) {
		p := testutil.Appointment()
		r := mongodb.NewAppointmentRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".appointments", mtest.FirstBatch))
		got, err := r.FindByID(ctx, p.ID)
		t := mt.T
		if !errors.Is(err, domain.ErrNotFound) || got != nil {
			t.Fatalf("%v %v", got, err)
		}
	})
	mt.Run("FindAll/success", func(mt *mtest.T) {
		p := testutil.Appointment()
		r := mongodb.NewAppointmentRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".appointments", mtest.FirstBatch, mongoDoc(mt.T, bson.M{"_id": p.ID.String(), "patient_id": p.PatientID.String(), "doctor_id": p.DoctorID.String(), "status": string(p.Status), "appointment_date": p.Date, "created_at": p.CreatedAt})))
		got, err := r.FindAll(ctx)
		t := mt.T
		if err != nil || len(got) != 1 || !reflect.DeepEqual(got[0], p) {
			t.Fatalf("got=%+v want=%+v err=%v", got, p, err)
		}
	})
	mt.Run("FindAll/empty", func(mt *mtest.T) {
		r := mongodb.NewAppointmentRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".appointments", mtest.FirstBatch))
		got, err := r.FindAll(ctx)
		t := mt.T
		if err != nil || len(got) != 0 {
			t.Fatalf("%v %v", got, err)
		}
	})
	mt.Run("FindByPatient/success", func(mt *mtest.T) {
		p := testutil.Appointment()
		r := mongodb.NewAppointmentRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".appointments", mtest.FirstBatch, mongoDoc(mt.T, bson.M{"_id": p.ID.String(), "patient_id": p.PatientID.String(), "doctor_id": p.DoctorID.String(), "status": string(p.Status), "appointment_date": p.Date, "created_at": p.CreatedAt})))
		got, err := r.FindByPatient(ctx, p.PatientID)
		t := mt.T
		if err != nil || len(got) != 1 || !reflect.DeepEqual(got[0], p) {
			t.Fatalf("got=%+v want=%+v err=%v", got, p, err)
		}
	})
	mt.Run("FindByPatient/empty", func(mt *mtest.T) {
		p := testutil.Appointment()
		r := mongodb.NewAppointmentRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".appointments", mtest.FirstBatch))
		got, err := r.FindByPatient(ctx, p.PatientID)
		t := mt.T
		if err != nil || len(got) != 0 {
			t.Fatalf("%v %v", got, err)
		}
	})
	mt.Run("FindActiveByDoctorAndDate/success", func(mt *mtest.T) {
		p := testutil.Appointment()
		r := mongodb.NewAppointmentRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".appointments", mtest.FirstBatch, mongoDoc(mt.T, bson.M{"_id": p.ID.String(), "patient_id": p.PatientID.String(), "doctor_id": p.DoctorID.String(), "status": string(p.Status), "appointment_date": p.Date, "created_at": p.CreatedAt})))
		got, err := r.FindActiveByDoctorAndDate(ctx, p.DoctorID, p.Date)
		t := mt.T
		if err != nil || !reflect.DeepEqual(got, p) {
			t.Fatalf("got=%+v want=%+v err=%v", got, p, err)
		}
	})
	mt.Run("FindActiveByDoctorAndDate/empty", func(mt *mtest.T) {
		p := testutil.Appointment()
		r := mongodb.NewAppointmentRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".appointments", mtest.FirstBatch))
		got, err := r.FindActiveByDoctorAndDate(ctx, p.DoctorID, p.Date)
		t := mt.T
		if !errors.Is(err, domain.ErrNotFound) || got != nil {
			t.Fatalf("%v %v", got, err)
		}
	})
	for _, n := range []int{0, 1} {
		mt.Run(fmtName(n), func(mt *mtest.T) {
			p := testutil.Appointment()
			r := mongodb.NewAppointmentRepository(mt.DB)
			mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".appointments", mtest.FirstBatch, bson.D{{Key: "n", Value: n}}))
			got, err := r.CheckSlotBusy(context.Background(), p.DoctorID, p.Date)
			if err != nil || got != (n > 0) {
				mt.Fatalf("%v %v", got, err)
			}
		})
	}
	mt.Run("CheckSlotBusy/error", func(mt *mtest.T) {
		p := testutil.Appointment()
		r := mongodb.NewAppointmentRepository(mt.DB)
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{Code: 123, Message: "injected"}))
		_, err := r.CheckSlotBusy(context.Background(), p.DoctorID, p.Date)
		assertStorageError(mt.T, err)
	})
}

func fmtName(n int) string {
	if n == 0 {
		return "CheckSlotBusy/free"
	}
	return "CheckSlotBusy/busy"
}

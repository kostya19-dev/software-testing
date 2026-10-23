package mongodb_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"

	"vitaline-clinic/internal/domain"
	"vitaline-clinic/internal/storage/mongodb"
	"vitaline-clinic/lab1/testutil"
)

func mongoDoc(t *testing.T, fields bson.M) bson.D {
	t.Helper()
	raw, err := bson.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	var d bson.D
	if err = bson.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	return d
}

func assertStorageError(t *testing.T, err error) {
	t.Helper()
	var commandError mongo.CommandError
	require.ErrorAs(t, err, &commandError)
	require.EqualValues(t, 123, commandError.Code)
}

func TestPatientMongoRepository(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("Save/success", func(mt *mtest.T) {
		p := testutil.Patient()
		r := mongodb.NewPatientRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}))
		err := r.Save(ctx, p)
		t := mt.T
		if err != nil {
			t.Fatal(err)
		}
	})
	mt.Run("Save/server_error", func(mt *mtest.T) {
		p := testutil.Patient()
		r := mongodb.NewPatientRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{Code: 123, Message: "injected"}))
		err := r.Save(ctx, p)
		t := mt.T
		assertStorageError(t, err)
	})
	mt.Run("Update/success", func(mt *mtest.T) {
		p := testutil.Patient()
		r := mongodb.NewPatientRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}))
		err := r.Update(ctx, p)
		t := mt.T
		if err != nil {
			t.Fatal(err)
		}
	})
	mt.Run("Update/not_found", func(mt *mtest.T) {
		p := testutil.Patient()
		r := mongodb.NewPatientRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 0}))
		err := r.Update(ctx, p)
		t := mt.T
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("%v", err)
		}
	})
	mt.Run("Delete/success", func(mt *mtest.T) {
		p := testutil.Patient()
		r := mongodb.NewPatientRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}))
		err := r.Delete(ctx, p.ID)
		t := mt.T
		if err != nil {
			t.Fatal(err)
		}
	})
	mt.Run("Delete/not_found", func(mt *mtest.T) {
		p := testutil.Patient()
		r := mongodb.NewPatientRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 0}))
		err := r.Delete(ctx, p.ID)
		t := mt.T
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("%v", err)
		}
	})
	mt.Run("FindByID/success", func(mt *mtest.T) {
		p := testutil.Patient()
		r := mongodb.NewPatientRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".patients", mtest.FirstBatch, mongoDoc(mt.T, bson.M{"_id": p.ID.String(), "full_name": p.FullName, "phone": p.Phone, "email": p.Email, "birth_date": p.BirthDate, "created_at": p.CreatedAt})))
		got, err := r.FindByID(ctx, p.ID)
		t := mt.T
		if err != nil || !reflect.DeepEqual(got, p) {
			t.Fatalf("got=%+v want=%+v err=%v", got, p, err)
		}
	})
	mt.Run("FindByID/empty", func(mt *mtest.T) {
		p := testutil.Patient()
		r := mongodb.NewPatientRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".patients", mtest.FirstBatch))
		got, err := r.FindByID(ctx, p.ID)
		t := mt.T
		if !errors.Is(err, domain.ErrNotFound) || got != nil {
			t.Fatalf("%v %v", got, err)
		}
	})
	mt.Run("FindByEmail/success", func(mt *mtest.T) {
		p := testutil.Patient()
		r := mongodb.NewPatientRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".patients", mtest.FirstBatch, mongoDoc(mt.T, bson.M{"_id": p.ID.String(), "full_name": p.FullName, "phone": p.Phone, "email": p.Email, "birth_date": p.BirthDate, "created_at": p.CreatedAt})))
		got, err := r.FindByEmail(ctx, p.Email)
		t := mt.T
		if err != nil || !reflect.DeepEqual(got, p) {
			t.Fatalf("got=%+v want=%+v err=%v", got, p, err)
		}
	})
	mt.Run("FindByEmail/empty", func(mt *mtest.T) {
		p := testutil.Patient()
		r := mongodb.NewPatientRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".patients", mtest.FirstBatch))
		got, err := r.FindByEmail(ctx, p.Email)
		t := mt.T
		if !errors.Is(err, domain.ErrNotFound) || got != nil {
			t.Fatalf("%v %v", got, err)
		}
	})
	mt.Run("FindAll/success", func(mt *mtest.T) {
		p := testutil.Patient()
		r := mongodb.NewPatientRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".patients", mtest.FirstBatch, mongoDoc(mt.T, bson.M{"_id": p.ID.String(), "full_name": p.FullName, "phone": p.Phone, "email": p.Email, "birth_date": p.BirthDate, "created_at": p.CreatedAt})))
		got, err := r.FindAll(ctx)
		t := mt.T
		if err != nil || len(got) != 1 || !reflect.DeepEqual(got[0], p) {
			t.Fatalf("got=%+v want=%+v err=%v", got, p, err)
		}
	})
	mt.Run("FindAll/empty", func(mt *mtest.T) {
		r := mongodb.NewPatientRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".patients", mtest.FirstBatch))
		got, err := r.FindAll(ctx)
		t := mt.T
		if err != nil || len(got) != 0 {
			t.Fatalf("%v %v", got, err)
		}
	})
}

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

func TestVisitMongoRepository(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("Save/success", func(mt *mtest.T) {
		p := testutil.Visit()
		r := mongodb.NewVisitRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}))
		err := r.Save(ctx, p)
		t := mt.T
		if err != nil {
			t.Fatal(err)
		}
	})
	mt.Run("Save/server_error", func(mt *mtest.T) {
		p := testutil.Visit()
		r := mongodb.NewVisitRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{Code: 123, Message: "injected"}))
		err := r.Save(ctx, p)
		t := mt.T
		assertStorageError(t, err)
	})
	mt.Run("Update/success", func(mt *mtest.T) {
		p := testutil.Visit()
		r := mongodb.NewVisitRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}))
		err := r.Update(ctx, p)
		t := mt.T
		if err != nil {
			t.Fatal(err)
		}
	})
	mt.Run("Update/not_found", func(mt *mtest.T) {
		p := testutil.Visit()
		r := mongodb.NewVisitRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 0}))
		err := r.Update(ctx, p)
		t := mt.T
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("%v", err)
		}
	})
	mt.Run("Delete/success", func(mt *mtest.T) {
		p := testutil.Visit()
		r := mongodb.NewVisitRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}))
		err := r.Delete(ctx, p.ID)
		t := mt.T
		if err != nil {
			t.Fatal(err)
		}
	})
	mt.Run("Delete/not_found", func(mt *mtest.T) {
		p := testutil.Visit()
		r := mongodb.NewVisitRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 0}))
		err := r.Delete(ctx, p.ID)
		t := mt.T
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("%v", err)
		}
	})
	mt.Run("FindByID/success", func(mt *mtest.T) {
		p := testutil.Visit()
		r := mongodb.NewVisitRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".visits", mtest.FirstBatch, mongoDoc(mt.T, bson.M{"_id": p.ID.String(), "appointment_id": p.AppointmentID.String(), "complaints": p.Complaints, "diagnosis": p.Diagnosis, "recommendations": p.Recommendations, "status": string(p.Status), "created_at": p.CreatedAt})))
		got, err := r.FindByID(ctx, p.ID)
		t := mt.T
		if err != nil || !reflect.DeepEqual(got, p) {
			t.Fatalf("got=%+v want=%+v err=%v", got, p, err)
		}
	})
	mt.Run("FindByID/empty", func(mt *mtest.T) {
		p := testutil.Visit()
		r := mongodb.NewVisitRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".visits", mtest.FirstBatch))
		got, err := r.FindByID(ctx, p.ID)
		t := mt.T
		if !errors.Is(err, domain.ErrNotFound) || got != nil {
			t.Fatalf("%v %v", got, err)
		}
	})
	mt.Run("FindByAppointment/success", func(mt *mtest.T) {
		p := testutil.Visit()
		r := mongodb.NewVisitRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".visits", mtest.FirstBatch, mongoDoc(mt.T, bson.M{"_id": p.ID.String(), "appointment_id": p.AppointmentID.String(), "complaints": p.Complaints, "diagnosis": p.Diagnosis, "recommendations": p.Recommendations, "status": string(p.Status), "created_at": p.CreatedAt})))
		got, err := r.FindByAppointment(ctx, p.AppointmentID)
		t := mt.T
		if err != nil || !reflect.DeepEqual(got, p) {
			t.Fatalf("got=%+v want=%+v err=%v", got, p, err)
		}
	})
	mt.Run("FindByAppointment/empty", func(mt *mtest.T) {
		p := testutil.Visit()
		r := mongodb.NewVisitRepository(mt.DB)
		ctx := context.Background()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, mt.DB.Name()+".visits", mtest.FirstBatch))
		got, err := r.FindByAppointment(ctx, p.AppointmentID)
		t := mt.T
		if !errors.Is(err, domain.ErrNotFound) || got != nil {
			t.Fatalf("%v %v", got, err)
		}
	})
}

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

func fmtName(n int) string {
	if n == 0 {
		return "CheckSlotBusy/free"
	}
	return "CheckSlotBusy/busy"
}

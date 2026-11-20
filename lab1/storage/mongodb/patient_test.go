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

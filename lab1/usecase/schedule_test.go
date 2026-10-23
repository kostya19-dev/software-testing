package usecase_test

import (
	"context"
	"errors"
	"testing"
	"vitaline-clinic/internal/usecase"
	"vitaline-clinic/lab1/testutil"
)

func TestScheduleServiceLondon(t *testing.T) {
	t.Run("CreateSlot/success", func(t *testing.T) {
		x := newServiceFixture()
		s := usecase.NewScheduleService(x.schedule)
		got, err := s.CreateSlot(context.Background(), scheduleDTO())
		if err != nil {
			t.Fatal(err)
		}
		if got.DoctorID != testutil.DoctorID || got.IsBooked || x.savedSlot == nil || x.savedSlot.IsBooked || x.savedSlot.Cabinet != "101" {
			t.Fatalf("unexpected response: %+v", got)
		}
	})
	t.Run("CreateSlot/schedule.Save", func(t *testing.T) {
		x := newServiceFixture()
		x.fail = "schedule.Save"
		s := usecase.NewScheduleService(x.schedule)
		_, err := s.CreateSlot(context.Background(), scheduleDTO())
		if !errors.Is(err, storageFailure) {
			t.Fatalf("expected original error: %v", err)
		}
	})
	t.Run("GetByDoctor/success", func(t *testing.T) {
		x := newServiceFixture()
		s := usecase.NewScheduleService(x.schedule)
		got, err := s.GetByDoctor(context.Background(), testutil.DoctorID)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].ID != testutil.ID {
			t.Fatalf("unexpected response: %+v", got)
		}
	})
	t.Run("GetByDoctor/schedule.FindByDoctorID", func(t *testing.T) {
		x := newServiceFixture()
		x.fail = "schedule.FindByDoctorID"
		s := usecase.NewScheduleService(x.schedule)
		_, err := s.GetByDoctor(context.Background(), testutil.DoctorID)
		if !errors.Is(err, storageFailure) {
			t.Fatalf("expected original error: %v", err)
		}
	})
	t.Run("GetFreeSlots/success", func(t *testing.T) {
		x := newServiceFixture()
		s := usecase.NewScheduleService(x.schedule)
		got, err := s.GetFreeSlots(context.Background(), testutil.DoctorID, testutil.Date)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].IsBooked {
			t.Fatalf("unexpected response: %+v", got)
		}
	})
	t.Run("GetFreeSlots/schedule.FindFreeSlots", func(t *testing.T) {
		x := newServiceFixture()
		x.fail = "schedule.FindFreeSlots"
		s := usecase.NewScheduleService(x.schedule)
		_, err := s.GetFreeSlots(context.Background(), testutil.DoctorID, testutil.Date)
		if !errors.Is(err, storageFailure) {
			t.Fatalf("expected original error: %v", err)
		}
	})
	t.Run("DeleteSlot/success", func(t *testing.T) {
		x := newServiceFixture()
		s := usecase.NewScheduleService(x.schedule)
		err := s.DeleteSlot(context.Background(), testutil.ID)
		if err != nil {
			t.Fatal(err)
		}

		if x.deletedSlotID != testutil.ID {
			t.Fatal("requested record was not deleted")
		}
	})
	t.Run("DeleteSlot/schedule.Delete", func(t *testing.T) {
		x := newServiceFixture()
		x.fail = "schedule.Delete"
		s := usecase.NewScheduleService(x.schedule)
		err := s.DeleteSlot(context.Background(), testutil.ID)
		if !errors.Is(err, storageFailure) {
			t.Fatalf("expected original error: %v", err)
		}
	})
}

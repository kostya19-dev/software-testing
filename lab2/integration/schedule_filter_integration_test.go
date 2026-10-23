package integration_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
	storage "vitaline-clinic/internal/storage/mongodb"
)

func TestScheduleFreeSlotsFilter(t *testing.T) {
	store, ctx := database(t)
	repo := storage.NewScheduleRepository(store.Database())
	doctor, otherDoctor := uuid.New(), uuid.New()
	day := tomorrow().Truncate(24 * time.Hour)
	slots := []*domain.ScheduleSlot{
		{ID: uuid.New(), DoctorID: doctor, Date: day, StartTime: day.Add(9 * time.Hour), EndTime: day.Add(10 * time.Hour), Cabinet: "101"},
		{ID: uuid.New(), DoctorID: otherDoctor, Date: day, StartTime: day.Add(9 * time.Hour), EndTime: day.Add(10 * time.Hour), Cabinet: "102"},
		{ID: uuid.New(), DoctorID: doctor, Date: day.Add(24 * time.Hour), StartTime: day.Add(33 * time.Hour), EndTime: day.Add(34 * time.Hour), Cabinet: "101"},
		{ID: uuid.New(), DoctorID: doctor, Date: day, StartTime: day.Add(10 * time.Hour), EndTime: day.Add(11 * time.Hour), Cabinet: "101", IsBooked: true},
	}
	for _, slot := range slots {
		must(t, repo.Save(ctx, slot))
	}

	found, err := repo.FindFreeSlots(ctx, doctor, day)
	must(t, err)
	if len(found) != 1 || found[0].ID != slots[0].ID {
		t.Fatalf("expected only the matching free slot, got %+v", found)
	}

	slots[0].IsBooked = true
	must(t, repo.Update(ctx, slots[0]))
	found, err = repo.FindFreeSlots(ctx, doctor, day)
	must(t, err)
	if len(found) != 0 {
		t.Fatal("a booked slot is still offered as free")
	}
}

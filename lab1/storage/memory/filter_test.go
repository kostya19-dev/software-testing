package memory_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
	"vitaline-clinic/internal/storage/memory"
	"vitaline-clinic/lab1/testutil"
)

func TestScheduleFreeSlotsFilter(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewScheduleRepository()
	matching := testutil.Schedule()
	otherDoctor := testutil.Schedule()
	otherDoctor.ID = uuid.MustParse("44444444-4444-4444-4444-444444444444")
	otherDoctor.DoctorID = testutil.PatientID
	otherDate := testutil.Schedule()
	otherDate.ID = uuid.MustParse("55555555-5555-5555-5555-555555555555")
	otherDate.Date = testutil.Date.AddDate(0, 0, 1)
	booked := testutil.Schedule()
	booked.ID = uuid.MustParse("66666666-6666-6666-6666-666666666666")
	booked.IsBooked = true
	for _, slot := range []*domain.ScheduleSlot{matching, otherDoctor, otherDate, booked} {
		mustSave(t, repo.Save(ctx, slot))
	}

	got, err := repo.FindFreeSlots(ctx, testutil.DoctorID, testutil.Date)

	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] == nil || got[0].ID != matching.ID {
		t.Fatalf("expected only available slot %s, got %+v", matching.ID, got)
	}
}

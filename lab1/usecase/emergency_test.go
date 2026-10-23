package usecase_test

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"testing"
	"time"
	"vitaline-clinic/internal/domain"
	"vitaline-clinic/internal/usecase"
	"vitaline-clinic/lab1/testutil"
)

func TestEmergencyServiceLondon(t *testing.T) {
	t.Run("Create/success", func(t *testing.T) {
		x := newServiceFixture()
		s := usecase.NewEmergencyService(x.emergency, x.appointment)
		got, err := s.Create(context.Background(), emergencyDTO())
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != domain.EmergencyStatusWaitingResponse || x.savedRequest == nil || x.savedRequest.Status != domain.EmergencyStatusWaitingResponse || x.savedOffer == nil || x.savedOffer.Status != domain.OfferStatusPending || !x.savedOffer.ProposedDate.Equal(testutil.Date.Add(30*time.Minute)) {
			t.Fatalf("unexpected response: %+v", got)
		}
	})
	t.Run("Create/emergency.SaveOffer", func(t *testing.T) {
		x := newServiceFixture()
		x.fail = "emergency.SaveOffer"
		s := usecase.NewEmergencyService(x.emergency, x.appointment)
		_, err := s.Create(context.Background(), emergencyDTO())
		if !errors.Is(err, storageFailure) {
			t.Fatalf("expected original error: %v", err)
		}
	})
	t.Run("GetRequests/success", func(t *testing.T) {
		x := newServiceFixture()
		s := usecase.NewEmergencyService(x.emergency, x.appointment)
		got, err := s.GetRequests(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].PatientID != testutil.PatientID {
			t.Fatalf("unexpected response: %+v", got)
		}
	})
	t.Run("GetRequests/emergency.FindAllRequests", func(t *testing.T) {
		x := newServiceFixture()
		x.fail = "emergency.FindAllRequests"
		s := usecase.NewEmergencyService(x.emergency, x.appointment)
		_, err := s.GetRequests(context.Background())
		if !errors.Is(err, storageFailure) {
			t.Fatalf("expected original error: %v", err)
		}
	})
	t.Run("GetOffers/success", func(t *testing.T) {
		x := newServiceFixture()
		s := usecase.NewEmergencyService(x.emergency, x.appointment)
		got, err := s.GetOffers(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].Status != domain.OfferStatusPending {
			t.Fatalf("unexpected response: %+v", got)
		}
	})
	t.Run("GetOffers/emergency.FindAllOffers", func(t *testing.T) {
		x := newServiceFixture()
		x.fail = "emergency.FindAllOffers"
		s := usecase.NewEmergencyService(x.emergency, x.appointment)
		_, err := s.GetOffers(context.Background())
		if !errors.Is(err, storageFailure) {
			t.Fatalf("expected original error: %v", err)
		}
	})
	t.Run("AcceptOffer/success", func(t *testing.T) {
		x := newServiceFixture()
		s := usecase.NewEmergencyService(x.emergency, x.appointment)
		got, err := s.AcceptOffer(context.Background(), testutil.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != domain.OfferStatusAccepted || x.savedRequest == nil || x.savedRequest.Status != domain.EmergencyStatusScheduled || x.savedOffer == nil || x.savedOffer.Status != domain.OfferStatusAccepted {
			t.Fatalf("unexpected response: %+v", got)
		}
	})
	t.Run("AcceptOffer/emergency.UpdateOffer", func(t *testing.T) {
		x := newServiceFixture()
		x.fail = "emergency.UpdateOffer"
		s := usecase.NewEmergencyService(x.emergency, x.appointment)
		_, err := s.AcceptOffer(context.Background(), testutil.ID)
		if !errors.Is(err, storageFailure) {
			t.Fatalf("expected original error: %v", err)
		}
	})
	t.Run("DeclineOffer/success", func(t *testing.T) {
		x := newServiceFixture()
		s := usecase.NewEmergencyService(x.emergency, x.appointment)
		got, err := s.DeclineOffer(context.Background(), testutil.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != domain.OfferStatusDeclined || x.savedOffer == nil || x.savedOffer.Status != domain.OfferStatusDeclined || x.savedRequest == nil || x.savedRequest.Status != domain.EmergencyStatusNeedsReview {
			t.Fatalf("unexpected response: %+v", got)
		}
	})
	t.Run("DeclineOffer/emergency.UpdateRequest", func(t *testing.T) {
		x := newServiceFixture()
		x.fail = "emergency.UpdateRequest"
		s := usecase.NewEmergencyService(x.emergency, x.appointment)
		_, err := s.DeclineOffer(context.Background(), testutil.ID)
		if !errors.Is(err, storageFailure) {
			t.Fatalf("expected original error: %v", err)
		}
	})
}

func TestEmergencyFreeSlot(t *testing.T) {
	for _, fail := range []string{"", "emergency.SaveRequest", "appointment.Save"} {
		name := fail
		if name == "" {
			name = "success"
		}
		t.Run(name, func(t *testing.T) {
			x := newServiceFixture()
			x.fail = fail
			x.appointment.FindActiveByDoctorAndDateFunc = func(context.Context, uuid.UUID, time.Time) (*domain.Appointment, error) {
				return nil, domain.ErrNotFound
			}
			s := usecase.NewEmergencyService(x.emergency, x.appointment)
			got, err := s.Create(context.Background(), emergencyDTO())
			if fail != "" {
				if !errors.Is(err, storageFailure) {
					t.Fatalf("%v", err)
				}
				return
			}
			if err != nil || got.Status != domain.EmergencyStatusScheduled || x.savedRequest == nil || x.savedRequest.Status != domain.EmergencyStatusScheduled || x.savedAppointment == nil || x.savedAppointment.PatientID != testutil.PatientID {
				t.Fatalf("%+v %v", got, err)
			}
		})
	}
}

func TestEmergencyProposedDateBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name                            string
		hour, minute, wantHour, wantDay int
		busy                            bool
	}{
		{"before_open", 7, 0, 9, 10, false}, {"closing", 16, 30, 9, 11, false}, {"no_free_slot", 10, 0, 0, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			x := newServiceFixture()
			x.appointment.CheckSlotBusyFunc = func(context.Context, uuid.UUID, time.Time) (bool, error) { return tc.busy, nil }
			s := usecase.NewEmergencyService(x.emergency, x.appointment)
			dto := emergencyDTO()
			dto.RequestedDate = time.Date(2099, 5, 10, tc.hour, tc.minute, 0, 0, time.UTC)
			_, err := s.Create(context.Background(), dto)
			if tc.busy {
				if !errors.Is(err, domain.ErrValidation) {
					t.Fatalf("%v", err)
				}
				if x.savedOffer != nil {
					t.Fatal("saved invalid offer")
				}
				return
			}
			if err != nil || x.savedOffer == nil || x.savedOffer.ProposedDate.Hour() != tc.wantHour || x.savedOffer.ProposedDate.Day() != tc.wantDay {
				t.Fatalf("%+v %v", x.savedOffer, err)
			}
		})
	}
}

func TestEmergencyOfferRejection(t *testing.T) {
	for _, method := range []string{"accept", "decline"} {
		for _, state := range []domain.RescheduleOfferStatus{domain.OfferStatusAccepted, domain.OfferStatusDeclined} {
			t.Run(method+string(state), func(t *testing.T) {
				x := newServiceFixture()
				x.emergency.FindOfferByIDFunc = func(context.Context, uuid.UUID) (*domain.RescheduleOffer, error) {
					o := testutil.RescheduleOffer()
					o.Status = state
					return o, nil
				}
				s := usecase.NewEmergencyService(x.emergency, x.appointment)
				act := s.AcceptOffer
				if method == "decline" {
					act = s.DeclineOffer
				}
				_, err := act(context.Background(), testutil.ID)
				if !errors.Is(err, domain.ErrInvalidStatus) || x.savedOffer != nil || x.savedAppointment != nil {
					t.Fatalf("%v", err)
				}
			})
		}
	}
}
